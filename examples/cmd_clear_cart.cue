package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ClearCart: em.#ChangeSlice & {
	name:  "ClearCart"
	actor: _actors.User
	trigger: em.#EndpointTrigger & {
		endpoint: {
			verb: "DELETE"
			params: {cartId: string}
			path: "/carts/{cartId}/items"
		}
	}
	command: {
		fields: {
			cartId:      string
		}
		query: {
			items: [{
				types: [_events.CartCreated]
				tags: [{tag: _tags.cart_id, value: fields.cartId}]
			}]
		}
	}
	emits: [
		_events.CartCleared,
	]
}
