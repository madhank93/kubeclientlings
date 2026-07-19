# KubeClientlings Website — Design

Date: 2026-07-19
Status: Approved

## Goal

Public website for KubeClientlings at `https://kubeclientlings.madhan.app`, following the
established pattern of golings.madhan.app and kubelings.madhan.app: a Starlight landing
page plus an interactive exercise catalog. Exercise popups show the broken exercise code
and a hint spoiler, but **never inline solution code** — they link to the solution file
on GitHub instead.

## Approach

Port the golings `web/` site (Astro 6 + Starlight + custom `catalog.astro` + Go
generator) and adapt it to this repo. Rejected alternatives: hand-written static catalog
(drifts out of sync with 52 exercises); kubelings docs port (module/incident structure is
the wrong shape for a topic/exercise catalog).

## Structure

New `web/` directory at repo root:

- `web/package.json`, `astro.config.mjs`, `tsconfig.json` — Astro 6 + Starlight +
  sitemap, patterned on golings.
- `web/gen/main.go` — generator, run from repo root as `go run ./web/gen`. Parses
  `info.toml` via the `kubeclientlings/exercises` package, reads `exercises/**` sources,
  and emits:
  - `web/src/data/catalog.ts` — typed catalog data for the catalog page
  - `web/src/data/lesson-details/<name>.md` — per-exercise detail markdown
  It fails loudly if any info.toml exercise is missing from the tier map or vice versa
  (coverage check, like golings).
- `web/src/content/docs/index.mdx` — landing page. Hero: "Learn client-go the rustlings
  way — 52 exercises, from clientset setup to controller-runtime operators." Sections:
  What is KubeClientlings, quick start (clone + mise), key features (real kind cluster,
  TUI watch mode, broken-on-purpose exercises, covers client-go through
  controller-runtime).
- `web/src/content/docs/getting-started.md` — install/run guide derived from README.
- `web/src/pages/catalog.astro` — interactive catalog: topic filter, text search,
  expandable rows grouped by tier. No mode filter (all 52 exercises are
  `mode = "compile"`).

## Topic tiers

Chip colors per tier, beginner → advanced:

1. Setup — `setup`
2. Core Workloads — `pods`, `deployments`, `services`
3. Requests & Watch — `options`, `watch`
4. Dynamic & CRDs — `dynamic`, `crds`
5. Controller Machinery — `informers`, `workqueue`, `controllers`
6. Advanced Writes — `ssa`, `subresources`, `finalizers`
7. Webhooks & Testing — `webhooks`, `testing`
8. Operator — `operator`

## Exercise popup (key difference from golings)

Each `lesson-details/<name>.md` contains:

1. The broken exercise source in a fenced code block.
2. The hint from info.toml behind a `<details>` spoiler.
3. A link — **not inline code**: "View solution on GitHub →" pointing at
   `https://github.com/madhank93/kubeclientlings/blob/main/solutions/<topic>/<name>/main.go`.

Solutions live on `main` in `solutions/`, so no solution-branch fetch is needed anywhere.

## Deployment

- `.github/workflows/pages.yml` — ported from golings minus the solution-branch step:
  checkout → setup-go → `go run ./web/gen` → setup-node → `npm ci` → `npm run build` →
  deploy to GitHub Pages. Triggers on pushes touching `web/**`, `info.toml`,
  `exercises/**`, or the workflow itself.
- `web/public/CNAME` containing `kubeclientlings.madhan.app`.
- Manual follow-ups for the user: add the DNS CNAME record and enable GitHub Pages
  (Actions source) in repo settings.

## Verification

- `go run ./web/gen` exits 0; coverage check passes; 52 detail files emitted.
- Generated detail files contain no solution code (grep guard in review).
- `npm run build` succeeds; catalog page renders 52 exercises.
- Spot-check: solution links resolve against the GitHub repo layout.
