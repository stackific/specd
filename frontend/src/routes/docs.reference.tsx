import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "@/components/common/page-placeholder"

export const Route = createFileRoute("/docs/reference")({
  component: () => (
    <PagePlaceholder
      title="Reference"
      description="CLI commands, project config, and JSON API."
    />
  ),
})
