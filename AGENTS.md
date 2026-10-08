# RentalCore Agent Rules

## Mandatory suite design contract

- Before any UI work, read `../docs/DESIGN_SYSTEM.md` and `../theme/README.md` in the `cores` umbrella, or the canonical documents in `github.com/nbt4/cores` for standalone work.
- `web/src/cores-theme.css`, `web/src/lib/cores-design.ts` and `web/static/css/cores-theme.css` are generated and must never be edited directly.
- Both the React client and legacy Go templates obey the same suite tokens. Orange/brown Rental-only UI accents, white “muted” text and independent scrollbar/select rules must not be reintroduced.
- Dashboard greetings must use `suiteGreeting()`. Shell geometry, typography, tables, forms, dropdowns and dashboards follow the common contract.
- Run the umbrella design check, frontend build and Go tests before release; update README documentation for visible behavior changes.
