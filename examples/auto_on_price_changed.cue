package examples

import "github.com/err0r500/event-modeling-dcb-spec/em"

OnPriceChanged: em.#AutomationSlice & {
	name: "OnPriceChanged"

	trigger: em.#ExternalEventTrigger & {
		externalEvent: {
			name: "IntegrationPriceChanged"
            source: "Pricing Context"
			fields: {
				productId: string
                oldPrice: int
				newPrice: int
			}
		}
	}

	command: em.#Command & {
		fields: em.#Field & {
			productId: string
			oldPrice: int
			newPrice: int
		}
		query: {}
	}

	emits: [_events.PriceChanged]
}
