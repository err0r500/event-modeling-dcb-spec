package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

SubmitCart: em.#ChangeSlice & {
    name:    "SubmitCart"
    actor:   _actors.User
    context: "Checkout"
    chapter: "Submit"
    trigger: em.#EndpointTrigger & {
        endpoint: {
            verb: "POST"
            params: {cartId: string}
            body: {}
            path: "/cart/{cartId}/submit"
        }
    }
    command: {
        fields: {cartId: string}
        query: {
            items: [{
                types: [
                _events.CartCreated,
                _events.ItemAdded,
                _events.ItemRemoved,
                _events.CartCleared,
                _events.CartDeleted]
                tags: [{tag: _tags.cart_id, value: fields.cartId}]
            }]
        }
    }
    emits: [_events.CartSubmitted]
    scenarios: []
}
