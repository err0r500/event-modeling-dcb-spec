package schema

// #StoryStep - Narrative reference to an existing slice
//
// Story steps allow reusing slices in narrative sequences without
// duplicating definitions. Use for user journeys and scenario flows.
//
// Fields:
//   kind: "story" - discriminator for Instant union
//   name: string - step identifier in the narrative
//   slice: #ChangeSlice | #ViewSlice - direct reference to the slice
//   description?: string - narrative context for this step
//   instance?: _ - concrete instance data (validated against slice schema)
//
// Example:
//   ViewEmptyCart: #StoryStep & {
//       slice: ViewCart
//       instance: {cartId: "cart-123", items: []}
//   }
#StoryStep: {
	kind: "story"
	name: string
	// Direct reference to the slice this step uses
	slice: #ChangeSlice | #ViewSlice
	// Narrative description
	description: string | *""
	// Optional concrete instance data
	instance?: _
}
