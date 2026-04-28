import { Outlet, createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/kb")({
  component: KBLayout,
})

function KBLayout() {
  return <Outlet />
}
