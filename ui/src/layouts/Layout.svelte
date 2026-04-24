<script>
import { navigate, router } from "../lib/router.svelte.js";
import { getTheme, toggleTheme } from "../lib/theme.js";

let theme = $state(getTheme());

function handleToggleTheme() {
  toggleTheme();
  theme = getTheme();
}

function handleNav(event, path) {
  event.preventDefault();
  navigate(path);
}

let { children } = $props();
</script>

<!-- Desktop sidebar: visible on large screens -->
<nav class="left l surface-container" aria-label="Main navigation">
  <header>
    <button class="circle extra transparent" aria-label="Toggle sidebar" aria-expanded="false">
      <i aria-hidden="true">menu</i>
    </button>
  </header>
  <div class="max"></div>
  <a
    href="/docs"
    class:active={router.path.startsWith("/docs")}
    onclick={(e) => handleNav(e, "/docs")}
  >
    <i aria-hidden="true">article</i>
    <span class="fw-5">Docs</span>
  </a>
  <button class="my-2 border circle" onclick={handleToggleTheme} aria-label="Toggle dark mode">
    <i aria-hidden="true">{theme === "dark" ? "light_mode" : "dark_mode"}</i>
  </button>
</nav>

<!-- Mobile/tablet top bar -->
<nav class="top s m left-align" aria-label="Mobile navigation">
  <button class="circle transparent" data-ui="#mobile-menu" aria-label="Open menu">
    <i aria-hidden="true">menu</i>
  </button>
  <div class="max"></div>
  <button class="circle border" onclick={handleToggleTheme} aria-label="Toggle dark mode">
    <i aria-hidden="true">{theme === "dark" ? "light_mode" : "dark_mode"}</i>
  </button>
</nav>

<!-- Mobile menu dialog -->
<dialog id="mobile-menu" class="p-5" aria-label="Mobile menu" style="width:min(360px, 92vw); max-width:none;">
  <nav class="vertical fw-5 text-2">
    <a
      href="/docs"
      class:active={router.path.startsWith("/docs")}
      onclick={(e) => { handleNav(e, "/docs"); ui("#mobile-menu"); }}
    >
      <i aria-hidden="true">school</i>
      <span class="ml-3">Tutorial</span>
    </a>
  </nav>
</dialog>

<main class="responsive" id="main-content">
  {@render children()}
</main>

<footer class="container padding" aria-labelledby="footer-heading">
  <h2 id="footer-heading" class="sr-only">Site footer</h2>
  <hr />
  <div class="grid top-align medium-margin">
    <div class="s12 l6">
      <a href="/" aria-label="specd home">
        <img class="logo-light" alt="" width="60" src="/logo.svg" />
        <img class="logo-dark" alt="" width="60" src="/logo-dark.svg" />
      </a>
      <p class="small-margin">
        Stackific Inc. is a software engineering practice focused on making AI genuinely useful.
        Stackific builds own products, works with startups and enterprises for shaping the AI
        architectures and implementation and AI engineering efficiency. Stackific also teaches real
        insight from work through <a href="https://stackdemy.com" target="_blank" rel="noopener noreferrer">Stackdemy</a>,
        their technical education platform.
      </p>
    </div>
    <nav class="s12 l6" aria-label="Footer">
      <div class="grid">
        <div class="s4">
          <h3 class="large-text bold bottom-margin">Social</h3>
          <ul class="no-padding" role="list">
            <li><a class="link" href="https://github.com/stackific/specd" target="_blank" rel="noopener noreferrer">GitHub</a></li>
            <li><a class="link" href="https://facebook.com/stackific" target="_blank" rel="noopener noreferrer">Facebook</a></li>
            <li><a class="link" href="https://instagram.com/stackific" target="_blank" rel="noopener noreferrer">Instagram</a></li>
            <li><a class="link" href="https://tiktok.com/@stackific" target="_blank" rel="noopener noreferrer">TikTok</a></li>
          </ul>
        </div>
        <div class="s4">
          <h3 class="large-text bold bottom-margin">Products</h3>
          <ul class="no-padding" role="list">
            <li><a class="link" href="https://stackdemy.com" target="_blank" rel="noopener noreferrer">Stackdemy</a></li>
          </ul>
        </div>
        <div class="s4">
          <h3 class="large-text bold bottom-margin">Legal</h3>
          <ul class="no-padding" role="list">
            <li><a class="link" href="https://stackific.com/privacy">Privacy Policy</a></li>
            <li><a class="link" href="https://stackific.com/terms">Terms of Service</a></li>
          </ul>
        </div>
      </div>
    </nav>
  </div>
</footer>
