# Sub2Video Design System

This is the source of truth for Sub2Video visual decisions. Read it before changing UI.

## Design revision for review (2026-10-08)

[UI design specification proposal](./ui-design-spec-2026-10-08.md) records the user's homepage/login feedback, verified typography issues, proposed type/control scales, layout diagrams and acceptance criteria. The homepage/authentication background and entrance-specific scales below have been implemented locally; broader product typography proposals remain for review. The user has explicitly clarified the full-background direction below; it supersedes the previous inset-media composition, with implementation and release evidence recorded separately. Adopt remaining confirmed details here alongside implementation and reviewed snapshots; do not maintain a competing design system.

### Full-background entrance (2026-10-08)

The homepage first screen and the entire login/register viewport use edge-to-edge video or a matching poster. Background media extends beneath navigation, text and forms, with zero outer margin, border or radius and no content max-width. Use proportional cover cropping without stretching. The 1760px maximum and safe gutters constrain foreground content and below-fold sections only; there must be no empty side bands, centered rounded media card or opaque full-height form-side panel. Desktop forms overlay the shared background; mobile retains a full-viewport poster/background with a single-column foreground form, not a short media header. Tall forms scroll naturally and remain usable with the keyboard open. Local dark scrims provide contrast without splitting the background. Actual delivered-video players continue to preserve the complete frame using contain. The current change implements this composition; local verification and 107 release evidence remain separate facts.

Entrance scales: desktop form 400px (420px at 1920px and above, 360px around 1024px), mobile within 16px safe gutters; form titles 28/38 desktop and 24/34 mobile, description 16/26, labels 14/20 at actual weight 500, input text 16/24 and 48px input height. Authentication primary buttons use the shared `xl` size (48px, 16/24, weight 500); the existing `lg` remains 44px for unaffected working surfaces. Shared input `lg` provides the larger fields without overriding component visuals in features. Password visibility and verification actions have 44px touch targets. Homepage scales follow the approved refinement below. Remaining general workspace scales below are unchanged.

### Approved homepage refinement (2026-10-09)

The user approved [the refined homepage prototype](./homepage-design-2026-10-09/refined/index.html). Its three-part composition supersedes the expanded homepage sections: full-screen brand film with one headline and one primary creation action; one mode-switching product illustration; simplified footer. Keep tutorials and help reachable through actual routes instead of expanding their contents on the homepage. Use the existing palette, fonts and tokens, not a second design system.

Homepage title: 64/80px desktop, 72/90px at 1920px and above, 40/52px mobile, 34/46px at 320px, 360–389px or short mobile screens. Keep the two semantic title lines intact, including when a native scrollbar reduces the usable width. Section title: 40/52px desktop, 28/40px mobile (26px at 320px). Mode labels: 20/30px desktop, 16/26px mobile. Mode explanation: 20/32px desktop, 18/28px mobile. Auxiliary labels and footer: 16/26–28px. Primary creation action: 18/28px, minimum height 56px desktop and 52px mobile. These entrance-specific scales do not alter workspace typography.

The mode illustration uses accessible Reka tabs for 文生视频、首尾帧、多图, one sentence per mode and a single workbench labelled 创作界面示意. Desktop input column is 340px, 280px at 768–1023px; reference preview fills the remainder. Mobile retains a compact description/material strip, then the reference preview and a real mode-specific creation link. Hide illustrative parameters on mobile, not all inputs. Reference media is explicitly labelled 参考画面, handles image failure, and is never presented as generated video. The approved static estimate-button illustration becomes a real 去创作 link; the homepage never calculates prices or submits tasks.

### Right-aligned public typography refinement (2026-10-09)

The user's follow-up supersedes the centered homepage headline/action. Keep the two-line headline at the right foreground axis; place the text-only 开始创作 link independently in the former bottom-right playback-control position, 32px from the desktop bottom and at least 24px from the mobile bottom/safe area. The title and action share the right axis with navigation and below-fold content. Mobile moves the headline down into the darker lower composition, with at least 24px clear space above the action. Keep the headline at 48/64px below 1024px, 40/52px on phones and 34/46px on narrow or short phones; other approved scales remain.

Public H1/H2 use the self-hosted `S2V Public Display` Medium (500) web subset derived from LXGW WenKai v1.522. Navigation, body, labels and actions use Source Han Sans SC; the Inter wordmark retains its brand role. Apply these families through `--s2v-font-public-display` and `--s2v-font-reading`; do not change workspace or authentication typography. Font source, subset coverage, hashes and SIL OFL license are documented under `frontend/public/fonts/`.

The homepage has no playback, pause or retry controls/captions. It plays muted video automatically after load/idle on eligible desktops, retains the poster on phones, reduced-motion/data-saving settings or playback failure, pauses offscreen/hidden, and resumes only while eligible. Genuine media failure must not trigger repeated requests. Login/registration keep their existing playback controls. Removing the homepage pause action is the user's explicit visual requirement; do not claim full conformance with continuous-animation pause-control accessibility criteria.

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

Borrow Xiaomi's disciplined alignment, generous media scale and simple hierarchy, while retaining Sub2Video's brand, truthful media labels and existing palette. Header, hero foreground, section grids and footer share one outer content axis. Public foreground content uses the same 1760px ceiling and fluid gutter; the confirmed full-background revision above exempts background media from this width limit. Use an independent full-width background layer, not a translated hero inside a narrow wrapper. Homepage headline and primary action use the right foreground axis as refined above; the mode illustration and footer share balanced gutters. Long reading text keeps a local line-length limit rather than narrowing the entire page.

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

The product homepage's first screen is a full-bleed film background with overlaid navigation, one title and one primary action. Below it is the single mode illustration and simplified footer (approved 2026-10-09 refinement). Do not restore expanded tutorials, steps, FAQ, repeated kickers or closing CTAs. Navigation targets `/#modes`, `/guides/coast-letter` and `/app/tickets`; mobile retains tutorials/help in the footer. Mode links preserve the selected mode through authentication without replacing an existing draft. Browsing public content must not request private workspace APIs or start estimates/tasks.

The homepage, login and registration use the same self-hosted anime brand film and matching poster described in [auth-entrance.md](./auth-entrance.md). Do not add visible media credits or decorative captions to the entrance; provenance remains documented in ATTRIBUTION.md. Homepage desktop playback begins after first paint/load; mobile, reduced-motion and data-saving visitors retain the poster without a playback control. Pause when hidden or offscreen. A playback error retains the poster and working creation/tutorial links. Authentication retains its separate playback/retry controls. Promotional copy must reflect shipped capabilities: individual video shots, current estimates, task delivery and external editing of a complete film. Do not invent prices, generated case studies, customer counts or available online payments.

Tutorial links contain trusted editorial IDs only. Authentication preserves the internal destination. Existing drafts require explicit application before replacing prompt/settings. Actual capabilities come from the creation catalog; unavailable tutorial modes cannot silently switch to another mode. Images must be supplied by the creator.

Successful submission opens `/app/works/:id`. Show an actual video only when the task has a ready delivery URL. Preserve download and save actions when browser playback fails; absence of a delivery URL is a distinct pending/unavailable state. Saving requires a chosen project and retains errors in the dialog. Open the returned folder through `/app/assets?project=...&folder=...`; the page must restore this location after refresh.

### Administration

Concise page heading, primary action, tabs/filters, records, pagination, modal editing. Remove decorative count cards where the list/tabs already answer the question. Internal supply/cost details remain administrator-only.

### Login and registration

`/auth/login` and `/auth/register` share one continuous full-viewport film background with a focused overlaid form, right-aligned on desktop and centered on smaller screens (2026-10-08 revision). A local scrim protects text contrast; it must not become an opaque full-height right panel. Use the existing client neutrals, amber action, wordmark and control recipes; no floating dashboard card, fake social sign-in or decorative status metrics. The media sourcing and authentication behavior in [auth-entrance.md](./auth-entrance.md) remain relevant, but its previous left-media/right-form and compact-mobile-media composition is superseded.

The entrance media is the user-supplied anime creator/studio brand film and matching first-frame poster (2026-10-08). Use `creator-studio-loop.mp4` and `creator-studio-poster.jpg` on all three entrances. Authentication retains the short main headline but removes the explanatory paragraph, brand-media caption and redundant footer tagline. Legal links and a quiet, text-only playback control share one footer; do not restore decorative media SVGs. Record each source and its own rights separately in `frontend/public/brand/auth/ATTRIBUTION.md`; do not transfer the historical coast license or authorship to brand media, or present either as a verified customer result. Keep cover cropping at 58% horizontally on desktop and 30% below 1024px so the creator's face stays within the mobile crop. The processed 9.625-second H.264 loop has no audio track, faststart metadata and a short crossfade; minor transition softness remains, so it is not a claim of perfect seamlessness. Reduced-motion, data-saving and small-screen users start with the poster. Media failure never blocks authentication. Original media processing evidence is recorded in [anime-auth-background-2026-10-08.md](../testing/anime-auth-background-2026-10-08.md); release evidence is recorded separately.

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
