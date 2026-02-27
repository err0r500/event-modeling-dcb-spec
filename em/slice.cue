package em

// #SliceType - Discriminator for slice behavior
//
// Values:
//   "change" - command slice that emits events (writes)
//   "view" - query slice that reads events (reads)
//   "automation" - event-triggered automation (no actor)
#SliceType: "change" | "view" | "automation"

#DevStatus: "specifying" | "todo" | "doing" | "done"

// #SliceBase - Common fields shared by all slices
//
// Fields:
//   kind: "slice" - discriminator for Instant union
//   name: string - unique slice identifier within the board
//   type: #SliceType - "change" or "view"
//   actor: #Actor - who triggers this slice (must exist in board.actors)
#SliceBase: {
	kind:   "slice"
	name!:  string
	type:   #SliceType
	actor!: #Actor
    devstatus: #DevStatus | *"specifying"
}

// #ChangeSlice - Command that emits events (write operation)
//
// Represents a state-changing action. The command queries existing events
// for consistency, then emits new events on success.
//
// Fields:
//   trigger: #Trigger - endpoint or externalEvent that triggers this command
//   command: #Command - the change intent with fields and DCB query
//   emits: [...#Event] - events this command can emit (must be defined in board.events)
//   scenarios: [...#GWT] - Given/When/Then test cases for this command
//
// Validation:
//   - command.fields must come from trigger fields, mapping, or command.computed
//   - emitted event fields must come from command.fields, mapping, or computed
//   - scenario command names must match slice command
#ChangeSlice: {
	#SliceBase
	type: "change"

	// Optional relative path to mockup/screenshot
	image?: string

	trigger!: #Trigger

	command!: #Command & {name: name}

	// Events emitted by this command
	emits!: [...#Event]

	// GWT scenarios with when validated against command.fields
	scenarios: [...#GWT & {when: command.fields}] | *[]
}

// #ViewSlice - Query that reads events (read operation)
//
// Represents a read-only projection. Queries events emitted by prior
// change slices in the flow to build a read model.
//
// Fields:
//   name: string - unique slice identifier within the board
//   actor: #Actor - who triggers this slice (must exist in board.actors)
//   endpoint: #Endpoint - HTTP API surface for this slice
//   readModel: #ReadModel - output schema with cardinality
//   query: #DCBQuery - which events to project (must be emitted before this slice)
//   dependentQuery?: #DependentQuery - optional second-phase query using extracted values
//   scenarios: [...] - test cases with type-checked expect matching readModel.fields
//
// Validation:
//   - queried events must be emitted by earlier change slices in flow
//   - readModel.fields must come from queried events, computed, or mapping
#ViewSlice: {
	#SliceBase
	type: "view"

	// Optional relative path to mockup/screenshot
	image?: string

	endpoint?: #Endpoint

	readModel!: #ReadModel

	// DCB query for events to build this view
	query!: #DCBQuery

	// Optional dependent query using values extracted from primary query
	dependentQuery?: #DependentQuery

	// Validate: dependentQuery.extract events must be in primary query.items[*].types
	if dependentQuery != _|_ {
		_queryEvents: or([ for item in query.items for e in item.types {e.eventType}])
		_validateExtractEvents: [
			for _, ext in dependentQuery.extract {
				ext.event.eventType & _queryEvents
			}
		]
	}

	// View scenarios with type-checked expect against readModel.fields/columns
	// name, given, query, expect
	// expect is a list for table cardinality, single struct for single cardinality
	scenarios!: [...{
		name:  string
		given: [...#EventInstance]
		query: #Field | *{}
		if readModel.cardinality == "single" {
			expect: readModel.fields
		}
		if readModel.cardinality == "table" {
			expect: [...readModel.columns]
		}
	}]
}

// #AutomationSlice - Event-triggered automation (no actor)
//
// Represents an automation triggered by an event, with no user actor.
// Renders in a dedicated automation lane.
//
// Fields:
//   trigger: #AutomationTrigger - external or internal event (no endpoint)
//   consumes: [...#ViewSlice] - views whose readModel fields are available to command
//   command: #Command - the automation logic (fields from trigger + consumed views)
//   emits: [...#Event] - events this automation can emit
//   scenarios: [...#GWT] - Given/When/Then test cases
#AutomationSlice: {
	kind:      "slice"
	name!:     string
	type:      "automation"
	devstatus: #DevStatus | *"specifying"

	// Optional relative path to mockup/screenshot
	image?: string

	trigger!: #AutomationTrigger

	// ReadModels consumed by this automation - their fields available to command
	consumes: [...#ReadModel] | *[]

	command!: #Command & {name: name}

	// Events emitted by this automation
	emits!: [...#Event]

	// GWT scenarios with when validated against command.fields
	scenarios: [...#GWT & {when: command.fields}] | *[]
}

// #Slice - Union of slice types (change, view, or automation)
//
// Use type field to discriminate:
//   type: "change" -> #ChangeSlice
//   type: "view" -> #ViewSlice
//   type: "automation" -> #AutomationSlice
#Slice: #ChangeSlice | #ViewSlice | #AutomationSlice

// #Instant - A moment in the event modeling flow
//
// The flow is an ordered list of instants. Each instant is either:
//   kind: "slice" -> #Slice (change or view operation)
//   kind: "story" -> #StoryStep (narrative reference to existing slice)
//
// Ordering matters: view slices can only query events emitted by
// earlier change slices in the flow.
#Instant: #Slice | #StoryStep
