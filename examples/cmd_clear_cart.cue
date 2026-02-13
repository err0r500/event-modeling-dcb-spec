package examples

import "github.com/fairway/eventmodelingspec/schema"

ClearCart: schema.#ChangeSlice & {
	name:  "ClearCart"
	actor: _actors.User
	trigger: schema.#EndpointTrigger & {
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
