package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

// Package-level definitions for use in separate files
_tags: [Name=string]: em.#Tag & {name: Name}
_tags: {
	item_id:    {param: "itemId", type: int}
	cart_id:    {param: "cartId", type: string}
	product_id: {param: "productId", type: string}
}

_actors: [Name=string]: em.#Actor & {name: Name}
_actors: {
	User:              {}
	InventoryEventBus: {}
}

cartBoard: em.#Board & {
	name:   "Shopping Cart"
	tags:   _tags
	events: _events
	actors: _actors

	contexts: [
		{
			name:        "Cart"
			description: "Shopping cart management — add, remove, clear items"
			chapters: [
				{
					name:        "BrowseAndFill"
					description: "Customer browses products and fills their cart"
					flow: [
						AddItem,
						ViewEmptyCart,
						RemoveItem,
						ClearCart,
					]
				},
				{
					name:        "Review"
					description: "Customer reviews cart contents before checkout"
					flow: [
						ViewCart,
					]
				},
			]
		},
		{
			name:        "Inventory"
			description: "Product inventory tracking and availability"
			chapters: [
				{
					name:        "BrowseAndFill"
					description: "Customer browses products — inventory side"
					flow: [
						ChangeInventory,
						ViewProductsInventories,
					]
				},
			]
		},
		{
			name:        "Checkout"
			description: "Cart submission and order finalization"
			chapters: [
				{
					name:        "Submit"
					description: "Customer submits the cart to place an order"
					flow: [
						SubmitCart,
					]
				},
			]
		},
	]
}
