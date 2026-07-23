import type { HighlighterCore, LanguageRegistration, ThemeRegistration } from "shiki/core";

export type ShikiLang = "json" | "javascript" | "plsql" | "graphql";
export type ShikiTheme = "vitesse-light" | "vitesse-dark" | "vitesse-black";

type LangModule = { default: LanguageRegistration[] };
type ThemeModule = { default: ThemeRegistration };

const langImporters: Record<ShikiLang, () => Promise<LangModule>> = {
  json: () => import("shiki/langs/json.mjs"),
  javascript: () => import("shiki/langs/javascript.mjs"),
  plsql: () => import("shiki/langs/plsql.mjs"),
  graphql: () => import("shiki/langs/graphql.mjs"),
};

const themeImporters: Record<ShikiTheme, () => Promise<ThemeModule>> = {
  "vitesse-light": () => import("shiki/themes/vitesse-light.mjs"),
  "vitesse-dark": () => import("shiki/themes/vitesse-dark.mjs"),
  "vitesse-black": () => import("shiki/themes/vitesse-black.mjs"),
};

let highlighterPromise: Promise<HighlighterCore> | null = null;
const loadedLangs = new Map<ShikiLang, Promise<void>>();
const loadedThemes = new Map<ShikiTheme, Promise<void>>();

function getHighlighter(): Promise<HighlighterCore> {
  if (!highlighterPromise) {
    highlighterPromise = Promise.all([import("shiki/core"), import("shiki/engine/javascript")])
      .then(([core, engine]) =>
        core.createHighlighterCore({ engine: engine.createJavaScriptRegexEngine() }),
      )
      .catch((error) => {
        highlighterPromise = null;
        throw error;
      });
  }
  return highlighterPromise;
}

function ensureLang(highlighter: HighlighterCore, lang: ShikiLang): Promise<void> {
  let pending = loadedLangs.get(lang);
  if (!pending) {
    pending = langImporters[lang]()
      .then((mod) => highlighter.loadLanguage(...mod.default))
      .catch((error) => {
        loadedLangs.delete(lang);
        throw error;
      });
    loadedLangs.set(lang, pending);
  }
  return pending;
}

function ensureTheme(highlighter: HighlighterCore, theme: ShikiTheme): Promise<void> {
  let pending = loadedThemes.get(theme);
  if (!pending) {
    pending = themeImporters[theme]()
      .then((mod) => highlighter.loadTheme(mod.default))
      .catch((error) => {
        loadedThemes.delete(theme);
        throw error;
      });
    loadedThemes.set(theme, pending);
  }
  return pending;
}

export async function highlightCode(
  code: string,
  lang: ShikiLang,
  theme: ShikiTheme,
): Promise<string> {
  const highlighter = await getHighlighter();
  await Promise.all([ensureLang(highlighter, lang), ensureTheme(highlighter, theme)]);
  return highlighter.codeToHtml(code, { lang, theme });
}
