package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ViewEmptyCart: em.#StoryStep & {
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
