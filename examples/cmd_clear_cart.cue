package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ClearCart: em.#ChangeSlice & {
	name:  "ClearCart"
	actor: _actors.User

	image: "./mockups/clear_cart.png"

	trigger: em.#EndpointTrigger & {
		endpoint: {
			verb: "DELETE"
			params: {cartId: string}
			path: "/carts/{cartId}/items"
            auth: {userId: string}
		}
	}

	command: {
		fields: {
			cartId: string
            shopperId: string
		}

        mapping: {shopperId: trigger.endpoint.auth.userId}

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
