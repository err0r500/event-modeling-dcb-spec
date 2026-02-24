package examples

import em "github.com/err0r500/event-modeling-dcb-spec/em"

ChangedPrices: em.#ViewSlice & {
	name:  "ChangedPrices"
	actor: _actors.User

	endpoint: em.#Endpoint & {
		verb: "GET"
		params: {
			productId: string
		}
		path: "/prices/products/{productId}"
	}

	query: {
		items: [
			{
				types: [_events.PriceChanged]
				tags: [{tag: _tags.product_id, value: endpoint.params.productId}]
			},
		]
	}

	readModel: em.#ReadModel & {
		name:        "ProductPrices"
		cardinality: "single"
		fields: {
			products: [...{
				productId: string
				oldPrice:  int
				newPrice:  int
			}]
		}
		mapping: {
			"products.productId": {event: _events.PriceChanged, field: "productId"}
			"products.oldPrice": {event: _events.PriceChanged, field: "oldPrice"}
			"products.newPrice": {event: _events.PriceChanged, field: "newPrice"}
		}
	}

	scenarios: []
}
