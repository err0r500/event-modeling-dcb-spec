package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

ChangeInventory: em.#ChangeSlice & {
	name:    "ChangeInventory"
	context: "Inventory"
	chapter: "BrowseAndFill"

	actor: _actors.InventoryEventBus

	trigger: em.#ExternalEventTrigger & {
		externalEvent: {
			name: "InventoryChanged"
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
