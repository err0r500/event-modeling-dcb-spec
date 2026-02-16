package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ViewOneItemCart: em.#StoryStep & {
	kind:  "story"
	name:  "view one item cart"

	slice: ViewCartItems

    image: "./mockups/one_item_cart_view.png"
	instance: ViewCartItems.readModel.fields & {
		cartId: "cart-abc"
		items: [
			{
				itemId:    "item-1"
				productId: "coffee"
				price:     1499
				quantity:  1
			},
		],
        totalPrice: 1499
	}
}
