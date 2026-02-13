```
go run ./cmd/render -file examples/cart.cue -outdir .board/ -watch

go run ./cmd/tui -dir .board/
cd web && BOARD_DIR=../.board npm run dev
```

## Using in Another Repo

Add dependency:
```bash
cue mod init your.module/name
cue mod get github.com/err0r500/event-modeling-dcb-spec@latest
```

Import schema:
```cue
package myboard

import "github.com/err0r500/event-modeling-dcb-spec/em"

em.#Board & {
    name: "MyDomain"
    // ...
}
```

## Validation Rules

### Board Structure

| Rule | Description |
|------|-------------|
| Actor existence | Actors referenced in slices must exist in `actors` |
| Event definition | Emitted events must be defined in `events` |
| Tag definition | All tags in DCB queries must exist in `tags` |

### Change Slice (Command)

| Rule | Description |
|------|-------------|
| Field source | Command fields must come from trigger (endpoint params/body or externalEvent), mapping, or computed |
| Field type | Types must match between source and command field |
| Emit field source | Event fields must come from command.fields, mapping, or computed |
| Emit field type | Types must match between source and event field |
| Path param consistency | Endpoint path params (e.g. `{cartId}`) must exist in params fields |

### View Slice (Query)

| Rule | Description |
|------|-------------|
| Event ordering | Can only query events emitted by earlier change slices in flow |
| ReadModel field source | Fields must come from queried events, computed, or mapping |
| Computed event queried | Computed source event must be in query |
| Computed field exists | Computed fields must exist in source event |
| Mapping event queried | Mapping source event must be in query |
| Mapping field exists | Mapping field must exist in source event |
| Mapping type match | ReadModel field type must match event field type |
| Dotted path resolution | Dotted paths (e.g. "items.price") must resolve to actual fields (Go) |
| Dotted path type | Resolved field type must match event field type (Go) |
| Path param consistency | Endpoint path params (e.g. `{cartId}`) must exist in params fields |
| Scenario given in query | View scenario `given` events must be in query types |

### DCB Query

| Rule | Description |
|------|-------------|
| Event has tags | Every event in query must have ALL required tags |
| Parameterized tag value | Tags with `param` require a `value` in queries (Go) |

### GWT Scenarios

| Rule | Description |
|------|-------------|
| Command name match | Scenario `when.name` must match slice command name |
| Given event in query | Given events must be in command's query types |
| Then event in emits | Success scenario `then.events` must be in slice's emits |
| Event value types | Event field values must match field types |


---
## Future Improvements

| Category | Check | Description |
|----------|-------|-------------|
| Structural | Non-empty emits | Change slice with empty emits is likely a mistake |
| Structural | Computed/mapping overlap | Same field shouldn't be both computed AND mapped |
| Type | Tag value type | Parameterized tag value should match tag.param type |
| Type | View endpoint response | Endpoint should reflect readModel (currently not linked) |
| Scenario | All emits tested | Every emitted event should appear in at least one success scenario |
| Scenario | Error scenario for queries | Commands with non-empty query should have error scenarios |
| DCB | Query not empty when needed | Command that emits to existing aggregate needs query to load state |

