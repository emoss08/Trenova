const SCRIPT = /<script\b[^>]*\btype="module"[^>]*\bsrc="([^"]+)"/g;
const PRELOAD = /<link\b[^>]*\brel="modulepreload"[^>]*\bhref="([^"]+)"/g;
const STYLESHEET = /<link\b[^>]*\brel="stylesheet"[^>]*\bhref="([^"]+)"/g;

const urls = (html: string, pattern: RegExp) =>
  [...html.matchAll(pattern)].map((match) => match[1]);

const isLocal = (url: string) => url.startsWith("/") && !url.startsWith("//");

/**
 * startupFiles lists the files index.html makes the browser fetch before it can render, and
 * the entry script among them: the one file whose name changes whenever the build does.
 */
export function startupFiles(html: string): {
  entry: string | null;
  scripts: string[];
  styles: string[];
} {
  const entries = urls(html, SCRIPT).filter(isLocal);
  return {
    entry: entries[0] ?? null,
    scripts: [...new Set([...entries, ...urls(html, PRELOAD)])].filter(isLocal),
    styles: [...new Set(urls(html, STYLESHEET))].filter(isLocal),
  };
}
