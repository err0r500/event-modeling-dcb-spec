Great — here’s a consolidated, end-to-end summary of the design and CUE patterns we discussed, written as a coherent architecture rather than scattered snippets.

⸻

1. Overall design goal

You’re using CUE as a declarative constraint system to model and validate:
	•	HTTP endpoints (routes)
	•	Commands triggered by those endpoints
	•	Event pipelines (emits → consumes)
	•	Cross-field and cross-object invariants

The core philosophy is:

Model structure explicitly, derive strings, and validate consistency by unification — not parsing.

This plays directly to CUE’s strengths.

⸻

2. Endpoints: structured, not parsed

Key decision

Instead of treating a URL path as a string with matchers (/post/:user_id), you model the endpoint as structured data, and derive the path from it.

This avoids:
	•	regex capture hacks
	•	brittle string parsing
	•	one-way validation

Canonical endpoint model

#Endpoint: {
  user_id: string
  path: "/post/\(user_id)"
}

Why this works well
	•	user_id is a first-class value
	•	path is guaranteed to stay consistent
	•	constraints flow both directions if needed
	•	no parsing or routing logic in CUE

This establishes the endpoint as a spec, not an artifact.

⸻

3. Commands: validated against endpoints

Commands are domain objects that must only reference values made available by the triggering endpoint.

Simple command model

#Command: {
  user_id: string
  // other fields...
}

Validation: command fields must come from endpoint

Explicit (per field):

command: {
  user_id: endpoint.user_id
}

Generic (scales better):

command: [k=string]: endpoint[k]

This enforces:
	•	every field present in command must exist in endpoint
	•	values must unify (same value, not just “present”)
	•	no accidental extra fields

This is structural referential integrity, not procedural checking.

⸻

4. Events: identity via eventType

Instead of string IDs or whole-object equality, events are identified by a stable identity field.

Event model

#Event: {
  eventType: string
  // optional payload/schema later
}

This is critical because:
	•	CUE validates via keys
	•	identity must be comparable
	•	whole-struct equality is not the right abstraction

⸻

5. Emits / consumes with helper types

To avoid repetition and keep intent clear:

#EventRef: {
  eventType: #Event.eventType
}

#Step: {
  emits:    [...#Event]
  consumes: [...#EventRef]
}

	•	emits = full events
	•	consumes = lightweight references
	•	shared identity definition

This keeps schemas DRY and readable.

⸻

6. Referential integrity across an ordered pipeline

Requirement

A step may only consume events that were emitted earlier in the list.

Core idea
	•	Represent “seen events” asunning as a set
	•	Accumulate it per index
	•	Validate consumes against the accumulated set

Running “seen emits” set

_seen: [i=int]: if i == 0 {
  {}
} else {
  _seen[i-1] & {
    [t=string]: steps[i-1].emits[_].eventType
  }
}

Validation

steps: [i=int]: {
  for c in consumes {
    c.eventType: _seen[i][c.eventType]
  }
}

This enforces:
	•	strict ordering
	•	no forward references
	•	no missing dependencies

All without loops, mutation, or explicit control flow.

⸻

7. Why this works so well in CUE

You consistently used these CUE idioms:

1. Identity as keys
	•	Sets are modeled as structs
	•	Presence = key existence
	•	Validation = unification against a key

2. Derivation over parsing
	•	Paths derived from fields
	•	No regex capture logic
	•	Constraints stay local and obvious

3. Generic constraints
	•	[k=string]: endpoint[k]
	•	reusable validation patterns
	•	scalable without boilerplate

4. Constraints, not execution
	•	No “if this then that”
	•	No evaluation order dependency
	•	Just “this must be true”

⸻

8. Architectural picture

Putting it all together:
	•	Endpoints declare what inputs exist
	•	Commands declare what inputs they require
	•	Events declare what facts are produced
	•	Pipelines declare legal flows of facts

CUE’s job is to ensure:
	•	no command references unavailable inputs
	•	no step consumes nonexistent events
	•	no path can drift from its parameters
	•	no invalid state can be represented

This is exactly the domain CUE excels at.

---

# information completeness

Let’s formalize information completeness precisely, in a way that is:
	•	operational (you can apply it to real systems),
	•	checkable (at least at the schema / design level),
	•	and aligned with your CUE-based modeling.

I’ll move from definitions → rules → implications → how to enforce.

⸻

1. Formal model (minimal)

We define the system in terms of time, facts, and decisions.

Entities
	•	Facts: atomic pieces of information (field values)
	•	Events: immutable records that introduce facts
	•	Commands: requests to make a decision / state transition
	•	Handlers: pure functions that transform (facts) → (events)

We assume time is discrete and ordered by events.

⸻

2. Causality and availability

Definition: Available information

At time t, the available information set A(t) is:

A(t) =
  facts provided by the triggering request
  ∪
  facts carried by events emitted at times < t

Nothing else is available.

No database reads.
No cache.
No “current state” unless reconstructed from events.

⸻

3. Information completeness (formal definition)

A command is information-complete if every fact it depends on is an element of A(t) at the time it is issued.

More formally:

Let:
	•	C be a command
	•	D(C) be the set of facts that influence the semantics of C
	•	A(t) be the available information set when C is evaluated

Then:

C is information-complete  ⇔  D(C) ⊆ A(t)

If any required fact is missing, the command is incomplete.

⸻

4. What does “depends on” mean (this is crucial)

A command depends on a fact if:
	•	changing that fact could change the outcome of the command, or
	•	the command handler reads it (directly or indirectly)

This includes:
	•	validation logic
	•	branching logic
	•	computed fields
	•	authorization decisions
	•	emitted event contents

This is a semantic dependency, not just a type dependency.

⸻

5. Completeness rules (practical laws)

Rule 1 — Explicit dependency rule

If a command handler reads a value, that value must be explicitly present in the command input or in consumed events.

No implicit reads.

⸻

Rule 2 — No ambient authority rule

A command may not rely on facts whose truth is established “elsewhere in the system” unless that truth is represented as an event.

This forbids:
	•	“the database already guarantees this”
	•	“we know payment happened”
	•	“this user must exist”

Unless there is an event that says so.

⸻

Rule 3 — Event sufficiency rule

An event must contain all facts required to interpret its meaning without external state.

Formally:

For any event E, D(E) ⊆ fields(E)

If a consumer must query something else to understand the event, the event is incomplete.

⸻

Rule 4 — Replay determinism rule

Replaying the same sequence of events and commands must produce the same outcomes.

If replay requires:
	•	current database state
	•	environment variables
	•	timing coincidences

→ information completeness is violated.

⸻

6. Command completeness vs event completeness

They are related but distinct.

Command completeness

Concerns inputs to decisions.

“Did the system know everything it needed at decision time?”

Event completeness

Concerns outputs of decisions.

“Does this event fully explain what happened and why?”

You need both.

⸻

7. Completeness lattice (strong → weak)

From strongest to weakest:
	1.	Strong completeness
All decisions are pure functions of:
	•	command input
	•	prior events
(ideal event sourcing)
	2.	Operational completeness
Decisions read models derived only from events
(acceptable in practice)
	3.	Implicit completeness (bad)
Decisions rely on external mutable state
(looks fine, breaks replay)

Your design is aiming at 1 → 2, and explicitly rejecting 3.

⸻

8. How your CUE model enforces completeness (structurally)

CUE cannot enforce runtime purity, but it enforces structural completeness:

1. Endpoint → Command binding

command: [k=string]: endpoint[k]

⇒ commands cannot invent fields
⇒ all command facts are traceable to the request

⸻

2. Emits → consumes ordering

consumes.eventType ∈ previously emitted eventTypes

⇒ commands/events can only depend on past facts
⇒ causality is explicit

⸻

3. No hidden identifiers

Using structured identity (eventType, params as fields) ensures:
	•	identity is explicit
	•	dependencies are visible
	•	no “stringly-typed magic”

⸻

9. What completeness does not require (important)

Information completeness does not mean:
	•	all data is present everywhere
	•	commands are huge blobs
	•	no derived values exist

It does mean:
	•	if a derived value affects behavior, its inputs must be present
	•	derivation must be deterministic and explicit

⸻

10. Completeness as a design test

Here’s a concrete test you can apply to any command or event:

“If I snapshot only the command/event and its declared dependencies, can I explain and reproduce its effect?”

If the answer is:
	•	yes → information complete
	•	no → there is hidden state

⸻

11. Final formal statement (clean)

You can use this as a design principle:

Information completeness is the property that every semantic dependency of a command or event is explicitly represented in its inputs or in causally prior events, such that the system’s behavior is reproducible, explainable, and free of hidden state.

This is exactly the level of rigor your current CUE-based design is moving toward.
