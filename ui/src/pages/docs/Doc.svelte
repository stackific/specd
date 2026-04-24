<script>
import { navigate } from "../../lib/router.svelte.js";

function handleNav(event, path) {
  event.preventDefault();
  navigate(path);
}
</script>

<section class="container">
  <h3>Documentation</h3>
  <div class="space"></div>
  <nav aria-label="Documentation pages">
    <a href="/docs/tutorial" class="link" onclick={(e) => handleNav(e, "/docs/tutorial")}>
      <i aria-hidden="true">school</i>
      <span>Tutorial</span>
    </a>
  </nav>
</section>
