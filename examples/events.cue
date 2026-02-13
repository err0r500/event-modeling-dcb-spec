package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

_events: [Type=string]: em.#Event & {eventType: Type}
_events: {
	CartCreated: {
		fields: {
			cartId: string
		}
		tags: [_tags.cart_id]
	}

	CartDeleted: {
		fields: {
			cartId: string
		}
		tags: [_tags.cart_id]
	}

	CartCleared: {
		fields: {
			cartId: string
		}
		tags: [_tags.cart_id]
	}

	ItemAdded: {
		fields: {
			cartId:      string
			itemId:      string
			productId:   string
			image:       string
			description: string
			price:       int
			quantity:    int
		}
		tags: [_tags.item_id, _tags.cart_id]
	}

	ItemRemoved: {
		fields: {
			cartId: string
			itemId: string
		}
		tags: [_tags.item_id, _tags.cart_id]
	}

	CartClosed: {
		fields: {
			cartId: string
		}
		tags: [_tags.cart_id]
	}

	InventoryChanged: {
		fields: {
			productId: string
			inventory: uint
		}
		tags: [_tags.product_id]
	}

	CartSubmitted: {
		fields: {
			cartId: string
		}
		tags: [_tags.cart_id]
	}
}
