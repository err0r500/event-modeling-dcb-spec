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
					types: [_events.ItemAdded, _events.ItemRemoved]
					tags: [
						{tag: _tags.cart_id, value: fields.cartId},
					]
				},
			]
		}
	}
	emits: [_events.CartSubmitted]
	scenarios: []
}
