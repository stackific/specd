import { Fragment } from "react"
import { Link, useLocation, useParams } from "@tanstack/react-router"
import { Binoculars } from "lucide-react"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { useSidebar } from "@/components/ui/sidebar"
import { activeNavItem } from "@/lib/nav"

type Crumb = { label: string; to?: string }

function buildCrumbs(
  pathname: string,
  params: Record<string, string | undefined>
): Array<Crumb> {
  const crumbs: Array<Crumb> = [{ label: "specd", to: "/welcome" }]
  const section = activeNavItem(pathname)

  if (section) {
    crumbs.push({ label: section.title, to: section.url })
  } else if (pathname === "/welcome") {
    crumbs.push({ label: "Welcome" })
    return crumbs
  } else if (pathname === "/search") {
    crumbs.push({ label: "Search" })
    return crumbs
  }

  if (pathname.startsWith("/docs/")) {
    if (pathname === "/docs/tutorial") crumbs.push({ label: "Tutorial" })
  } else if (params.id) {
    crumbs.push({ label: params.id })
  }

  return crumbs
}

export function PageHeader() {
  const location = useLocation()
  const params = useParams({ strict: false })
  const crumbs = buildCrumbs(location.pathname, params)
  const { toggleSidebar, state } = useSidebar()

  return (
    <header className="sticky top-0 z-10 flex h-14 shrink-0 items-center gap-2 border-b bg-background/80 px-3 backdrop-blur sm:px-4">
      <Button
        variant="ghost"
        size="icon"
        className="-ml-1"
        onClick={toggleSidebar}
        aria-label={
          state === "expanded" ? "Hide quick search" : "Show quick search"
        }
        aria-expanded={state === "expanded"}
      >
        <Binoculars aria-hidden="true" />
      </Button>
      <Separator
        orientation="vertical"
        className="mr-2 self-center! data-[orientation=vertical]:h-4"
      />
      <Breadcrumb className="min-w-0 flex-1">
        <BreadcrumbList className="flex-nowrap">
          {crumbs.map((c, i) => {
            const isLast = i === crumbs.length - 1
            return (
              <Fragment key={`${c.label}-${i}`}>
                <BreadcrumbItem className="min-w-0">
                  {isLast || !c.to ? (
                    <BreadcrumbPage className="truncate">
                      {c.label}
                    </BreadcrumbPage>
                  ) : (
                    <BreadcrumbLink asChild className="truncate">
                      <Link to={c.to}>{c.label}</Link>
                    </BreadcrumbLink>
                  )}
                </BreadcrumbItem>
                {!isLast && <BreadcrumbSeparator />}
              </Fragment>
            )
          })}
        </BreadcrumbList>
      </Breadcrumb>
    </header>
  )
}
