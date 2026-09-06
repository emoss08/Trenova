import { useTheme } from "@trenova/shared/components/theme-provider";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useState } from "react";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import {
  createHighlighterCore,
  type HighlighterCore,
  type LanguageInput,
  type ThemeInput,
} from "shiki/core";

type ResolvedTheme = "light" | "dark";
type SupportedLang = "json" | "plsql" | "javascript" | "graphql";
type DarkTheme = "vitesse-black" | "vitesse-dark";
type ShikiTheme = DarkTheme | "vitesse-light";

// Grammars and themes are fetched per code block rather than bundled together.
// The four grammars come to roughly 750 kB — the GraphQL one alone embeds the
// TypeScript grammar — and any given block needs exactly one of them.
const LANGUAGE_LOADERS: Record<SupportedLang, LanguageInput> = {
  json: () => import("shiki/langs/json.mjs"),
  plsql: () => import("shiki/langs/plsql.mjs"),
  javascript: () => import("shiki/langs/javascript.mjs"),
  graphql: () => import("shiki/langs/graphql.mjs"),
};

const THEME_LOADERS: Record<ShikiTheme, ThemeInput> = {
  "vitesse-light": () => import("shiki/themes/vitesse-light.mjs"),
  "vitesse-black": () => import("shiki/themes/vitesse-black.mjs"),
  "vitesse-dark": () => import("shiki/themes/vitesse-dark.mjs"),
};

let highlighterPromise: Promise<HighlighterCore> | null = null;
const pendingLanguages = new Map<SupportedLang, Promise<void>>();
const pendingThemes = new Map<ShikiTheme, Promise<void>>();

function getHighlighter() {
  highlighterPromise ??= createHighlighterCore({
    themes: [],
    langs: [],
    engine: createJavaScriptRegexEngine(),
  });
  return highlighterPromise;
}

/**
 * Resolves a highlighter that has exactly the requested grammar and theme
 * registered. Each grammar and theme is fetched and registered once per page.
 */
async function getHighlighterFor(lang: SupportedLang, theme: ShikiTheme) {
  const highlighter = await getHighlighter();

  let language = pendingLanguages.get(lang);
  if (!language) {
    language = highlighter.loadLanguage(LANGUAGE_LOADERS[lang]);
    pendingLanguages.set(lang, language);
  }

  let themeRegistration = pendingThemes.get(theme);
  if (!themeRegistration) {
    themeRegistration = highlighter.loadTheme(THEME_LOADERS[theme]);
    pendingThemes.set(theme, themeRegistration);
  }

  await Promise.all([language, themeRegistration]);
  return highlighter;
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
      const shikiTheme: ShikiTheme = resolvedTheme === "dark" ? darkTheme : "vitesse-light";
      const highlighter = await getHighlighterFor(lang, shikiTheme);
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
