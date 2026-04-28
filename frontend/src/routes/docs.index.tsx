import { Link, createFileRoute } from "@tanstack/react-router"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

export const Route = createFileRoute("/docs/")({
  component: Docs,
})

function Docs() {
  return (
    <div className="container mx-auto max-w-5xl space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Docs</h1>
        <p className="text-muted-foreground">
          Guides for getting the most out of specd.
        </p>
      </header>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Link to="/docs/tutorial" className="block h-full">
          <Card className="h-full transition-colors hover:bg-accent">
            <CardHeader>
              <CardTitle>Tutorial</CardTitle>
              <CardDescription>
                From init to your first spec and task.
              </CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              Walks through the spec-driven workflow.
            </CardContent>
          </Card>
        </Link>
        <Link to="/docs/user-guide" className="block h-full">
          <Card className="h-full transition-colors hover:bg-accent">
            <CardHeader>
              <CardTitle>User guide</CardTitle>
              <CardDescription>Day-to-day workflows and tips.</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              Specs, tasks, knowledge base, and search in depth.
            </CardContent>
          </Card>
        </Link>
        <Link to="/docs/reference" className="block h-full">
          <Card className="h-full transition-colors hover:bg-accent">
            <CardHeader>
              <CardTitle>Reference</CardTitle>
              <CardDescription>CLI, config, and JSON API.</CardDescription>
            </CardHeader>
            <CardContent className="text-sm text-muted-foreground">
              Commands, project config keys, and `/api/*` endpoints.
            </CardContent>
          </Card>
        </Link>
      </div>
    </div>
  )
}
