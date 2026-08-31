# Sub2Video Engineering Rules

## Project Scope

Sub2Video is a creator-facing video generation product. It is not an API gateway.
Do not add client-facing API keys, provider accounts, channel management, proxy
management, subscriptions, plugins, or API quota pages.

Before changing UI, read `docs/ui/design-system.md`. That file is the single
source of truth for visual tokens, component ownership, responsive behavior, and
AI-assisted changes.

Visual chrome follows `docs/原型设计/sub2video-client.html` and
`docs/原型设计/sub2video-admin.html`. Menus, routes, and the works/assets model
follow `docs/产品方案总览.md`. Do not copy HTML sidebar names into Vue.

Recipe-level visual alignment with those HTML files must update Playwright
snapshots in the same change. Do not update snapshots only to hide drift.

## Frontend Stack

- Vue 3 + TypeScript + Vite
- Vue Router 4 + Pinia
- Tailwind CSS 3.4 as the styling implementation layer
- Reka UI for headless interaction primitives
- `tailwind-variants` for component variants
- `clsx` + `tailwind-merge` through `shared/lib/cn.ts`
- Lucide Vue icons
- Mock data only until backend integration is explicitly requested

## Layer Ownership

- `shared/styles` and `tailwind.config.ts`: design tokens only.
- `shared/ui`: reusable visual components and their variants.
- `entities` and `features`: domain behavior and business components.
- `widgets` and `pages`: composition and layout only.
- Pages may use layout utilities such as `grid`, `flex`, `gap`, and `max-w`.
- Pages must not define colors, shadows, borders, radii, or button recipes.
- Do not override shared component visuals with arbitrary `class` values.
- Add a public variant only when at least two business surfaces need it.

## Tailwind Rules

- Never use arbitrary visual values such as `bg-[#123456]` or `px-[13px]` in
  pages, features, widgets, or entities.
- Never use `!important` for application UI.
- Use `cn()` for class merging and `tailwind-variants` for stateful variants.
- Keep `@apply` limited to global reset and cross-component utilities such as
  focus rings. Do not use it for business components.

## Validation

Every UI change must pass:

1. `pnpm lint:check`
2. `pnpm typecheck`
3. relevant unit tests
4. desktop and mobile browser inspection
5. visual regression tests when a shared component or layout changes

Do not update Playwright snapshots just to hide an unintended visual change.
Snapshot updates require an explicit reviewable change.

## Repository Safety

- Keep mock data deterministic.
- Do not add secrets, credentials, generated build output, or logs.
- Keep changes scoped to the Sub2Video project.
- Do not modify the existing `sub2api` worktree for this project.
