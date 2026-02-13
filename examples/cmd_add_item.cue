package examples

import "github.com/fairway/eventmodelingspec/schema"

ViewEmptyCart: schema.#StoryStep & {
	kind:  "story"
	name:  "view empty cart"
	slice: ViewCart
	instance: ViewCart.readModel.fields & {
		cartId: "cart-abc"
		items: [
			{
				itemId:    "item-1"
				productId: "coffee"
				price:     1499
				quantity:  1
			},
		]
	}
}

AddItem: schema.#ChangeSlice & {
	name:  "AddItem"
	actor: _actors.User
	image: "mockups/add_item.png"

	trigger: schema.#EndpointTrigger & {
		endpoint: {
			verb: "POST"
			params: {cartId: string}
			body: {
				productId:   string
				description: string
				imageURL:    string
				itemId:      string
				price:       int
			}
			path: "/carts/{cartId}/items"
		}
	}

	command: schema.#Command & {
		fields: {
			cartId:      string
			productId:   string
			quantity:    int
			description: string
			image:       string
			itemId:      string
			price:       int
		}

		mapping: {image: trigger.endpoint.body.imageURL}

		computed: {quantity: "default quantity"}

		query: {
			items: [
				{
					types: [_events.CartCreated, _events.ItemAdded]
					tags: [{tag: _tags.cart_id, value: fields.cartId}]
				},
				{
					types: [_events.InventoryChanged]
					tags: [{tag: _tags.product_id, value: fields.productId}]
				},
			]
		}
	}

	emits: [
		_events.CartCreated,
		_events.ItemAdded,
	]

	scenarios: [
		{
			name: "OK: Add item automatically opens the cart"
			given: [_events.CartCreated & {fields: {cartId: "abc"}}]
			when: {}
			then: {
				success: true
				events: [_events.CartCreated, _events.ItemAdded]
			}
		},
		{
			name: "Err: duplicate cart creation"
			given: [_events.CartCreated & {fields: {cartId: "abc"}}]
			when: {cartId: "abc"}
			then: {
				success: false
				error:   "already created"
			}
		},
		{
			name: "Err: 2 carts can't have the same Id"
			given: [
				_events.CartCreated & {fields: {cartId: "abc"}},
			]
			when: {cartId: "abc"}
			then: {
				success: false
				error:   "conflict"
			}
		},
		{
			name: "Err: max 3 items per cart"
			given: [
				_events.CartCreated,
				_events.ItemAdded,
				_events.ItemAdded,
				_events.ItemAdded,
			]
			when: {}
			then: {
				success: false
				error:   "can't add more than 3 items"
			}
		},
		{
			name: "Err: can't add item if empty inventory"
			given: [
				_events.CartCreated,
				_events.InventoryChanged & {fields: {productId: "abc", inventory: 0}},
			]
			when: {}
			then: {
				success: false
			}
		},
	]
}
