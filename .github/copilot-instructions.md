# Copilot Chat Repository Instructions

Before answering or changing code in this repository, read these documents first:

1. [docs/development-norms.md](../docs/development-norms.md)
2. [docs/customization-plan.md](../docs/customization-plan.md)
3. [docs/project-structure.md](../docs/project-structure.md)
4. [docs/change-log.md](../docs/change-log.md)

## Working rules

- Prefer minimal changes.
- Keep new logic in new files when possible.
- Touch existing source files only for small hooks, wiring, or registration.
- Keep any new behavior documented.
- If a change depends on current runtime state or unknown types, inspect the nearby code first and then implement the smallest safe patch.

## Project priorities

- Preserve the original author's code style and structure where possible.
- Keep customization code easy to reapply after upstream updates.
- Use the documentation files above as the source of truth for customization decisions.
