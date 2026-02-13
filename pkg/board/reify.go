package board

import (
	"fmt"
	"regexp"
	"strings"

	"cuelang.org/go/cue"
)

// ReifyBoard transforms a loaded Board into a compact JSON-serializable map.
func ReifyBoard(b *Board) map[string]any {
	flow := make([]any, 0, len(b.Flow))
	for _, item := range b.Flow {
		flow = append(flow, reifyInstant(item))
	}
	return map[string]any{
		"name": b.Name,
		"flow": flow,
	}
}

// BoardManifest is the top-level manifest written to board.json.
type BoardManifest struct {
	Name   string      `json:"name"`
	Flow   []FlowEntry `json:"flow"`
	Errors []string    `json:"errors,omitempty"`
}

// FlowEntry is one entry in the manifest's flow table of contents.
type FlowEntry struct {
	Index       int            `json:"index"`
	Kind        string         `json:"kind"`
	Type        string         `json:"type,omitempty"`
	Name        string         `json:"name"`
	File        string         `json:"file,omitempty"`
	SliceRef    string         `json:"sliceRef,omitempty"`
	Description string         `json:"description,omitempty"`
	Instance    map[string]any `json:"instance,omitempty"`
}

// ReifyBoardFiles splits a board into a manifest + per-slice data maps.
// Stories are inline in the manifest only (no separate file).
// Returns manifest, slice data, and list of image paths to copy.
func ReifyBoardFiles(b *Board, errors []string) (BoardManifest, map[string]map[string]any, []string) {
	manifest := BoardManifest{
		Name:   b.Name,
		Errors: errors,
	}
	slices := make(map[string]map[string]any)
	seen := make(map[string]int) // for dedup filenames
	var images []string

	for i, item := range b.Flow {
		entry := FlowEntry{
			Index: i,
			Kind:  item.Kind,
			Type:  item.Type,
			Name:  item.Name,
		}

		switch item.Kind {
		case "slice":
			data := reifyInstant(item)
			filename := sanitizeFilename(item.Name, seen) + ".json"
			entry.File = filename
			slices[filename] = data
			// Collect image if present
			if img, ok := data["image"].(string); ok && img != "" {
				images = append(images, img)
			}
		case "story":
			entry.SliceRef = item.SliceRef
			storyData := reifyStory(item.CUEValue)
			if desc, ok := storyData["description"].(string); ok {
				entry.Description = desc
			}
			if inst, ok := storyData["instance"].(map[string]any); ok {
				entry.Instance = inst
			}
		}

		manifest.Flow = append(manifest.Flow, entry)
	}

	return manifest, slices, images
}

var nonAlnum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// sanitizeFilename converts a name to a safe filename, deduplicating collisions.
func sanitizeFilename(name string, seen map[string]int) string {
	base := nonAlnum.ReplaceAllString(name, "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "unnamed"
	}
	seen[base]++
	if seen[base] > 1 {
		base = fmt.Sprintf("%s_%d", base, seen[base])
	}
	return base
}

func reifyInstant(item FlowItem) map[string]any {
	switch item.Kind {
	case "slice":
		switch item.Type {
		case "change":
			return reifyChangeSlice(item.CUEValue)
		case "view":
			return reifyViewSlice(item.CUEValue)
		}
	case "story":
		return reifyStory(item.CUEValue)
	}
	return map[string]any{"kind": item.Kind, "name": item.Name}
}

func reifyChangeSlice(v cue.Value) map[string]any {
	sliceName := getString(v, "name")
	out := map[string]any{
		"kind":      "slice",
		"type":      "change",
		"name":      sliceName,
		"actor":     getString(v, "actor.name"),
		"trigger":   reifyTrigger(v.LookupPath(cue.ParsePath("trigger"))),
		"command":   reifyCommand(v.LookupPath(cue.ParsePath("command"))),
		"emits":     reifyEmits(v.LookupPath(cue.ParsePath("emits"))),
		"scenarios": reifyGWTScenarios(v.LookupPath(cue.ParsePath("scenarios")), sliceName),
	}
	if img := getString(v, "image"); img != "" {
		out["image"] = img
	}
	if ds := getString(v, "devstatus"); ds != "" {
		out["devstatus"] = ds
	}
	return out
}

func reifyViewSlice(v cue.Value) map[string]any {
	out := map[string]any{
		"kind":      "slice",
		"type":      "view",
		"name":      getString(v, "name"),
		"actor":     getString(v, "actor.name"),
		"endpoint":  reifyEndpoint(v.LookupPath(cue.ParsePath("endpoint"))),
		"query":     reifyQueryItems(v.LookupPath(cue.ParsePath("query.items"))),
		"readModel": reifyReadModel(v.LookupPath(cue.ParsePath("readModel"))),
		"scenarios": reifyViewScenarios(v.LookupPath(cue.ParsePath("scenarios"))),
	}
	if img := getString(v, "image"); img != "" {
		out["image"] = img
	}
	if ds := getString(v, "devstatus"); ds != "" {
		out["devstatus"] = ds
	}
	return out
}

func reifyStory(v cue.Value) map[string]any {
	out := map[string]any{
		"kind":     "story",
		"name":     getString(v, "name"),
		"sliceRef": getString(v, "slice.name"),
	}
	if desc := getString(v, "description"); desc != "" {
		out["description"] = desc
	}
	if inst := v.LookupPath(cue.ParsePath("instance")); inst.Exists() && inst.Err() == nil {
		if cv, ok := reifyConcreteValue(inst).(map[string]any); ok && len(cv) > 0 {
			out["instance"] = cv
		}
	}
	return out
}

func reifyTrigger(v cue.Value) map[string]any {
	kind := getString(v, "kind")
	out := map[string]any{"kind": kind}

	if kind == "endpoint" {
		out["endpoint"] = reifyEndpoint(v.LookupPath(cue.ParsePath("endpoint")))
	} else if kind == "externalEvent" {
		out["externalEvent"] = reifyExternalEvent(v.LookupPath(cue.ParsePath("externalEvent")))
	}
	return out
}

func reifyExternalEvent(v cue.Value) map[string]any {
	return map[string]any{
		"name":   getString(v, "name"),
		"fields": reifyFields(v.LookupPath(cue.ParsePath("fields"))),
	}
}

func reifyEndpoint(v cue.Value) map[string]any {
	out := map[string]any{
		"verb": getString(v, "verb"),
		"path": getString(v, "path"),
	}
	if params := reifyFields(v.LookupPath(cue.ParsePath("params"))); len(params) > 0 {
		out["params"] = params
	}
	if body := reifyFields(v.LookupPath(cue.ParsePath("body"))); len(body) > 0 {
		out["body"] = body
	}
	return out
}

func reifyCommand(v cue.Value) map[string]any {
	out := map[string]any{}
	if fields := reifyFields(v.LookupPath(cue.ParsePath("fields"))); len(fields) > 0 {
		out["fields"] = fields
	}
	if mapping := reifyCommandMapping(v.LookupPath(cue.ParsePath("mapping"))); len(mapping) > 0 {
		out["mapping"] = mapping
	}
	if query := reifyQueryItems(v.LookupPath(cue.ParsePath("query.items"))); len(query) > 0 {
		out["query"] = query
	}
	if comp := reifyCommandComputed(v.LookupPath(cue.ParsePath("computed"))); len(comp) > 0 {
		out["computed"] = comp
	}
	return out
}

// reifyCommandComputed extracts command computed fields: {fieldName: {type, description}}
func reifyCommandComputed(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		fv := iter.Value()
		item := map[string]any{}
		// Extract type
		if typeVal := fv.LookupPath(cue.ParsePath("type")); typeVal.Exists() {
			item["type"] = reifyFieldType(typeVal)
		}
		// Extract description (concrete string)
		if desc := getString(fv, "description"); desc != "" {
			item["description"] = desc
		}
		if len(item) > 0 {
			out[label] = item
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// reifyCommandMapping extracts command-level mapping: cmdField -> source path string
func reifyCommandMapping(v cue.Value) map[string]string {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		// Extract the path reference as string (e.g. "trigger.endpoint.body.image")
		val := iter.Value()
		// Try to get a readable path from the CUE reference
		path := formatCUEPath(val)
		if path != "" {
			out[label] = path
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// formatCUEPath extracts a readable path string from a CUE value reference
func formatCUEPath(v cue.Value) string {
	// Try direct ReferencePath first
	_, path := v.ReferencePath()
	if len(path.Selectors()) > 0 {
		return cleanMappingPath(path.String())
	}
	// For unified values, use Expr to get operands and find the reference
	_, args := v.Expr()
	for _, arg := range args {
		_, p := arg.ReferencePath()
		if len(p.Selectors()) > 0 {
			return cleanMappingPath(p.String())
		}
	}
	return ""
}

// cleanMappingPath extracts the relative path (trigger.*, command.*, etc.)
func cleanMappingPath(fullPath string) string {
	// Look for common prefixes and extract relative part
	prefixes := []string{"trigger.", "command.", "fields."}
	for _, prefix := range prefixes {
		if idx := strings.Index(fullPath, prefix); idx >= 0 {
			return fullPath[idx:]
		}
	}
	return fullPath
}

func reifyQueryItems(v cue.Value) []any {
	iter, err := v.List()
	if err != nil {
		return nil
	}
	var items []any
	for iter.Next() {
		items = append(items, reifyQueryItem(iter.Value()))
	}
	return items
}

func reifyQueryItem(v cue.Value) map[string]any {
	// types: extract event type names
	var types []string
	typesVal := v.LookupPath(cue.ParsePath("types"))
	if iter, err := typesVal.List(); err == nil {
		for iter.Next() {
			if et := getString(iter.Value(), "eventType"); et != "" {
				types = append(types, et)
			}
		}
	}

	// tags
	var tags []any
	tagsVal := v.LookupPath(cue.ParsePath("tags"))
	if iter, err := tagsVal.List(); err == nil {
		for iter.Next() {
			tv := iter.Value()
			tag := map[string]any{}
			// Could be a bare #Tag or a #TagRef {tag: #Tag, value: ...}
			tagField := tv.LookupPath(cue.ParsePath("tag"))
			if tagField.Exists() && tagField.Err() == nil {
				// TagRef form
				tag["tag"] = getString(tagField, "name")
				if param := getString(tagField, "param"); param != "" {
					tag["param"] = param
				}
			} else {
				// Bare tag
				tag["tag"] = getString(tv, "name")
				if param := getString(tv, "param"); param != "" {
					tag["param"] = param
				}
			}
			tags = append(tags, tag)
		}
	}

	if tags == nil {
		tags = []any{}
	}
	return map[string]any{
		"types": types,
		"tags":  tags,
	}
}

func reifyEmits(v cue.Value) []any {
	iter, err := v.List()
	if err != nil {
		return nil
	}
	var emits []any
	for iter.Next() {
		ev := iter.Value()
		item := map[string]any{
			"type":   getString(ev, "eventType"),
			"fields": reifyFields(ev.LookupPath(cue.ParsePath("fields"))),
		}

		// tags as string names
		var tagNames []string
		tagsVal := ev.LookupPath(cue.ParsePath("tags"))
		if ti, err := tagsVal.List(); err == nil {
			for ti.Next() {
				if n := getString(ti.Value(), "name"); n != "" {
					tagNames = append(tagNames, n)
				}
			}
		}
		item["tags"] = tagNames

		// mapping (optional, skip if empty)
		if mapping := reifyMappingEmit(ev.LookupPath(cue.ParsePath("mapping"))); len(mapping) > 0 {
			item["mapping"] = mapping
		}

		emits = append(emits, item)
	}
	return emits
}

// reifyMappingEmit extracts emit-level mapping: eventField -> commandFieldExpr
func reifyMappingEmit(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		// The mapping value is typically a reference to a command field
		// Try to extract as concrete string; if not, render the type
		val := iter.Value()
		if val.IsConcrete() {
			if s, err := val.String(); err == nil {
				out[label] = s
				continue
			}
		}
		// Non-concrete: just note the type
		out[label] = reifyFieldType(val)
	}
	return out
}

func reifyGWTScenarios(v cue.Value, sliceName string) []any {
	iter, err := v.List()
	if err != nil {
		return nil
	}
	var scenarios []any
	for iter.Next() {
		sv := iter.Value()
		s := map[string]any{
			"name":  getString(sv, "name"),
			"given": reifyEventInstances(sv.LookupPath(cue.ParsePath("given"))),
			"when":  reifyWhen(sv.LookupPath(cue.ParsePath("when")), sliceName),
			"then":  reifyOutcome(sv.LookupPath(cue.ParsePath("then"))),
		}
		scenarios = append(scenarios, s)
	}
	return scenarios
}

func reifyViewScenarios(v cue.Value) []any {
	iter, err := v.List()
	if err != nil {
		return nil
	}
	var scenarios []any
	for iter.Next() {
		sv := iter.Value()
		s := map[string]any{
			"name":   getString(sv, "name"),
			"given":  reifyEventInstances(sv.LookupPath(cue.ParsePath("given"))),
			"query":  reifyConcreteValue(sv.LookupPath(cue.ParsePath("query"))),
			"expect": reifyConcreteValue(sv.LookupPath(cue.ParsePath("expect"))),
		}
		scenarios = append(scenarios, s)
	}
	return scenarios
}

func reifyEventInstances(v cue.Value) []any {
	iter, err := v.List()
	if err != nil {
		return nil
	}
	var out []any
	for iter.Next() {
		out = append(out, reifyEventInstance(iter.Value()))
	}
	if out == nil {
		return []any{}
	}
	return out
}

func reifyEventInstance(v cue.Value) any {
	et := getString(v, "eventType")
	if et == "" {
		return reifyConcreteValue(v)
	}

	// Extract concrete field values (if any)
	fieldsVal := v.LookupPath(cue.ParsePath("fields"))
	var concreteFields map[string]any
	if fieldsVal.Exists() && fieldsVal.Err() == nil {
		concreteFields = extractConcreteFields(fieldsVal)
	}

	// Check for fromFuture flag
	fromFuture := false
	ff := v.LookupPath(cue.ParsePath("fromFuture"))
	if ff.Exists() {
		if b, err := ff.Bool(); err == nil && b {
			fromFuture = true
		}
	}

	// If no concrete values and no fromFuture, return bare event type
	if len(concreteFields) == 0 && !fromFuture {
		return et
	}

	// Return full form with values
	item := map[string]any{"type": et}
	if len(concreteFields) > 0 {
		item["values"] = concreteFields
	}
	if fromFuture {
		item["fromFuture"] = true
	}
	return item
}

// extractConcreteFields extracts only concrete (non-type) values from fields
func extractConcreteFields(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		fv := iter.Value()
		// Only include if the value is concrete
		if fv.IsConcrete() {
			if cv := reifyConcreteValue(fv); cv != nil {
				out[label] = cv
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func reifyWhen(v cue.Value, sliceName string) map[string]any {
	out := map[string]any{
		"command": sliceName,
	}
	// when is now just #Field (the values directly)
	cv := reifyConcreteValue(v)
	if m, ok := cv.(map[string]any); ok && len(m) > 0 {
		out["values"] = m
	}
	return out
}

func reifyOutcome(v cue.Value) map[string]any {
	successVal := v.LookupPath(cue.ParsePath("success"))
	success, _ := successVal.Bool()
	out := map[string]any{"success": success}
	if success {
		out["events"] = reifyEventInstances(v.LookupPath(cue.ParsePath("events")))
	} else {
		if errStr := getString(v, "error"); errStr != "" {
			out["error"] = errStr
		}
	}
	return out
}

func reifyReadModel(v cue.Value) map[string]any {
	out := map[string]any{
		"name":        getString(v, "name"),
		"cardinality": getString(v, "cardinality"),
		"fields":      reifyFieldsDeep(v.LookupPath(cue.ParsePath("fields"))),
	}

	// mapping: field -> "Event.field"
	if mapping := reifyReadModelMapping(v.LookupPath(cue.ParsePath("mapping"))); len(mapping) > 0 {
		out["mapping"] = mapping
	}

	// computed
	if computed := reifyReadModelComputed(v.LookupPath(cue.ParsePath("computed"))); len(computed) > 0 {
		out["computed"] = computed
	}

	return out
}

func reifyReadModelMapping(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		mv := iter.Value()
		eventType := getString(mv, "event.eventType")
		field := getString(mv, "field")
		out[label] = eventType + "." + field
	}
	return out
}

func reifyReadModelComputed(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		cv := iter.Value()
		eventType := getString(cv, "event.eventType")
		var fields []string
		fieldsVal := cv.LookupPath(cue.ParsePath("fields"))
		if fi, err := fieldsVal.List(); err == nil {
			for fi.Next() {
				if s, err := fi.Value().String(); err == nil {
					fields = append(fields, s)
				}
			}
		}
		out[label] = map[string]any{
			"event":  eventType,
			"fields": fields,
		}
	}
	return out
}

// selectorLabel returns the unquoted field name from a CUE selector.
// CUE quotes keys containing dots, e.g. `"items.price"` → `items.price`.
func selectorLabel(sel cue.Selector) string {
	s := sel.String()
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return s[1 : len(s)-1]
	}
	return s
}

// reifyFields extracts struct fields as {"name": "type"} (flat, type names only).
func reifyFields(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		out[label] = reifyFieldType(iter.Value())
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// reifyFieldsDeep extracts struct fields preserving nested structs and arrays.
func reifyFieldsDeep(v cue.Value) map[string]any {
	if !v.Exists() || v.Err() != nil {
		return nil
	}
	iter, err := v.Fields(cue.Optional(true))
	if err != nil {
		return nil
	}
	out := map[string]any{}
	for iter.Next() {
		label := selectorLabel(iter.Selector())
		if len(label) > 0 && label[0] == '_' {
			continue
		}
		out[label] = reifyFieldTypeDeep(iter.Value())
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// reifyFieldType returns "string", "int", "float", "bool" for scalars.
func reifyFieldType(v cue.Value) any {
	switch v.IncompleteKind() {
	case cue.StringKind:
		return "string"
	case cue.IntKind:
		return "int"
	case cue.FloatKind, cue.NumberKind:
		return "float"
	case cue.BoolKind:
		return "bool"
	case cue.StructKind:
		return reifyFields(v)
	case cue.ListKind:
		return reifyListType(v)
	default:
		return v.IncompleteKind().String()
	}
}

// reifyFieldTypeDeep like reifyFieldType but recurses into nested structs.
func reifyFieldTypeDeep(v cue.Value) any {
	switch v.IncompleteKind() {
	case cue.StringKind:
		return "string"
	case cue.IntKind:
		return "int"
	case cue.FloatKind, cue.NumberKind:
		return "float"
	case cue.BoolKind:
		return "bool"
	case cue.StructKind:
		return reifyFieldsDeep(v)
	case cue.ListKind:
		return reifyListTypeDeep(v)
	default:
		return v.IncompleteKind().String()
	}
}

func reifyListType(v cue.Value) any {
	if v.Allows(cue.AnyIndex) {
		elem := v.LookupPath(cue.MakePath(cue.AnyIndex))
		if elem.Exists() {
			return []any{reifyFieldType(elem)}
		}
	}
	return []any{}
}

func reifyListTypeDeep(v cue.Value) any {
	if v.Allows(cue.AnyIndex) {
		elem := v.LookupPath(cue.MakePath(cue.AnyIndex))
		if elem.Exists() {
			return []any{reifyFieldTypeDeep(elem)}
		}
	}
	return []any{}
}

// reifyConcreteValue extracts concrete JSON values from CUE (for scenario data).
func reifyConcreteValue(v cue.Value) any {
	if !v.Exists() {
		return nil
	}

	switch v.IncompleteKind() {
	case cue.StringKind:
		if s, err := v.String(); err == nil {
			return s
		}
		return nil
	case cue.IntKind:
		if n, err := v.Int64(); err == nil {
			return n
		}
		return nil
	case cue.FloatKind, cue.NumberKind:
		if f, err := v.Float64(); err == nil {
			return f
		}
		return nil
	case cue.BoolKind:
		if b, err := v.Bool(); err == nil {
			return b
		}
		return nil
	case cue.NullKind:
		return nil
	case cue.StructKind:
		iter, err := v.Fields(cue.Optional(true))
		if err != nil {
			return nil
		}
		out := map[string]any{}
		for iter.Next() {
			label := selectorLabel(iter.Selector())
			if len(label) > 0 && label[0] == '_' {
				continue
			}
			if val := reifyConcreteValue(iter.Value()); val != nil {
				out[label] = val
			}
		}
		return out
	case cue.ListKind:
		iter, err := v.List()
		if err != nil {
			return nil
		}
		var out []any
		for iter.Next() {
			out = append(out, reifyConcreteValue(iter.Value()))
		}
		if out == nil {
			return []any{}
		}
		return out
	}
	return nil
}

