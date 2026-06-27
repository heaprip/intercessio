---
tags:
  - карта
---

# Repository Guidelines

## Collaboration Contract

This repository is developed with AI assistants, but the human author owns all
product and architecture decisions. Assistants are expected to challenge
assumptions, distinguish an accepted decision from an idea, and explain
trade-offs instead of silently turning suggestions into architecture.

**The author writes the code.** The assistant prepares design, specifications
and documentation, and does not add Go files unless asked for a specific piece.
A specification is finished when the author can implement from it without asking
what was meant.

An assistant is never the author of a project decision. Unless the author gives
an explicit instruction for a specific piece of work, assistants draft only
inside `docs/proposals/`, and every draft carries `Статус: НЕ ПРИНЯТО` in its
header. Moving material into the other directories, and setting any decision to
`Accepted`, is done by the author alone. In a decision record, `Proposed by`
names whoever drafted it; `Accepted by` is filled in by the author only, and an
assistant must leave it empty. A generated document that claims the author's
decision is a defect, not a shortcut — it makes later sessions build on
authority that was never granted.

## Documentation Layers

Documents are split by **what makes them wrong**, not by topic. See
[[docs/decisions/documentation-layers|documentation-layers]].

| Layer | Directory | Goes stale when |
| --- | --- | --- |
| Intent | `docs/guide/` | the game itself changes — almost never |
| Vocabulary | `docs/concepts/` | the language changes |
| Decisions | `docs/decisions/` | never — superseded, not edited |
| Design | `docs/design/` | the core changes |
| Implementation | `docs/impl/` | the stack changes — disposable |
| Sources | `docs/research/` | verification happens |

Two invariants. The second is enforced by `scripts/check-docs.sh`; the first is
not machine-checkable and rests on discipline:

1. a `guide/` chapter is understandable without following any link;
2. `guide/` never links into `impl/`.

Identifiers are **file-name slugs**, not numbers. Links are wikilinks of the
form `[[docs/concepts/period|период]]`, pathed from the vault root because
`README` is not unique. Numbers survive only as `guide/` chapter ordering.
Documents carry no dates: Git holds chronology, and decision order is expressed
by `Зависит от` and `Заменяет`.

## Before a Material Change

Read, in this order:

1. [[docs/guide/06-where-we-are|guide/06-where-we-are.md]] — active focus;
2. [[docs/guide/01-what-it-is|guide/01]] and the chapters relevant to the change;
3. [[docs/decisions/README|decisions/README.md]] and any relevant decision;
4. [[docs/open-questions|open-questions.md]] before resolving an ambiguity by
   assumption.

If conversation context may have been compacted, do not reconstruct missing
decisions from plausibility. Re-read the sources above. If they do not answer
the question unambiguously, or if two sources conflict, stop and ask the author
before turning the assumption into code or durable documentation.

## Documentation Is Part of the Change

Use the matrix in [[docs/README|docs/README.md]]. In particular:

- update `concepts/` when domain language changes;
- add a new failure to [[docs/guide/04-what-goes-wrong|chapter 4]] together
  with the check that detects it;
- create or supersede a decision for a cross-cutting, costly-to-reverse choice;
- update [[docs/guide/06-where-we-are|chapter 6]] when the active work changes;
- record unresolved choices in [[docs/open-questions|open-questions.md]], not
  as facts;
- never rewrite an accepted decision to make history look current: supersede it;
- distinguish explicitly between an accepted decision, a working hypothesis,
  research evidence, and an assistant recommendation.

Run `./scripts/check-docs.sh` after documentation changes. The script checks
structure and links; it does not replace human review of meaning.

Project documents are written in Russian for the current author. Code,
identifiers, commit messages, and externally consumed technical contracts are
written in English unless a domain term has no adequate translation.

## Project Structure & Module Organization

Keep executable entry points under `cmd/<name>/main.go`. Put application code
under `internal/`, organized first by bounded context or cohesive capability
rather than by technical layer. Add `pkg/` only when an external Go consumer
actually exists. Do not create global `interfaces`, `models`, `services`,
`utils`, or `common` packages.

Place test files beside the code they exercise using the `_test.go` suffix.
Store test fixtures in package-local `testdata/` directories. Put non-code
resources in a clearly named directory such as `assets/`.

## Build, Test, and Development Commands

- `go run ./cmd/<name>` runs a local executable.
- `go build ./...` compiles every package and command.
- `go test ./...` runs the complete test suite.
- `go test -race ./...` checks tests for data races.
- `go vet ./...` reports suspicious constructs.
- `gofmt -w .` formats Go source files in place.

If a task runner is added, keep these commands as the underlying behavior and
document its targets here.

## Coding Style & Naming Conventions

Follow idiomatic Go and accept `gofmt` output. Use short, lowercase package
names without underscores. Exported identifiers use `PascalCase`; unexported
identifiers use `camelCase`. Name files by responsibility. Keep packages
focused, avoid stuttering names, and add doc comments to exported APIs. Return
errors with useful context and preserve wrapped errors with `%w`.

Define interfaces where they are consumed, keep them minimal, and introduce
them only for an observed substitution. There are exactly four such places, and
they are named in [[docs/guide/05-how-it-is-built|chapter 5]]; elsewhere,
ports-for-the-future are forbidden. Start design from behavior, invariants,
examples, and use-case contracts — not from inventories of ports.

Domain code must not depend on HTTP, database, message-broker, model SDK, or
serialization types. Domain events are past-tense facts; commands express an
intent and may be rejected.

## Testing Guidelines

Use Go's `testing` package and table-driven tests where multiple cases share
behavior. Name tests `TestFunction_Scenario` and benchmarks `BenchmarkFunction`.
Tests must be deterministic, isolated, and free of live network dependencies.
Add regression tests with bug fixes. Levels and the project-specific checks —
synthetic programs for the evaluator, period replay, and the substitutability
test — are described in [[docs/design/testing|design/testing.md]].

## Commit & Pull Request Guidelines

Use concise, imperative subjects such as `Add request validation`; keep
unrelated changes in separate commits. Pull requests should explain the
motivation and behavior change, list validation commands run, and link relevant
issues. Ensure formatting, tests, vetting, and race checks pass before
requesting review.

@RTK.md
