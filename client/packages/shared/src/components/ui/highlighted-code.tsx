import {
  codeLanguage,
  createMatcher,
  findMatches,
  highlightLines,
  highlightSpans,
  lineText,
  markSegments,
  type CodeLine,
  type CodeRole,
  type CodeSpan,
} from "@trenova/shared/lib/highlight/code";
import { cn } from "@trenova/shared/lib/utils";
import { Fragment, memo, type MouseEvent, type ReactNode, useMemo } from "react";

// Each role takes a syntax token (styles/tokens.css), so code follows the theme through CSS
// alone and is never highlighted twice. Comments and punctuation sit back in the text greys,
// and a diff's added and removed lines read as success and danger.
const ROLE_CLASS: Record<CodeRole, string> = {
  attr: "text-syntax-property",
  "code-inline": "text-syntax-string",
  command: "text-syntax-keyword",
  comment: "text-foreground-subtle italic",
  deleted: "text-danger-foreground",
  function: "text-syntax-function",
  heading: "text-syntax-meta",
  inserted: "text-success-foreground",
  keyword: "text-syntax-keyword",
  link: "text-syntax-string",
  literal: "text-syntax-number",
  meta: "text-syntax-meta",
  number: "text-syntax-number",
  operator: "text-foreground-muted",
  property: "text-syntax-property",
  selector: "text-syntax-keyword",
  string: "text-syntax-string",
  tag: "text-syntax-type",
  type: "text-syntax-type",
  variable: "text-syntax-function",
};

const MARK_CLASS = "rounded-sm bg-warning-subtle text-warning-subtle-foreground";

export type HighlightedCodeProps = {
  code: string;
  /** A language name or fence label; one the app has no grammar for is shown as plain text. */
  language?: string;
  /**
   * Roles from the language's own tokenizer, in place of a grammar: Tariff is coloured from the
   * rules module's `tariff.tokens`, so the browser holds no second grammar for it.
   */
  spans?: readonly CodeSpan[];
  className?: string;
  /** Marks every case-insensitive occurrence, across token boundaries. */
  query?: string;
  lineClassName?: string;
  /** Drawn after a line's code, given the line's text and its zero-based index. */
  lineEnd?: (text: string, index: number) => ReactNode;
  onLineClick?: (index: number) => void;
};

type LineProps = {
  line: CodeLine;
  index: number;
  matcher: RegExp | null;
  className?: string;
  lineEnd?: (text: string, index: number) => ReactNode;
};

const Line = memo(function Line({ line, index, matcher, className, lineEnd }: LineProps) {
  const text = lineText(line);
  const segments = markSegments(line, findMatches(text, matcher));
  return (
    <span data-line={index} className={cn("inline-block min-w-full", className)}>
      {segments.map((segment, i) =>
        segment.marked ? (
          <mark key={i} className={MARK_CLASS}>
            {segment.text}
          </mark>
        ) : segment.role ? (
          <span key={i} className={ROLE_CLASS[segment.role]}>
            {segment.text}
          </span>
        ) : (
          <Fragment key={i}>{segment.text}</Fragment>
        ),
      )}
      {lineEnd?.(text, index)}
    </span>
  );
});

/**
 * Code coloured by syntactic role. It renders elements, never markup strings, so nothing in the
 * code, a search or a decoration can become HTML.
 */
export function HighlightedCode({
  code,
  language,
  spans,
  className,
  query = "",
  lineClassName,
  lineEnd,
  onLineClick,
}: HighlightedCodeProps) {
  const lines = useMemo(
    () => (spans ? highlightSpans(code, spans) : highlightLines(code, language)),
    [code, language, spans],
  );
  const matcher = useMemo(() => createMatcher(query), [query]);

  const handleClick = onLineClick
    ? (event: MouseEvent<HTMLElement>) => {
        const line = (event.target as Element).closest("[data-line]");
        if (line) onLineClick(Number(line.getAttribute("data-line")));
      }
    : undefined;

  return (
    <pre
      data-language={(spans ? language : codeLanguage(language)) ?? undefined}
      className={cn("font-code text-xs leading-relaxed text-foreground", className)}
    >
      <code onClick={handleClick}>
        {lines.map((line, index) => (
          <Fragment key={index}>
            {index > 0 && "\n"}
            <Line
              line={line}
              index={index}
              matcher={matcher}
              className={lineClassName}
              lineEnd={lineEnd}
            />
          </Fragment>
        ))}
      </code>
    </pre>
  );
}
