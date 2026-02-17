package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ViewEmptyCart: em.#StoryStep & {
	kind:  "story"
	name:  "view empty cart"

	slice: ViewCartItems

    image: "./mockups/empty_cart_view.png"
	instance: ViewCartItems.readModel.fields & {
		cartId: "cart-abc"
		items: [ ],
        totalPrice: 0
	}
}

AddOneItemCartStory: em.#ChangeStoryStep & {
	kind:  "story"
	name:  "one empty cart"

	slice: AddItem
    emits: [_events.ItemAdded & {fields: {cartId: "abc"}}]
}
