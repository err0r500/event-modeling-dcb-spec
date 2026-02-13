package examples

import "github.com/fairway/eventmodelingspec/schema"

RemoveItem: schema.#ChangeSlice & {
	name:  "RemoveItem"
	actor: _actors.User

    image: "./mockups/one_item_cart.png"

	trigger: schema.#EndpointTrigger & {
		endpoint: {
			verb: "DELETE"
			params: {cartId: string, itemId: string}
			path: "/carts/{cartId}/items/{itemId}"
		}
	}

	command: {
		fields: {
			cartId: string
			itemId: string
		}
		query: {
			items: [{
				types: [_events.CartCreated, _events.ItemAdded, _events.ItemRemoved]
				tags: [{tag: _tags.cart_id, value: fields.cartId}]
			}]
		}
	}

	emits: [
		_events.ItemRemoved,
	]

	scenarios: [
		{
			name: "OK: happy case"
			given: [
				_events.CartCreated & {fields: {cartId: "abc"}},
				_events.ItemAdded & {fields: {cartId: "abc", itemId: "item1"}},
			]
			when: {cartId: "abc", itemId: "item1"}
			then: {
				success: true
				events: [_events.ItemRemoved]
			}
		},
		{
			name: "OK: already removed"
			given: [
				_events.CartCreated & {fields: {cartId: "abc"}},
				_events.ItemAdded & {fields: {cartId: "abc", itemId: "item1"}},
				_events.ItemRemoved & {fields: {cartId: "abc", itemId: "item1"}},
			]
			when: {cartId: "abc", itemId: "item1"}
			then: {
				success: true
				events: []
			}
		},

		{
			name: "Err: wrong cart"
			given: [
				_events.CartCreated & {fields: {cartId: "abc"}},
				_events.ItemAdded & {fields: {cartId: "def", itemId: "item1"}},
			]
			when: {cartId: "abc", itemId: "item1"}
			then: {
				success: false
				error: "the item doesn't belong to this cart"
			}
		},
	]
}
