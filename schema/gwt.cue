package schema

// #EventInstance - Reference to an event with optional field values
//
// Usage:
//   Bare event: _events.CartCreated
//   With values: _events.CartCreated & {fields: {cartId: "abc"}}
//   Race condition: _events.CartCreated & {fromFuture: true}
//
// Field values are type-checked against the event's fields definition.
#EventInstance: #Event

// #Outcome - Union of scenario results (success or error)
//
// Use success field to discriminate:
//   success: true -> #SuccessOutcome (events emitted)
//   success: false -> #ErrorOutcome (error message)
#Outcome: #SuccessOutcome | #ErrorOutcome

// #SuccessOutcome - Command succeeded and emitted events
//
// Fields:
//   success: true - discriminator
//   events: [...#EventInstance] - events emitted on success
#SuccessOutcome: {
	success: true
	// Events emitted on success, e.g. [_events.CartCreated]
	events: [...#EventInstance]
}

// #ErrorOutcome - Command failed with error
//
// Fields:
//   success: false - discriminator
//   error: string - error message describing failure reason
#ErrorOutcome: {
	success: false
	// Error message describing why command failed
	error: string
}

// #GWTSuccess - Given/When/Then scenario for success cases
//
// Tests that a command succeeds and emits expected events given prior state.
//
// Fields:
//   name: string - scenario description
//   given: [...#EventInstance] - prior events (empty [] for fresh state)
//   when: #Field - command input values (validated against command.fields in slice)
//   then: #SuccessOutcome - expected success with emitted events
//
// Example:
//   {name: "Add first item", given: [_events.CartCreated],
//    when: {quantity: 1},
//    then: {success: true, events: [_events.ItemAdded]}}
#GWTSuccess: {
	// Scenario name - describes the business case being tested
	name: string
	// Prior events that set up the state (empty [] for fresh state)
	given: [...#EventInstance]
	// Command input values (validated against command.fields in slice)
	when: #Field
	// Expected success outcome with emitted events
	then: #SuccessOutcome
}

// #GWTError - Given/When/Then scenario for error cases
//
// Tests that a command fails with expected error given prior state.
// Can include fromFuture events to test race conditions.
//
// Fields:
//   name: string - scenario description
//   given: [...#EventInstance] - prior events (can include fromFuture: true)
//   when: #Field - command input values (validated against command.fields in slice)
//   then: #ErrorOutcome - expected error with message
//
// Example:
//   {name: "Cannot add to closed cart",
//    given: [_events.CartCreated, _events.CartClosed],
//    when: {},
//    then: {success: false, error: "cart is closed"}}
#GWTError: {
	// Scenario name - describes the error case being tested
	name: string
	// Prior events (can include fromFuture: true for race conditions)
	given: [...#EventInstance]
	// Command input values (validated against command.fields in slice)
	when: #Field
	// Expected error outcome
	then: #ErrorOutcome
}

// #GWT - Given/When/Then scenario union (success or error)
//
// Use then.success to discriminate:
//   then.success: true -> #GWTSuccess
//   then.success: false -> #GWTError
#GWT: #GWTSuccess | #GWTError

// #ViewScenario - Test case for a view/read model
//
// Tests that given events produce expected view output.
//
// Fields:
//   name: string - scenario description
//   given: [...#EventInstance] - events that build view state
//   query?: #Field - query parameters
//   expect: string - description of expected result
//
// Example:
//   {name: "Show cart with 2 items",
//    given: [_events.CartCreated, _events.ItemAdded, _events.ItemAdded],
//    query: {cartId: "abc"},
//    expect: "Returns cart with 2 items and total price"}
#ViewScenario: {
	// Scenario name
	name: string
	// Prior events that build up the view state
	given: [...#EventInstance]
	// Query parameters for the view
	query: #Field | *{}
	// Description of expected result
	expect: string
}
