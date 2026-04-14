# Vendored CSS / fonts

This directory holds files served as-is at `/vendor/*`. They are NOT processed
by Vite/LightningCSS — they're shipped to `dist/` verbatim.

## What's here

- `beer.min.css` — BeerCSS framework (Material Design 3)
- `material-symbols-rounded.woff2` — referenced by `beer.min.css` via relative URL
- `material-symbols-outlined.woff2`
- `material-symbols-sharp.woff2`
- `material-symbols-subset.woff2`

## Why vendored

We deliberately keep BeerCSS OUT of our LightningCSS bundle so our single
combined custom CSS stays small and our cache invalidates only when our own
styles change. BeerCSS is loaded as a separate `<link>` from `Layout.astro`.

## Regenerating after a BeerCSS update

```bash
pnpm update beercss
cp node_modules/beercss/dist/cdn/beer.min.css public/vendor/
cp node_modules/beercss/dist/cdn/material-symbols-*.woff2 public/vendor/
```

(pnpm hoists `beercss` to a deeply nested path under `node_modules/.pnpm/...`,
so the actual command is closer to:
`cp node_modules/.pnpm/beercss@*/node_modules/beercss/dist/cdn/beer.min.css public/vendor/`)
