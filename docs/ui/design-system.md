# Sub2Video Design System

This document is the single source of truth for the Sub2Video UI. Cursor,
Codex, Copilot, and human contributors must read it before changing UI.

## Product Surfaces

- Client: creator workflow, assets, works, wallet, support, and profile.
- Admin: workflow supply, providers, pricing, tasks, users, support, and system
  operations.
- Client and admin use separate layouts and route namespaces. They share
  primitives, but not navigation or page responsibilities.

## Visual Language

The reference HTML uses a compact white top bar, a white 240px sidebar, a soft
slate canvas, fine slate borders, restrained shadows, and dense information
surfaces. Client emphasis is indigo; admin emphasis is emerald.

### Typography

```css
font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI",
  "Microsoft YaHei", sans-serif;
font-family: "JetBrains Mono", "SFMono-Regular", Consolas, monospace;
```

Inter and JetBrains Mono should be self-hosted under `public/fonts` in production
with `font-display: swap`. The Chinese system fallback remains necessary.

### Token Ownership

Design tokens define values, not component behavior. Component recipes decide
how those values are used. Pages only compose components and layout.

| Decision | Owner | Examples |
| --- | --- | --- |
| Brand colors, type, radii, shadows | `shared/styles/tokens.css` and `tailwind.config.ts` | primary, canvas, card radius |
| Button states and sizes | `shared/ui/S2VButton.vue` | primary, danger, loading |
| Card density and emphasis | `shared/ui/S2VCard.vue` | default, metric, highlighted |
| Page grid and responsive composition | `widgets/` and `pages/` | columns, gaps, max width |
| Product-specific special treatment | `features/` | upload card, task refund action |

## Themes

```html
<div data-surface="client">
<div data-surface="admin">
```

Client tokens use indigo. Admin tokens use emerald. Shared components consume
semantic `s2v-*` variables and do not branch on client/admin in templates.

## Component Rules

Use a shared component when the visual pattern repeats. Use a feature component
when the behavior or content is product-specific. Do not expose arbitrary visual
class overrides on shared components.

Add a public variant only when at least two business surfaces use the same
semantic state. One-off visual exceptions belong in a feature component.

Preferred usage:

```vue
<S2VButton variant="primary" size="lg">开始生成</S2VButton>
<S2VCard variant="metric">...</S2VCard>
```

Avoid:

```vue
<button class="rounded-xl bg-indigo-600 px-4 py-2 text-white">...</button>
<S2VCard class="border-2 border-dashed">...</S2VCard>
```

## Tailwind Rules

- Tailwind is used inside shared and feature components, not as a page-level
  visual language.
- Pages may use layout utilities (`grid`, `flex`, `gap`, `max-w`, `overflow`).
- Pages may not use raw color, shadow, border, radius, or control recipes.
- No arbitrary visual values (`bg-[#...]`, `px-[...]`, `rounded-[...]`).
- `@apply` is limited to global reset and shared focus utilities.
- Class merging goes through `shared/lib/cn.ts`.

## Responsive Contract

- Minimum supported width: 320px.
- Desktop client/admin: 240px sidebar and top bar.
- Client mobile: top bar plus bottom navigation for overview, create, works,
  wallet, and profile; secondary pages live in the mobile menu.
- Admin mobile: top bar plus grouped drawer; tables become cards or explicit
  horizontal scroll regions.
- Interactive targets are at least 44px on touch devices.
- No page may introduce horizontal overflow.

## UI Change Decision Tree

1. Global brand or primitive value? Change tokens.
2. Same component behavior everywhere? Change the shared component recipe.
3. Repeated semantic state? Add a named variant.
4. One business surface only? Create a feature component.
5. Only layout/order/columns? Change the widget or page composition.

## Visual Acceptance

Reference HTML files are visual baselines only. Production pages are Vue
components. Playwright screenshots cover client/admin desktop (1440x900) and
mobile (390x844). Use deterministic mock data, local fonts/assets, disabled
animations, and `document.fonts.ready` before capture.
