import { stepsFromExchanges, type ToolStep } from "./activity";
import type { ThreadEntry } from "./thread-view";
import { isWebTool, parseToolResult, WEB_READ_TOOL } from "./tool-presentation";

export { isWebTool, WEB_READ_TOOL, WEB_SEARCH_TOOL } from "./tool-presentation";

/** A page the agent found or read on the web during a reply. */
export type WebSource = {
  url: string;
  title: string;
  site: string;
  official: boolean;
  /** The page's publication day, `YYYY-MM-DD`, when the page gave one. */
  published: string;
  excerpt: string;
  /** The day the agent retrieved it, `YYYY-MM-DD`. */
  retrievedOn: string;
  /** The answer links to it. */
  cited: boolean;
  /** The agent opened the page in full rather than reading search passages. */
  read: boolean;
};

/**
 * The address a source is matched by: scheme and host case-folded, a leading
 * `www.` and a trailing slash dropped, and the fragment ignored, so the link
 * a model writes finds the result it copied it from.
 */
export function sourceKey(raw: string): string {
  try {
    const url = new URL(raw.trim());
    const host = url.hostname.toLowerCase().replace(/^www\./u, "");
    const path = url.pathname.replace(/\/+$/u, "");

    return `${host}${path}${url.search}`;
  } catch {
    return raw.trim().toLowerCase();
  }
}

/** A site as a person reads it: the host without `www.`. */
export function siteOf(raw: string): string {
  try {
    return new URL(raw.trim()).hostname.toLowerCase().replace(/^www\./u, "");
  } catch {
    return "";
  }
}

type RawResult = Record<string, unknown>;

function text(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function firstExcerpt(value: unknown): string {
  if (!Array.isArray(value)) {
    return "";
  }
  for (const entry of value) {
    const passage = text(entry)
      .split(/\n\.\.\.\n|\n/u)
      .map((part) => part.trim())
      .find((part) => part.length >= 40);
    if (passage) {
      return passage;
    }
  }

  return "";
}

function toSource(raw: RawResult, retrievedOn: string, read: boolean): WebSource | null {
  const url = text(raw.url);
  if (!/^https?:\/\//iu.test(url)) {
    return null;
  }

  return {
    url,
    title: text(raw.title) || siteOf(url),
    site: text(raw.site) || siteOf(url),
    official: raw.source === "official",
    published: text(raw.published),
    excerpt: read ? text(raw.text).slice(0, 280) : firstExcerpt(raw.excerpts),
    retrievedOn,
    cited: false,
    read,
  };
}

/** The pages one web step returned, in the order it returned them. */
export function sourcesOfStep(step: ToolStep): WebSource[] {
  if (!isWebTool(step.name) || step.status === "running" || step.status === "failed") {
    return [];
  }

  const parsed = parseToolResult(step.content);
  if (parsed.kind !== "json" || parsed.value === null || typeof parsed.value !== "object") {
    return [];
  }

  const document = parsed.value as RawResult;
  const retrievedOn = text(document.retrievedOn);
  if (step.name === WEB_READ_TOOL) {
    const page = toSource(document, retrievedOn, true);
    return page ? [page] : [];
  }

  const results = Array.isArray(document.results) ? (document.results as RawResult[]) : [];
  const sources: WebSource[] = [];
  for (const result of results) {
    const source =
      result && typeof result === "object" ? toSource(result, retrievedOn, false) : null;
    if (source) {
      sources.push(source);
    }
  }

  return sources;
}

/** Every web address the answer links to, in the order it first links to them. */
function citedKeys(answer: string): string[] {
  const keys: string[] = [];
  const seen = new Set<string>();
  for (const match of answer.matchAll(/\]\((https?:\/\/[^\s)]+)\)|<(https?:\/\/[^\s>]+)>/giu)) {
    const key = sourceKey(match[1] ?? match[2] ?? "");
    if (!seen.has(key)) {
      seen.add(key);
      keys.push(key);
    }
  }

  return keys;
}

/**
 * What a reply drew on from the web: every page its searches returned or it
 * read, once each. The pages the answer links to come first, in the order it
 * cites them; the rest follow in the order they were found. A page the agent
 * both found and read is kept as read, with its search excerpt.
 */
export function webSourcesOf(steps: readonly ToolStep[], answer: string): WebSource[] {
  const byKey = new Map<string, WebSource>();
  for (const step of steps) {
    for (const source of sourcesOfStep(step)) {
      const key = sourceKey(source.url);
      const known = byKey.get(key);
      if (!known) {
        byKey.set(key, source);
        continue;
      }
      if (source.read) {
        byKey.set(key, {
          ...known,
          read: true,
          title: source.title || known.title,
          published: source.published || known.published,
          excerpt: known.excerpt || source.excerpt,
        });
      }
    }
  }

  const cited: WebSource[] = [];
  for (const key of citedKeys(answer)) {
    const source = byKey.get(key);
    if (source) {
      cited.push({ ...source, cited: true });
      byKey.delete(key);
    }
  }

  return [...cited, ...byKey.values()];
}

/** Whether a reply used the web at all, whatever its searches found. */
export function usedTheWeb(steps: readonly ToolStep[]): boolean {
  return steps.some((step) => isWebTool(step.name));
}

export type SourceIndex = ReadonlyMap<string, WebSource>;

export function indexSources(sources: readonly WebSource[]): SourceIndex {
  return new Map(sources.map((source) => [sourceKey(source.url), source]));
}

export function sourceFor(index: SourceIndex, href: string | undefined): WebSource | undefined {
  return href ? index.get(sourceKey(href)) : undefined;
}

/** What one written step of a saved reply knows of the web. */
export type StepSources = {
  /** Every page the reply found up to and including this step. */
  sources: WebSource[];
  /** This step is the reply's answer, the one that lists its sources. */
  answer: boolean;
};

/**
 * The web sources each written step of a saved reply drew on. A reply saved
 * as several steps searches in one and answers in a later one, so the pages
 * are gathered across the reply up to the step that cites them, and a
 * question or a decision starts a new reply. Only the last step that says
 * anything is the answer; the sources are listed once, under it.
 */
export function replyWebSources(entries: readonly ThreadEntry[]): Map<string, StepSources> {
  const byMessage = new Map<string, StepSources>();
  let steps: ToolStep[] = [];
  let answer: StepSources | null = null;

  const closeReply = () => {
    if (answer) {
      answer.answer = true;
    }
    steps = [];
    answer = null;
  };

  for (const entry of entries) {
    if (entry.kind !== "assistant") {
      closeReply();
      continue;
    }
    steps = steps.concat(
      stepsFromExchanges(entry.tools, entry.message.createdAt).filter((step) =>
        isWebTool(step.name),
      ),
    );
    if (entry.message.content.trim() === "" || steps.length === 0) {
      continue;
    }
    answer = { sources: webSourcesOf(steps, entry.message.content), answer: false };
    byMessage.set(entry.message.id, answer);
  }
  closeReply();

  return byMessage;
}
