package em

// #Field - Generic typed field schema placeholder
//
// Use CUE native types to define field schemas.
// Examples:
//   fields: { userId: string, quantity: int }
//   fields: { items: [...{name: string, price: number}] }
#Field: {
	[string]: _
}

// #Tag - Event stream partitioning tag
//
// Tags enable Dynamic Consistency Boundaries (DCB) by partitioning event
// streams. Events with the same tag values form a consistency boundary.
//
// Fields:
//   name: string - tag identifier (auto-filled from key in board.tags)
//   param?: string - if set, queries must provide a value for this tag
//   type: _ - type constraint for tag values (default: any)
//
// Examples:
//   Simple: {name: "cart"} - category tag, no value needed
//   Parameterized: {name: "cartId", param: "cartId", type: string}
//   Typed: {name: "quantity", param: "qty", type: int}
#Tag: {
	name:     string
	param?:   string // if set, queries must provide a value
	type:     _       // type constraint for values (e.g., string, int)
	_tagName: name
}

// #Event - Domain event definition
//
// Events are immutable facts that capture state changes. Closed to allow
// short-form EventInstance references.
//
// Fields:
//   eventType: string - unique event name (auto-filled from key in board.events)
//   fields: #Field - event payload schema
//   tags: [...#Tag] - tags for DCB partitioning (events with same tag values = boundary)
//   computed?: #Field - fields derived at emit time, not from command
//   mapping?: #Field - rename command fields when emitting (eventField -> cmdField)
#Event: close({
	eventType: string
	fields!:   #Field
	tags!: [...#Tag]
	// Fields not from command (computed) - field name → description
	computed: {[string]: string} | *{}
	// Field mapping: eventField -> command.fields.x (for emit overrides)
	mapping: #Field | *{}
	// For scenario testing: true if event occurs AFTER this slice (race conditions)
	fromFuture: bool | *false
})

// #Actor - Entity that triggers commands or consumes views
//
// Actors represent users, systems, or services that interact with the domain.
// Must be defined in board.actors and referenced by slices.
//
// Fields:
//   name: string - unique actor identifier (auto-filled from key in board.actors)
//
// Examples: "Customer", "Admin", "PaymentService", "ScheduledJob"
#Actor: {
	name: string
}

// #ExternalEvent - Event from outside the system that triggers a command
//
// External events come from other systems/domains and trigger commands.
// Command fields must come from external event fields, mapping, or computed.
//
// Fields:
//   name: string - external event identifier
//   source: string - origin system/context (e.g., "InventoryContext", "PricingContext")
//   fields: #Field - payload schema
#ExternalEvent: {
	name!:   string
	source!: string
	fields!: #Field
}

// #Trigger - Union of command trigger types
//
// Commands can be triggered by:
//   kind: "endpoint" -> HTTP request
//   kind: "externalEvent" -> event from external system
//   kind: "internalEvent" -> event from within this board (automation)
#Trigger: #EndpointTrigger | #ExternalEventTrigger | #InternalEventTrigger

// #AutomationTrigger - Triggers for automation slices (no endpoint)
//
// Automations can only be triggered by events, not HTTP endpoints.
#AutomationTrigger: #ExternalEventTrigger | #InternalEventTrigger

#EndpointTrigger: {
	kind:     "endpoint"
	endpoint: #Endpoint
}

#ExternalEventTrigger: {
	kind:          "externalEvent"
	externalEvent: #ExternalEvent
}

// #InternalEventTrigger - Automation trigger from internal event
//
// Triggers a command when an event is emitted within this board.
// Command fields must come from event fields, mapping, or computed.
// The event must have been emitted before this slice (causality).
#InternalEventTrigger: {
	kind:          "internalEvent"
	internalEvent: #Event
}

// #Command - Intent to change domain state
//
// Commands represent actions that may emit events. Fields must come from
// endpoint (params/body), mapping, or be computed.
//
// Fields:
//   name: string - command identifier (e.g., "AddToCart")
//   fields: #Field - command input schema (must match endpoint inputs)
//   query: #DCBQuery - events to load for consistency check before emitting
//   computed?: #Field - fields derived at runtime (e.g., timestamp, generated IDs)
//   mapping?: #Field - rename endpoint fields (cmdField: endpoint.params.x or endpoint.body.x)
#Command: {
	name:    string
	fields!: #Field

	query!: #DCBQuery
	// Fields not from endpoint (computed) - field name → description
	computed: {[string]: string} | *{}
	// Field mapping: cmdField -> endpoint.params.x or endpoint.body.x
	mapping: #Field | *{}
}

// #ComputedField - ReadModel field derived from event fields
//
// For view fields that aggregate or transform event data.
// No type checking performed (transformations may change types).
//
// Fields:
//   event: #Event - source event reference
//   fields: [...string] - event field names this computation derives from
//
// Example: count derived from ItemAdded events
#ComputedField: {
	event:  #Event
	fields: [...string] // event field names this value derives from
}

// #MappedField - ReadModel field renamed from event field
//
// For view fields that directly copy an event field with a different name.
// Type must match between readModel field and event field.
//
// Fields:
//   event: #Event - source event reference
//   field: string - event field name to copy from
//
// Example: displayName mapped from event.name
#MappedField: {
	event: #Event
	field: string // event field name this is mapped from
}

// #Chapter - Narrative segment of the event modeling timeline
//
// A chapter is a narrative cut in the timeline that makes the model
// readable for humans. Like a chapter in a scenario or an act in a film:
// a coherent mini-story from point A to point B.
//
// Fields:
//   name: string - chapter identifier
//   description?: string - optional narrative description
//   flow: [...#Instant] - ordered sequence of slices and story steps in this chapter
//
// Examples: "Sign-up", "First purchase", "Refund"
#Chapter: {
	name:        string
	description: string | *""
	flow: [...#Instant]
}

// #Context - Bounded context grouping on the event modeling board
//
// A context groups chapters that belong to the same responsibility boundary.
// Equivalent to a DDD bounded context expressed on the storyboard
// (swimlane or zone surrounding columns).
//
// Fields:
//   name: string - context identifier
//   description?: string - optional description of the context's responsibility
//   chapters: [...#Chapter] - ordered narrative chapters within this context
//
// Examples: "Billing", "Inventory", "Cart", "Payments"
#Context: {
	name:        string
	description: string | *""
	chapters: [...#Chapter]
}

// #ReadModel - View projection schema
//
// Defines what a view slice returns. Fields come from queried events,
// computed values, or mapped renames.
//
// Fields:
//   name: string - read model identifier
//   cardinality: "single" | "table" - one result or collection
//   fields: #Field - output schema
//   computed?: {[string]: #ComputedField} - aggregated/transformed fields
//   mapping?: {[string]: #MappedField} - renamed fields (type-checked)
#ReadModel: {
	name!:        string
	cardinality!: "single" | "table"
	persistence!: "transient" | "persistent" | "transient"
	if cardinality == "single" {
		fields!:  #Field
		_schema: fields
	}
	if cardinality == "table" {
		columns!: #Field
		_schema:  columns
	}
	// Fields derived/aggregated from events (no type check)
	computed: {[string]: #ComputedField} | *{}
	// Fields renamed from event fields (type must match)
	mapping: {[string]: #MappedField} | *{}
}
