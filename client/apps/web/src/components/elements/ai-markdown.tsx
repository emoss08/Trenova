import { isAppPath } from "@/lib/app-path";
import { artifactRefId } from "@/lib/artifact-ref";
import { splitMarkdownBlocks } from "@/lib/markdown-blocks";
import { ShikiCodeBlock } from "@trenova/shared/components/ui/shiki-code-block";
import { cn } from "@trenova/shared/lib/utils";
import {
  Children,
  createContext,
  Fragment,
  memo,
  use,
  useMemo,
  useState,
  type ComponentProps,
  type ReactNode,
} from "react";
import ReactMarkdown, {
  defaultUrlTransform,
  type Components,
  type Options as MarkdownOptions,
} from "react-markdown";
import { Link, useInRouterContext } from "react-router";
import { useKatexPlugin } from "@/lib/katex-plugin";
import { markdownHasMath, prepareMarkdown } from "@/lib/markdown-prepare";
import remarkBreaks from "remark-breaks";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { rehypeNumericColumns } from "./rehype-numeric-columns";
import { remarkDeskSubset } from "./remark-desk-subset";
import { rehypeStreamWords } from "./rehype-stream-words";

type PluggableList = NonNullable<MarkdownOptions["rehypePlugins"]>;

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

/** The language a fence was labelled with, as written: "sql", "json". */
function fenceLabel(className: string | undefined): string {
  return /language-([\w+#.-]+)/iu.exec(className ?? "")?.[1] ?? "text";
}

/** The fence a still-open math block is shown in, as its raw text, until it closes. */
export const RAW_MATH_FENCE = "dk-math-raw";

function CopyCode({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      className={cn("md-code-copy", copied && "md-ok")}
      onClick={() => {
        void navigator.clipboard?.writeText(code).then(() => {
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1400);
        });
      }}
    >
      {copied && (
        <svg
          width="11"
          height="11"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2.6"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden
        >
          <path d="M5 12.5l4.5 4.5L19 7" />
        </svg>
      )}
      {copied ? "Copied" : "Copy"}
    </button>
  );
}

/**
 * A fenced block: its language as a label, a way to copy it, and the code,
 * highlighted where the language is one the app highlights. A block still
 * arriving shows what has come so far.
 */
function CodeBlock({ className, children }: ComponentProps<"code">) {
  const code = textOf(children).replace(/\n$/u, "");
  const lang = resolveLang(className);
  const label = fenceLabel(className);
  if (label === RAW_MATH_FENCE) {
    return <pre className="md-math-raw">{code}</pre>;
  }

  return (
    <div className="md-code bg-sunken rounded-surface my-2.5 overflow-hidden">
      <div className="md-code-h">
        <span>{label}</span>
        <CopyCode code={code} />
      </div>
      {lang ? (
        <ShikiCodeBlock code={code} lang={lang} className="text-xs" />
      ) : (
        <pre className="scrollbar-overlay overflow-x-auto p-3 font-mono text-xs leading-relaxed">
          <code>{code}</code>
        </pre>
      )}
    </div>
  );
}

const LINK_CLASS = "text-foreground underline underline-offset-2";

/**
 * A link in a reply. A page of this app opens in place, the way every other
 * link in the app does, so "open [Rate matrices](/billing/…)" keeps the
 * person in their session; anything else opens in a new tab. An unsafe
 * scheme never gets here: react-markdown empties it first.
 */
/**
 * Draws a link the surrounding surface knows more about than its address,
 * such as a web page the agent searched, or returns null to leave it a plain
 * link. The renderer only ever sees links to the web, never an app path.
 */
export type MarkdownLinkRenderer = (href: string, children: ReactNode) => ReactNode | null;

export const MarkdownLinkContext = createContext<MarkdownLinkRenderer | null>(null);

/**
 * Draws an artifact a reply names in its sentence, or returns null when the
 * surface does not know that artifact, which leaves the link its words.
 */
export type ArtifactLinkRenderer = (id: string, children: ReactNode) => ReactNode | null;

export const ArtifactLinkContext = createContext<ArtifactLinkRenderer | null>(null);

export function MarkdownLink({ href, children }: ComponentProps<"a">) {
  const inRouter = useInRouterContext();
  const renderLink = use(MarkdownLinkContext);
  const renderArtifact = use(ArtifactLinkContext);
  const artifactId = artifactRefId(href);
  if (artifactId !== null || href?.startsWith("artifact:")) {
    const drawn =
      artifactId !== null && renderArtifact ? renderArtifact(artifactId, children) : null;
    return drawn ?? <span className="font-medium">{children}</span>;
  }
  if (inRouter && isAppPath(href)) {
    return (
      <Link to={href} className={LINK_CLASS}>
        {children}
      </Link>
    );
  }

  const known = renderLink && href ? renderLink(href, children) : null;
  if (known) {
    return known;
  }

  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className={LINK_CLASS}>
      {children}
    </a>
  );
}

/**
 * An image in a reply is never loaded. The browser fetches an image the
 * moment it renders, without a click, so a reply that a document or an email
 * talked into writing `![](https://host/?d=…)` would send whatever it put in
 * the address to that host as soon as the reply was read. It is shown as a
 * link to open deliberately, and only its words are drawn.
 */
function MarkdownImage({ src, alt }: ComponentProps<"img">) {
  const label = alt?.trim() || "image";
  if (typeof src !== "string" || src === "") {
    return <span>{label}</span>;
  }

  return <MarkdownLink href={src}>{label}</MarkdownLink>;
}

/**
 * The type a reply is set in. Headings carry the heading weight and keep the
 * case they were written in; a table takes the house row rhythm and
 * hairlines, so a reply's figures read like a table anywhere else in the
 * product; a list breathes a little between its items.
 */
const components: Components = {
  p: ({ children }) => <p className="my-2 leading-relaxed first:mt-0 last:mb-0">{children}</p>,
  h1: ({ children }) => (
    <h3 className="md-h md-h1 mt-4 mb-1.5 text-base font-semibold first:mt-0">{children}</h3>
  ),
  h2: ({ children }) => (
    <h3 className="md-h md-h2 mt-3.5 mb-1.5 text-sm font-semibold first:mt-0">{children}</h3>
  ),
  h3: ({ children }) => (
    <h4 className="md-h md-h3 mt-3 mb-1 text-sm font-semibold first:mt-0">{children}</h4>
  ),
  h4: ({ children }) => (
    <h5 className="md-h md-h4 mt-2 mb-1 text-sm font-medium first:mt-0">{children}</h5>
  ),
  h5: ({ children }) => (
    <h6 className="md-h md-h5 mt-2 mb-1 text-sm font-medium first:mt-0">{children}</h6>
  ),
  h6: ({ children }) => (
    <h6 className="md-h md-h6 text-muted-foreground mt-2 mb-1 text-xs font-medium first:mt-0">
      {children}
    </h6>
  ),
  ul: ({ children }) => (
    <ul className="marker:text-foreground-subtle my-2 list-disc space-y-1 pl-5">{children}</ul>
  ),
  ol: ({ children }) => (
    <ol className="marker:text-foreground-subtle my-2 list-decimal space-y-1 pl-5 marker:tabular-nums">
      {children}
    </ol>
  ),
  li: ({ children, className }) => (
    <li
      className={cn("pl-0.5 leading-relaxed", className?.includes("task-list-item") && "md-task")}
    >
      {children}
    </li>
  ),
  // A checklist's box: drawn, never editable, since the reply is a record.
  input: ({ type, checked }) =>
    type === "checkbox" ? (
      <span
        className={cn("md-check", checked && "md-checked")}
        role="img"
        aria-label={checked ? "Done" : "Not done"}
      />
    ) : null,
  del: ({ children }) => <del className="text-muted-foreground">{children}</del>,
  strong: ({ children }) => <strong className="font-semibold">{children}</strong>,
  em: ({ children }) => <em>{children}</em>,
  a: ({ href, children }) => <MarkdownLink href={href}>{children}</MarkdownLink>,
  img: ({ src, alt }) => <MarkdownImage src={src} alt={alt} />,
  blockquote: ({ children }) => (
    <blockquote className="border-border text-muted-foreground my-2.5 border-l-2 pl-3">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="border-border-subtle my-4" />,
  table: ({ children }) => (
    <div className="md-table border-border scrollbar-overlay rounded-surface my-2.5 overflow-x-auto border">
      <table className="w-full border-collapse text-xs">{children}</table>
    </div>
  ),
  thead: ({ children }) => <thead className="bg-sunken">{children}</thead>,
  th: ({ children, className, style }) => (
    <th
      className={cn(
        "border-border-subtle text-foreground-muted h-(--row-head-h) border-b px-(--cell-px) text-left align-middle font-medium whitespace-nowrap",
        className,
      )}
      style={style}
    >
      {children}
    </th>
  ),
  tr: ({ children }) => <tr className="last:[&>td]:border-b-0">{children}</tr>,
  td: ({ children, className, style }) => (
    <td
      className={cn(
        "border-border-subtle h-(--row-h-compact) border-b px-(--cell-px) py-1 align-top tabular-nums",
        className,
      )}
      style={style}
    >
      {children}
    </td>
  ),
  code: ({ className, children, ...props }) => {
    const text = textOf(children);
    // Math the typesetter has not been read for yet: shown as written, in the
    // box an unfinished block uses, until the reply is drawn again with it.
    if (typeof className === "string" && className.includes("language-math")) {
      return className.includes("math-display") ? (
        <pre className="md-math-raw">{text}</pre>
      ) : (
        <span className="md-math-pending">{text}</span>
      );
    }
    const isBlock = typeof className === "string" && className.includes("language-");
    if (isBlock || text.includes("\n")) {
      return <CodeBlock className={className}>{children}</CodeBlock>;
    }

    return (
      <code
        className="md-icode bg-sunken rounded-control px-1 py-0.5 font-mono text-[0.85em]"
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
  overrides,
  deskSubset = false,
}: {
  content: string;
  className?: string;
  /** Elements a surface draws its own way, such as a link it reads as something else. */
  overrides?: Components;
  /** Keeps to the Desk's markdown set; see DESK_REMARK_PLUGINS. */
  deskSubset?: boolean;
}) {
  const merged = useMemo(
    () => (overrides ? { ...components, ...overrides } : components),
    [overrides],
  );
  const prepared = useMemo(() => prepareMarkdown(content), [content]);
  const rehypePlugins = rehypePluginsWith(useKatexPlugin(markdownHasMath(prepared)));

  return (
    <div className={cn("text-sm wrap-break-word", className)}>
      <ReactMarkdown
        remarkPlugins={deskSubset ? DESK_REMARK_PLUGINS : REMARK_PLUGINS}
        rehypePlugins={rehypePlugins}
        components={merged}
        urlTransform={urlTransform}
      >
        {prepared}
      </ReactMarkdown>
    </div>
  );
});

const REMARK_PLUGINS = [remarkGfm, remarkMath, remarkBreaks];

/**
 * The Desk's reply set: the same, less what its design leaves out (bare
 * addresses as links, footnotes, images, indented code). Other surfaces keep
 * the broader set they were built for.
 */
const DESK_REMARK_PLUGINS = [...REMARK_PLUGINS, remarkDeskSubset];

/** A reference definition line, "[id]: url", wherever it sits in a reply. */
const DEFINITION = /^ {0,3}\[[^\]\n]+\]:[ \t]*\S.*$/gmu;

/** What every reply's markup goes through: figures aligned. */
const REHYPE_PLUGINS: PluggableList = [[rehypeNumericColumns, { className: "md-num" }]];

let withMath: { katex: PluggableList[number]; plugins: PluggableList } | null = null;

/**
 * The reply's plugins, with the math typesetter first once it has been read.
 * One list for the tab, so a block's plugins keep their identity and it is
 * not parsed again for nothing.
 */
function rehypePluginsWith(katex: PluggableList[number] | null): PluggableList {
  if (katex === null) {
    return REHYPE_PLUGINS;
  }
  if (withMath?.katex !== katex) {
    withMath = { katex, plugins: [katex, ...REHYPE_PLUGINS] };
  }
  return withMath.plugins;
}

/**
 * react-markdown empties any address with a scheme it does not know, which
 * would turn a link to an artifact into an empty one. That scheme is kept;
 * it never reaches the browser as an address, only as a badge or as words.
 */
function urlTransform(url: string): string {
  return artifactRefId(url) !== null ? url : defaultUrlTransform(url);
}

/** One top-level block, parsed again only when its own text changes. */
const MarkdownBlock = memo(function MarkdownBlock({
  content,
  merged,
  rehypePlugins,
  deskSubset,
}: {
  content: string;
  merged: Components;
  rehypePlugins: PluggableList;
  deskSubset: boolean;
}) {
  return (
    <ReactMarkdown
      remarkPlugins={deskSubset ? DESK_REMARK_PLUGINS : REMARK_PLUGINS}
      rehypePlugins={rehypePlugins}
      components={merged}
      urlTransform={urlTransform}
    >
      {content}
    </ReactMarkdown>
  );
});

/**
 * AiMarkdown for a reply still arriving. Parsing the whole reply on every
 * token costs the square of its length; cut into its top-level blocks, every
 * block but the last is final and parsed once. The blocks are siblings in one
 * container, joined by the line break one parse puts between them, so what is
 * drawn is what one parse would draw.
 */
export const StreamingAiMarkdown = memo(function StreamingAiMarkdown({
  content,
  className,
  overrides,
  wordClassName,
  caretClassName,
  deskSubset = false,
}: {
  content: string;
  className?: string;
  overrides?: Components;
  deskSubset?: boolean;
  /** Wraps each word in a span of this class, so a surface can bring words in as they land. */
  wordClassName?: string;
  /** Ends the reply in a caret of this class, after its last word. */
  caretClassName?: string;
}) {
  const prepared = useMemo(() => prepareMarkdown(content, { streaming: true }), [content]);
  const blocks = useMemo(() => splitMarkdownBlocks(prepared), [prepared]);
  // Each block is parsed on its own, so a reference link in one block would
  // not find its definition in another. Every block carries the reply's
  // definitions; they draw nothing themselves.
  const definitions = useMemo(() => (prepared.match(DEFINITION) ?? []).join("\n"), [prepared]);
  const merged = useMemo(
    () => (overrides ? { ...components, ...overrides } : components),
    [overrides],
  );
  const basePlugins = rehypePluginsWith(useKatexPlugin(markdownHasMath(prepared)));
  const wordPlugins = useMemo<PluggableList>(
    () =>
      wordClassName
        ? [...basePlugins, [rehypeStreamWords, { className: wordClassName }]]
        : basePlugins,
    [basePlugins, wordClassName],
  );
  const lastPlugins = useMemo<PluggableList>(
    () =>
      wordClassName || caretClassName
        ? [
            ...basePlugins,
            [rehypeStreamWords, { className: wordClassName ?? "", caret: caretClassName }],
          ]
        : basePlugins,
    [basePlugins, caretClassName, wordClassName],
  );

  return (
    <div className={cn("text-sm wrap-break-word", className)}>
      {blocks.map((block, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- blocks only ever grow at the end
        <Fragment key={index}>
          {index > 0 && "\n"}
          <MarkdownBlock
            content={definitions === "" ? block : `${block}\n\n${definitions}`}
            merged={merged}
            rehypePlugins={index === blocks.length - 1 ? lastPlugins : wordPlugins}
            deskSubset={deskSubset}
          />
        </Fragment>
      ))}
    </div>
  );
});
