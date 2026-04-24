/** Minimal History API router using Svelte 5 runes. */

export const router = $state({ path: window.location.pathname });

// Redirect "/" to the default route from the API.
if (router.path === "/") {
  fetch("/api/meta/default-route")
    .then((r) => r.json())
    .then((data) => {
      if (data.default_route) {
        navigate(data.default_route);
      }
    })
    .catch(() => {});
}

/** Navigate to a new path via the History API. */
export function navigate(path) {
  if (path === router.path) return;
  window.history.pushState(null, "", path);
  router.path = path;
}

// Handle browser back/forward navigation.
window.addEventListener("popstate", () => {
  router.path = window.location.pathname;
});
