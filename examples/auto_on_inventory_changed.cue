package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

OnInventoryChanged: em.#AutomationSlice & {
	name: "OnInventoryChanged"

	trigger: {
		kind: "externalEvent"
		externalEvent: {
			name:   "InventoryChanged"
			source: "Inventory Context"
			fields: {
				productId: string
				inventory: int
			}
		}
	}

	command: em.#Command & {
		fields: em.#Field & {
			productId: string
			inventory: int
		}
		query: {}
	}

	emits: [_events.InventoryChanged]
}
