import { useTheme } from "@/components/theme-provider";
import { cn } from "@/lib/utils";
import { useEffect, useState } from "react";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import { createHighlighterCore, type HighlighterCore } from "shiki/core";
import vitesseBlack from "shiki/themes/vitesse-black.mjs";
import vitesseDark from "shiki/themes/vitesse-dark.mjs";
import vitesseLight from "shiki/themes/vitesse-light.mjs";
import jsonLanguage from "shiki/langs/json.mjs";
import javascriptLanguage from "shiki/langs/javascript.mjs";
import plsqlLanguage from "shiki/langs/plsql.mjs";

type ResolvedTheme = "light" | "dark";
type SupportedLang = "json" | "plsql" | "javascript";
type DarkTheme = "vitesse-black" | "vitesse-dark";

let highlighterPromise: Promise<HighlighterCore> | null = null;

function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighterCore({
      themes: [vitesseLight, vitesseBlack, vitesseDark],
      langs: [jsonLanguage, plsqlLanguage, javascriptLanguage],
      engine: createJavaScriptRegexEngine(),
    });
  }
  return highlighterPromise;
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

export function ShikiCodeBlock({
  code,
  lang,
  className,
  darkTheme = "vitesse-black",
  transformHtml,
}: {
  code: string;
  lang: SupportedLang;
  className?: string;
  darkTheme?: DarkTheme;
  transformHtml?: (html: string) => string;
}) {
  const resolvedTheme = useResolvedTheme();
  const [html, setHtml] = useState<string>("");

  useEffect(() => {
    let cancelled = false;

    async function render() {
      const highlighter = await getHighlighter();
      const shikiTheme = resolvedTheme === "dark" ? darkTheme : "vitesse-light";
      let rendered = highlighter.codeToHtml(code, {
        lang,
        theme: shikiTheme,
      });

      if (transformHtml) {
        rendered = transformHtml(rendered);
      }

      if (!cancelled) {
        setHtml(rendered);
      }
    }

    void render();

    return () => {
      cancelled = true;
    };
  }, [code, lang, darkTheme, resolvedTheme, transformHtml]);

  if (!html) {
    return (
      <div className={cn("rounded-md bg-muted/50 p-2", className)}>
        <pre className="font-mono text-xs text-muted-foreground">{code}</pre>
      </div>
    );
  }

  return (
    <div
      className={cn(
        "shiki-code-block rounded-md [&_pre]:m-0! [&_pre]:rounded-md! [&_pre]:border-0! [&_pre]:p-2! [&_pre]:text-xs!",
        className,
      )}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}
