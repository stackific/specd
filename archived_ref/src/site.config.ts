/**
 * Site-wide configuration.
 *
 * Single source of truth for SEO defaults, social-media metadata, brand
 * identity, and any other property that should be consistent across every
 * page of the site. Imported by `src/layouts/Layout.astro` and anywhere
 * else that needs canonical site values.
 *
 * Per-page overrides are accepted via the Layout's props — anything not
 * supplied falls back to the values defined here.
 */

export interface SiteConfig {
  /** Production hostname (no trailing slash). */
  url: string;
  /** Brand name as it should appear in titles, og:site_name, JSON-LD. */
  name: string;
  /** Short tagline used as default description and twitter:description. */
  tagline: string;
  /** Default page description (longer than tagline). */
  description: string;
  /** Default page title used when a page does not provide one. */
  defaultTitle: string;
  /** Format for the document <title> when a per-page title is given. `%s` is replaced with the page title. */
  titleTemplate: string;
  /** Site language tag (BCP 47). */
  locale: string;
  /** Two-letter locale for og:locale (e.g. "en_US"). */
  ogLocale: string;
  /** Default Open Graph image, served from /public. Must be at least 1200×630. */
  ogImage: string;
  /** Width of the OG image in px. */
  ogImageWidth: number;
  /** Height of the OG image in px. */
  ogImageHeight: number;
  /** Alt text for the OG image. */
  ogImageAlt: string;
  /** Twitter handle (with @). Empty string if none. */
  twitterHandle: string;
  /** Brand color (used by theme-color meta and hex JSON-LD). */
  themeColor: string;
  /** Author / organization name for JSON-LD and meta author. */
  author: string;
  /** Whether the site should be indexed by search engines. */
  indexable: boolean;
  /** Logo paths (absolute, served from /public). */
  logo: {
    light: string;
    dark: string;
    /** Square / icon variant used in JSON-LD logo. */
    square: string;
  };
  /** Social profiles for JSON-LD sameAs and footer links. */
  social: {
    github?: string;
    linkedin?: string;
    twitter?: string;
    facebook?: string;
    instagram?: string;
    tiktok?: string;
    youtube?: string;
  };
}

export const siteConfig: SiteConfig = {
  url: "https://stackific.com",
  name: "Stackific",
  tagline: "Purposeful AI. From products to implementation to education.",
  description:
    "Stackific is a software engineering practice focused on making AI genuinely useful — building products, advising teams, and teaching practitioners.",
  defaultTitle: "Stackific — Purposeful AI",
  titleTemplate: "%s — Stackific",
  locale: "en",
  ogLocale: "en_US",
  ogImage: "/og-image.png",
  ogImageWidth: 1200,
  ogImageHeight: 630,
  ogImageAlt: "Stackific — Purposeful AI",
  twitterHandle: "",
  themeColor: "#1447E6",
  author: "Stackific Inc.",
  indexable: true,
  logo: {
    light: "/logo.svg",
    dark: "/logo-dark.svg",
    square: "/logo.svg",
  },
  social: {
    github: "https://github.com",
    facebook: "https://facebook.com",
    instagram: "https://instagram.com",
    tiktok: "https://tiktok.com",
  },
};

/**
 * Build the document title for a given page title.
 * Falls back to `defaultTitle` if no page title is provided.
 */
export function buildTitle(pageTitle?: string): string {
  if (!pageTitle) return siteConfig.defaultTitle;
  if (pageTitle === siteConfig.name) return siteConfig.defaultTitle;
  return siteConfig.titleTemplate.replace("%s", pageTitle);
}

/**
 * JSON-LD Organization schema. Embed via <script type="application/ld+json">.
 */
export function organizationJsonLd() {
  const sameAs = Object.values(siteConfig.social).filter(Boolean);
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: siteConfig.name,
    url: siteConfig.url,
    logo: new URL(siteConfig.logo.square, siteConfig.url).href,
    description: siteConfig.description,
    ...(sameAs.length ? { sameAs } : {}),
  };
}

/**
 * JSON-LD WebSite schema with SearchAction (sitelinks search box).
 */
export function websiteJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: siteConfig.name,
    url: siteConfig.url,
    inLanguage: siteConfig.locale,
  };
}
