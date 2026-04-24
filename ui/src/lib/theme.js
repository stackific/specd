const STORAGE_KEY = "specd-theme";
const SEED_COLOR = "#1c4bea";

/** Generate the material color palette and restore the saved mode. */
export async function initTheme() {
  await ui("theme", SEED_COLOR);

  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "light" || saved === "dark") {
    ui("mode", saved);
  } else {
    const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;
    ui("mode", prefersDark ? "dark" : "light");
  }
}

/** Toggle between light and dark mode, persisting the choice. */
export function toggleTheme() {
  const isDark = document.body.classList.contains("dark");
  const next = isDark ? "light" : "dark";
  ui("mode", next);
  localStorage.setItem(STORAGE_KEY, next);
}

/** Return the current mode. */
export function getTheme() {
  return document.body.classList.contains("dark") ? "dark" : "light";
}
