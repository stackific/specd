import { Outlet, createRootRoute } from "@tanstack/react-router"
import { AppShell } from "@/components/shell/app-shell"

export const Route = createRootRoute({
  notFoundComponent: () => (
    <main className="container mx-auto p-6">
      <h1 className="text-3xl font-semibold tracking-tight">404</h1>
      <p className="text-muted-foreground">
        The requested page could not be found.
      </p>
    </main>
  ),
  component: RootComponent,
})

function RootComponent() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  )
}
