# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Event modeling specification framework using CUE as a declarative constraint engine. Boards define domain flows with explicit causality: commands, events, views, actors, and GWT scenarios. Two CLI tools: interactive TUI explorer and ASCII renderer.

## Commands

```bash
go build ./cmd/tui            # Build TUI app
go build ./cmd/render          # Build render CLI
go test ./...                  # Run all tests
go test -run TestValidation    # Run specific test (schema_test.go)
```

TUI: `go run ./cmd/tui -file examples/cart.cue [-board name]`
Render: `go run ./cmd/render -file examples/cart.cue -name <instant> [-board name]`

## Architecture

```
cmd/tui/         → Bubbletea TUI entry point (file watcher, table view, detail view)
cmd/render/      → ASCII box renderer for individual instants
pkg/board/       → Board loading: CUE file → Go Board struct via cue.Value unification
pkg/render/      → Rendering + validation (ValidateBoard, validateParameterizedTags)
pkg/tui/         → TUI model, styles, detail view, item definitions
em/              → CUE schemas: board.cue (main), types.cue, gwt.cue, slice.cue, story.cue, dcb.cue, endpoint.cue
examples/        → Shopping cart domain example (split across multiple .cue files)
schema_test.go   → 12 validation tests covering board structure, ordering, references, GWT
```

## Key Types

**Go:** `Board{Name, Value(cue.Value), Flow[]FlowItem}` — FlowItem has Kind (slice/story), Type (change/view), CUEValue.

**CUE schema hierarchy:** `#Board` → `flow: [...]#Instant` where `#Instant` is `#ChangeSlice | #ViewSlice | #StoryStep`. Change slices have `#Command` + `#Endpoint` + `#GWT` scenarios. Events use `#DCBQuery` for tag-based filtering.

## Design Principles

From `chat.md` — the authoritative design document:

- **Structured over parsed**: Endpoints/fields are typed data, not parsed strings
- **Declarative validation**: CUE unification, not imperative checks
- **Information completeness**: Command fields must come from endpoint inputs or computed fields
- **Explicit causality**: `board.cue` tracks `_emittedBefore` per flow index — view slices can only query previously-emitted events
- **DCB pattern**: Dynamic Consistency Boundaries — (event has ANY of types) AND (has ALL of tags)

## CUE Schema Validation

`board.cue` is the most complex file (~180 lines). It pre-computes emitted event sets at each flow index to enforce ordering. Key constraints: actor existence, command-endpoint field compatibility, event field sourcing (command/mapping/computed), story step slice references, tag validation.
