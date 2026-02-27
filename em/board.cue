package em

import (
	"list"
	"regexp"
	"strings"
)

// #Board - Event Modeling Board (complete domain model)
//
// The board is the top-level container for an event-modeled domain.
// It defines all entities and their relationships with explicit causality.
//
// Structure: Board → Contexts → Chapters → Flow (slices & story steps)
//
// Fields:
//   name: string - board identifier
//   tags: {[Name]: #Tag} - all tags for DCB partitioning (key becomes tag.name)
//   events: {[Type]: #Event} - all events (key becomes event.eventType)
//   actors: {[Name]: #Actor} - all actors (key becomes actor.name)
//   contexts: [...#Context] - bounded contexts, each containing ordered chapters
//   flow: (computed) - flat ordered sequence derived from contexts → chapters → flow
//
// Validation (automatic):
//   - Actors referenced in slices must exist in actors
//   - Events emitted by change slices must exist in events
//   - View slices can only query events emitted by earlier change slices
//   - Command fields must come from endpoint or be computed
//   - Story steps must reference existing slice names
//   - DCB query tags must exist and events must have required tags
#Board: {
	name: string

	// All tags used in the system
	tags: [Name=string]: #Tag & {name: Name}

	// All events defined in the system
	events: [Type=string]: #Event & {eventType: Type}

	// All actors
	actors: [Name=string]: #Actor & {name: Name}

	// Contexts contain chapters which contain the flow
	contexts: [...#Context]

	// Computed flat flow from all contexts → chapters → flow
	_allFlow: [ for ctx in contexts for ch in ctx.chapters for inst in ch.flow {inst}]
	flow: _allFlow

	// --- HELPERS ---

	// Extract only slices from flow
	_slices: [for inst in flow if inst.kind == "slice" {inst}]

	// Extract only story steps from flow
	_storySteps: [for inst in flow if inst.kind == "story" {inst}]

	// Slice names list
	_sliceNameList: [for s in _slices {s.name}]

	// Event type list
	_eventTypeList: [for k, _ in events {k}]

	// Tag list
	_tagList: [for k, _ in tags {k}]

	// Actor list
	_actorList: [for k, _ in actors {k}]

	// Map: eventType -> list of tag names
	_eventTagMap: {for k, e in events {(k): [for t in e.tags {t.name}]}}

	// --- VALIDATION ---

	// Build set of emitted eventTypes up to each flow index
	// Change and automation slices emit events
	_emittedBefore: {
		"0": {}
		for i, inst in flow if i > 0 {
			"\(i)": _emittedBefore["\(i-1)"] & {
				if flow[i-1].kind == "slice" {
					if flow[i-1].type == "change" || flow[i-1].type == "automation" {
						for e in flow[i-1].emits {
							(e.eventType): true
						}
					}
				}
			}
		}
	}

	// Pre-compute list versions
	_emittedBeforeLists: {
		for i, _ in flow {
			"\(i)": [for k, _ in _emittedBefore["\(i)"] {k}]
		}
	}

	// Validate slices - actor must be defined (except automation slices)
	for i, inst in flow if inst.kind == "slice" {
		if inst.type != "automation" {
			_actorValid: list.Contains(_actorList, inst.actor.name) & true
		}
	}

	// Validate change slices
	for i, inst in flow if inst.kind == "slice" {
		if inst.type == "change" {
			// Emitted events must be defined
			for e in inst.emits {
				_emitDefined: list.Contains(_eventTypeList, e.eventType) & true
			}

			// Command fields must come from trigger, mapping, or be computed
			let _computedFields = [for k, _ in inst.command.computed {k}]
			let _mappedFields = [for k, _ in inst.command.mapping {k}]

			// Endpoint trigger: fields from params/body/auth
			if inst.trigger.kind == "endpoint" {
				let _paramFields = [for k, _ in inst.trigger.endpoint.params {k}]
				let _bodyFields = [for k, _ in inst.trigger.endpoint.body {k}]
				let _authFields = [for k, _ in inst.trigger.endpoint.auth {k}]
				for fieldName, fieldType in inst.command.fields {
					let inParams = list.Contains(_paramFields, fieldName)
					let inBody = list.Contains(_bodyFields, fieldName)
					let inAuth = list.Contains(_authFields, fieldName)
					let isComputed = list.Contains(_computedFields, fieldName)
					let inMapping = list.Contains(_mappedFields, fieldName)
					("slice_\(inst.name)_field_\(fieldName)_must_come_from_trigger"): (inParams | inBody | inAuth | isComputed | inMapping) & true

					// Type validation (skip computed)
					if isComputed == false && inMapping == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.command.mapping[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inParams == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.trigger.endpoint.params[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inBody == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.trigger.endpoint.body[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inAuth == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.trigger.endpoint.auth[fieldName] & fieldType
					}
				}

				// Validate endpoint path params exist in params
				if strings.Contains(inst.trigger.endpoint.path, "{") {
					let _pathParams = [for m in regexp.FindAllSubmatch("\\{(\\w+)\\}", inst.trigger.endpoint.path, -1) {m[1]}]
					for p in _pathParams {
						("slice_\(inst.name)_endpoint_path_param_\(p)_must_be_in_params"): list.Contains(_paramFields, p) & true
					}
				}
			}

			// ExternalEvent trigger: fields from externalEvent.fields
			if inst.trigger.kind == "externalEvent" {
				let _extFields = [for k, _ in inst.trigger.externalEvent.fields {k}]
				for fieldName, fieldType in inst.command.fields {
					let inExt = list.Contains(_extFields, fieldName)
					let isComputed = list.Contains(_computedFields, fieldName)
					let inMapping = list.Contains(_mappedFields, fieldName)
					("slice_\(inst.name)_field_\(fieldName)_must_come_from_trigger"): (inExt | isComputed | inMapping) & true

					// Type validation (skip computed)
					if isComputed == false && inMapping == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.command.mapping[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inExt == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.trigger.externalEvent.fields[fieldName] & fieldType
					}
				}
			}

			// InternalEvent trigger: fields from internalEvent.fields, causality check
			if inst.trigger.kind == "internalEvent" {
				// Causality: the triggering event must be emitted before this slice
				("slice_\(inst.name)_internalEvent_\(inst.trigger.internalEvent.eventType)_must_be_emitted_before"): list.Contains(_emittedBeforeLists["\(i)"], inst.trigger.internalEvent.eventType) & true

				let _intFields = [for k, _ in inst.trigger.internalEvent.fields {k}]
				for fieldName, fieldType in inst.command.fields {
					let inInt = list.Contains(_intFields, fieldName)
					let isComputed = list.Contains(_computedFields, fieldName)
					let inMapping = list.Contains(_mappedFields, fieldName)
					("slice_\(inst.name)_field_\(fieldName)_must_come_from_trigger"): (inInt | isComputed | inMapping) & true

					// Type validation (skip computed)
					if isComputed == false && inMapping == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.command.mapping[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inInt == true {
						("slice_\(inst.name)_field_\(fieldName)_type"): inst.trigger.internalEvent.fields[fieldName] & fieldType
					}
				}
			}

			// Validate emitted event fields come from command, mapping, or computed
			for e in inst.emits {
				let _cmdFields = [for k, _ in inst.command.fields {k}]
				let _mappedFields = [for k, _ in e.mapping {k}]
				let _eventComputedFields = [for k, _ in e.computed {k}]
				for eventFieldName, eventFieldType in e.fields {
					let inCmd = list.Contains(_cmdFields, eventFieldName)
					let inMapping = list.Contains(_mappedFields, eventFieldName)
					let isComputed = list.Contains(_eventComputedFields, eventFieldName)
					("slice_\(inst.name)_emit_\(e.eventType)_field_\(eventFieldName)_source"): (inCmd | inMapping | isComputed) & true

					// Type compatibility (skip computed)
					if isComputed == false {
						if inMapping == true {
							// Type check against mapped field value
							("slice_\(inst.name)_emit_\(e.eventType)_field_\(eventFieldName)_type"): e.mapping[eventFieldName] & eventFieldType
						}
						if inMapping == false && inCmd == true {
							// Type check against same-name command field
							("slice_\(inst.name)_emit_\(e.eventType)_field_\(eventFieldName)_type"): inst.command.fields[eventFieldName] & eventFieldType
						}
					}
				}
			}

			// Validate scenario command names match slice command
			for si, s in inst.scenarios {
				("slice_\(inst.name)_scenario\(si)_command_must_match"): s.when.name & inst.command.name
			}

			// Validate scenario given events are in query types (including dependent query)
			let _queryEventTypes = [
				for qi in inst.command.query.items
				for e in qi.types {e.eventType},
			]
			let _depQueryEventTypes = [
				if inst.command.dependentQuery != _|_
				for qi in inst.command.dependentQuery.items
				for e in qi.types {e.eventType},
			]
			let _allQueryEventTypes = list.Concat([_queryEventTypes, _depQueryEventTypes])
			for si, s in inst.scenarios {
				for ge in s.given {
					("slice_\(inst.name)_scenario\(si)_given_\(ge.eventType)_must_be_in_query"): list.Contains(_allQueryEventTypes, ge.eventType) & true
				}
			}

			// Validate scenario then.events are in emits (success scenarios only)
			let _emitEventTypes = [for e in inst.emits {e.eventType}]
			for si, s in inst.scenarios {
				if s.then.success == true {
					for te in s.then.events {
						("slice_\(inst.name)_scenario\(si)_then_\(te.eventType)_must_be_in_emits"): list.Contains(_emitEventTypes, te.eventType) & true
					}
				}
			}

			// Validate DCB query
			for qi in inst.command.query.items {
				// All tags must be defined
				for tref in qi.tags {
					_dcbTagValid: list.Contains(_tagList, tref._tagName) & true
				}

				// Satisfiability: EVERY event must have ALL required tags
				for e in qi.types {
					for tref in qi.tags {
						("slice_\(inst.name)_event_\(e.eventType)_must_have_tag_\(tref._tagName)"): list.Contains(_eventTagMap[e.eventType], tref._tagName) & true
					}
				}
			}

			// Validate dependent query if present
			if inst.command.dependentQuery != _|_ {
				for qi in inst.command.dependentQuery.items {
					// All tags must be defined
					for tref in qi.tags {
						_dcbTagValid: list.Contains(_tagList, tref._tagName) & true
					}

					// Satisfiability: EVERY event must have ALL required tags
					for e in qi.types {
						for tref in qi.tags {
							("slice_\(inst.name)_depquery_event_\(e.eventType)_must_have_tag_\(tref._tagName)"): list.Contains(_eventTagMap[e.eventType], tref._tagName) & true
						}
					}
				}
			}
		}
	}

	// Validate automation slices (similar to change, but no actor/endpoint)
	for i, inst in flow if inst.kind == "slice" {
		if inst.type == "automation" {
			// Emitted events must be defined
			for e in inst.emits {
				_emitDefined: list.Contains(_eventTypeList, e.eventType) & true
			}

			// Command fields must come from trigger, consumed readModels, mapping, or be computed
			let _computedFields = [for k, _ in inst.command.computed {k}]
			let _mappedFields = [for k, _ in inst.command.mapping {k}]
			let _consumedFields = [for v in inst.consumes for k, _ in v._schema {k}]

			// ExternalEvent trigger: fields from externalEvent.fields or consumed readModels
			if inst.trigger.kind == "externalEvent" {
				let _extFields = [for k, _ in inst.trigger.externalEvent.fields {k}]
				for fieldName, fieldType in inst.command.fields {
					let inExt = list.Contains(_extFields, fieldName)
					let inConsumed = list.Contains(_consumedFields, fieldName)
					let isComputed = list.Contains(_computedFields, fieldName)
					let inMapping = list.Contains(_mappedFields, fieldName)
					("automation_\(inst.name)_field_\(fieldName)_must_come_from_trigger"): (inExt | inConsumed | isComputed | inMapping) & true

					// Type validation (skip computed)
					if isComputed == false && inMapping == true {
						("automation_\(inst.name)_field_\(fieldName)_type"): inst.command.mapping[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inExt == true {
						("automation_\(inst.name)_field_\(fieldName)_type"): inst.trigger.externalEvent.fields[fieldName] & fieldType
					}
				}
			}

			// InternalEvent trigger: fields from internalEvent.fields or consumed readModels, causality check
			if inst.trigger.kind == "internalEvent" {
				// Causality: the triggering event must be emitted before this slice
				("automation_\(inst.name)_internalEvent_\(inst.trigger.internalEvent.eventType)_must_be_emitted_before"): list.Contains(_emittedBeforeLists["\(i)"], inst.trigger.internalEvent.eventType) & true

				let _intFields = [for k, _ in inst.trigger.internalEvent.fields {k}]
				for fieldName, fieldType in inst.command.fields {
					let inInt = list.Contains(_intFields, fieldName)
					let inConsumed = list.Contains(_consumedFields, fieldName)
					let isComputed = list.Contains(_computedFields, fieldName)
					let inMapping = list.Contains(_mappedFields, fieldName)
					("automation_\(inst.name)_field_\(fieldName)_must_come_from_trigger"): (inInt | inConsumed | isComputed | inMapping) & true

					// Type validation (skip computed)
					if isComputed == false && inMapping == true {
						("automation_\(inst.name)_field_\(fieldName)_type"): inst.command.mapping[fieldName] & fieldType
					}
					if isComputed == false && inMapping == false && inInt == true {
						("automation_\(inst.name)_field_\(fieldName)_type"): inst.trigger.internalEvent.fields[fieldName] & fieldType
					}
				}
			}

			// Collect consumed readModel fields (ReadModel has _schema directly, not nested in readModel)
			let _consumedViewFields = [
				for v in inst.consumes
				for k, _ in v._schema {k},
			]

			// Validate emitted event fields come from command, consumed readModels, mapping, or computed
			for e in inst.emits {
				let _cmdFields = [for k, _ in inst.command.fields {k}]
				let _mappedFields = [for k, _ in e.mapping {k}]
				let _eventComputedFields = [for k, _ in e.computed {k}]
				for eventFieldName, eventFieldType in e.fields {
					let inCmd = list.Contains(_cmdFields, eventFieldName)
					let inConsumed = list.Contains(_consumedViewFields, eventFieldName)
					let inMapping = list.Contains(_mappedFields, eventFieldName)
					let isComputed = list.Contains(_eventComputedFields, eventFieldName)
					("automation_\(inst.name)_emit_\(e.eventType)_field_\(eventFieldName)_source"): (inCmd | inConsumed | inMapping | isComputed) & true

					// Type compatibility (skip computed)
					if isComputed == false {
						if inMapping == true {
							("automation_\(inst.name)_emit_\(e.eventType)_field_\(eventFieldName)_type"): e.mapping[eventFieldName] & eventFieldType
						}
						if inMapping == false && inCmd == true {
							("automation_\(inst.name)_emit_\(e.eventType)_field_\(eventFieldName)_type"): inst.command.fields[eventFieldName] & eventFieldType
						}
					}
				}
			}

			// Validate scenario command names match slice command
			for si, s in inst.scenarios {
				("automation_\(inst.name)_scenario\(si)_command_must_match"): s.when.name & inst.command.name
			}

			// Validate scenario given events are in query types (including dependent query)
			let _queryEventTypes = [
				for qi in inst.command.query.items
				for e in qi.types {e.eventType},
			]
			let _depQueryEventTypes = [
				if inst.command.dependentQuery != _|_
				for qi in inst.command.dependentQuery.items
				for e in qi.types {e.eventType},
			]
			let _allQueryEventTypes = list.Concat([_queryEventTypes, _depQueryEventTypes])
			for si, s in inst.scenarios {
				for ge in s.given {
					("automation_\(inst.name)_scenario\(si)_given_\(ge.eventType)_must_be_in_query"): list.Contains(_allQueryEventTypes, ge.eventType) & true
				}
			}

			// Validate scenario then.events are in emits (success scenarios only)
			let _emitEventTypes = [for e in inst.emits {e.eventType}]
			for si, s in inst.scenarios {
				if s.then.success == true {
					for te in s.then.events {
						("automation_\(inst.name)_scenario\(si)_then_\(te.eventType)_must_be_in_emits"): list.Contains(_emitEventTypes, te.eventType) & true
					}
				}
			}

			// Validate DCB query
			for qi in inst.command.query.items {
				// All tags must be defined
				for tref in qi.tags {
					_dcbTagValid: list.Contains(_tagList, tref._tagName) & true
				}

				// Satisfiability: EVERY event must have ALL required tags
				for e in qi.types {
					for tref in qi.tags {
						("automation_\(inst.name)_event_\(e.eventType)_must_have_tag_\(tref._tagName)"): list.Contains(_eventTagMap[e.eventType], tref._tagName) & true
					}
				}
			}

			// Validate dependent query if present
			if inst.command.dependentQuery != _|_ {
				for qi in inst.command.dependentQuery.items {
					// All tags must be defined
					for tref in qi.tags {
						_dcbTagValid: list.Contains(_tagList, tref._tagName) & true
					}

					// Satisfiability: EVERY event must have ALL required tags
					for e in qi.types {
						for tref in qi.tags {
							("automation_\(inst.name)_depquery_event_\(e.eventType)_must_have_tag_\(tref._tagName)"): list.Contains(_eventTagMap[e.eventType], tref._tagName) & true
						}
					}
				}
			}
		}
	}

	// Validate view slices
	for i, inst in flow if inst.kind == "slice" {
		if inst.type == "view" {
			// Validate DCB query
			for qi in inst.query.items {
				// All tags must be defined
				for tref in qi.tags {
					_dcbTagValid: list.Contains(_tagList, tref._tagName) & true
				}

				// Satisfiability: EVERY event must have ALL required tags
				for e in qi.types {
					for tref in qi.tags {
						("slice_\(inst.name)_event_\(e.eventType)_must_have_tag_\(tref._tagName)"): list.Contains(_eventTagMap[e.eventType], tref._tagName) & true
					}
				}
			}

			// ReadModel fields must come from queried events (including dependent query) or be computed/mapped
			// Dotted paths (e.g. "items.price") cover their parent field (e.g. "items")
			let _queriedEventFieldNames = [
				for qi in inst.query.items
				for evt in qi.types
				for k, _ in events[evt.eventType].fields {k},
			]
			let _depQueriedEventFieldNames = [
				if inst.dependentQuery != _|_
				for qi in inst.dependentQuery.items
				for evt in qi.types
				for k, _ in events[evt.eventType].fields {k},
			]
			let _allQueriedEventFieldNames = list.Concat([_queriedEventFieldNames, _depQueriedEventFieldNames])
			let _rmMappedFields = [for k, _ in inst.readModel.mapping {k}]
			let _rmComputedFields = [for k, _ in inst.readModel.computed {k}]
			for fieldName, _ in inst.readModel._schema {
				let inEvents = list.Contains(_allQueriedEventFieldNames, fieldName)
				let inMapping = list.Contains(_rmMappedFields, fieldName)
				let isComputed = list.Contains(_rmComputedFields, fieldName)
				// Check if any dotted path starts with this field name
				let _hasDottedMapping = list.Contains([for k, _ in inst.readModel.mapping if strings.HasPrefix(k, fieldName+".") {true}], true)
				let _hasDottedComputed = list.Contains([for k, _ in inst.readModel.computed if strings.HasPrefix(k, fieldName+".") {true}], true)
				("view_\(inst.name)_field_\(fieldName)_must_come_from_events_or_computed"): (inEvents | inMapping | isComputed | _hasDottedMapping | _hasDottedComputed) & true
			}

			// Validate computed fields: event must be queried (including dependent query), fields must exist in event
			let _queriedEventTypes = [
				for qi in inst.query.items
				for evt in qi.types {evt.eventType},
			]
			let _depQueriedEventTypes = [
				if inst.dependentQuery != _|_
				for qi in inst.dependentQuery.items
				for evt in qi.types {evt.eventType},
			]
			let _allQueriedEventTypes = list.Concat([_queriedEventTypes, _depQueriedEventTypes])
			for computedName, comp in inst.readModel.computed {
				("view_\(inst.name)_computed_\(computedName)_event_must_be_queried"): list.Contains(_allQueriedEventTypes, comp.event.eventType) & true

				let _eventFieldNames = [for k, _ in events[comp.event.eventType].fields {k}]
				for _, f in comp.fields {
					("view_\(inst.name)_computed_\(computedName)_field_\(f)_must_exist_in_event"): list.Contains(_eventFieldNames, f) & true
				}
			}

			// Validate mapped fields: event queried (including dependent query), field exists, type matches
			// Note: dotted paths (e.g. "items.price") are validated in Go, not here
			for mappedName, m in inst.readModel.mapping {
				("view_\(inst.name)_mapping_\(mappedName)_event_must_be_queried"): list.Contains(_allQueriedEventTypes, m.event.eventType) & true

				let _eventFieldNames = [for k, _ in events[m.event.eventType].fields {k}]
				("view_\(inst.name)_mapping_\(mappedName)_field_\(m.field)_must_exist_in_event"): list.Contains(_eventFieldNames, m.field) & true

				// Type must match between readModel field and event field (skip dotted paths - validated in Go)
				let _isDotted = strings.Contains(mappedName, ".")
				if !_isDotted {
					("view_\(inst.name)_mapping_\(mappedName)_type"): inst.readModel._schema[mappedName] & events[m.event.eventType].fields[m.field]
				}
			}

			// Validate view scenario given events are in query types (including dependent query)
			for si, s in inst.scenarios {
				for ge in s.given {
					("view_\(inst.name)_scenario\(si)_given_\(ge.eventType)_must_be_in_query"): list.Contains(_allQueriedEventTypes, ge.eventType) & true
				}
			}

			// Validate endpoint path params exist in params (only if endpoint defined)
			if inst.endpoint != _|_ {
				if strings.Contains(inst.endpoint.path, "{") {
					let _pathParams = [for m in regexp.FindAllSubmatch("\\{(\\w+)\\}", inst.endpoint.path, -1) {m[1]}]
					let _endpointParams = [for k, _ in inst.endpoint.params {k}]
					for p in _pathParams {
						("view_\(inst.name)_endpoint_path_param_\(p)_must_be_in_params"): list.Contains(_endpointParams, p) & true
					}
				}
			}

			// Validate dependent query if present
			if inst.dependentQuery != _|_ {
				for qi in inst.dependentQuery.items {
					// All tags must be defined
					for tref in qi.tags {
						_dcbTagValid: list.Contains(_tagList, tref._tagName) & true
					}

					// Satisfiability: EVERY event must have ALL required tags
					for e in qi.types {
						for tref in qi.tags {
							("view_\(inst.name)_depquery_event_\(e.eventType)_must_have_tag_\(tref._tagName)"): list.Contains(_eventTagMap[e.eventType], tref._tagName) & true
						}
					}
				}
			}
		}
	}

}
