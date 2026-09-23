import { isAppPath } from "@/lib/app-path";
import { ShikiCodeBlock } from "@trenova/shared/components/ui/shiki-code-block";
import { cn } from "@trenova/shared/lib/utils";
import { Children, memo, type ComponentProps, type ReactNode } from "react";
import ReactMarkdown, { type Components } from "react-markdown";
import { Link, useInRouterContext } from "react-router";
import remarkGfm from "remark-gfm";

const HIGHLIGHTED_LANGS = new Set(["json", "javascript", "graphql", "plsql"]);

type HighlightedLang = "json" | "javascript" | "graphql" | "plsql";

const LANG_ALIASES: Record<string, HighlightedLang> = {
  js: "javascript",
  jsonc: "json",
  sql: "plsql",
  gql: "graphql",
};

function resolveLang(className: string | undefined): HighlightedLang | null {
  const match = /language-([a-z0-9]+)/iu.exec(className ?? "");
  if (!match) {
    return null;
  }
  const raw = match[1].toLowerCase();
  const lang = LANG_ALIASES[raw] ?? raw;

  return HIGHLIGHTED_LANGS.has(lang) ? (lang as HighlightedLang) : null;
}

/** The text inside a code node: react-markdown hands it as one or more strings. */
function textOf(children: ReactNode): string {
  return Children.toArray(children)
    .map((child) => (typeof child === "string" || typeof child === "number" ? String(child) : ""))
    .join("");
}

function CodeBlock({ className, children }: ComponentProps<"code">) {
  const code = textOf(children).replace(/\n$/u, "");
  const lang = resolveLang(className);

  if (lang) {
    return <ShikiCodeBlock code={code} lang={lang} className="my-2 text-xs" />;
  }

  return (
    <pre className="bg-muted my-2 overflow-x-auto rounded-md p-3 font-mono text-xs leading-relaxed">
      <code>{code}</code>
    </pre>
  );
}

const LINK_CLASS = "text-foreground underline underline-offset-2";

/**
 * A link in a reply. A page of this app opens in place, the way every other
 * link in the app does, so "open [Rate matrices](/billing/…)" keeps the
 * person in their session; anything else opens in a new tab. An unsafe
 * scheme never gets here: react-markdown empties it first.
 */
function MarkdownLink({ href, children }: ComponentProps<"a">) {
  const inRouter = useInRouterContext();
  if (inRouter && isAppPath(href)) {
    return (
      <Link to={href} className={LINK_CLASS}>
        {children}
      </Link>
    );
  }

  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={LINK_CLASS}>
      {children}
    </a>
  );
}

const components: Components = {
  p: ({ children }) => <p className="my-1.5 leading-relaxed first:mt-0 last:mb-0">{children}</p>,
  h1: ({ children }) => (
    <h3 className="mt-3 mb-1.5 text-base font-semibold first:mt-0">{children}</h3>
  ),
  h2: ({ children }) => (
    <h3 className="mt-3 mb-1.5 text-sm font-semibold first:mt-0">{children}</h3>
  ),
  h3: ({ children }) => <h4 className="mt-2 mb-1 text-sm font-semibold first:mt-0">{children}</h4>,
  h4: ({ children }) => <h5 className="mt-2 mb-1 text-sm font-medium first:mt-0">{children}</h5>,
  ul: ({ children }) => <ul className="my-1.5 list-disc space-y-0.5 pl-5">{children}</ul>,
  ol: ({ children }) => <ol className="my-1.5 list-decimal space-y-0.5 pl-5">{children}</ol>,
  li: ({ children }) => <li className="leading-relaxed">{children}</li>,
  strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
  em: ({ children }) => <em>{children}</em>,
  a: ({ href, children }) => <MarkdownLink href={href}>{children}</MarkdownLink>,
  blockquote: ({ children }) => (
    <blockquote className="border-border text-muted-foreground my-2 border-l-2 pl-3">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="border-border my-3" />,
  table: ({ children }) => (
    <div className="my-2 overflow-x-auto rounded-md border">
      <table className="w-full border-collapse text-xs">{children}</table>
    </div>
  ),
  thead: ({ children }) => <thead className="bg-muted/60">{children}</thead>,
  th: ({ children }) => (
    <th className="border-border border-b px-2 py-1.5 text-left font-medium">{children}</th>
  ),
  td: ({ children }) => (
    <td className="border-border border-b px-2 py-1.5 align-top last:border-b-0">{children}</td>
  ),
  code: ({ className, children, ...props }) => {
    const isBlock = typeof className === "string" && className.includes("language-");
    const text = textOf(children);
    if (isBlock || text.includes("\n")) {
      return <CodeBlock className={className}>{children}</CodeBlock>;
    }

    return (
      <code
        className="bg-muted rounded-md px-1 py-0.5 font-mono text-[0.85em]"
        {...(props as ComponentProps<"code">)}
      >
        {children}
      </code>
    );
  },
  pre: ({ children }) => <>{children}</>,
};

/**
 * Renders model prose the way the rest of the application renders text.
 *
 * The mapping is explicit rather than a typography plugin so the markdown a
 * model produces lands in the same type scale and colours as the surrounding
 * page. Raw HTML is never rendered: react-markdown drops it by default, and a
 * model reading customer records must not be able to inject markup.
 */
export const AiMarkdown = memo(function AiMarkdown({
  content,
  className,
}: {
  content: string;
  className?: string;
}) {
  return (
    <div className={cn("text-sm wrap-break-word", className)}>
      <ReactMarkdown remarkPlugins={[remarkGfm]} components={components}>
        {content}
      </ReactMarkdown>
    </div>
  );
});
