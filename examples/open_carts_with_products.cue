package examples

import em "github.com/err0r500/event-modeling-dcb-spec/em"

OpenCartsWithProducts: em.#ViewSlice & {
	name:  "OpenCartsWithProducts"
	actor: _actors.User

	query: {
		items: [
			{
				types: [
					_events.CartCreated,
					_events.CartSubmitted,
					_events.ItemAdded,
					_events.CartCleared,
					_events.ItemRemoved,
				]
			},
		]
	}

	readModel: em.#ReadModel & {
		name:        "OpenCartsWithProducts"
		cardinality: "table"
		persistence: "persistent"

		columns: {
			productId: string
			cartId:    string
			itemId:    string
		}
	}

	scenarios: [
		{
			name: "one item"
			given: [
				_events.CartCreated & {fields: {cartId: "abc"}},
				_events.ItemAdded & {fields: {itemId: "item1", productId: "product-1"}},
			]
			expect: [{
				cartId:    "abc"
				itemId:    "item-1"
				productId: "prod-1"
			}]
		},
		{
			name: "two item, same product id"
			given: [
				_events.CartCreated & {fields: {cartId: "abc"}},
				_events.ItemAdded & {fields: {itemId: "item1", productId: "product-1"}},
				_events.ItemAdded & {fields: {itemId: "item2", productId: "product-1"}},
			]
			expect: [
				{
					cartId:    "abc"
					itemId:    "item-1"
					productId: "prod-1"
				},
				{
					cartId:    "abc"
					itemId:    "item-2"
					productId: "prod-1"
				},
			]
		},
		{
			name: "cart cleared"
			given: [
				_events.CartCreated,
				_events.ItemAdded,
				_events.CartCleared,
			]
			expect: []
		},
		{
			name: "cart submitted"
			given: [
				_events.CartCreated,
				_events.ItemAdded,
				_events.CartSubmitted,
			]
			expect: []
		},
	]
}
