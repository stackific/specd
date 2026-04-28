import { useLocation, useNavigate } from "@tanstack/react-router"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { RouteContextPane } from "@/components/shell/route-context-pane"
import { NAV_ITEMS, UTILITY_NAV_ITEMS, activeNavItem } from "@/lib/nav"

export function AppSidebar() {
  const navigate = useNavigate()
  const location = useLocation()
  const active = activeNavItem(location.pathname)

  return (
    <Sidebar
      collapsible="icon"
      className="overflow-hidden *:data-[sidebar=sidebar]:flex-row"
    >
      <Sidebar
        collapsible="none"
        className="w-[calc(var(--sidebar-width-icon)+1px)]! border-r"
      >
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton
                size="lg"
                asChild
                className="md:h-8 md:p-0"
                tooltip={{ children: "specd", hidden: false }}
              >
                <button
                  type="button"
                  onClick={() => navigate({ to: "/welcome" })}
                  aria-label="Go to welcome"
                >
                  <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
                    <svg
                      viewBox="0 0 600 600"
                      fill="currentColor"
                      aria-hidden="true"
                      className="size-5"
                    >
                      <path d="M32 410L572.953 412.922L386.797 189.471L378.392 187.858L490.5 374.5L26.012 404.852L32 410Z" />
                    </svg>
                  </div>
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">specd</span>
                    <span className="truncate text-xs">Spec-driven dev</span>
                  </div>
                </button>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent className="px-1.5 md:px-0">
              <SidebarMenu>
                {NAV_ITEMS.map((item) => {
                  const isActive = active?.url === item.url
                  return (
                    <SidebarMenuItem key={item.url}>
                      <SidebarMenuButton
                        tooltip={{ children: item.title, hidden: false }}
                        isActive={isActive}
                        onClick={() => navigate({ to: item.url })}
                        className="px-2.5 md:px-2"
                        aria-label={item.title}
                        aria-current={isActive ? "page" : undefined}
                      >
                        <item.icon aria-hidden="true" />
                        <span>{item.title}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          <SidebarMenu>
            {UTILITY_NAV_ITEMS.map((item) => {
              const isActive = active?.url === item.url
              return (
                <SidebarMenuItem key={item.url}>
                  <SidebarMenuButton
                    tooltip={{ children: item.title, hidden: false }}
                    isActive={isActive}
                    onClick={() => navigate({ to: item.url })}
                    className="px-2.5 md:px-2"
                    aria-label={item.title}
                    aria-current={isActive ? "page" : undefined}
                  >
                    <item.icon aria-hidden="true" />
                    <span>{item.title}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              )
            })}
          </SidebarMenu>
        </SidebarFooter>
      </Sidebar>

      <Sidebar collapsible="none" className="hidden flex-1 md:flex">
        <RouteContextPane />
      </Sidebar>
    </Sidebar>
  )
}
