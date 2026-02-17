package em

// #ChangeStoryStep - Narrative reference to an existing change slice
//
// Fields:
//   kind: "story" - discriminator for Instant union
//   name: string - step identifier in the narrative
//   slice: #ChangeSlice - direct reference to the change slice
//   description?: string - narrative context for this step
//   image?: string - optional illustration
//   emits?: [...#EventInstance] - concrete event instances emitted in this step
#ChangeStoryStep: {
	kind: "story"
	name: string
	slice: #ChangeSlice
	description: string | *""
	image?:      string
	emits?: [...#EventInstance]
}

// #ViewStoryStep - Narrative reference to an existing view slice
//
// Fields:
//   kind: "story" - discriminator for Instant union
//   name: string - step identifier in the narrative
//   slice: #ViewSlice - direct reference to the view slice
//   description?: string - narrative context for this step
//   image?: string - optional illustration
//   instance?: slice.readModel.fields - concrete read model instance for this step
#ViewStoryStep: {
	kind: "story"
	name: string
	slice: #ViewSlice
	description: string | *""
	image?:      string
	instance?:   slice.readModel.fields
}

// #StoryStep - Union of story step types
#StoryStep: #ChangeStoryStep | #ViewStoryStep
