package examples

import "github.com/fairway/eventmodelingspec/schema"

ChangeInventory: schema.#ChangeSlice & {
	name: "ChangeInventory"

	actor: _actors.InventoryEventBus

	trigger: schema.#ExternalEventTrigger & {
		externalEvent: {
			name: "InventoryChanged"
			fields: {
				productId: string
				inventory: int
			}
		}
	}

	command: schema.#Command & {
		fields: schema.#Field & {
			productId: string
			inventory: int
		}
		query: {}
	}
	emits: [_events.InventoryChanged]
}
