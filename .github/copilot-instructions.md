# Sub2Video UI Rules

Read `docs/ui/design-system.md` before changing UI.

Use Vue 3 + TypeScript + Pinia + Vue Router. Reuse `src/shared/ui` components.
Pages may compose layout utilities but must not hard-code visual decisions such
as colors, shadows, radii, borders, or button recipes. Do not use arbitrary
Tailwind values. Use `cn()` for class merging and `tailwind-variants` for
variants. Keep client and admin routes and layouts separate. Run typecheck and
lint after changes.
