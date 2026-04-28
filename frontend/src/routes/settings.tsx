import { useEffect, useState } from "react"
import { createFileRoute } from "@tanstack/react-router"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { useStore } from "@nanostores/react"
import { z } from "zod"
import { toast } from "sonner"
import type { StartpageChoice } from "@/lib/api/meta"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ApiError } from "@/lib/api"
import { getMeta } from "@/lib/api/meta"
import { setDefaultRoute } from "@/lib/api/settings"
import { $theme } from "@/lib/stores/ui"

export const Route = createFileRoute("/settings")({
  component: SettingsPage,
})

function SettingsPage() {
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [choices, setChoices] = useState<Array<StartpageChoice>>([])
  const [defaultValue, setDefaultValue] = useState<string>("")
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    document.title = "Settings — specd"
  }, [])

  useEffect(() => {
    const ctrl = new AbortController()
    setLoading(true)
    setLoadError(null)
    getMeta(ctrl.signal)
      .then((meta) => {
        setChoices(meta.startpage_choices)
        setDefaultValue(meta.default_route)
      })
      .catch((err: unknown) => {
        if (err instanceof DOMException && err.name === "AbortError") return
        const msg =
          err instanceof Error ? err.message : "Failed to load settings"
        setLoadError(msg)
      })
      .finally(() => {
        if (!ctrl.signal.aborted) setLoading(false)
      })
    return () => ctrl.abort()
  }, [reloadKey])

  return (
    <div className="container mx-auto max-w-3xl space-y-6 p-6">
      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Settings</h1>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>Appearance</CardTitle>
          <CardDescription>
            Choose a theme. System follows your OS preference.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ThemeToggle />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Startpage</CardTitle>
          <CardDescription>
            Where the home page redirects on load.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-3">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-9 w-full max-w-sm" />
              <Skeleton className="h-9 w-24" />
            </div>
          ) : loadError ? (
            <div className="space-y-3" role="alert">
              <p className="text-sm text-destructive">{loadError}</p>
              <Button
                type="button"
                variant="outline"
                onClick={() => setReloadKey((k) => k + 1)}
              >
                Retry
              </Button>
            </div>
          ) : (
            <DefaultRouteForm choices={choices} defaultValue={defaultValue} />
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function ThemeToggle() {
  const theme = useStore($theme)
  return (
    <Tabs
      value={theme}
      onValueChange={(v) => $theme.set(v as "light" | "dark" | "system")}
    >
      <TabsList aria-label="Theme">
        <TabsTrigger value="system">System</TabsTrigger>
        <TabsTrigger value="light">Light</TabsTrigger>
        <TabsTrigger value="dark">Dark</TabsTrigger>
      </TabsList>
    </Tabs>
  )
}

function DefaultRouteForm({
  choices,
  defaultValue,
}: {
  choices: Array<StartpageChoice>
  defaultValue: string
}) {
  const allowed = choices.map((c) => c.route) as [string, ...Array<string>]
  const schema = z.object({
    default_route: z
      .string()
      .min(1, "Pick a route")
      .refine((v) => allowed.includes(v), { message: "Unknown route" }),
  })
  type FormValues = z.infer<typeof schema>

  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { default_route: defaultValue },
  })

  async function onSubmit(values: FormValues) {
    try {
      await setDefaultRoute(values.default_route)
      toast.success("Saved")
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? err.message
          : err instanceof Error
            ? err.message
            : "Failed to save default route"
      toast.error(msg)
    }
  }

  const saving = form.formState.isSubmitting

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        aria-busy={saving}
        className="flex w-full flex-col gap-4"
      >
        <FormField
          control={form.control}
          name="default_route"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Route</FormLabel>
              <Select
                value={field.value}
                onValueChange={field.onChange}
                disabled={saving}
              >
                <FormControl>
                  <SelectTrigger
                    className="w-full sm:w-72"
                    aria-label="Default route"
                  >
                    <SelectValue placeholder="Select a route" />
                  </SelectTrigger>
                </FormControl>
                <SelectContent>
                  {choices.map((c) => (
                    <SelectItem key={c.route} value={c.route}>
                      {c.title}{" "}
                      <span className="text-muted-foreground">({c.route})</span>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <FormMessage />
            </FormItem>
          )}
        />
        <div>
          <Button type="submit" disabled={saving || !form.formState.isValid}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </Form>
  )
}
