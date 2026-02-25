package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

// Package-level definitions for use in separate files
_tags: [Name=string]: em.#Tag & {name: Name}
_tags: {
	item_id: {param: "itemId", type: int}
	shopper_id: {param: "shopper_id", type: string}
	cart_id: {param: "cartId", type: string}
	product_id: {param: "productId", type: string}
}

_actors: [Name=string]: em.#Actor & {name: Name}
_actors: {
	InventoryEventBus: {}
	User: {}
}

cartBoard: em.#Board & {
	name:   "Shopping Cart"
	tags:   _tags
	events: _events
	actors: _actors

	contexts: [
		{
			name:        "Shopping"
			description: "Shopping cart context"
			chapters: [
				{
					name:        "Cart Items"
					description: "Customer browses products and fills their cart"
					flow: [
						AddItem,
						ViewOneItemCart,
						RemoveItem,
						ViewEmptyCart,
						AddOneItemCartStory,
						ViewOneItemCart,
						ClearCart,
						ViewEmptyCart,
						AddOneItemCartStory,
						ViewCartItems,
					]
				},
				{
					name:        "Inventory"
					description: "Handles changes from the inventory context"
					flow: [
						OnInventoryChanged,
						ViewProductsInventories,
					]
				},
				{
					name:        "Price Change"
					description: "Handles changes from the pricing context"
					flow: [
						OnPriceChanged,
						ChangedPrices,
						OpenCartsWithProducts,
						ArchiveItems,
					]
				},
				{
					name:        "Submit Cart"
					description: "Customer submits the cart"
					flow: [
						SubmitCart,
						AutoCloseCart,
					]
				},
			]
		},
		{
			name: "other context"
		},
	]
}
