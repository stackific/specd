export interface SiteConfig {
  url: string;
  name: string;
  tagline: string;
  description: string;
  defaultTitle: string;
  titleTemplate: string;
  locale: string;
  ogLocale: string;
  ogImage: string;
  ogImageWidth: number;
  ogImageHeight: number;
  ogImageAlt: string;
  twitterHandle: string;
  themeColor: string;
  author: string;
  indexable: boolean;
  logo: {
    light: string;
    dark: string;
    square: string;
  };
  social: Record<string, string>;
}

export const siteConfig: SiteConfig = {
  url: "https://stackific.com/specd",
  name: "specd",
  tagline: "Persistent memory for AI coding agents— from Stackific",
  description: "Give your AI coding agent a memory that survives the session. specd stores specs, tasks, and reference docs in markdown with powerful search and citations.",
  defaultTitle: "specd",
  titleTemplate: "%s — specd",
  locale: "en",
  ogLocale: "en_US",
  ogImage: "/og-image.png",
  ogImageWidth: 1200,
  ogImageHeight: 630,
  ogImageAlt: "specd",
  twitterHandle: "",
  themeColor: "#1447E6",
  author: "specd",
  indexable: true,
  logo: {
    light: "/logo.svg",
    dark: "/logo-dark.svg",
    square: "/logo.svg",
  },
  social: {
    github: "https://github.com/stackific/specd",
  },
};

export function buildTitle(pageTitle?: string): string {
  if (!pageTitle) return siteConfig.defaultTitle;
  if (pageTitle === siteConfig.name) return siteConfig.defaultTitle;
  return siteConfig.titleTemplate.replace("%s", pageTitle);
}

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

export function websiteJsonLd() {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: siteConfig.name,
    url: siteConfig.url,
    inLanguage: siteConfig.locale,
  };
}
