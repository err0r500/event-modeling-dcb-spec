package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ArchiveItems: em.#AutomationSlice & {
	name: "ArchiveItems"

	trigger: em.#InternalEventTrigger & {
		internalEvent: _events.PriceChanged
	}

    consumes: [OpenCartsWithProducts.readModel]

	command: em.#Command & {
		// Only trigger fields - productId comes from PriceChanged
		fields: em.#Field & {
			productId: string
		}
		query: {
			items: [
				{
					types: [_events.ItemAdded]
					tags: [{tag: _tags.product_id, value: fields.productId}]
				},
			]
		}
	}

	// cartId, itemId come from consumed readModel.columns
	emits: [_events.ItemArchived & {
		mapping: {
			cartId: consumes[0].columns.cartId
			itemId: consumes[0].columns.itemId
		}
	}]
}
