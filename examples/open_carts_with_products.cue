package examples

import em "github.com/err0r500/event-modeling-dcb-spec/em"

OpenCartsWithProducts: em.#ViewSlice & {
	name:  "OpenCartsWithProducts"
	actor: _actors.User

	endpoint: em.#Endpoint & {
		verb: "GET"
		path: "/open-carts/{productId}"
		params: {productId: string}
	}

	query: {
		items: [
			{
				types: [_events.CartCreated, _events.CartSubmitted, _events.ItemAdded, _events.CartCleared, _events.ItemRemoved]
                tags: []
			},
		]
	}

	readModel: em.#ReadModel & {
		name:        "OpenCartsWithProducts"
		cardinality: "table"
		fields: {
			cartId: string
			itemId: string
		}
		mapping: {
			cartId: {event: _events.ItemAdded, field: "cartId"}
			itemId: {event: _events.ItemAdded, field: "itemId"}
		}
	}

	scenarios: []
}
