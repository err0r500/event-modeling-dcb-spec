package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

SubmitCart: em.#ChangeSlice & {
	name:  "SubmitCart"
	actor: _actors.User
	trigger: em.#EndpointTrigger & {
		endpoint: {
			verb: "POST"
			params: {cartId: string}
			auth: {userId: string}
			path: "/cart/{cartId}/submit"
		}
	}
	command: {
		fields: {
			shopperId: string
			cartId:    string
		}

		mapping: {
			shopperId: trigger.endpoint.auth.userId
		}

		query: {
			items: [
				{
					types: [_events.CartCreated, _events.CartCleared, _events.CartDeleted]
					tags: [
						{tag: _tags.cart_id, value: fields.cartId},
						{tag: _tags.shopper_id, value: fields.shopperId},
					]
				},
				{
					types: [_events.ItemAdded, _events.ItemRemoved, _events.ItemArchived, _events.CartSubmitted]
					tags: [
						{tag: _tags.cart_id, value: fields.cartId},
					]
				},
			]
		}
	}
	emits: [_events.CartSubmitted]
	scenarios: [
		{
			name: "one item in cart"
			given: [_events.CartCreated, _events.ItemAdded]
			when: {}
			then: {
				success: true
				events: [_events.CartSubmitted]
			}
		},
        {
            name: "empty cart"
            given: [_events.CartCreated]
            when: {}
            then: {
                success: false
                error: "Cart cannot be empty"
            }
        },
        {
            name: "inventory changed"
            given: [_events.CartCreated, _events.ItemAdded & {fields: {itemId: "item-abc"}}, _events.ItemArchived & {fields: {itemId: "item-abc"}}]
            when: {}
            then: {
                success: false
                error: "Inventory has changed for one or more items in the cart"
            }
        },
        {
            name: "cart already submitted"
            given: [_events.CartCreated, _events.ItemAdded, _events.CartSubmitted]
            when: {}
            then: {
                success: false
                error: "Cart has already been submitted"
            }
        },
	]
}
