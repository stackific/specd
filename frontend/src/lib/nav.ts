import {
  BookOpenIcon,
  FileTextIcon,
  GraduationCapIcon,
  ListChecksIcon,
  SearchIcon,
  SettingsIcon,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"

export type NavItem = {
  title: string
  url: string
  icon: LucideIcon
  /** Routes whose pathname starts with one of these prefixes count as active. */
  matchPrefixes: Array<string>
}

/** Primary nav items shown in the sidebar rail. */
export const NAV_ITEMS: Array<NavItem> = [
  {
    title: "Tasks",
    url: "/tasks",
    icon: ListChecksIcon,
    matchPrefixes: ["/tasks"],
  },
  {
    title: "Specs",
    url: "/specs",
    icon: FileTextIcon,
    matchPrefixes: ["/specs"],
  },
  {
    title: "Knowledge",
    url: "/kb",
    icon: BookOpenIcon,
    matchPrefixes: ["/kb"],
  },
  {
    title: "Search",
    url: "/search",
    icon: SearchIcon,
    matchPrefixes: ["/search"],
  },
]

/** Utility nav items shown top-right in the page header. */
export const UTILITY_NAV_ITEMS: Array<NavItem> = [
  {
    title: "Docs",
    url: "/docs",
    icon: GraduationCapIcon,
    matchPrefixes: ["/docs"],
  },
  {
    title: "Settings",
    url: "/settings",
    icon: SettingsIcon,
    matchPrefixes: ["/settings"],
  },
]

const ALL_NAV_ITEMS = [...NAV_ITEMS, ...UTILITY_NAV_ITEMS]

export function activeNavItem(pathname: string): NavItem | undefined {
  return ALL_NAV_ITEMS.find((item) =>
    item.matchPrefixes.some(
      (p) => pathname === p || pathname.startsWith(p + "/")
    )
  )
}
