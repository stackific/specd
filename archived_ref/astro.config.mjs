// @ts-check
import { defineConfig } from "astro/config";
import cloudflare from "@astrojs/cloudflare";
import { fileURLToPath } from "node:url";
import { readFile, writeFile, readdir } from "node:fs/promises";
import path from "node:path";

// Astro integration: purge unused selectors from OUR bundled CSS only.
// BeerCSS ships from public/vendor/beer.min.css as its own <link> and is
// NOT touched — its runtime-added classes (active, dark, field states, etc.)
// would otherwise be stripped. We only purge dist/_astro/style.<hash>.css.
function purgeOwnCss() {
  return {
    name: "purge-own-css",
    hooks: {
      "astro:build:done": async ({ dir }) => {
        const { PurgeCSS } = await import("purgecss");
        const distDir = fileURLToPath(dir);
        const astroDir = path.join(distDir, "_astro");
        let entries;
        try {
          entries = await readdir(astroDir);
        } catch {
          return;
        }
        const cssFiles = entries
          .filter((f) => f.startsWith("style.") && f.endsWith(".css"))
          .map((f) => path.join(astroDir, f));
        if (cssFiles.length === 0) return;

        // Walk dist/ for every emitted HTML page; PurgeCSS will keep any
        // selector referenced by any of them.
        async function walk(d) {
          const out = [];
          for (const e of await readdir(d, { withFileTypes: true })) {
            const p = path.join(d, e.name);
            if (e.isDirectory()) out.push(...(await walk(p)));
            else if (e.name.endsWith(".html")) out.push(p);
          }
          return out;
        }
        const htmlFiles = await walk(distDir);

        const before = (await Promise.all(cssFiles.map((f) => readFile(f, "utf8")))).reduce(
          (n, s) => n + s.length,
          0,
        );

        const results = await new PurgeCSS().purge({
          content: htmlFiles,
          css: cssFiles,
          // Custom extractor: allow colons in class names so our
          // responsive variants (`m:p-3`, `l:text-4`) are detected.
          // Also allows the dot-prefixed and slash characters used by
          // a few Tailwind-style patterns we may add later.
          defaultExtractor: (content) => content.match(/[\w-/:]+(?<!:)/g) || [],
          // Keep BeerCSS-style runtime classes that JS toggles at runtime,
          // and Material Symbols glyph names which appear as element text.
          safelist: {
            standard: [/^is-/, /^active$/, /^dark$/, /^ms$/],
            deep: [/active$/, /dark$/],
            greedy: [/^field/, /^chip/, /^dialog/, /^nav/, /^snackbar/],
          },
        });

        let after = 0;
        for (const r of results) {
          await writeFile(r.file, r.css);
          after += r.css.length;
        }
        const saved = before - after;
        // eslint-disable-next-line no-console
        console.log(
          `[purge-own-css] ${cssFiles.length} file(s): ${before} → ${after} bytes (-${saved}, ${((saved / before) * 100).toFixed(1)}%)`,
        );
      },
    },
  };
}

// https://astro.build/config
export default defineConfig({
  site: "https://stackific.com",
  output: "static",
  adapter: cloudflare({ imageService: "compile" }),
  integrations: [purgeOwnCss()],
  build: {
    // Force external <link rel="stylesheet"> in head — no inlined <style> tags.
    inlineStylesheets: "never",
  },
  vite: {
    css: {
      // Use LightningCSS as the CSS transformer (not just the minifier).
      // This enables native @custom-media support — see src/styles/media.css.
      transformer: "lightningcss",
      lightningcss: {
        drafts: {
          customMedia: true,
        },
      },
    },
    build: {
      // Combine all our custom CSS into ONE bundled file.
      // BeerCSS is served separately from public/vendor/ via a direct <link> in Layout.
      cssCodeSplit: false,
      // Minify with esbuild instead of lightningcss: when the Cloudflare
      // adapter triggers an SSR build, Vite passes its SSR build target
      // ("es2024") to LightningCSS minify, which throws "Unsupported target
      // es2024". esbuild minify avoids that path. LightningCSS is still the
      // CSS *transformer* (see vite.css.transformer above) so @custom-media
      // and other modern features still work.
      cssMinify: "esbuild",
    },
  },
});
