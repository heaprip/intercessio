# Repository Guidelines

## Project Memory & Collaboration Contract

This repository is developed with AI assistants, but the human author owns all
product and architecture decisions. Assistants are expected to challenge
assumptions, distinguish an accepted decision from an idea, and explain
trade-offs instead of silently turning suggestions into architecture.

An assistant is never the author of a project decision. Assistants may draft
only inside `docs/proposals/`, and every draft must carry `Статус: НЕ ПРИНЯТО`
in its header. Moving material into `docs/` or `.agents/memory/`, and setting
any ADR to `Accepted`, is done by the author alone. In an ADR, `Proposed by`
names whoever drafted it; `Accepted by` is filled in by the author only, and an
assistant must leave it empty. A generated document that claims the author's
decision is a defect, not a shortcut — it makes later sessions build on
authority that was never granted.

Before making a material change, read:

1. `.agents/memory/current.md` for the active focus;
2. `docs/product/vision.md` and `docs/architecture/principles.md` for stable
   context;
3. `docs/decisions/README.md` and any ADRs relevant to the change;
4. `.agents/memory/open-questions.md` before resolving an ambiguity by
   assumption.

If conversation context may have been compacted, do not reconstruct missing
decisions from plausibility. Re-read the sources above. If they do not answer
the question unambiguously, or if two sources conflict, stop and ask the author
before turning the assumption into code or durable documentation.

Documentation changes are part of the implementation, not follow-up work. Use
the change matrix in `docs/README.md`. In particular:

- update the glossary when domain language changes;
- update the context map when ownership or boundaries change;
- create or supersede an ADR for a cross-cutting, costly-to-reverse decision;
- update `.agents/memory/current.md` when the active work or next step changes;
- record unresolved choices in `.agents/memory/open-questions.md`, not as facts;
- never rewrite an accepted ADR to make history look current: supersede it.
- distinguish explicitly between an accepted decision, a working hypothesis,
  research evidence, and an assistant recommendation.

Run `./scripts/check-docs.sh` after documentation changes. The script checks
structure and metadata; it does not replace human review of meaning.

Project documents are written in Russian for the current author. Code,
identifiers, commit messages, and externally consumed technical contracts are
written in English unless a domain term has no adequate translation.

## Project Structure & Module Organization

Keep executable entry points under `cmd/<name>/main.go`. Put application code
under `internal/`, organized first by bounded context or cohesive capability
rather than by technical layer. Add
`pkg/` only when an external Go consumer actually exists. Do not create global
`interfaces`, `models`, `services`, `utils`, or `common` packages.

Place test files beside the code they exercise using the `_test.go` suffix.
Store test fixtures in package-local `testdata/` directories; Go tooling ignores
these directories during normal package discovery. Put durable project
knowledge in `docs/`, short-lived agent context in `.agents/memory/`, and
non-code resources in a clearly named directory such as `assets/`.

## Build, Test, and Development Commands

Until project-specific automation exists, use standard Go commands:

- `go run ./cmd/<name>` runs a local executable.
- `go build ./...` compiles every package and command.
- `go test ./...` runs the complete test suite.
- `go test -race ./...` checks tests for data races.
- `go vet ./...` reports suspicious constructs.
- `gofmt -w .` formats Go source files in place.

If a `Makefile` or task runner is added, keep these commands as the underlying behavior and document its targets here.

## Coding Style & Naming Conventions

Follow idiomatic Go and accept `gofmt` output (tabs for Go indentation). Use short, lowercase package names without underscores. Exported identifiers use `PascalCase`; unexported identifiers use `camelCase`. Name files by responsibility, for example `server.go` and `config_test.go`. Keep packages focused, avoid stuttering names, and add doc comments to exported APIs. Return errors with useful context and preserve wrapped errors with `%w` where appropriate.

Define interfaces where they are consumed, keep them minimal, and introduce
them only for an observed substitution or boundary. Start design from behavior,
invariants, examples, and use-case contracts—not from inventories of ports.
Domain code must not depend on HTTP, database, message-broker, LLM SDK, or
serialization types. Domain events are past-tense facts; commands express an
intent and may be rejected.

## Testing Guidelines

Use Go's `testing` package and table-driven tests where multiple cases share behavior. Name tests `TestFunction_Scenario` and benchmarks `BenchmarkFunction`. Tests should be deterministic, isolated, and avoid live network dependencies. Add regression tests with bug fixes. No coverage threshold is established yet; use `go test -cover ./...` to monitor coverage and prioritize meaningful branch coverage.

## Commit & Pull Request Guidelines

No repository-specific commit convention is established yet. Use concise, imperative subjects such as `Add request validation`; keep unrelated changes in separate commits. Pull requests should explain the motivation and behavior change, list validation commands run, and link relevant issues. Include logs or screenshots only when they clarify user-visible or operational changes. Ensure formatting, tests, vetting, and race checks pass before requesting review.

@RTK.md
