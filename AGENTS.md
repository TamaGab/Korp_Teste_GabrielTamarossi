## Agent skills

### Issue tracker

Issues are tracked in Linear under the `korp_teste` team and are uploaded only when explicitly requested. See `docs/agents/issue-tracker.md`.

### Triage labels

The repository uses the five default triage labels. See `docs/agents/triage-labels.md`.

### Domain docs

Domain documentation uses a single-context layout. See `docs/agents/domain.md`.

## Project

Technical challenge for an internship position.

The application consists of:

- Angular frontend
- Go backend with Gin
- PostgreSQL
- Docker Compose
- Two microservices:
  - `inventory-service`: products and stock
  - `billing-service`: invoices and billing

## Main Rule

Keep the solution simple.

This is a technical challenge, not a production-scale system. Do not overengineer or introduce abstractions, libraries, architectural layers, or infrastructure unless they solve a concrete requirement.

Prefer clear, idiomatic, maintainable code.

## Development Approach

Work incrementally.

For each task:

1. Inspect the existing implementation.
2. Change only what is necessary for the requested feature.
3. Do not implement future features prematurely.
4. Build and test the affected parts.
5. Preserve existing working behavior.

Do not refactor unrelated code unless necessary.

## Backend

Use:

- Go
- Gin
- PostgreSQL
- pgx/v5
- Go Modules
- `log/slog` when logging is needed

Each microservice is an independent Go module and owns its own database.

Services must communicate through HTTP APIs. Never access another service's database directly.

Prefer feature-oriented packages such as:

```text
internal/product/
internal/stock/
internal/invoice/
```

Do not create layers such as `domain`, `service`, `repository`, interfaces, factories, or abstractions unless the current implementation genuinely benefits from them.

## Frontend

The Angular application was generated using the official Angular CLI.

Do not recreate or replace the Angular workspace.

Use modern Angular conventions, including:

- Standalone Components
- Angular Router
- HttpClient
- Reactive Forms
- RxJS
- Angular Material when visual components are needed

Use CSS for custom styling.

Do not introduce NgRx, Tailwind, or other libraries unless explicitly requested or clearly necessary.

## Docker

The complete application must be runnable with:

```bash
docker compose up --build
```

A user cloning the repository should not need Go, Node.js, Angular CLI, npm, or PostgreSQL installed locally to run the project.

Development tools may still be used locally during development.

## Code Style

All code identifiers must be in English:

- variables
- functions
- types
- structs
- methods
- filenames
- package names
- API fields

Comments, when genuinely useful, should be written in Brazilian Portuguese.

All user-facing text must be written in Brazilian Portuguese, including:

frontend labels
buttons
form validation messages
notifications
dialogs
success messages
error messages shown to the user

Internal code identifiers must remain in English.

Developer-facing logs, internal error codes, API field names, function names, variables, types, and package names should remain in English unless explicitly requested otherwise.

The application's target users are Brazilian, so the frontend language must be Brazilian Portuguese.
Avoid obvious comments. Prefer self-explanatory names.

## Scope

Follow the requested task exactly.

Do not proactively implement:

- optional requirements
- future endpoints
- authentication
- additional infrastructure
- speculative abstractions

unless explicitly requested.

When multiple valid approaches exist, prefer the simplest one that correctly satisfies the technical challenge requirements.
