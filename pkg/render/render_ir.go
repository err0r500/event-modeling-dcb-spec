package render

import (
	"fmt"
	"strings"
)

// RenderSliceIR renders a slice from its IR (map[string]any) as ASCII box art.
func RenderSliceIR(data map[string]any, width int) (string, error) {
	kind := getStr(data, "kind")
	switch kind {
	case "slice":
		sliceType := getStr(data, "type")
		if sliceType == "view" {
			return renderViewSliceIR(data, width)
		}
		return renderChangeSliceIR(data, width)
	case "story":
		return renderStoryIR(data, width)
	default:
		return "", fmt.Errorf("unknown kind: %s", kind)
	}
}

func renderChangeSliceIR(data map[string]any, width int) (string, error) {
	box := NewBox(width)

	name := getStr(data, "name")
	sliceType := getStr(data, "type")
	actor := getStr(data, "actor")
	box.AddLine(fmt.Sprintf("  SLICE: %s (%s)  │  Actor: %s", name, sliceType, actor))

	if img := getStr(data, "image"); img != "" {
		box.AddLine(fmt.Sprintf("  📷 %s", img))
		if ascii := renderImageASCII(img, width); ascii != "" {
			box.AddLine("")
			for line := range strings.SplitSeq(ascii, "\n") {
				box.AddLine("  " + line)
			}
			box.AddLine("")

		}
	}

	// Trigger
	trigger := getMap(data, "trigger")
	box.AddSection()
	triggerKind := getStr(trigger, "kind")
	if triggerKind == "endpoint" {
		ep := getMap(trigger, "endpoint")
		box.AddLine(fmt.Sprintf("  %s %s", getStr(ep, "verb"), getStr(ep, "path")))

		if body := getMap(ep, "body"); len(body) > 0 {
			box.AddLine("    body:")
			for k, v := range body {
				box.AddLine(fmt.Sprintf("      - %s: %s", k, irTypeStr(v)))
			}
		}
	} else if triggerKind == "externalEvent" {
		ext := getMap(trigger, "externalEvent")
		box.AddLine(fmt.Sprintf("  External Event: %s", getStr(ext, "name")))

		if fields := getMap(ext, "fields"); len(fields) > 0 {
			box.AddLine("    fields:")
			for k, v := range fields {
				box.AddLine(fmt.Sprintf("      - %s: %s", k, irTypeStr(v)))
			}
		}
	}

	// Command
	cmd := getMap(data, "command")
	box.AddSection()
	box.AddLine(fmt.Sprintf("  Command: %s", name))
	box.AddLine("    fields:")
	if fields := getMap(cmd, "fields"); len(fields) > 0 {
		for k, v := range fields {
			box.AddLine(fmt.Sprintf("      - %s: %s", k, irTypeStr(v)))
		}
	}

	// Command mapping
	if mapping := getMapStr(cmd, "mapping"); len(mapping) > 0 {
		box.AddLine("    mapping:")
		for k, v := range mapping {
			box.AddLine(fmt.Sprintf("      - %s ← %s", k, v))
		}
	}

	// Command computed
	if computed := getMap(cmd, "computed"); len(computed) > 0 {
		box.AddLine("    computed:")
		for k, v := range computed {
			cm, _ := v.(map[string]any)
			typ := getStr(cm, "type")
			desc := getStr(cm, "description")
			if desc != "" {
				box.AddLine(fmt.Sprintf("      - %s: %s (%s)", k, typ, desc))
			} else {
				box.AddLine(fmt.Sprintf("      - %s: %s", k, typ))
			}
		}
	}

	// DCB Query
	if query := getSlice(cmd, "query"); len(query) > 0 {
		box.AddLine("    Query:")
		for _, qi := range query {
			for _, line := range formatQueryItemIR(qi) {
				box.AddLine(fmt.Sprintf("      - %s", line))
			}
		}
	}

	// Emits
	box.AddSection()
	box.AddLine("  Emits:")
	if emits := getSlice(data, "emits"); len(emits) > 0 {
		for _, e := range emits {
			em, _ := e.(map[string]any)
			box.AddLine(fmt.Sprintf("    %s", getStr(em, "type")))
			if fields := getMap(em, "fields"); len(fields) > 0 {
				for k, v := range fields {
					box.AddLine(fmt.Sprintf("      - %s: %s", k, irTypeStr(v)))
				}
			}
		}
	}

	// Scenarios
	if scenarios := getSlice(data, "scenarios"); len(scenarios) > 0 {
		box.AddSection()
		box.AddLine("  Scenarios:")
		for _, s := range scenarios {
			sm, _ := s.(map[string]any)
			box.AddLine(fmt.Sprintf("    • %s", getStr(sm, "name")))
			box.AddLine(fmt.Sprintf("      Given: %s", formatGivenIR(getSlice(sm, "given"))))
			when := getMap(sm, "when")
			box.AddLine(fmt.Sprintf("      When:  %s %s", getStr(when, "command"), formatValuesIR(getMap(when, "values"))))
			then := getMap(sm, "then")
			if getBool(then, "success") {
				box.AddLine(fmt.Sprintf("      Then:  ✓ %s", formatGivenIR(getSlice(then, "events"))))
			} else {
				box.AddLine(fmt.Sprintf("      Then:  ✗ %s", getStr(then, "error")))
			}
		}
	}

	return box.Render(), nil
}

func renderViewSliceIR(data map[string]any, width int) (string, error) {
	box := NewBox(width)

	name := getStr(data, "name")
	actor := getStr(data, "actor")
	box.AddLine(fmt.Sprintf("  VIEW: %s  │  Actor: %s", name, actor))

	// Image (optional)
	if img := getStr(data, "image"); img != "" {
		box.AddLine(fmt.Sprintf("  📷 %s", img))
		if ascii := renderImageASCII(img, width); ascii != "" {
			box.AddSection()
			for _, line := range normalizeIndent(ascii) {
				box.AddLine("  " + line)
			}
		}
	}

	// Endpoint
	ep := getMap(data, "endpoint")
	box.AddSection()
	box.AddLine(fmt.Sprintf("  %s %s", getStr(ep, "verb"), getStr(ep, "path")))
	if params := getMap(ep, "params"); len(params) > 0 {
		box.AddLine("    params:")
		for k, v := range params {
			box.AddLine(fmt.Sprintf("      - %s: %s", k, irTypeStr(v)))
		}
	}

	// ReadModel
	rm := getMap(data, "readModel")
	box.AddSection()
	box.AddLine(fmt.Sprintf("  ReadModel: %s (%s)", getStr(rm, "name"), getStr(rm, "cardinality")))
	box.AddLine("    fields:")
	if fields := getMap(rm, "fields"); len(fields) > 0 {
		renderFieldsIR(fields, "      ", box)
	}

	// Mapping
	if mapping := getMap(rm, "mapping"); len(mapping) > 0 {
		box.AddLine("    mapping:")
		for k, v := range mapping {
			box.AddLine(fmt.Sprintf("      - %s ← %s", k, irTypeStr(v)))
		}
	}

	// Computed
	if computed := getMap(rm, "computed"); len(computed) > 0 {
		box.AddLine("    computed:")
		for k, v := range computed {
			cm, _ := v.(map[string]any)
			event := getStr(cm, "event")
			fields := getSlice(cm, "fields")
			var fieldStrs []string
			for _, f := range fields {
				if s, ok := f.(string); ok {
					fieldStrs = append(fieldStrs, s)
				}
			}
			box.AddLine(fmt.Sprintf("      - %s: %s (%s)", k, event, strings.Join(fieldStrs, ", ")))
		}
	}

	// Query
	box.AddSection()
	box.AddLine("  Query:")
	if query := getSlice(data, "query"); len(query) > 0 {
		for _, qi := range query {
			for _, line := range formatQueryItemIR(qi) {
				box.AddLine(fmt.Sprintf("    - %s", line))
			}
		}
	}

	// Scenarios
	if scenarios := getSlice(data, "scenarios"); len(scenarios) > 0 {
		box.AddSection()
		box.AddLine("  Scenarios:")
		for _, s := range scenarios {
			sm, _ := s.(map[string]any)
			box.AddLine(fmt.Sprintf("    • %s", getStr(sm, "name")))
			box.AddLine(fmt.Sprintf("      Given: %s", formatGivenIR(getSlice(sm, "given"))))
			if q := getMap(sm, "query"); len(q) > 0 {
				box.AddLine(fmt.Sprintf("      Query: %s", formatValuesIR(q)))
			}
			box.AddLine("      Expect: {")
			if expect := getMap(sm, "expect"); len(expect) > 0 {
				for k, v := range expect {
					box.AddLine(fmt.Sprintf("        %s: %s", k, formatAnyIR(v)))
				}
			}
			box.AddLine("      }")
		}
	}

	return box.Render(), nil
}

func renderStoryIR(data map[string]any, width int) (string, error) {
	box := NewBox(width)
	sliceRef := getStr(data, "sliceRef")
	desc := getStr(data, "description")
	box.AddLine(fmt.Sprintf("  STORY: refs %s", sliceRef))
	if desc != "" {
		box.AddLine(fmt.Sprintf("  \"%s\"", desc))
	}
	return box.Render(), nil
}

// --- helpers ---

func getStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	r, _ := v.(map[string]any)
	return r
}

func getMapStr(m map[string]any, key string) map[string]string {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	raw, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string)
	for k, val := range raw {
		if s, ok := val.(string); ok {
			out[k] = s
		}
	}
	return out
}

func getSlice(m map[string]any, key string) []any {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok {
		return nil
	}
	r, _ := v.([]any)
	return r
}

func getBool(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	b, _ := v.(bool)
	return b
}

func irTypeStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		return "struct"
	case []any:
		if len(t) > 0 {
			return fmt.Sprintf("[%s]", irTypeStr(t[0]))
		}
		return "[]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func renderFieldsIR(fields map[string]any, indent string, box *Box) {
	for k, v := range fields {
		switch t := v.(type) {
		case map[string]any:
			box.AddLine(fmt.Sprintf("%s- %s:", indent, k))
			renderFieldsIR(t, indent+"    ", box)
		case []any:
			if len(t) == 1 {
				if inner, ok := t[0].(map[string]any); ok {
					box.AddLine(fmt.Sprintf("%s- %s: [", indent, k))
					for ik, iv := range inner {
						box.AddLine(fmt.Sprintf("%s    %s: %s", indent, ik, irTypeStr(iv)))
					}
					box.AddLine(fmt.Sprintf("%s  ]", indent))
				} else {
					box.AddLine(fmt.Sprintf("%s- %s: [%s]", indent, k, irTypeStr(t[0])))
				}
			} else {
				box.AddLine(fmt.Sprintf("%s- %s: []", indent, k))
			}
		default:
			box.AddLine(fmt.Sprintf("%s- %s: %s", indent, k, irTypeStr(v)))
		}
	}
}

func formatQueryItemIR(qi any) []string {
	m, ok := qi.(map[string]any)
	if !ok {
		return nil
	}
	typesRaw := getSlice(m, "types")
	tagsRaw := getSlice(m, "tags")

	var tags []string
	for _, t := range tagsRaw {
		tm, ok := t.(map[string]any)
		if !ok {
			continue
		}
		tagName := getStr(tm, "tag")
		if param := getStr(tm, "param"); param != "" {
			tags = append(tags, fmt.Sprintf("%s=<binding>", tagName))
		} else {
			tags = append(tags, tagName)
		}
	}

	var lines []string
	for _, et := range typesRaw {
		s, _ := et.(string)
		if len(tags) > 0 {
			lines = append(lines, fmt.Sprintf("[%s, tagged: %s]", s, strings.Join(tags, " AND ")))
		} else {
			lines = append(lines, fmt.Sprintf("[%s]", s))
		}
	}
	return lines
}

func formatGivenIR(items []any) string {
	if len(items) == 0 {
		return "(none)"
	}
	var parts []string
	for _, item := range items {
		switch t := item.(type) {
		case string:
			parts = append(parts, t)
		case map[string]any:
			et := getStr(t, "type")
			if vals := getMap(t, "values"); len(vals) > 0 {
				et += " " + formatValuesIR(vals)
			}
			parts = append(parts, et)
		}
	}
	return strings.Join(parts, ", ")
}

func formatValuesIR(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s: %s", k, formatAnyIR(v)))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func formatAnyIR(v any) string {
	switch t := v.(type) {
	case string:
		return fmt.Sprintf("%q", t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case bool:
		return fmt.Sprintf("%t", t)
	case []any:
		var items []string
		for _, item := range t {
			items = append(items, formatAnyIR(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case map[string]any:
		return formatValuesIR(t)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}
