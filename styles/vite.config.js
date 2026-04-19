import { defineConfig } from "vite";
import { readFile, writeFile, readdir } from "node:fs/promises";
import path from "node:path";

// Vite plugin: purge unused selectors from our bundled CSS only.
// BeerCSS ships from public/vendor/beer.min.css as its own <link> and is
// NOT touched — its runtime-added classes (active, dark, field states, etc.)
// would otherwise be stripped. We only purge dist/assets/style.<hash>.css.
// Content is scanned from Go templates in ../templates/.
function purgeOwnCss() {
  return {
    name: "purge-own-css",
    apply: "build",
    async closeBundle() {
      const { PurgeCSS } = await import("purgecss");
      const assetsDir = path.resolve(__dirname, "dist", "assets");
      let entries;
      try {
        entries = await readdir(assetsDir);
      } catch {
        return;
      }
      const cssFiles = entries
        .filter((f) => f.endsWith(".css"))
        .map((f) => path.join(assetsDir, f));
      if (cssFiles.length === 0) return;

      // Walk Go templates for class usage.
      const templateDir = path.resolve(__dirname, "..", "templates");
      async function walk(d) {
        const out = [];
        let items;
        try {
          items = await readdir(d, { withFileTypes: true });
        } catch {
          return out;
        }
        for (const e of items) {
          const p = path.join(d, e.name);
          if (e.isDirectory()) out.push(...(await walk(p)));
          else if (e.name.endsWith(".html")) out.push(p);
        }
        return out;
      }
      const templateFiles = await walk(templateDir);

      const before = (
        await Promise.all(cssFiles.map((f) => readFile(f, "utf8")))
      ).reduce((n, s) => n + s.length, 0);

      const results = await new PurgeCSS().purge({
        content: templateFiles,
        css: cssFiles,
        // Custom extractor preserves colon-prefixed responsive classes (m:p-3, l:text-4).
        defaultExtractor: (content) => content.match(/[\w-/:]+(?<!:)/g) || [],
        safelist: {
          standard: [/^is-/, /^active$/, /^dark$/, /^ms$/, /^sd-drop-/],
          deep: [/active$/, /dark$/, /kr-/],
          greedy: [/^field/, /^chip/, /^dialog/, /^nav/, /^snackbar/],
        },
      });

      let after = 0;
      for (const r of results) {
        await writeFile(r.file, r.css);
        after += r.css.length;
      }
      const saved = before - after;
      console.log(
        `[purge-own-css] ${cssFiles.length} file(s): ${before} → ${after} bytes (-${saved}, ${((saved / before) * 100).toFixed(1)}%)`
      );
    },
  };
}

export default defineConfig({
  css: {
    transformer: "lightningcss",
    lightningcss: {
      drafts: {
        customMedia: true,
      },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    cssCodeSplit: false,
    cssMinify: "esbuild",
    rollupOptions: {
      input: "src/entry.js",
      output: {
        // CSS bundle lands in dist/assets/style.<hash>.css.
        // The JS shim is discarded by the Go embed — only CSS matters.
        assetFileNames: "assets/[name].[hash][extname]",
        entryFileNames: "assets/[name].[hash].js",
      },
    },
  },
  publicDir: "public",
  plugins: [purgeOwnCss()],
});
