import { useEffect } from "react"
import { Link, createFileRoute } from "@tanstack/react-router"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import rehypeSanitize from "rehype-sanitize"
import tutorialMd from "@/content/tutorial.md?raw"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"

export const Route = createFileRoute("/docs/tutorial")({
  component: TutorialPage,
})

function TutorialPage() {
  useEffect(() => {
    document.title = "Tutorial — specd"
  }, [])

  return (
    <div className="container mx-auto max-w-3xl space-y-6 p-6">
      <Breadcrumb>
        <BreadcrumbList>
          <BreadcrumbItem>
            <BreadcrumbLink asChild>
              <Link to="/docs">Docs</Link>
            </BreadcrumbLink>
          </BreadcrumbItem>
          <BreadcrumbSeparator />
          <BreadcrumbItem>
            <BreadcrumbPage>Tutorial</BreadcrumbPage>
          </BreadcrumbItem>
        </BreadcrumbList>
      </Breadcrumb>

      <header className="space-y-2">
        <h1 className="text-3xl font-semibold tracking-tight">Tutorial</h1>
        <p className="text-muted-foreground">
          From init to your first spec, task, and the web UI.
        </p>
      </header>

      <article className="prose dark:prose-invert max-w-none">
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          rehypePlugins={[rehypeSanitize]}
          components={{
            h2: ({ children, ...props }) => (
              <h2
                className="mt-8 scroll-mt-20 border-b pb-2 text-2xl font-semibold tracking-tight first:mt-0"
                {...props}
              >
                {children}
              </h2>
            ),
            h3: ({ children, ...props }) => (
              <h3
                className="mt-6 scroll-mt-20 text-xl font-semibold tracking-tight"
                {...props}
              >
                {children}
              </h3>
            ),
            p: ({ children, ...props }) => (
              <p className="my-4 leading-7" {...props}>
                {children}
              </p>
            ),
            ul: ({ children, ...props }) => (
              <ul className="my-4 list-disc space-y-1 pl-6" {...props}>
                {children}
              </ul>
            ),
            ol: ({ children, ...props }) => (
              <ol className="my-4 list-decimal space-y-1 pl-6" {...props}>
                {children}
              </ol>
            ),
            li: ({ children, ...props }) => (
              <li className="leading-7" {...props}>
                {children}
              </li>
            ),
            a: ({ children, href, ...props }) => (
              <a
                href={href}
                className="font-medium text-primary underline underline-offset-4 hover:no-underline"
                {...props}
              >
                {children}
              </a>
            ),
            blockquote: ({ children, ...props }) => (
              <blockquote
                className="my-4 border-l-4 border-border bg-muted/40 px-4 py-2 text-muted-foreground italic"
                {...props}
              >
                {children}
              </blockquote>
            ),
            code: ({ className, children, ...props }) => {
              const isBlock =
                typeof className === "string" &&
                className.startsWith("language-")
              if (isBlock) {
                return (
                  <code className={`${className} font-mono text-sm`} {...props}>
                    {children}
                  </code>
                )
              }
              return (
                <code
                  className="rounded border bg-muted px-1.5 py-0.5 font-mono text-[0.85em]"
                  {...props}
                >
                  {children}
                </code>
              )
            },
            pre: ({ children, ...props }) => (
              <pre
                className="my-4 overflow-x-auto rounded-md border bg-muted p-4 font-mono text-sm leading-6"
                {...props}
              >
                {children}
              </pre>
            ),
            hr: (props) => <hr className="my-8 border-border" {...props} />,
            table: ({ children, ...props }) => (
              <div className="my-4 w-full overflow-x-auto">
                <table className="w-full border-collapse text-sm" {...props}>
                  {children}
                </table>
              </div>
            ),
            th: ({ children, ...props }) => (
              <th
                className="border-b px-3 py-2 text-left font-semibold"
                {...props}
              >
                {children}
              </th>
            ),
            td: ({ children, ...props }) => (
              <td className="border-b px-3 py-2 align-top" {...props}>
                {children}
              </td>
            ),
          }}
        >
          {tutorialMd}
        </ReactMarkdown>
      </article>
    </div>
  )
}
