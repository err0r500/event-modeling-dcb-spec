package examples

import (
	s "github.com/err0r500/event-modeling-dcb-spec/em"
	"list"
)

ViewCartItems: s.#ViewSlice & {
	name:  "ViewCartItems"
	actor: _actors.User

	endpoint: s.#Endpoint & {
        verb: "GET"
		params: {
			cartId: string
		}
		path: "/carts/{cartId}"
	}

	query: {
		items: [
        {
			types: [_events.CartCreated, _events.ItemAdded, _events.ItemRemoved, _events.CartCleared]
			tags: [{tag: _tags.cart_id, value: endpoint.params.cartId}]
		},
        {types: [_events.ItemAdded], tags: []},
        ]
	}

	readModel: s.#ReadModel & {
		name:        "CartItemsView"
		cardinality: "single"
		fields: {
			cartId: string
			items: [...{
				itemId:    string
				productId: string
				price:     int
				quantity:  int
			}]
			totalPrice: int
		}
		mapping: {
			"items.itemId": {event: _events.ItemAdded, field: "itemId"}
			"items.productId": {event: _events.ItemAdded, field: "productId"}
			"items.price": {event: _events.ItemAdded, field: "price"}
			"items.quantity": {event: _events.ItemAdded, field: "quantity"}
			totalPrice: {event: _events.ItemAdded, field: "price"}
		}
	}

	scenarios: [
		{
			name: "Ex: Empty cart"
			given: [_events.CartCreated, _events.ItemAdded]
			query: {cartId: "abc"}
			expect: {
				cartId: "abc"
				items: []
				totalPrice: 0
			}
		},
		{
			name: "Ex: Cart with one item"
			given: [_events.CartCreated,
				_events.ItemAdded & {fields: {price: 10, quantity: 2}}]
			query: {cartId: "abc"}
			expect: {
				cartId: "abc"
				items: [{
					itemId:    "item-1"
					productId: "prod-1"
					price:     10
					quantity:  2
				}]
				totalPrice: items[0].price * items[0].quantity
			}
		},
		{
			name: "Ex: Cart with 2 items"
			given: [_events.CartCreated,
				_events.ItemAdded & {fields: {itemId: "item-1", price: 999}},
				_events.ItemAdded & {fields: {itemId: "item-2", price: 999}}]
			query: {cartId: "abc"}
			expect: {
				cartId: "abc"
				items: [{
					itemId:    "item-1"
					productId: "prod-1"
					price:     999
					quantity:  1
				}, {
					itemId:    "item-2"
					productId: "prod-2"
					price:     999
					quantity:  1
				},
				]
				totalPrice: list.Sum([for item in items {item.price * item.quantity}])
			}
		},
	]
}
