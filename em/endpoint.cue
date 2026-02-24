package em

// #Endpoint - HTTP API surface for a slice
//
// Structured representation of an HTTP endpoint. Command fields must
// come from endpoint params, body, or auth (plus any computed fields).
//
// Fields:
//   verb: #HTTPVerb - HTTP method
//   params: #Field - URL path/query parameters as typed fields
//   body: #Field - request body fields (typically empty for GET)
//   auth: #Field - auth context fields (e.g., JWT claims, identity info)
//   path: string - URL path pattern (e.g., "/carts/{cartId}/items")
//
// Example:
//   endpoint: {
//     verb: "POST"
//     params: {cartId: string}
//     body: {productId: string, quantity: int}
//     auth: {userId: string, roles: [...string]}
//     path: "/carts/{cartId}/items"
//   }
#Endpoint: {
	verb!:   "GET" | "POST" | "PUT" | "PATCH" | "DELETE"
	params!: #Field
	body:    #Field | *{}
	auth:    #Field | *{}
	path!:   string
}
