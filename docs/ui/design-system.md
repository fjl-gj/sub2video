# Sub2Video Design System

This is the source of truth for Sub2Video visual decisions. Read it before changing UI.

## Current baseline: commercial studio (2026-09-25)

The user explicitly requested a redesign referencing 小云雀、可灵 AI、有戏 AI, replacing the earlier generic dashboard styling. The current visual baseline is [studio-redesign.html](./studio-redesign.html); exported design images cover creation, overview, administration and mobile. [studio-redesign.md](./studio-redesign.md) records the research, decisions and acceptance criteria.

The earlier files under docs/原型设计 are retained as historical prototypes. Their indigo/emerald palette, rain animation and dashboard metric recipes are superseded by this baseline. Do not combine the old and new visual languages.

Product menus, routes, task/order/asset boundaries and business contracts remain unchanged. User-facing names remain 首页概览、创作、广场、任务、资产、回收站、钱包、帮助与工单. Do not add subscriptions, agents, API keys or features merely because a reference product has them.

The connected creator journey is defined in [creator-journey.html](./creator-journey.html) and [creator-journey.md](./creator-journey.md). The public product homepage `/` and editorial tutorial `/guides/:id` precede the authenticated workspace. `/app/overview` remains the private overview. 广场 now lists the same clearly labelled editorial tutorials; generic sample videos are not presented as generated case studies.

## One language, two surfaces

Client: a dark creator workspace focused on input materials, video results and iteration.
Admin: a light operational workspace focused on configuration, readable data, filtering and edits.

Both surfaces use the same typography, wordmark, warm amber brand family, icon language, spacing, control recipes and status meanings. A difference in working background must not become two unrelated brands.

| Token | Client | Admin / light baseline |
| --- | --- | --- |
| Canvas | 20 22 22 | 245 246 244 |
| Surface | 27 30 28 | 255 255 255 |
| Ink | 244 244 240 | 37 44 38 |
| Body | 216 222 216 | 54 63 56 |
| Muted | 162 170 166 | 100 112 103 |
| Line | 50 56 50 | 219 226 216 |
| Primary | 232 185 129 | 132 88 43 |
| Primary foreground | 37 34 28 | 255 255 255 |
| Card radius | 12px | 12px |
| Control / nav radius | 8px | 8px |

Brand accent is reserved for the principal action, selected navigation and focus. Statuses use semantic success/warning/danger tokens. Do not reuse brand color for success. Media overlays use a separate dark media-overlay token; ink becomes light in the client and cannot be used as a black overlay.

Use neutral surfaces, fine borders, little or no shadow. No ornamental neon gradients, sparkle icons on every action, fake badges, decorative canvas animation or repeated containers inside containers. Image/video content supplies the visual richness.

## Typography and layout

- Source Han Sans SC for reading; Inter for the wordmark; JetBrains Mono selectively for money and IDs.
- Keep self-hosted fonts under frontend/public/fonts. Use existing system CJK fallbacks when a subset lacks a character.
- Page title: 25px / 500 on desktop, 22px on mobile. Section title: 15–16px / 500.
- Main controls and reading text: 13–14px; auxiliary labels: 11–12px. Avoid long text at 10px.
- Desktop header: 64px. Sidebar: 200px. Workspace content uses a fluid 24–40px horizontal gutter and a 1760px maximum; center it within the area remaining after the sidebar. Reserve scrollbars on both edges to keep that axis stable.
- Client nav row: 42px; admin nav row: 36px. Primary touch actions: at least 44px.
- Mobile header: 56px; 16px content gutters. Client retains bottom navigation with safe-area padding.
- Minimum supported width: 320px. Mobile drawers must close on navigation and Escape.
- Admin data tables scroll inside their container; never squeeze seven columns into vertical text.

### Responsive alignment refinement (2026-09-29)

Borrow Xiaomi's disciplined alignment, generous media scale and simple hierarchy, while retaining Sub2Video's brand, truthful media labels and existing palette. Header, hero, section grids and footer share one outer content axis. Public content uses the same 1760px ceiling and fluid gutter; do not mix a separately translated viewport-width hero with a narrow body. Homepage headline and actions are centered; tutorial/process/FAQ sections use balanced columns. Long reading text keeps a local line-length limit rather than narrowing the entire page.

Workspace pages are named inline-size containers. Creation changes to a single column when actual content width is at most 840px; results and asset folders collapse based on their available area, not only viewport width. This prevents a desktop sidebar plus an internal sidebar or three narrow cards at tablet widths. Keep input before preview on phones, 16px outer gutters and clearance for the bottom navigation/safe area. Grid symmetry means consistent axes and spacing, not forcing forms and video previews to identical widths.

Verify 320, 390, 768, 1024, 1440 and 1920px viewports. Check both geometry (alignment, overflow, usable controls) and actual screenshots; desktop/mobile snapshot baselines alone miss the tablet and wide-screen gaps.

## Core page composition

### Creation

Desktop: an input column scaling from 320 to 420px on the left, flexible preview/results on the right. Mobile: mode and creation inputs precede the preview.

Input sequence: fixed mode, required source materials, prompt, explicitly labelled settings, estimate and submit. Carrier/model remain visible preferences. Supplier, API/workflow, cost and upstream identifiers remain hidden.

Empty preview: explain the next step. An image is a reference preview, never a simulated movie. Show video controls only for an actual delivered video URL. Recent results come from API data and navigate to real task details.

Estimate states: waiting, calculating, available, failed. A missing estimate is never ¥0.00. Superseded responses must not overwrite the latest selection.

### Overview and assets

Overview: three existing mode entrances, recent delivered video, active tasks, then compact balance/storage information. Modes use the real creation catalog for availability. No manufactured tasks or promotional statistics.

Assets: project identity, file count and recent context; preserve current folder model and actions. Use actual covers when available or a clear folder treatment. Never use sample photos as a user's work.

### Public tutorials and delivery

Public pages share the client tokens and wordmark. Use editorial headings, composition references, a small tutorial library and concrete preparation steps. Reference photos must be labelled as such, never as delivered video or verified generation results.

The product homepage uses a wide film stage, three separate mode entrances, an editorial tutorial section, a four-step creation/delivery explanation and keyboard-accessible FAQ. Keep the hierarchy and imagery distinct from a dashboard. Public navigation targets `#modes`, `#guides` and `#process`; mode links preserve the selected mode through authentication without replacing an existing draft. Browsing public content must not request private workspace APIs or start estimates/tasks.

The homepage reuses the self-hosted licensed reference footage described in [auth-entrance.md](./auth-entrance.md), with visible source/license credit. Desktop playback begins after first paint/load; mobile, reduced-motion and data-saving visitors see the poster until they choose to play. Pause when hidden or offscreen. A playback error retains the poster, retry control and working creation/tutorial links. Promotional copy must reflect shipped capabilities: individual video shots, current estimates, task delivery and external editing of a complete film. Do not invent prices, generated case studies, customer counts or available online payments.

Tutorial links contain trusted editorial IDs only. Authentication preserves the internal destination. Existing drafts require explicit application before replacing prompt/settings. Actual capabilities come from the creation catalog; unavailable tutorial modes cannot silently switch to another mode. Images must be supplied by the creator.

Successful submission opens `/app/works/:id`. Show an actual video only when the task has a ready delivery URL. Preserve download and save actions when browser playback fails; absence of a delivery URL is a distinct pending/unavailable state. Saving requires a chosen project and retains errors in the dialog. Open the returned folder through `/app/assets?project=...&folder=...`; the page must restore this location after refresh.

### Administration

Concise page heading, primary action, tabs/filters, records, pagination, modal editing. Remove decorative count cards where the list/tabs already answer the question. Internal supply/cost details remain administrator-only.

### Login and registration

`/auth/login` and `/auth/register` share a film-led entrance: a large left-hand media panel and a focused right-hand form on desktop, a compact media header above the form below 900px. Use the existing client neutrals, amber action, wordmark and control recipes; no floating dashboard card, fake social sign-in or decorative status metrics. See [auth-entrance.md](./auth-entrance.md).

Self-host licensed footage and its poster. Keep visible authorship/license credit and identify reference footage honestly; it is not a generated result or customer case study. Muted playback has a visible pause control. Reduced-motion, data-saving and small-screen users start with the poster. Media failure never blocks authentication.

Forms support native keyboard submission, password managers, password visibility and meaningful errors. Register only after public settings are loaded and registration is enabled; require explicit terms agreement. Preserve the internal destination across login/register, and distinguish an expired session from a temporary connection failure. A visually complete entrance does not establish backend, email delivery or commercial acceptance.

## Ownership and implementation

| Responsibility | Owner |
| --- | --- |
| Global values | shared/styles/tokens.css + tailwind.config.ts |
| Shared chrome / button / nav / dialog recipes | shared/ui |
| Domain visuals and behavior | features / entities |
| Composition only | pages / widgets |

Continue using Vue 3, TypeScript, Vite, Vue Router, Pinia, Tailwind 3.4, Reka UI, tailwind-variants, cn(), and Lucide.

Pages may compose layout utilities. They must not define visual color/shadow/radius/button recipes. No arbitrary visual Tailwind values, !important, or feature-specific overrides of shared components. Add public component variants only when at least two surfaces need them.

Feature CSS must be scoped by a feature root or unique namespace. Never rely on route CSS loading order. Dialogs and other portals inherit the current surface from the document root, including after client/admin navigation.

Existing legacy recipes in tokens.css are retained for unaffected components; place new component recipes in shared/ui or the owning feature rather than expanding that legacy block.

## Truthful states and accessibility

- All actionable controls have meaningful names and visible focus.
- Upload, estimate, submission, preview, empty, loading and error states have distinct copy.
- Balance, tasks and downloads come from APIs; no mock success in production pages.
- Keep diagnostic supplier details out of client error messages.
- Use proper video/image media, handle media failure, and preserve download/detail actions where supported.
- Preserve keyboard interaction and Reka focus management in modals.
- Design prototype photos and amounts are labelled examples; prototype is not a backend acceptance environment.

## Verification

Every UI change must pass lint:check, typecheck, relevant interaction tests, desktop/mobile inspection, and visual regression when primitives/layout change.

Screenshots use deterministic fixtures and locally loaded fonts. Intentional baseline updates must be reviewed visually. Do not mask layout mistakes by increasing screenshot tolerances.

UI mocks establish layout and interaction contracts only. They do not verify PostgreSQL/Redis, supplier execution, wallet accounting, real delivery or commercial readiness.
