// Each `components.*` override below destructures `children` from the inner
// element props; ESLint flags this as shadowing the outer `Markdown`
// component's own `children` prop. The shadow is intentional — these are
// per-tag renderers, and matching React's standard prop name keeps the
// code idiomatic. Disable the rule for this file rather than rename one
// side or the other.
/* eslint-disable no-shadow */
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import rehypeSanitize from "rehype-sanitize"

import { cn } from "@/lib/utils"

// Markdown renders sanitized GitHub-flavoured markdown with explicit
// component overrides. The Tailwind `prose` plugin is not loaded in this
// project, so styling has to live on each tag here rather than relying on
// `@tailwindcss/typography`.
export function Markdown({
  children,
  className,
}: {
  children: string
  className?: string
}) {
  return (
    <article className={cn("max-w-none", className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeSanitize]}
        components={{
          h1: ({ children, ...props }) => (
            <h1
              className="mt-8 scroll-mt-20 text-3xl font-semibold tracking-tight first:mt-0"
              {...props}
            >
              {children}
            </h1>
          ),
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
          h4: ({ children, ...props }) => (
            <h4 className="mt-4 text-base font-semibold" {...props}>
              {children}
            </h4>
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
          code: ({ className: cls, children, ...props }) => {
            const isBlock =
              typeof cls === "string" && cls.startsWith("language-")
            if (isBlock) {
              return (
                <code className={`${cls} font-mono text-sm`} {...props}>
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
        {children}
      </ReactMarkdown>
    </article>
  )
}
