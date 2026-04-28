import { createFileRoute } from "@tanstack/react-router"
import { PagePlaceholder } from "@/components/common/page-placeholder"

export const Route = createFileRoute("/docs/user-guide")({
  component: () => (
    <PagePlaceholder
      title="User guide"
      description="Day-to-day workflows and tips."
    />
  ),
})
