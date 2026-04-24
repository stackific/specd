<script>
import Layout from "./layouts/Layout.svelte";
import { router } from "./lib/router.svelte.js";
import Docs from "./pages/docs/Doc.svelte";
import Tutorial from "./pages/docs/Tutorial.svelte";
</script>

<Layout>
  {#if router.path === "/docs/tutorial"}
    <Tutorial />
  {:else if router.path === "/docs"}
    <Docs />
  {:else}
    <section class="container p-5">
      <h3>Welcome to specd</h3>
      <div class="space"></div>
      <p>Select a page from the navigation.</p>
    </section>
  {/if}
</Layout>
