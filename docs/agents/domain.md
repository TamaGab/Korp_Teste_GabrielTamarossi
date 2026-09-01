# Domain Docs

## Before exploring, read these

- `CONTEXT.md` at the repository root.
- Relevant ADRs under `docs/adr/`.

If these files do not exist, proceed silently. `/domain-modeling` creates them lazily when terminology or decisions are resolved.

## File structure

This repository uses a single-context layout:

```
/
├── CONTEXT.md
└── docs/adr/
```

## Use the glossary's vocabulary

Use terms as defined in `CONTEXT.md` in issues, tests, designs, and implementation. Avoid synonyms that the glossary explicitly rejects. Treat missing terminology as a possible domain-modeling gap.

## Flag ADR conflicts

Explicitly identify any proposal that contradicts an existing ADR rather than silently overriding it.
