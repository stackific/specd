import { useEffect } from "react"
import { useStore } from "@nanostores/react"
import type { ReactNode } from "react"
import { TooltipProvider } from "@/components/ui/tooltip"
import { Toaster } from "@/components/ui/sonner"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { AppSidebar } from "@/components/shell/app-sidebar"
import { PageHeader } from "@/components/shell/page-header"
import { $sidebarOpen, $theme } from "@/lib/stores/ui"

export function AppShell({ children }: { children: ReactNode }) {
  const open = useStore($sidebarOpen)
  const theme = useStore($theme)

  useEffect(() => {
    const root = document.documentElement
    const prefersDark = window.matchMedia(
      "(prefers-color-scheme: dark)"
    ).matches
    const isDark = theme === "dark" || (theme === "system" && prefersDark)
    root.classList.toggle("dark", isDark)
  }, [theme])

  return (
    <TooltipProvider delayDuration={0}>
      <SidebarProvider
        open={open}
        onOpenChange={(value) => $sidebarOpen.set(value)}
        style={{ "--sidebar-width": "22rem" } as React.CSSProperties}
      >
        <AppSidebar />
        <SidebarInset className="min-w-0">
          <PageHeader />
          <div className="min-w-0 flex-1 overflow-x-hidden">{children}</div>
        </SidebarInset>
        <Toaster richColors position="bottom-right" />
      </SidebarProvider>
    </TooltipProvider>
  )
}
