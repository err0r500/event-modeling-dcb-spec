package em

// #TagRef - Tag reference with optional value for parameterized tags
//
// Two forms:
//   Simple: tags.cart - bare tag (category, no value)
//   Parameterized: {tag: tags.cartId, value: command.fields.cartId}
//
// Fields:
//   tag: #Tag - reference to tag from board.tags
//   value?: tag.type - required if tag.param is set, must match tag's type
#TagRef: {
	tag:      #Tag
	value?:   tag.type
	_tagName: tag.name
}

// #QueryItem - Single DCB query clause
//
// Matches events where:
//   - Event type is ANY of the listed types (OR/union)
//   - AND event has ALL of the listed tags (AND/intersection)
//
// Fields:
//   types: [...#Event] - event types to match (OR semantics)
//   tags: [...#Tag | #TagRef] - required tags (AND semantics)
//
// Example: Get all cart events for a specific cart
//   {types: [_events.CartCreated, _events.ItemAdded],
//    tags: [{tag: tags.cartId, value: command.fields.cartId}]}
#QueryItem: {
	types!: [...#Event]          // OR - event matches if ANY (reference board events)
	tags!: [...#Tag | #TagRef]   // AND - event must have ALL
}

// #DCBQuery - Dynamic Consistency Boundary query
//
// Defines which events to load for consistency checking (commands) or
// projection (views). Multiple items are OR'd together.
//
// Fields:
//   items: [...#QueryItem] - query clauses (OR'd together)
//
// Pattern: (event has ANY of types) AND (has ALL of tags)
#DCBQuery: {
	items: [...#QueryItem]
}
