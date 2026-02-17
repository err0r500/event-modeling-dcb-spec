package examples

import (
	s "github.com/err0r500/event-modeling-dcb-spec/em"
)

ViewProductsInventories: s.#ViewSlice & {
	name:  "ViewProductsInventories"
	actor: _actors.User

    image: "./mockups/view_inventories.png"

	endpoint: s.#Endpoint & {
        verb: "GET"
		params: {
			productId: string
		}
		path: "/inventories/products/{productId}"
	}

	query: {
		items: [
        {
			types: [_events.InventoryChanged]
			tags: [{tag: _tags.product_id, value: endpoint.params.productId}]
		},
        ]
	}

	readModel: s.#ReadModel & {
		name:        "ProductInventories"
		cardinality: "single"
		fields: {
			products: [...{
				productId:    string
				quantity:  int
			}]
		}
		mapping: {
			"products.productId": {event: _events.InventoryChanged, field: "productId"}
			"products.quantity": {event: _events.InventoryChanged, field: "inventory"}
		}
	}

	scenarios: []
}
