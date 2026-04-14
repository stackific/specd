# Astro + BeerCSS site template

A reusable Astro starter built around BeerCSS (Material Design 3) and a small custom CSS framework. Static by default, with optional Cloudflare Workers SSR for a contact API route.

## Stack

- [Astro](https://astro.build) (static output)
- [BeerCSS](https://www.beercss.com) — Material Design 3 components
- A custom token + utility CSS framework (see `src/styles/README.md`)
- LightningCSS as the CSS transformer (`@custom-media` support) and esbuild as the minifier
- PurgeCSS as a post-build step on our own bundle (BeerCSS is left untouched)
- Optional: `@astrojs/cloudflare` adapter for the contact form API route
- Optional: AWS SES (via inline SigV4 over `fetch`) and Cloudflare Turnstile for the contact form

## Requirements

- Node.js >= 22.12
- pnpm

## Install

```sh
pnpm install
```

## Develop

```sh
pnpm dev
```

## Build

```sh
pnpm build
pnpm preview
```

## Project structure

```
public/
├── vendor/beer.min.css        BeerCSS, served as its own <link>
├── logo.svg, favicon.*, og-image.png
└── robots.txt

src/
├── site.config.ts             Single source of truth for site identity / SEO
├── layouts/Layout.astro       Global layout (head, nav, footer slot)
├── styles/                    CSS framework foundation (see styles/README.md)
├── components/                Co-located .astro + .css components
└── pages/                     Co-located .astro + .css pages
    └── api/contact.ts         Optional: contact form server route

scripts/
├── gen-utilities.mjs          Generate src/styles/utilities.css
└── gen-colors.mjs             Generate the --c-* tonal palettes from a seed

astro.config.mjs               Build pipeline + custom PurgeCSS integration
CLAUDE.md                      Strict rules for AI agents working in this repo
docs/
├── BEERCSS.md                 BeerCSS reference
└── UTILITIES.md               Utility class quick reference
```

## CSS framework

Read `src/styles/README.md` first if you're going to touch any styles. Short version:

- BeerCSS is vendored to `public/vendor/beer.min.css` and shipped as its own `<link>`. Don't `import "beercss"` from anywhere.
- All custom CSS bundles into ONE file (`vite.build.cssCodeSplit: false`).
- Component and page CSS lives in co-located `.css` files imported from each `.astro` frontmatter — never inside a `<style>` block.
- Spacing, sizing, typography, and color come from generated utility classes (`p-3`, `mt--1`, `text-4`, `lh-3`, `m:p-5`, `l:text-6`, `c-primary-40`). BeerCSS owns components only.
- Every selector in custom CSS must be namespaced (`c-<name>`, `<2-letter>-*`, `home-*`, …) — the bundle is global and unprefixed classes will collide.
- The full rules an agent must follow live in `CLAUDE.md`.

## Templating this project

This repo is designed to be cloned and adapted into new sites. The CSS framework, build pipeline, and `src/styles/` foundation are project-agnostic. The "Per-project find-and-replace checklist" at the bottom of `CLAUDE.md` lists exactly what to update when you start a new project from this template.

The minimum updates are:

1. Update `src/site.config.ts` (name, url, description, og image, theme color).
2. Update `astro.config.mjs` `site:` and `public/robots.txt` sitemap URL.
3. Replace `public/logo*`, `public/favicon*`, `public/og-image.png`.
4. Regenerate the color palette: `pnpm gen:colors '#NEWHEX'` and paste into `src/styles/tokens.css`.
5. Update `package.json` `name`.
6. Replace pages under `src/pages/` and components under `src/components/`.

Decide whether the new project needs the contact form (see below). If not, delete `src/pages/api/contact.ts`, the contact dialog markup and JS in `Layout.astro`, the `@astrojs/cloudflare` adapter from `astro.config.mjs`, and `wrangler.jsonc`.

---

## Optional: contact form (Turnstile + AWS SES)

The "Work with us" form in `Layout.astro` is wired to a server route at `/api/contact` that validates input, verifies a Cloudflare Turnstile token, and sends an email via AWS SES (SES v2 API, signed with SigV4 — no AWS SDK dependency, so it runs in Cloudflare Workers without Node compat shims).

If you don't need this, delete it (see "Templating this project" above). If you do, follow the setup below.

### Environment variables

Create a `.env` at the project root for local dev:

```dotenv
# --- Cloudflare Turnstile ---
# Public site key (exposed to the browser; the PUBLIC_ prefix is required by Astro).
PUBLIC_TURNSTILE_SITE_KEY=0x4AAAAAAA__your_site_key__
# Server-side secret (NEVER expose to the browser).
TURNSTILE_SECRET_KEY=0x4AAAAAAA__your_secret__

# --- AWS SES ---
AWS_SES_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
# Verified SES sender (must be verified in the SES console / domain)
CONTACT_FROM_ADDRESS=no-reply@example.com
# Recipient
CONTACT_TO_ADDRESS=info@example.com
```

For local dev with the Cloudflare adapter, also add the non-`PUBLIC_` secrets to a `.dev.vars` file at the project root (wrangler / the Cloudflare adapter loads this automatically and exposes it on the worker `env`):

```dotenv
TURNSTILE_SECRET_KEY=0x4AAAAAAA__your_secret__
AWS_SES_REGION=us-east-1
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...
CONTACT_FROM_ADDRESS=no-reply@example.com
CONTACT_TO_ADDRESS=info@example.com
```

`PUBLIC_TURNSTILE_SITE_KEY` must go in a regular `.env` file instead — Astro inlines `PUBLIC_*` vars into client-side code at build time.

For production on Cloudflare, set the AWS credentials and Turnstile secret with `wrangler secret put <NAME>`, and the non-secret values (region, from / to addresses) as plain `vars` entries in `wrangler.jsonc`. At runtime, `src/pages/api/contact.ts` reads them via `import { env } from "cloudflare:workers"` (the Astro v6 replacement for the now-removed `Astro.locals.runtime.env`).

### Setting up Turnstile

1. In the Cloudflare dashboard go to **Turnstile** and create a widget.
2. Add `localhost` and your production domain to the allowed hostnames.
3. Copy the **Site key** into `PUBLIC_TURNSTILE_SITE_KEY` and the **Secret key** into `TURNSTILE_SECRET_KEY`.

### Setting up AWS SES

1. In the AWS SES console, verify the **sending domain** (or at minimum the `CONTACT_FROM_ADDRESS` identity).
2. If your account is still in the SES sandbox, also verify `CONTACT_TO_ADDRESS` so SES will deliver to it.
3. Create an IAM user with a policy that grants only `ses:SendEmail` on the verified identity (least privilege). Use its access key + secret as `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`.
4. Set `AWS_SES_REGION` to the region where the identity is verified (e.g. `us-east-1`).
