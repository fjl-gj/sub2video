# Sub2Video Design System

This document is the single source of truth for the Sub2Video UI. Cursor,
Codex, Copilot, and human contributors must read it before changing UI.

## Visual vs product IA

Two sources, two jobs. Do not mix them.

| Concern | Source of truth | Do not |
| --- | --- | --- |
| Color, type, radius, shadow, density, chrome | `docs/原型设计/sub2video-client.html` and `docs/原型设计/sub2video-admin.html` | Invent a third visual language |
| Menus, routes, works/assets object model | `docs/产品方案总览.md` | Copy HTML sidebar names (广场 / 创作台 / 作品集) into Vue |

The HTML files are visual baselines only. Production pages are Vue. Keep product
IA (创作 / 创意园 / 作品 / 资产) even while matching HTML chrome, type, and
component recipes.

## AI change checklist

```text
1. Menu / route / works·assets model → 产品方案总览.md (never the HTML sidebar)
2. Color / font / radius / shadow / spacing → tokens.css (values from the HTML)
3. Button / card / nav item / page header / dialog / toast → shared/ui
4. One-off business chrome → features
5. Columns, order, max-width → widgets / pages (layout utilities only)
```

Before editing UI: read this file, then the matching HTML surface. After a
recipe-level visual change, update Playwright snapshots in the same change and
name the HTML baseline in the review.

## Product Surfaces

- Client: creator workflow, assets, works, wallet, support, and profile.
- Admin: workflow supply, providers, pricing, tasks, users, support, and system
  operations.
- Client and admin use separate layouts and route namespaces. They share
  primitives, but not navigation or page responsibilities.

## Visual Language

Compact frosted white top bar (`h-14`, `bg-white/95`, `backdrop-blur-md`), white
240px sidebar, soft slate canvas (`slate-50/70`), fine `slate-200/80` borders,
micro-shadows, dense information surfaces.

Client emphasis is indigo (`#4f46e5`). Admin emphasis is emerald (`#059669`).
Health, online, and success states stay green on both surfaces. Do not paint the
GPU cluster pill with `--s2v-primary` on the client.

### Typography

Self-host Inter and JetBrains Mono under `frontend/public/fonts` with
`font-display: swap`. Load them with `@font-face` and Latin `unicode-range` so
Latin letters and numerals never fall through to a CJK face.

```css
/* Latin + numerals only. Do not put Microsoft YaHei on this family. */
font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
  sans-serif;

/* CJK runs use the next face per glyph, not the whole line. */
font-family: Inter, "PingFang SC", "Microsoft YaHei", sans-serif;

font-family: "JetBrains Mono", "SFMono-Regular", Consolas, monospace;
```

Money, IDs, duration, resolution, GPU counts, and nav trailing tags use
`font-mono`. Do not declare `font-mono` without the Mono `@font-face` files.

### Token values (from the HTML, measured)

| Token | Value |
| --- | --- |
| Canvas | `248 250 252` at 70% (`slate-50/70`) |
| Surface | `255 255 255` |
| Ink | `15 23 42` (`slate-900`) for titles; body `30 41 59` (`slate-800`) |
| Muted | `100 116 139` / labels `148 163 184` |
| Line | `226 232 240` at 80% |
| Client primary | `79 70 229` (`indigo-600`); active nav text `67 56 202` |
| Admin primary | `5 150 105` (`emerald-600`); active nav text `4 120 87` |
| Card radius | **16px** (`rounded-2xl`) |
| Control radius | **8px** nav item / chips; **12px** buttons (`rounded-xl`) |
| Nav item | height **36px**, icon slot **20px**, icon **16px**, label **13px** |
| Page title | **20px** / `font-bold` |
| Content | `max-w-6xl` (1152px), main padding 32px |
| Micro-shadow | `0 1px 3px rgba(0,0,0,0.04), 0 1px 2px -1px rgba(0,0,0,0.02)` |
| Top bar | 56px, frosted white, no solid slab |

`--s2v-radius` is the card radius (16px). Buttons use `rounded-xl` (12px). Nav
items use `rounded-lg` (8px). Do not collapse these three into one number.

### Token Ownership

Design tokens define values, not component behavior. Component recipes decide
how those values are used. Pages only compose components and layout.

| Decision | Owner | Examples |
| --- | --- | --- |
| Brand colors, type, radii, shadows | `shared/styles/tokens.css` and `tailwind.config.ts` | primary, canvas, card radius |
| Button states and sizes | `shared/ui/S2VButton.vue` | primary, danger, loading |
| Card density and emphasis | `shared/ui/S2VCard.vue` | default, metric, muted, highlight |
| Nav item slot and badges | `shared/ui/S2VNavItem.vue` | active, trailing mono tag |
| Page header chrome | `shared/ui/S2VPageHeader.vue` | badge + 20px title + rule |
| Dialog / toast | `shared/ui/S2VDialog.vue`, `S2VToast.vue` | confirm, success, danger |
| Page grid and responsive composition | `widgets/` and `pages/` | columns, gaps, max width |
| Product-specific special treatment | `features/` | upload card, task refund action |

## Themes

```html
<div data-surface="client">
<div data-surface="admin">
```

Client tokens use indigo. Admin tokens use emerald. Shared components consume
semantic `s2v-*` variables and do not branch on client/admin in templates.
Success / online / GPU health always use `--s2v-success`, never `--s2v-primary`.

## Closed components

Use a shared component when the visual pattern repeats. Use a feature component
when the behavior or content is product-specific. Do not expose arbitrary visual
class overrides on shared components.

Add a public variant only when at least two business surfaces use the same
semantic state. One-off visual exceptions belong in a feature component.

| Component | Recipe |
| --- | --- |
| `S2VButton` | `primary` is solid brand fill, no grey border, `font-bold`, micro-shadow. `secondary` is white + `slate-200` border + micro-shadow. `variant` is a real Vue prop. |
| `S2VCard` | 16px radius, micro-shadow, `slate-200/80` border. `metric` is the overview stat cell. Pages must not pass `rounded-*`, `shadow-*`, `border-*`, or `bg-*`. |
| `S2VNavItem` | Only sidebar row. Height 36px, 20px icon slot, 13px label, 8px radius. Optional trailing `font-mono` tag. |
| `S2VPageHeader` | Row with optional badge, 20px title, bottom border. Not eyebrow + 23px + long subcopy. |
| `S2VBadge` | Small rounded square (`rounded`), not a full pill, except status dots. Use semantic tones. |
| `S2VDialog` | Reka UI headless + frosted center modal. Destructive actions use this, not toast. |
| `S2VToast` | Bottom-right system toast with success / warning / danger tones. |
| `S2VInput` / `S2VSelect` | Shared control radius. Pages must not draw raw inputs. |

Preferred usage:

```vue
<S2VButton variant="primary" size="lg">开始生成</S2VButton>
<S2VCard variant="metric">...</S2VCard>
<S2VPageHeader badge="S2V Video Studio" title="欢迎回来，李维工作室" />
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
- Never use `!important` for application UI.

## Responsive Contract

- Minimum supported width: 320px.
- Desktop client/admin: 240px sidebar and 56px top bar.
- Desktop nav / toolbar rows: **36px**. Do not stretch sidebar items to 44px.
- Client mobile: top bar plus bottom navigation for overview, create, works,
  wallet, and profile; secondary pages live in the mobile menu.
- Admin mobile: top bar plus grouped drawer; no four-item bottom bar. Tables
  become cards or explicit horizontal scroll regions.
- Touch primary targets (bottom nav, main CTA) are at least 44px.
- No page may introduce horizontal overflow.

## UI Change Decision Tree

1. Global brand or primitive value? Change tokens.
2. Same component behavior everywhere? Change the shared component recipe.
3. Repeated semantic state? Add a named variant.
4. One business surface only? Create a feature component.
5. Only layout/order/columns? Change the widget or page composition.

## Visual Acceptance

Playwright screenshots cover client/admin desktop (1440x900) and mobile
(390x844). Use deterministic mock data, local fonts/assets, `document.fonts.ready`,
and disable animations in the screenshot. Runtime UI still uses `badge-pulse` for
GPU online; do not delete the animation to satisfy tests.

Recipe-level alignment with the HTML baselines **must** update snapshots in the
same change. Do not update snapshots only to hide an unintended drift.
