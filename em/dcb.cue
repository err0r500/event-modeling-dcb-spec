package em

// #TagRef - Tag reference with optional value for parameterized tags
//
// Two forms:
//   Simple: tags.cart - bare tag (category, no value)
//   Parameterized: {tag: tags.cartId, value: command.fields.cartId}
//   FromExtract: {tag: tags.discountCode, fromExtract: "discountCode"} (only in dependentQuery)
//
// Fields:
//   tag: #Tag - reference to tag from board.tags
//   value?: tag.type - required if tag.param is set, must match tag's type
//   fromExtract?: string - name from dependentQuery.extract (mutually exclusive with value)
#TagRef: {
	tag:          #Tag
	value?:       tag.type
	fromExtract?: string
	_tagName:     tag.name
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
	tags: [...#Tag | #TagRef] | *[]   // AND - event must have ALL
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

// #QueryExtract - Extract values from primary query results for dependent query
//
// Maps a name to an event type and field, allowing the dependent query
// to use values extracted from events returned by the primary query.
//
// Example:
//   extract: {
//     discountCode: {event: _events.CartCreated, field: "discountCode"}
//   }
#QueryExtract: {
	[name=string]: {
		event: #Event
		field: string
	}
}

// #DependentQuery - Second-phase query using extracted values
//
// Enables a two-phase query pattern where the first query retrieves events,
// values are extracted from those events, and a second query uses those
// extracted values as tag parameters.
//
// Fields:
//   extract: #QueryExtract - values to extract from primary query events
//   items: [...#QueryItem] - query clauses using fromExtract in tags
//
// Example:
//   dependentQuery: {
//     extract: {discountCode: {event: _events.CartCreated, field: "discountCode"}}
//     items: [{
//       types: [_events.DiscountCreated]
//       tags: [{tag: _tags.discount_code, fromExtract: "discountCode"}]
//     }]
//   }
#DependentQuery: {
	extract: #QueryExtract
	items: [...#QueryItem]
}
