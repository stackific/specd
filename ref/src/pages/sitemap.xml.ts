import type { APIRoute } from "astro";

/**
 * Static sitemap generator. Auto-discovers every .astro page in
 * src/pages and emits a standard sitemap.xml. Uses Astro's site
 * URL from astro.config.mjs.
 */
export const GET: APIRoute = ({ site }) => {
  if (!site) {
    return new Response("Missing `site` config in astro.config.mjs", { status: 500 });
  }

  // Discover all page files at build time.
  const modules = import.meta.glob("./**/*.astro", { eager: true });

  const urls = Object.keys(modules)
    .map((path) => {
      // ./index.astro       → /
      // ./about.astro       → /about
      // ./blog/[slug].astro → skipped (dynamic)
      const route = path
        .replace(/^\.\//, "")
        .replace(/\.astro$/, "")
        .replace(/\/index$/, "")
        .replace(/^index$/, "");
      return route;
    })
    // Skip dynamic routes (anything with brackets)
    .filter((route) => !route.includes("["))
    .filter((route) => route !== "sitemap.xml")
    .filter((route) => route !== "widgets")
    .map((route) => new URL(route, site).href)
    .sort();

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${urls.map((url) => `  <url><loc>${url}</loc></url>`).join("\n")}
</urlset>
`;

  return new Response(xml, {
    headers: { "Content-Type": "application/xml; charset=utf-8" },
  });
};
