package examples

import "github.com/fairway/eventmodelingspec/schema"

// Package-level definitions for use in separate files
_tags: [Name=string]: schema.#Tag & {name: Name}
_tags: {
	item_id:    {param: "itemId", type: int}
	cart_id:    {param: "cartId", type: string}
	product_id: {param: "productId", type: string}
}

_actors: [Name=string]: schema.#Actor & {name: Name}
_actors: {
	User: {}
    InventoryEventBus: {}
}

cartBoard: schema.#Board & {
	name:   "Shopping Cart"
	tags:   _tags
	events: _events
	actors: _actors

	flow: [
		AddItem,
        ViewEmptyCart,
        RemoveItem,
        ClearCart,
        ViewCart,
        ChangeInventory,
        ViewProductsInventories,
        SubmitCart
	]
}
