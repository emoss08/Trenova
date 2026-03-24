import { useTheme } from "@/components/theme-provider";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { useEffect, useMemo, useState } from "react";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import vitesseBlack from "shiki/themes/vitesse-black.mjs";
import vitesseLight from "shiki/themes/vitesse-light.mjs";
import jsonLanguage from "shiki/langs/json.mjs";
import { createHighlighterCore } from "shiki/core";

const SHIKI_THEME_LIGHT = "vitesse-light";
const SHIKI_THEME_DARK = "vitesse-black";

type ResolvedTheme = "light" | "dark";

let shikiHighlighterPromise: Promise<{
  codeToHtml: (code: string, options: { lang: string; theme: string }) => string;
}> | null = null;

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
}: {
  value: unknown;
  className?: string;
}) {
  const resolvedTheme = useResolvedTheme();
  const [html, setHtml] = useState<string>("");

  const jsonCode = useMemo(() => {
    try {
      return JSON.stringify(value ?? null, null, 2);
    } catch {
      return '"Unable to serialize value"';
    }
  }, [value]);

  const scrollHeight = useMemo(() => {
    const lineCount = jsonCode.split("\n").length;
    const estimated = lineCount * 18 + 24;
    return Math.min(Math.max(estimated, 56), 288);
  }, [jsonCode]);

  useEffect(() => {
    let cancelled = false;

    async function render() {
      const highlighter = await getShikiHighlighter();
      const shikiTheme =
        resolvedTheme === "dark" ? SHIKI_THEME_DARK : SHIKI_THEME_LIGHT;
      const renderedHtml = highlighter.codeToHtml(jsonCode, {
        lang: "json",
        theme: shikiTheme,
      });

      if (!cancelled) {
        setHtml(injectSensitiveBadgesIntoHtml(renderedHtml));
      }
    }

    void render();

    return () => {
      cancelled = true;
    };
  }, [jsonCode, resolvedTheme]);

  if (!html) {
    return (
      <ScrollArea
        className={cn(
          "rounded-md border border-border/80 bg-muted/30",
          className,
        )}
        style={{ height: scrollHeight }}
      >
        <pre className="p-3 font-mono text-xs text-foreground">{jsonCode}</pre>
      </ScrollArea>
    );
  }

  return (
    <ScrollArea
      className={cn(
        "audit-shiki-json rounded-md border border-border/80",
        "[&_pre]:m-0! [&_pre]:rounded-none! [&_pre]:border-0! [&_pre]:p-3! [&_pre]:text-xs!",
        className,
      )}
      style={{ height: scrollHeight }}
    >
      <div dangerouslySetInnerHTML={{ __html: html }} />
    </ScrollArea>
  );
}
