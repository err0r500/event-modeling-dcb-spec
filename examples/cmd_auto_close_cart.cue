package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

// AutoCloseCart - Automation triggered by CartSubmitted event
// Demonstrates automation slice (no actor, event-triggered)
AutoCloseCart: em.#AutomationSlice & {
	name: "AutoCloseCart"

	trigger: em.#InternalEventTrigger & {
		internalEvent: _events.CartSubmitted
	}

	command: em.#Command & {
		fields: em.#Field & {
			cartId: string
		}
		query: {
			items: [{
				types: [_events.CartSubmitted]
				tags: [_tags.cart_id]
			}]
		}
	}

	emits: [_events.CartClosed]
}
