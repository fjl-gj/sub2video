# Sub2Video UI Rules

Read `docs/ui/design-system.md` before changing UI. Visual chrome follows its
current commercial-studio baseline. `docs/原型设计/` contains historical references.
Menus, routes, and the works/assets model follow `docs/产品方案总览.md`. Do not
copy HTML sidebar names into Vue.

Use Vue 3 + TypeScript + Pinia + Vue Router. Reuse `src/shared/ui` components.
Pages may compose layout utilities but must not hard-code visual decisions such
as colors, shadows, radii, borders, or button recipes. Do not use arbitrary
Tailwind values. Use `cn()` for class merging and `tailwind-variants` for
variants. Keep client and admin routes and layouts separate. Use semantic status
tokens; do not use the brand accent to mean success. Run typecheck and lint after
changes. Recipe-level visual alignment must update Playwright snapshots in the
same change.
