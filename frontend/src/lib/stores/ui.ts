import { atom } from "nanostores"

const STORAGE_PREFIX = "specd-"

type Theme = "light" | "dark" | "system"

function readLocalStorage(key: string, fallback: string): string {
  if (typeof window === "undefined") return fallback
  return window.localStorage.getItem(STORAGE_PREFIX + key) ?? fallback
}

function writeLocalStorage(key: string, value: string) {
  if (typeof window === "undefined") return
  window.localStorage.setItem(STORAGE_PREFIX + key, value)
}

export const $theme = atom<Theme>(readLocalStorage("theme", "system") as Theme)
$theme.subscribe((value) => writeLocalStorage("theme", value))

export const $sidebarOpen = atom<boolean>(false)
