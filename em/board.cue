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
// Fields:
//   name: string - board identifier
//   tags: {[Name]: #Tag} - all tags for DCB partitioning (key becomes tag.name)
//   events: {[Type]: #Event} - all events (key becomes event.eventType)
//   actors: {[Name]: #Actor} - all actors (key becomes actor.name)
//   flow: [...#Instant] - ordered sequence of slices and story steps
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

	// Bounded contexts (responsibility boundaries)
	contexts: [Name=string]: #Context & {name: Name} | *{}

	// Narrative chapters (timeline segments)
	chapters: [Name=string]: #Chapter & {name: Name} | *{}

	// Ordered flow of instants (slices and story steps)
	flow: [...#Instant]

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

	// Context list
	_contextList: [for k, _ in contexts {k}]

	// Chapter list
	_chapterList: [for k, _ in chapters {k}]

	// Map: eventType -> list of tag names
	_eventTagMap: {for k, e in events {(k): [for t in e.tags {t.name}]}}

	// --- VALIDATION ---

	// Build set of emitted eventTypes up to each flow index
	// Only change slices emit events
	_emittedBefore: {
		"0": {}
		for i, inst in flow if i > 0 {
			"\(i)": _emittedBefore["\(i-1)"] & {
				if flow[i-1].kind == "slice" {
					if flow[i-1].type == "change" {
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

	// Validate slices - actor must be defined
	for i, inst in flow if inst.kind == "slice" {
		_actorValid: list.Contains(_actorList, inst.actor.name) & true
	}

	// Validate slices - context must be defined (if set)
	for i, inst in flow if inst.kind == "slice" {
		if inst.context != _|_ {
			_contextValid: list.Contains(_contextList, inst.context) & true
		}
	}

	// Validate slices - chapter must be defined (if set)
	for i, inst in flow if inst.kind == "slice" {
		if inst.chapter != _|_ {
			_chapterValid: list.Contains(_chapterList, inst.chapter) & true
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

			// Endpoint trigger: fields from params/body
			if inst.trigger.kind == "endpoint" {
				let _paramFields = [for k, _ in inst.trigger.endpoint.params {k}]
				let _bodyFields = [for k, _ in inst.trigger.endpoint.body {k}]
				for fieldName, fieldType in inst.command.fields {
					let inParams = list.Contains(_paramFields, fieldName)
					let inBody = list.Contains(_bodyFields, fieldName)
					let isComputed = list.Contains(_computedFields, fieldName)
					let inMapping = list.Contains(_mappedFields, fieldName)
					("slice_\(inst.name)_field_\(fieldName)_must_come_from_trigger"): (inParams | inBody | isComputed | inMapping) & true

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

			// Validate scenario given events are in query types
			let _queryEventTypes = [
				for qi in inst.command.query.items
				for e in qi.types {e.eventType},
			]
			for si, s in inst.scenarios {
				for ge in s.given {
					("slice_\(inst.name)_scenario\(si)_given_\(ge.eventType)_must_be_in_query"): list.Contains(_queryEventTypes, ge.eventType) & true
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

				// Ordering: all events in query must be emitted before
				for e in qi.types {
					("slice_\(inst.name)_event_\(e.eventType)_must_be_emitted_before"): list.Contains(_emittedBeforeLists["\(i)"], e.eventType) & true
				}
			}

			// ReadModel fields must come from queried events or be computed/mapped
			// Dotted paths (e.g. "items.price") cover their parent field (e.g. "items")
			let _queriedEventFieldNames = [
				for qi in inst.query.items
				for evt in qi.types
				for k, _ in events[evt.eventType].fields {k},
			]
			let _rmMappedFields = [for k, _ in inst.readModel.mapping {k}]
			let _rmComputedFields = [for k, _ in inst.readModel.computed {k}]
			for fieldName, _ in inst.readModel.fields {
				let inEvents = list.Contains(_queriedEventFieldNames, fieldName)
				let inMapping = list.Contains(_rmMappedFields, fieldName)
				let isComputed = list.Contains(_rmComputedFields, fieldName)
				// Check if any dotted path starts with this field name
				let _hasDottedMapping = list.Contains([for k, _ in inst.readModel.mapping if strings.HasPrefix(k, fieldName+".") {true}], true)
				let _hasDottedComputed = list.Contains([for k, _ in inst.readModel.computed if strings.HasPrefix(k, fieldName+".") {true}], true)
				("view_\(inst.name)_field_\(fieldName)_must_come_from_events_or_computed"): (inEvents | inMapping | isComputed | _hasDottedMapping | _hasDottedComputed) & true
			}

			// Validate computed fields: event must be queried, fields must exist in event
			let _queriedEventTypes = [
				for qi in inst.query.items
				for evt in qi.types {evt.eventType},
			]
			for computedName, comp in inst.readModel.computed {
				("view_\(inst.name)_computed_\(computedName)_event_must_be_queried"): list.Contains(_queriedEventTypes, comp.event.eventType) & true

				let _eventFieldNames = [for k, _ in events[comp.event.eventType].fields {k}]
				for _, f in comp.fields {
					("view_\(inst.name)_computed_\(computedName)_field_\(f)_must_exist_in_event"): list.Contains(_eventFieldNames, f) & true
				}
			}

			// Validate mapped fields: event queried, field exists, type matches
			// Note: dotted paths (e.g. "items.price") are validated in Go, not here
			for mappedName, m in inst.readModel.mapping {
				("view_\(inst.name)_mapping_\(mappedName)_event_must_be_queried"): list.Contains(_queriedEventTypes, m.event.eventType) & true

				let _eventFieldNames = [for k, _ in events[m.event.eventType].fields {k}]
				("view_\(inst.name)_mapping_\(mappedName)_field_\(m.field)_must_exist_in_event"): list.Contains(_eventFieldNames, m.field) & true

				// Type must match between readModel field and event field (skip dotted paths - validated in Go)
				let _isDotted = strings.Contains(mappedName, ".")
				if !_isDotted {
					("view_\(inst.name)_mapping_\(mappedName)_type"): inst.readModel.fields[mappedName] & events[m.event.eventType].fields[m.field]
				}
			}

			// Validate view scenario given events are in query types
			for si, s in inst.scenarios {
				for ge in s.given {
					("view_\(inst.name)_scenario\(si)_given_\(ge.eventType)_must_be_in_query"): list.Contains(_queriedEventTypes, ge.eventType) & true
				}
			}

			// Validate endpoint path params exist in params
			if strings.Contains(inst.endpoint.path, "{") {
				let _pathParams = [for m in regexp.FindAllSubmatch("\\{(\\w+)\\}", inst.endpoint.path, -1) {m[1]}]
				let _endpointParams = [for k, _ in inst.endpoint.params {k}]
				for p in _pathParams {
					("view_\(inst.name)_endpoint_path_param_\(p)_must_be_in_params"): list.Contains(_endpointParams, p) & true
				}
			}
		}
	}

}
