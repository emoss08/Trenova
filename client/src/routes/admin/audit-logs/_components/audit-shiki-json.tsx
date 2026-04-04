import { useTheme } from "@/components/theme-provider";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { useDebounce } from "@/hooks/use-debounce";
import { cn } from "@/lib/utils";
import { SearchIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import vitesseBlack from "shiki/themes/vitesse-black.mjs";
import vitesseLight from "shiki/themes/vitesse-light.mjs";
import jsonLanguage from "shiki/langs/json.mjs";
import { createHighlighterCore } from "shiki/core";

const SHIKI_THEME_LIGHT = "vitesse-light";
const SHIKI_THEME_DARK = "vitesse-black";
const LINE_HEIGHT_PX = 18;
const PADDING_PX = 24;
const MIN_HEIGHT = 56;
const MAX_HEIGHT = 288;

type ResolvedTheme = "light" | "dark";

let shikiHighlighterPromise: Promise<{
  codeToHtml: (code: string, options: { lang: string; theme: string }) => string;
}> | null = null;

function escapeRegExp(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function injectSensitiveBadgesIntoHtml(html: string): string {
  const badge =
    '<span title="Sensitive data omitted" class="ml-1 inline-flex h-4 select-none items-center rounded border border-amber-500/40 bg-amber-500/15 px-1 text-[10px] leading-none text-amber-700 dark:text-amber-300">Sensitive</span>';

  return html
    .split("\n")
    .map((line) => {
      if (!/(\[REDACTED\]|\*{4,})/.test(line)) {
        return line;
      }

      if (line.includes(">Sensitive</span>")) {
        return line;
      }

      return `${line}${badge}`;
    })
    .join("\n");
}

function injectSearchHighlightsIntoHtml(
  html: string,
  query: string,
): { html: string; count: number } {
  if (!query) return { html, count: 0 };

  const escaped = escapeRegExp(query);
  const regex = new RegExp(escaped, "gi");
  const parts = html.split(/(<[^>]*>)/);
  let count = 0;

  const result = parts
    .map((part) => {
      if (part.startsWith("<")) return part;
      return part.replace(regex, (match) => {
        count++;
        return `<mark class="bg-yellow-300/50 dark:bg-yellow-500/30 rounded-sm">${match}</mark>`;
      });
    })
    .join("");

  return { html: result, count };
}

function buildJsonPathIndex(value: unknown): Map<number, string> {
  const map = new Map<number, string>();
  let line = 0;

  function walk(val: unknown, path: string) {
    if (val === null || typeof val !== "object") {
      return;
    }

    if (Array.isArray(val)) {
      line++; // opening [
      for (let i = 0; i < val.length; i++) {
        const childPath = `${path}[${i}]`;
        const child = val[i];
        if (child !== null && typeof child === "object") {
          walk(child, childPath);
        } else {
          map.set(line, childPath);
          line++;
        }
      }
      line++; // closing ]
    } else {
      const entries = Object.entries(val);
      line++; // opening {
      for (const [key, child] of entries) {
        const childPath = `${path}.${key}`;
        if (child !== null && typeof child === "object") {
          map.set(line, childPath);
          walk(child, childPath);
        } else {
          map.set(line, childPath);
          line++;
        }
      }
      line++; // closing }
    }
  }

  walk(value, "$");
  return map;
}

async function getShikiHighlighter() {
  if (!shikiHighlighterPromise) {
    shikiHighlighterPromise = createHighlighterCore({
      themes: [vitesseLight, vitesseBlack],
      langs: [jsonLanguage],
      engine: createJavaScriptRegexEngine(),
    });
  }

  return shikiHighlighterPromise;
}

function useResolvedTheme(): ResolvedTheme {
  const { theme } = useTheme();
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>("light");

  useEffect(() => {
    if (theme === "dark" || theme === "light") {
      setResolvedTheme(theme);
      return;
    }

    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const update = () => setResolvedTheme(media.matches ? "dark" : "light");

    update();
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, [theme]);

  return resolvedTheme;
}

export function ShikiJsonBlock({
  value,
  className,
  searchable = false,
  copyPath = false,
}: {
  value: unknown;
  className?: string;
  searchable?: boolean;
  copyPath?: boolean;
}) {
  const resolvedTheme = useResolvedTheme();
  const [baseHtml, setBaseHtml] = useState<string>("");
  const [searchQuery, setSearchQuery] = useState("");
  const debouncedQuery = useDebounce(searchQuery, 150);
  const codeRef = useRef<HTMLDivElement>(null);
  const { copy } = useCopyToClipboard();

  const hasToolbar = searchable || copyPath;

  const jsonCode = useMemo(() => {
    try {
      return JSON.stringify(value ?? null, null, 2);
    } catch {
      return '"Unable to serialize value"';
    }
  }, [value]);

  const scrollHeight = useMemo(() => {
    const lineCount = jsonCode.split("\n").length;
    const estimated = lineCount * LINE_HEIGHT_PX + PADDING_PX;
    return Math.min(Math.max(estimated, MIN_HEIGHT), MAX_HEIGHT);
  }, [jsonCode]);

  const pathIndex = useMemo(() => {
    if (!copyPath) return new Map<number, string>();
    return buildJsonPathIndex(value);
  }, [value, copyPath]);

  useEffect(() => {
    let cancelled = false;

    async function render() {
      const highlighter = await getShikiHighlighter();
      const shikiTheme = resolvedTheme === "dark" ? SHIKI_THEME_DARK : SHIKI_THEME_LIGHT;
      const renderedHtml = highlighter.codeToHtml(jsonCode, {
        lang: "json",
        theme: shikiTheme,
      });

      if (!cancelled) {
        setBaseHtml(injectSensitiveBadgesIntoHtml(renderedHtml));
      }
    }

    void render();

    return () => {
      cancelled = true;
    };
  }, [jsonCode, resolvedTheme]);

  const { html: displayHtml, count: matchCount } = useMemo(() => {
    if (!baseHtml) return { html: "", count: 0 };
    return injectSearchHighlightsIntoHtml(baseHtml, debouncedQuery);
  }, [baseHtml, debouncedQuery]);

  const handleCodeClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (!copyPath || pathIndex.size === 0) return;

      const container = codeRef.current;
      if (!container) return;

      const pre = container.querySelector("pre");
      if (!pre) return;

      const rect = pre.getBoundingClientRect();
      const y = e.clientY - rect.top;
      const lineNumber = Math.floor(y / LINE_HEIGHT_PX);
      const path = pathIndex.get(lineNumber);

      if (path) {
        void copy(path, { withToast: true });
      }
    },
    [copyPath, pathIndex, copy],
  );

  const scrollAreaClasses = hasToolbar
    ? cn(
        "audit-shiki-json",
        "[&_pre]:m-0! [&_pre]:rounded-none! [&_pre]:border-0! [&_pre]:p-3! [&_pre]:text-xs!",
      )
    : cn(
        "audit-shiki-json rounded-md border border-border/80",
        "[&_pre]:m-0! [&_pre]:rounded-none! [&_pre]:border-0! [&_pre]:p-3! [&_pre]:text-xs!",
        className,
      );

  if (!baseHtml) {
    const fallbackClasses = hasToolbar
      ? "bg-muted/30"
      : cn("rounded-md border border-border/80 bg-muted/30", className);

    return hasToolbar ? (
      <div className={cn("overflow-hidden rounded-md border border-border/80", className)}>
        {searchable && (
          <div className="flex items-center gap-2 border-b border-border/60 px-2 py-1.5">
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search JSON..."
              leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
              className="h-6 bg-transparent text-xs"
            />
          </div>
        )}
        <ScrollArea className={fallbackClasses} style={{ height: scrollHeight }}>
          <pre className="p-3 font-mono text-xs text-foreground">{jsonCode}</pre>
        </ScrollArea>
      </div>
    ) : (
      <ScrollArea className={fallbackClasses} style={{ height: scrollHeight }}>
        <pre className="p-3 font-mono text-xs text-foreground">{jsonCode}</pre>
      </ScrollArea>
    );
  }

  if (!hasToolbar) {
    return (
      <ScrollArea className={scrollAreaClasses} style={{ height: scrollHeight }}>
        <div dangerouslySetInnerHTML={{ __html: displayHtml }} />
      </ScrollArea>
    );
  }

  return (
    <div className={cn("overflow-hidden rounded-md border border-border/80", className)}>
      <div className="flex items-center gap-2 border-b border-border/60 px-2 py-1.5">
        {searchable && (
          <Input
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search JSON..."
            leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
            rightElement={
              debouncedQuery ? (
                <span className="pr-1 text-[10px] text-muted-foreground">
                  {matchCount} {matchCount === 1 ? "match" : "matches"}
                </span>
              ) : undefined
            }
            className="h-6 bg-transparent text-xs"
          />
        )}
        {copyPath && (
          <span className="shrink-0 text-[10px] text-muted-foreground">
            Click line to copy path
          </span>
        )}
      </div>
      <ScrollArea className={scrollAreaClasses} style={{ height: scrollHeight }}>
        <div
          ref={codeRef}
          onClick={handleCodeClick}
          className={cn(copyPath && "cursor-pointer [&_pre_.line:hover]:bg-muted/50")}
          dangerouslySetInnerHTML={{ __html: displayHtml }}
        />
      </ScrollArea>
    </div>
  );
}
