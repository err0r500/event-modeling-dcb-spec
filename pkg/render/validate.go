package render

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/errors"
)

// Error codes by category:
//
//	E1xx - Command/Change slice
//	E2xx - View slice
//	E3xx - DCB query
//	E4xx - GWT scenarios
const (
	// Command errors
	ErrCmdFieldSource  = "E101" // field must come from trigger
	ErrCmdFieldType    = "E102" // field type mismatch
	ErrEmitFieldSource = "E103" // emit field must come from command
	ErrEmitFieldType   = "E104" // emit field type mismatch
	ErrCmdPathParam    = "E105" // path param not in params

	// View errors
	ErrEventOrdering   = "E201" // event must be emitted before
	ErrViewFieldSource = "E202" // field must come from events
	ErrComputedEvent   = "E203" // computed event not in query
	ErrComputedField   = "E204" // computed field not in event
	ErrMappingEvent    = "E205" // mapping event not in query
	ErrMappingField    = "E206" // mapping field not in event
	ErrMappingType     = "E207" // mapping type mismatch
	ErrDottedPath      = "E208" // dotted path doesn't resolve
	ErrDottedType      = "E209" // dotted path type mismatch
	ErrViewPathParam   = "E210" // path param not in params

	// DCB errors
	ErrEventMissingTag  = "E301" // event missing required tag
	ErrTagRequiresValue = "E302" // parameterized tag requires value

	// Dependent query errors
	ErrDepExtractEventNotInQuery = "E311" // extract event not in primary query
	ErrDepExtractFieldNotInEvent = "E312" // extract field not in event
	ErrDepFromExtractAndValue    = "E313" // cannot have both fromExtract and value
	ErrDepFromExtractInPrimary   = "E314" // fromExtract only allowed in dependent query

	// Scenario errors
	ErrScenarioGiven     = "E401" // given event not in query
	ErrScenarioThen      = "E402" // then event not in emits
	ErrScenarioType      = "E403" // event value type mismatch
	ErrViewScenarioGiven = "E404" // view scenario given not in query

	// Actor errors
	ErrActorUndefined = "E501" // actor not defined in board.actors
	ErrActorMissing   = "E502" // actor field missing from slice
)

var (
	// Pattern: slice_AddItem_event_CartCreated_must_have_tag_carts
	tagErrorPattern = regexp.MustCompile(`slice_(\w+)_event_(\w+)_must_have_tag_(\w+)`)
	// Pattern: slice_AddItem_event_CartCreated_must_be_emitted_before_or_by_self
	orderErrorPattern = regexp.MustCompile(`slice_(\w+)_event_(\w+)_must_be_emitted_before`)
	// Pattern: slice_AddItem_tag_cart_id_requires_value
	tagValuePattern = regexp.MustCompile(`slice_(\w+)_tag_(\w+)_requires_value`)
	// Pattern: slice_CreateCart_field_cartId_must_come_from_endpoint_or_be_computed
	fieldSourcePattern = regexp.MustCompile(`slice_(\w+)_field_(\w+)_must_come_from`)
	// Pattern: slice_AddItem_field_quantity_type_incompatible
	cmdTypeIncompatiblePattern = regexp.MustCompile(`slice_(\w+)_field_(\w+)_type`)
	// Pattern: slice_CreateCart_emit_CartCreated_field_cartId_source
	emitFieldSourcePattern = regexp.MustCompile(`slice_(\w+)_emit_(\w+)_field_(\w+)_source`)
	// Pattern: slice_CreateCart_emit_CartCreated_field_cartId_type
	emitTypeIncompatiblePattern = regexp.MustCompile(`slice_(\w+)_emit_(\w+)_field_(\w+)_type`)
	// Pattern: view_ReadA_computed_total_event_must_be_queried
	computedEventPattern = regexp.MustCompile(`view_(\w+)_computed_(\w+)_event_must_be_queried`)
	// Pattern: view_ReadA_computed_total_field_bogus_must_exist_in_event
	computedFieldPattern = regexp.MustCompile(`view_(\w+)_computed_(\w+)_field_(\w+)_must_exist_in_event`)
	// Pattern: view_ReadA_field_X_must_come_from_events_or_computed
	viewFieldSourcePattern = regexp.MustCompile(`view_(\w+)_field_(\w+)_must_come_from_events_or_computed`)
	// Pattern: view_ReadA_mapping_totalPrice_event_must_be_queried
	mappingEventPattern = regexp.MustCompile(`view_(\w+)_mapping_(\w+)_event_must_be_queried`)
	// Pattern: view_ReadA_mapping_totalPrice_field_amount_must_exist_in_event
	mappingFieldPattern = regexp.MustCompile(`view_(\w+)_mapping_(\w+)_field_(\w+)_must_exist_in_event`)
	// Pattern: view_ReadA_mapping_totalPrice_type
	mappingTypePattern = regexp.MustCompile(`view_(\w+)_mapping_(\w+)_type`)
	// Pattern: _validValues.fieldName: conflicting values X and Y (mismatched types A and B)
	scenarioTypeMismatchPattern = regexp.MustCompile(`_validValues\.(\w+): conflicting values (\S+) and (\w+) \(mismatched types (\w+) and (\w+)\)`)
	// Pattern: view_ReadA_scenarioN_given_EventType_must_be_in_query
	viewScenarioGivenPattern = regexp.MustCompile(`view_(\w+)_scenario(\d+)_given_(\w+)_must_be_in_query`)
	// Pattern: slice_AddItem_scenarioN_given_EventType_must_be_in_query
	sliceScenarioGivenPattern = regexp.MustCompile(`slice_(\w+)_scenario(\d+)_given_(\w+)_must_be_in_query`)
	// Pattern: slice_AddItem_scenarioN_then_EventType_must_be_in_emits
	scenarioThenPattern = regexp.MustCompile(`slice_(\w+)_scenario(\d+)_then_(\w+)_must_be_in_emits`)
	// Pattern: slice_AddItem_endpoint_path_param_cartId_must_be_in_params (or view_)
	pathParamPattern = regexp.MustCompile(`(slice|view)_(\w+)_endpoint_path_param_(\w+)_must_be_in_params`)
	// Pattern: _actorValid
	actorValidPattern = regexp.MustCompile(`_actorValid`)
	// Pattern: board.flow.N.actor: field is required but not present
	actorMissingPattern = regexp.MustCompile(`flow\.\d+\.actor: field is required`)
)

var (
	// Type mismatch path patterns for friendly formatting
	cmdFieldTypeRe  = regexp.MustCompile(`slice_(\w+)_field_(\w+)_type`)
	emitFieldTypeRe = regexp.MustCompile(`slice_(\w+)_emit_(\w+)_field_(\w+)_type`)
	mappingTypeRe   = regexp.MustCompile(`view_(\w+)_mapping_(\w+)_type`)
	scenarioTypeRe  = regexp.MustCompile(`_validValues\.(\w+)`)
)

// formatTypeMismatch returns (code, friendly message) for a type mismatch path
func formatTypeMismatch(path, expectedType, gotType, gotValue string) (string, string) {
	if m := emitFieldTypeRe.FindStringSubmatch(path); m != nil {
		return ErrEmitFieldType, fmt.Sprintf("slice %q emits %q: field %q type mismatch -> event expects <%s> command declares <%s>", m[1], m[2], m[3], gotType, expectedType)
	}
	if m := cmdFieldTypeRe.FindStringSubmatch(path); m != nil {
		return ErrCmdFieldType, fmt.Sprintf("slice %q field %q: command expects %s, trigger provides %s", m[1], m[2], gotType, expectedType)
	}
	if m := mappingTypeRe.FindStringSubmatch(path); m != nil {
		return ErrMappingType, fmt.Sprintf("view %q mapping %q: type mismatch (expected %s, got %s)", m[1], m[2], expectedType, gotType)
	}
	if m := scenarioTypeRe.FindStringSubmatch(path); m != nil {
		return ErrScenarioType, fmt.Sprintf("scenario field %s: expected %s, got %s (%s)", m[1], expectedType, gotType, gotValue)
	}
	return ErrScenarioType, fmt.Sprintf("%s: expected %s, got %s (%s)", path, expectedType, gotType, gotValue)
}

// fmtErr formats an error with code, message, and optional location
func fmtErr(code, msg, loc string) string {
	if loc != "" {
		return fmt.Sprintf("%s: %s [%s]", code, msg, loc)
	}
	return fmt.Sprintf("%s: %s", code, msg)
}

// FormatCUEError takes a CUE error and returns a user-friendly message with position info
func FormatCUEError(err error) string {
	if err == nil {
		return ""
	}

	fullErrStr := err.Error()

	// Look for type mismatch pattern in full error string
	typeMismatchRe := regexp.MustCompile(`(\w+(?:\.\w+)*): conflicting values (\S+) and (\w+) \(mismatched types (\w+) and (\w+)\)`)
	if matches := typeMismatchRe.FindAllStringSubmatch(fullErrStr, -1); len(matches) > 0 {
		var results []string
		seen := make(map[string]bool)
		for _, match := range matches {
			path := match[1]
			gotValue := match[2]
			expectedType := match[3]
			gotType := match[4]
			code, msg := formatTypeMismatch(path, expectedType, gotType, gotValue)
			formatted := fmtErr(code, msg, "")
			if !seen[formatted] {
				seen[formatted] = true
				results = append(results, formatted)
			}
		}
		return strings.Join(results, "\n")
	}

	// Collect all errors from the error list
	allErrs := errors.Errors(err)
	seen := make(map[string]bool)
	var results []string

	for _, e := range allErrs {
		code, msg := formatSingleError(e)
		if code == "" {
			continue // Skip noise
		}
		pos := extractPosition(e)
		formatted := fmtErr(code, msg, pos)
		if !seen[formatted] {
			seen[formatted] = true
			results = append(results, formatted)
		}
	}

	if len(results) == 0 {
		return err.Error()
	}
	return strings.Join(results, "\n")
}

// extractPosition gets file:line:col from a CUE error
func extractPosition(err errors.Error) string {
	positions := errors.Positions(err)
	if len(positions) == 0 {
		return ""
	}
	p := positions[0]
	if p.Filename() == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d:%d", p.Filename(), p.Line(), p.Column())
}

// formatSingleError formats a single error message, returns (code, message)
func formatSingleError(err errors.Error) (string, string) {
	msg := err.Error()

	// Skip disjunction noise
	if strings.Contains(msg, "empty disjunction") || strings.Contains(msg, "field not allowed") || strings.Contains(msg, "incompatible list lengths") {
		return "", ""
	}

	// DCB: event missing tag
	if match := tagErrorPattern.FindStringSubmatch(msg); match != nil {
		return ErrEventMissingTag, fmt.Sprintf("slice %q query: event %q must have tag %q", match[1], match[2], match[3])
	}

	// View: event ordering
	if match := orderErrorPattern.FindStringSubmatch(msg); match != nil {
		return ErrEventOrdering, fmt.Sprintf("slice %q query: event %q must be emitted by an earlier slice", match[1], match[2])
	}

	// DCB: tag requires value
	if match := tagValuePattern.FindStringSubmatch(msg); match != nil {
		return ErrTagRequiresValue, fmt.Sprintf("slice %q query: tag %q must have a value (parameterized tag)", match[1], match[2])
	}

	// Command: emit field source (check before general field patterns)
	if match := emitFieldSourcePattern.FindStringSubmatch(msg); match != nil {
		return ErrEmitFieldSource, fmt.Sprintf("slice %q emit %q: field %q must come from command or computed", match[1], match[2], match[3])
	}

	// Command: emit field type (check before general type patterns)
	if match := emitTypeIncompatiblePattern.FindStringSubmatch(msg); match != nil {
		return ErrEmitFieldType, fmt.Sprintf("slice %q emit %q: field %q must match command field type", match[1], match[2], match[3])
	}

	// Command: field source
	if match := fieldSourcePattern.FindStringSubmatch(msg); match != nil {
		return ErrCmdFieldSource, fmt.Sprintf("slice %q command: field %q must come from trigger, needs a mapping or be computed", match[1], match[2])
	}

	// Command: field type
	if match := cmdTypeIncompatiblePattern.FindStringSubmatch(msg); match != nil {
		return ErrCmdFieldType, fmt.Sprintf("slice %q command: field %q must match trigger field type", match[1], match[2])
	}

	// View: field source
	if match := viewFieldSourcePattern.FindStringSubmatch(msg); match != nil {
		return ErrViewFieldSource, fmt.Sprintf("view %q readModel: field %q must come from queried events or computed", match[1], match[2])
	}

	// View: computed event not queried
	if match := computedEventPattern.FindStringSubmatch(msg); match != nil {
		return ErrComputedEvent, fmt.Sprintf("view %q computed %q: source event must be in query", match[1], match[2])
	}

	// View: computed field not in event
	if match := computedFieldPattern.FindStringSubmatch(msg); match != nil {
		return ErrComputedField, fmt.Sprintf("view %q computed %q: field %q must exist in source event", match[1], match[2], match[3])
	}

	// View: mapping event not queried
	if match := mappingEventPattern.FindStringSubmatch(msg); match != nil {
		return ErrMappingEvent, fmt.Sprintf("view %q mapping %q: source event must be in query", match[1], match[2])
	}

	// View: mapping field not in event
	if match := mappingFieldPattern.FindStringSubmatch(msg); match != nil {
		return ErrMappingField, fmt.Sprintf("view %q mapping %q: field %q must exist in source event", match[1], match[2], match[3])
	}

	// View: mapping type mismatch
	if match := mappingTypePattern.FindStringSubmatch(msg); match != nil {
		return ErrMappingType, fmt.Sprintf("view %q mapping %q: must match source event field type", match[1], match[2])
	}

	// Scenario: event value type mismatch
	if match := scenarioTypeMismatchPattern.FindStringSubmatch(msg); match != nil {
		return ErrScenarioType, fmt.Sprintf("scenario field %s: must be %s, got %s (%s)", match[1], match[3], match[4], match[2])
	}

	// Scenario: view given event not in query
	if match := viewScenarioGivenPattern.FindStringSubmatch(msg); match != nil {
		return ErrViewScenarioGiven, fmt.Sprintf("view %q scenario %s: given event %q must be in query", match[1], match[2], match[3])
	}

	// Scenario: slice given event not in query
	if match := sliceScenarioGivenPattern.FindStringSubmatch(msg); match != nil {
		return ErrScenarioGiven, fmt.Sprintf("slice %q scenario %s: given event %q must be in query", match[1], match[2], match[3])
	}

	// Scenario: then event not in emits
	if match := scenarioThenPattern.FindStringSubmatch(msg); match != nil {
		return ErrScenarioThen, fmt.Sprintf("slice %q scenario %s: then event %q must be in emits", match[1], match[2], match[3])
	}

	// Endpoint: path param missing
	if match := pathParamPattern.FindStringSubmatch(msg); match != nil {
		kind := match[1]
		code := ErrCmdPathParam
		if kind == "view" {
			code = ErrViewPathParam
		}
		return code, fmt.Sprintf("%s %q endpoint: path param {%s} must be in params", kind, match[2], match[3])
	}

	// Actor: not defined in board.actors
	if actorValidPattern.MatchString(msg) {
		return ErrActorUndefined, "slice actor must be defined in board.actors"
	}

	// Actor: missing from slice
	if actorMissingPattern.MatchString(msg) {
		return ErrActorMissing, "slice must have an actor field"
	}

	// Return raw error if no pattern matches (avoid hiding errors)
	return "E000", msg
}

// ValidateBoard validates a board and returns formatted error messages
func ValidateBoard(board cue.Value) []string {
	var errs []string

	// CUE validation happens automatically - we just need to check for errors
	if err := board.Validate(); err != nil {
		for line := range strings.SplitSeq(err.Error(), "\n") {
			if line == "" {
				continue
			}
			formatted := FormatCUEError(fmt.Errorf("%s", line))
			if formatted != "" && !slices.Contains(errs, formatted) {
				errs = append(errs, formatted)
			}
		}
	}

	// Additional Go validation: actor must be present and defined
	errs = append(errs, validateActors(board)...)

	// Additional Go validation: parameterized tags must have values
	errs = append(errs, validateParameterizedTags(board)...)

	// Additional Go validation: dotted paths in mapping/computed must resolve
	errs = append(errs, validateDottedPaths(board)...)

	// Additional Go validation: dependent query constraints
	errs = append(errs, validateDependentQueries(board)...)

	return errs
}

// validateActors checks that each slice has an actor and it's defined in board.actors
func validateActors(board cue.Value) []string {
	var errs []string

	// Build list of defined actors
	actorNames := make(map[string]bool)
	actorsVal := board.LookupPath(cue.ParsePath("actors"))
	if iter, err := actorsVal.Fields(); err == nil {
		for iter.Next() {
			actorNames[iter.Selector().Unquoted()] = true
		}
	}

	flowVal := board.LookupPath(cue.ParsePath("flow"))
	flowIter, err := flowVal.List()
	if err != nil {
		return errs
	}

	for flowIter.Next() {
		inst := flowIter.Value()
		kind := getString(inst, "kind")
		if kind != "slice" {
			continue
		}

		// Automation slices don't have actors
		sliceType := getString(inst, "type")
		if sliceType == "automation" {
			continue
		}

		sliceName := getString(inst, "name")
		actorVal := inst.LookupPath(cue.ParsePath("actor"))

		if !actorVal.Exists() || actorVal.Err() != nil {
			errs = append(errs, fmtErr(ErrActorMissing, fmt.Sprintf("slice %q must have an actor", sliceName), ""))
			continue
		}

		actorName := getString(actorVal, "name")
		if actorName == "" || !actorNames[actorName] {
			errs = append(errs, fmtErr(ErrActorUndefined, fmt.Sprintf("slice %q actor %q not defined in board.actors", sliceName, actorName), ""))
		}
	}

	return errs
}

// validateDottedPaths checks that dotted paths in mapping/computed resolve to actual fields
func validateDottedPaths(board cue.Value) []string {
	var errs []string

	eventsVal := board.LookupPath(cue.ParsePath("events"))
	flowVal := board.LookupPath(cue.ParsePath("flow"))
	flowIter, err := flowVal.List()
	if err != nil {
		return errs
	}

	for flowIter.Next() {
		inst := flowIter.Value()
		kind := getString(inst, "kind")
		if kind != "slice" {
			continue
		}

		sliceType := getString(inst, "type")
		if sliceType != "view" {
			continue
		}

		sliceName := getString(inst, "name")
		fieldsVal := inst.LookupPath(cue.ParsePath("readModel.fields"))

		// Check mapping paths
		mappingVal := inst.LookupPath(cue.ParsePath("readModel.mapping"))
		if iter, err := mappingVal.Fields(); err == nil {
			for iter.Next() {
				pathKey := iter.Selector().Unquoted()
				if !strings.Contains(pathKey, ".") {
					continue
				}

				fieldType, ok := resolveDottedPathType(fieldsVal, pathKey)
				if !ok {
					errs = append(errs, fmtErr(ErrDottedPath, fmt.Sprintf("view %q mapping %q: path must resolve to a field in readModel", sliceName, pathKey), ""))
					continue
				}

				m := iter.Value()
				eventType := getString(m, "event.eventType")
				eventFieldName := getString(m, "field")
				eventFieldType := eventsVal.LookupPath(cue.ParsePath(eventType + ".fields." + eventFieldName))

				if eventFieldType.Exists() && eventFieldType.Err() == nil {
					unified := fieldType.Unify(eventFieldType)
					if unified.Err() != nil {
						errs = append(errs, fmtErr(ErrDottedType, fmt.Sprintf("view %q mapping %q: must match source event field type", sliceName, pathKey), ""))
					}
				}
			}
		}

		// Check computed paths
		computedVal := inst.LookupPath(cue.ParsePath("readModel.computed"))
		if iter, err := computedVal.Fields(); err == nil {
			for iter.Next() {
				pathKey := iter.Selector().Unquoted()
				if !strings.Contains(pathKey, ".") {
					continue
				}
				_, ok := resolveDottedPathType(fieldsVal, pathKey)
				if !ok {
					errs = append(errs, fmtErr(ErrDottedPath, fmt.Sprintf("view %q computed %q: path must resolve to a field in readModel", sliceName, pathKey), ""))
				}
			}
		}
	}

	return errs
}

// resolveDottedPathType checks if a dotted path like "items.price" resolves in fields
func resolveDottedPathType(fields cue.Value, path string) (cue.Value, bool) {
	parts := strings.Split(path, ".")
	current := fields

	for _, part := range parts {
		next := current.LookupPath(cue.ParsePath(part))
		if next.Exists() && next.Err() == nil {
			current = next
			continue
		}

		if current.IncompleteKind() == cue.ListKind {
			listIter, err := current.List()
			if err == nil && listIter.Next() {
				elem := listIter.Value()
				next = elem.LookupPath(cue.ParsePath(part))
				if next.Exists() && next.Err() == nil {
					current = next
					continue
				}
			}
			elem := current.LookupPath(cue.MakePath(cue.AnyIndex))
			if elem.Exists() && elem.Err() == nil {
				next = elem.LookupPath(cue.ParsePath(part))
				if next.Exists() && next.Err() == nil {
					current = next
					continue
				}
			}
		}

		return cue.Value{}, false
	}

	return current, true
}

// validateParameterizedTags checks that parameterized tags have values in queries
func validateParameterizedTags(board cue.Value) []string {
	var errs []string

	paramTags := make(map[string]bool)
	tagsVal := board.LookupPath(cue.ParsePath("tags"))
	if iter, err := tagsVal.Fields(); err == nil {
		for iter.Next() {
			tagName := iter.Selector().Unquoted()
			paramVal := iter.Value().LookupPath(cue.ParsePath("param"))
			if paramVal.Exists() && paramVal.Err() == nil {
				paramTags[tagName] = true
			}
		}
	}

	flowVal := board.LookupPath(cue.ParsePath("flow"))
	flowIter, err := flowVal.List()
	if err != nil {
		return errs
	}

	for flowIter.Next() {
		inst := flowIter.Value()
		kind := getString(inst, "kind")
		if kind != "slice" {
			continue
		}

		sliceName := getString(inst, "name")
		sliceType := getString(inst, "type")

		var queryPath string
		if sliceType == "change" {
			queryPath = "command.query.items"
		} else {
			queryPath = "query.items"
		}

		queryVal := inst.LookupPath(cue.ParsePath(queryPath))
		if qIter, err := queryVal.List(); err == nil {
			for qIter.Next() {
				item := qIter.Value()
				tagsVal := item.LookupPath(cue.ParsePath("tags"))
				if tIter, err := tagsVal.List(); err == nil {
					for tIter.Next() {
						tagRef := tIter.Value()
						tagName := getString(tagRef, "tag.name")
						if paramTags[tagName] {
							valueVal := tagRef.LookupPath(cue.ParsePath("value"))
							if !valueVal.Exists() || valueVal.Err() != nil {
								errs = append(errs, fmtErr(ErrTagRequiresValue, fmt.Sprintf("slice %q query: tag %q must have a value (parameterized tag)", sliceName, tagName), ""))
							}
						}
					}
				}
			}
		}
	}

	return errs
}

// validateDependentQueries checks dependent query constraints:
// 1. extract.event must be in primary query
// 2. extract.field must exist in that event
// 3. TagRef cannot have both fromExtract and value
// 4. primary query cannot use fromExtract
func validateDependentQueries(board cue.Value) []string {
	var errs []string

	eventsVal := board.LookupPath(cue.ParsePath("events"))
	flowVal := board.LookupPath(cue.ParsePath("flow"))
	flowIter, err := flowVal.List()
	if err != nil {
		return errs
	}

	for flowIter.Next() {
		inst := flowIter.Value()
		kind := getString(inst, "kind")
		if kind != "slice" {
			continue
		}

		sliceName := getString(inst, "name")
		sliceType := getString(inst, "type")

		// Determine paths based on slice type
		var queryPath, depQueryPath string
		if sliceType == "change" || sliceType == "automation" {
			queryPath = "command.query.items"
			depQueryPath = "command.dependentQuery"
		} else if sliceType == "view" {
			queryPath = "query.items"
			depQueryPath = "dependentQuery"
		} else {
			continue
		}

		// Check primary query for fromExtract (not allowed)
		queryVal := inst.LookupPath(cue.ParsePath(queryPath))
		if qIter, err := queryVal.List(); err == nil {
			for qIter.Next() {
				item := qIter.Value()
				tagsVal := item.LookupPath(cue.ParsePath("tags"))
				if tIter, err := tagsVal.List(); err == nil {
					for tIter.Next() {
						tagRef := tIter.Value()
						fromExtract := tagRef.LookupPath(cue.ParsePath("fromExtract"))
						if fromExtract.Exists() && fromExtract.Err() == nil {
							errs = append(errs, fmtErr(ErrDepFromExtractInPrimary, fmt.Sprintf("slice %q: fromExtract only allowed in dependentQuery, not primary query", sliceName), ""))
						}
					}
				}
			}
		}

		// Check dependent query if present
		depQueryVal := inst.LookupPath(cue.ParsePath(depQueryPath))
		if !depQueryVal.Exists() || depQueryVal.Err() != nil {
			continue
		}

		// Build set of event types in primary query
		primaryEventTypes := make(map[string]bool)
		if qIter, err := queryVal.List(); err == nil {
			for qIter.Next() {
				item := qIter.Value()
				typesVal := item.LookupPath(cue.ParsePath("types"))
				if tIter, err := typesVal.List(); err == nil {
					for tIter.Next() {
						evtType := getString(tIter.Value(), "eventType")
						primaryEventTypes[evtType] = true
					}
				}
			}
		}

		// Validate extract: event in primary, field exists
		extractVal := depQueryVal.LookupPath(cue.ParsePath("extract"))
		if iter, err := extractVal.Fields(); err == nil {
			for iter.Next() {
				extractName := iter.Selector().Unquoted()
				ext := iter.Value()

				evtType := getString(ext, "event.eventType")
				fieldName := getString(ext, "field")

				// Check event is in primary query
				if !primaryEventTypes[evtType] {
					errs = append(errs, fmtErr(ErrDepExtractEventNotInQuery, fmt.Sprintf("slice %q dependentQuery.extract.%s: event %q must be in primary query", sliceName, extractName, evtType), ""))
				}

				// Check field exists in event
				eventFieldsVal := eventsVal.LookupPath(cue.ParsePath(evtType + ".fields"))
				fieldVal := eventFieldsVal.LookupPath(cue.ParsePath(fieldName))
				if !fieldVal.Exists() || fieldVal.Err() != nil {
					errs = append(errs, fmtErr(ErrDepExtractFieldNotInEvent, fmt.Sprintf("slice %q dependentQuery.extract.%s: field %q not in event %q", sliceName, extractName, fieldName, evtType), ""))
				}
			}
		}

		// Validate dependent query items: tags cannot have both value and fromExtract
		depItemsVal := depQueryVal.LookupPath(cue.ParsePath("items"))
		if dIter, err := depItemsVal.List(); err == nil {
			for dIter.Next() {
				item := dIter.Value()
				tagsVal := item.LookupPath(cue.ParsePath("tags"))
				if tIter, err := tagsVal.List(); err == nil {
					for tIter.Next() {
						tagRef := tIter.Value()
						valueVal := tagRef.LookupPath(cue.ParsePath("value"))
						fromExtractVal := tagRef.LookupPath(cue.ParsePath("fromExtract"))

						hasValue := valueVal.Exists() && valueVal.Err() == nil
						hasFromExtract := fromExtractVal.Exists() && fromExtractVal.Err() == nil

						if hasValue && hasFromExtract {
							tagName := getString(tagRef, "tag.name")
							errs = append(errs, fmtErr(ErrDepFromExtractAndValue, fmt.Sprintf("slice %q dependentQuery: tag %q cannot have both value and fromExtract", sliceName, tagName), ""))
						}
					}
				}
			}
		}
	}

	return errs
}
