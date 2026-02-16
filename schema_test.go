package eventmodelingspec

import (
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/err0r500/event-modeling-dcb-spec/pkg/render"
)

func TestValidBoard(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

_events: [Type=string]: em.#Event & {eventType: Type}
_events: {
	EventA: {fields: {}, tags: []}
}

_sliceA: em.#ChangeSlice & {
	kind: "slice"
	name: "SliceA"
	type: "change"
	actor: {name: "User"}
	trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/test"}}
	command: {name: "CmdA", fields: {}, query: {items: []}}
	emits: [_events.EventA]
	scenarios: []
}

board: em.#Board & {
	name: "Test Board"
	tags: {
		mytag: {name: "mytag"}
	}
	events: _events
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				_sliceA,
				{
					kind: "story"
					name: "story step"
					slice: _sliceA
					description: "Test story"
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidDCBTag(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {
		realtag: {name: "realtag"}
	}
	events: {
		TestEvent: {eventType: "TestEvent", fields: {}, tags: [{name: "realtag"}]}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [{
				kind: "slice"
				name: "TestSlice"
				type: "change"
				actor: {name: "User"}
				trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/test"}}
				command: {
					name: "TestCmd"
					fields: {}
					query: {
						items: [{
							types: [events.TestEvent]
							tags: [{tag: {name: "faketag"}}]  // doesn't exist
						}]
					}
				}
				emits: [events.TestEvent]
				scenarios: []
			}]
		}]
	}]
}
`
	assertInvalid(t, src, "_dcbTagValid")
}

func TestValidCommandQueriesEventEmittedLater(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {}, tags: []}
		EventB: {eventType: "EventB", fields: {}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "SliceOne"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/one"}}
					command: {
						name: "CmdOne"
						fields: {}
						query: {
							items: [{
								types: [events.EventB]  // EventB emitted later - now allowed
								tags: []
							}]
						}
					}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "SliceTwo"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/two"}}
					command: {
						name: "CmdTwo"
						fields: {}
						query: {
							items: [{
								types: [events.EventA]
								tags: []
							}]
						}
					}
					emits: [events.EventB]
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidStoryRefNonexistentSlice(t *testing.T) {
	// With direct slice references, CUE catches undefined references at build time
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		TestEvent: {eventType: "TestEvent", fields: {}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "RealSlice"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/test"}}
					command: {name: "TestCmd", fields: {}, query: {items: []}}
					emits: [events.TestEvent]
					scenarios: []
				},
				{
					kind: "story"
					name: "invalid story"
					slice: _nonExistent  // CUE catches undefined reference
					description: "Invalid"
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "_nonExistent")
}

func TestValidStoryRefFutureSlice(t *testing.T) {
	// Direct CUE references work regardless of declaration order
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

_events: [Type=string]: em.#Event & {eventType: Type}
_events: {
	TestEvent: {fields: {}, tags: []}
}

_futureSlice: em.#ChangeSlice & {
	kind: "slice"
	name: "FutureSlice"
	type: "change"
	actor: {name: "User"}
	trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/test"}}
	command: {name: "TestCmd", fields: {}, query: {items: []}}
	emits: [_events.TestEvent]
	scenarios: []
}

board: em.#Board & {
	name: "Test"
	tags: {}
	events: _events
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "story"
					name: "story before slice"
					slice: _futureSlice  // OK to reference slice defined elsewhere
					description: "Valid"
				},
				_futureSlice,
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidActorNotDefined(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		TestEvent: {eventType: "TestEvent", fields: {}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [{
				kind: "slice"
				name: "TestSlice"
				type: "change"
				actor: {name: "Admin"}  // Admin not defined in actors
				trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/test"}}
				command: {name: "TestCmd", fields: {}, query: {items: []}}
				emits: [events.TestEvent]
				scenarios: []
			}]
		}]
	}]
}
`
	assertInvalid(t, src, "_actorValid")
}

func TestValidFutureEventInGWT(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

gwt: em.#GWT & {
	name: "Valid GWT with future event"
	given: [{
		eventType: "SomeEvent"
		fields: {}
		tags: []
		fromFuture: true  // allowed in both success and error scenarios
	}]
	when: {}
	then: {
		success: false
		error: "Expected error"
	}
}
`
	assertValid(t, src)
}

func TestValidFieldsFromEndpoint(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		PaymentMade: {eventType: "PaymentMade", fields: {userId: string, amount: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [{
				kind: "slice"
				name: "TestSlice"
				type: "change"
				actor: {name: "User"}
				trigger: {kind: "endpoint", endpoint: {
					verb: "POST"
					params: {userId: string}
					body: {amount: int}
					path: "/users/{userId}/pay"
				}}
				command: {
					name: "Pay"
					fields: {
						userId: string
						amount: int
					}
					query: {items: []}
				}
				emits: [events.PaymentMade]
				scenarios: []
			}]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestValidViewReadModelFromEvents(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string, amount: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {amount: int}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string, amount: int}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, amount: int}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidViewReadModelFieldNotFromEvents(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, bogusField: string}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "view_ReadA_field_bogusField_must_come_from_events_or_computed")
}

func TestValidViewReadModelWithComputed(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string, amount: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {amount: int}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string, amount: int}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, totalSpent: int}
						computed: {totalSpent: {event: events.EventA, fields: ["amount"]}}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidComputedFieldNotInEvent(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, total: int}
						computed: {total: {event: events.EventA, fields: ["bogusField"]}}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "view_ReadA_computed_total_field_bogusField_must_exist_in_event")
}

func TestInvalidComputedEventNotQueried(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
		EventB: {eventType: "EventB", fields: {amount: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "EmitA"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/a"}}
					command: {name: "CmdA", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "EmitB"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {amount: int}, path: "/b"}}
					command: {name: "CmdB", fields: {amount: int}, query: {items: []}}
					emits: [events.EventB]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, total: int}
						// EventB is NOT in query — only EventA is queried
						computed: {total: {event: events.EventB, fields: ["amount"]}}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "view_ReadA_computed_total_event_must_be_queried")
}

func TestValidViewMappingTypeMatch(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string, amount: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {amount: int}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string, amount: int}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, totalPrice: int}
						mapping: {totalPrice: {event: events.EventA, field: "amount"}}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidViewMappingTypeMismatch(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string, amount: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {amount: int}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string, amount: int}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, totalPrice: string}
						// totalPrice is string but amount is int — type mismatch
						mapping: {totalPrice: {event: events.EventA, field: "amount"}}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "view_ReadA_mapping_totalPrice_type")
}

func TestInvalidViewMappingFieldNotInEvent(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {
						name: "ViewA"
						cardinality: "single"
						fields: {userId: string, total: int}
						mapping: {total: {event: events.EventA, field: "noSuchField"}}
					}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "view_ReadA_mapping_total_field_noSuchField_must_exist_in_event")
}

func TestValidDottedPathMapping(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		ItemAdded: {eventType: "ItemAdded", fields: {itemId: string, price: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {itemId: string, price: int}, path: "/items"}}
					command: {name: "AddItem", fields: {itemId: string, price: int}, query: {items: []}}
					emits: [events.ItemAdded]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ViewItems"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/items"}
					readModel: {
						name: "ItemsView"
						cardinality: "single"
						fields: {
							items: [{itemId: string, price: int}]
						}
						mapping: {
							"items.itemId": {event: events.ItemAdded, field: "itemId"}
							"items.price": {event: events.ItemAdded, field: "price"}
						}
					}
					query: {items: [{types: [events.ItemAdded], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidDottedPathMapping(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		ItemAdded: {eventType: "ItemAdded", fields: {itemId: string, price: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {itemId: string, price: int}, path: "/items"}}
					command: {name: "AddItem", fields: {itemId: string, price: int}, query: {items: []}}
					emits: [events.ItemAdded]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ViewItems"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/items"}
					readModel: {
						name: "ItemsView"
						cardinality: "single"
						fields: {
							items: [{itemId: string, price: int}]
						}
						mapping: {
							"items.nonexistent": {event: events.ItemAdded, field: "itemId"}
						}
					}
					query: {items: [{types: [events.ItemAdded], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalidGo(t, src, "items.nonexistent", "must resolve to a field")
}

func TestInvalidDottedPathTypeMismatch(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		ItemAdded: {eventType: "ItemAdded", fields: {itemId: string, price: int}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {itemId: string, price: int}, path: "/items"}}
					command: {name: "AddItem", fields: {itemId: string, price: int}, query: {items: []}}
					emits: [events.ItemAdded]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ViewItems"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/items"}
					readModel: {
						name: "ItemsView"
						cardinality: "single"
						fields: {
							items: [{itemId: string, price: int}]
						}
						mapping: {
							// itemId is string but price is int - type mismatch
							"items.itemId": {event: events.ItemAdded, field: "price"}
						}
					}
					query: {items: [{types: [events.ItemAdded], tags: []}]}
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalidGo(t, src, "items.itemId", "must match source event field type")
}

// Helper functions

// buildResult holds the CUE value and any errors from loading/building
type buildResult struct {
	value cue.Value
	err   error
}

func buildValue(t *testing.T, src string) buildResult {
	t.Helper()

	absDir, err := filepath.Abs(".")
	if err != nil {
		return buildResult{err: err}
	}

	overlay := map[string]load.Source{
		filepath.Join(absDir, "test_input.cue"): load.FromString(src),
	}

	cfg := &load.Config{
		Dir:     absDir,
		Overlay: overlay,
	}

	instances := load.Instances([]string{"./test_input.cue"}, cfg)
	if len(instances) == 0 {
		return buildResult{err: err}
	}

	inst := instances[0]
	if inst.Err != nil {
		return buildResult{err: inst.Err}
	}

	ctx := cuecontext.New()
	v := ctx.BuildInstance(inst)
	if v.Err() != nil {
		return buildResult{err: v.Err()}
	}

	return buildResult{value: v}
}

func assertValid(t *testing.T, src string) {
	t.Helper()
	res := buildValue(t, src)
	if res.err != nil {
		t.Errorf("expected valid, got build error: %v", res.err)
		return
	}
	if err := res.value.Validate(cue.Concrete(false)); err != nil {
		t.Errorf("expected valid, got validation error: %v", err)
	}
}

func assertInvalid(t *testing.T, src string, expectedErrContains string) {
	t.Helper()
	res := buildValue(t, src)

	// Check build error first
	if res.err != nil {
		if !strings.Contains(res.err.Error(), expectedErrContains) {
			t.Errorf("expected error containing %q, got build error: %v", expectedErrContains, res.err)
		}
		return
	}

	// Check validation error
	err := res.value.Validate(cue.Concrete(false))
	if err == nil {
		t.Errorf("expected invalid (containing %q), but validation passed", expectedErrContains)
		return
	}
	if !strings.Contains(err.Error(), expectedErrContains) {
		t.Errorf("expected error containing %q, got validation error: %v", expectedErrContains, err)
	}
}

func assertInvalidGo(t *testing.T, src string, pathContains string, errContains string) {
	t.Helper()
	res := buildValue(t, src)
	if res.err != nil {
		t.Errorf("expected valid CUE build, got build error: %v", res.err)
		return
	}

	// CUE validation should pass (dotted paths not checked by CUE)
	if err := res.value.Validate(cue.Concrete(false)); err != nil {
		t.Errorf("expected CUE validation to pass, got: %v", err)
		return
	}

	// Go validation should catch it
	board := res.value.LookupPath(cue.ParsePath("board"))
	errs := render.ValidateBoard(board)
	if len(errs) == 0 {
		t.Errorf("expected Go validation error containing %q and %q, but none found", pathContains, errContains)
		return
	}

	found := false
	for _, e := range errs {
		if strings.Contains(e, pathContains) && strings.Contains(e, errContains) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error containing %q and %q, got: %v", pathContains, errContains, errs)
	}
}

func TestValidPathParamInParams(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "CreateUser"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/users/{userId}"}}
					command: {name: "Cmd", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidPathParamMissing(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "CreateUser"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/users/{userId}"}}
					command: {name: "Cmd", fields: {}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "path_param_userId")
}

func TestValidViewScenarioGivenInQuery(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "Emit"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {name: "View", cardinality: "single", fields: {userId: string}}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: [
						{name: "ok", given: [events.EventA], query: {}, expect: {userId: "abc"}}
					]
				},
			]
		}]
	}]
}
`
	assertValid(t, src)
}

func TestInvalidViewScenarioGivenNotInQuery(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {userId: string}, tags: []}
		EventB: {eventType: "EventB", fields: {}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [{
		name: "Default"
		chapters: [{
			name: "Main"
			flow: [
				{
					kind: "slice"
					name: "EmitA"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {userId: string}, body: {}, path: "/test"}}
					command: {name: "Cmd", fields: {userId: string}, query: {items: []}}
					emits: [events.EventA]
					scenarios: []
				},
				{
					kind: "slice"
					name: "EmitB"
					type: "change"
					actor: {name: "User"}
					trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/test2"}}
					command: {name: "Cmd2", fields: {}, query: {items: []}}
					emits: [events.EventB]
					scenarios: []
				},
				{
					kind: "slice"
					name: "ReadA"
					type: "view"
					actor: {name: "User"}
					endpoint: {verb: "GET", params: {}, body: {}, path: "/test"}
					readModel: {name: "View", cardinality: "single", fields: {userId: string}}
					query: {items: [{types: [events.EventA], tags: []}]}
					scenarios: [
						{name: "bad", given: [events.EventB], query: {}, expect: {userId: "abc"}}
					]
				},
			]
		}]
	}]
}
`
	assertInvalid(t, src, "given_EventB_must_be_in_query")
}

func TestValidBoardWithMultipleContextsAndChapters(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {
		EventA: {eventType: "EventA", fields: {}, tags: []}
		EventB: {eventType: "EventB", fields: {}, tags: []}
	}
	actors: {
		User: {name: "User"}
	}
	contexts: [
		{
			name: "Billing"
			description: "Handles payments"
			chapters: [
				{
					name: "Onboarding"
					description: "User signs up"
					flow: [{
						kind: "slice"
						name: "SliceA"
						type: "change"
						actor: {name: "User"}
						trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/a"}}
						command: {name: "CmdA", fields: {}, query: {items: []}}
						emits: [events.EventA]
						scenarios: []
					}]
				},
			]
		},
		{
			name: "Shipping"
			description: "Handles deliveries"
			chapters: [
				{
					name: "Fulfillment"
					flow: [{
						kind: "slice"
						name: "SliceB"
						type: "change"
						actor: {name: "User"}
						trigger: {kind: "endpoint", endpoint: {verb: "POST", params: {}, body: {}, path: "/b"}}
						command: {name: "CmdB", fields: {}, query: {items: []}}
						emits: [events.EventB]
						scenarios: []
					}]
				},
			]
		},
	]
}
`
	assertValid(t, src)
}

func TestValidBoardWithEmptyContexts(t *testing.T) {
	src := `
package test

import "github.com/err0r500/event-modeling-dcb-spec/em"

board: em.#Board & {
	name: "Test"
	tags: {}
	events: {}
	actors: {}
	contexts: []
}
`
	assertValid(t, src)
}
