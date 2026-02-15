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
	User: {}
    InventoryEventBus: {}
}

_contexts: [Name=string]: em.#Context & {name: Name}
_contexts: {
	Cart:      {description: "Shopping cart management — add, remove, clear items"}
	Inventory: {description: "Product inventory tracking and availability"}
	Checkout:  {description: "Cart submission and order finalization"}
}

_chapters: [Name=string]: em.#Chapter & {name: Name}
_chapters: {
	BrowseAndFill: {description: "Customer browses products and fills their cart"}
	Review:        {description: "Customer reviews cart contents before checkout"}
	Submit:        {description: "Customer submits the cart to place an order"}
}

cartBoard: em.#Board & {
	name:     "Shopping Cart"
	tags:     _tags
	events:   _events
	actors:   _actors
	contexts: _contexts
	chapters: _chapters

	flow: [
		AddItem,
		ViewEmptyCart,
		RemoveItem,
		ClearCart,
		ViewCart,
		ChangeInventory,
		ViewProductsInventories,
		SubmitCart,
	]
}
