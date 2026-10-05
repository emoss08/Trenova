import {
  isWebTool,
  siteOf,
  sourceKey,
  sourcesOfStep,
  WEB_SEARCH_TOOL,
} from "@/components/assistant/web-sources";
import type { ToolStep } from "@/components/assistant/activity";
import type { TurnState } from "@/components/assistant/turn-stream";

/** A page the reply found on the web, as the Desk draws it. */
export type DeskWebSource = {
  url: string;
  /** The site, without scheme or `www.`: "eia.gov". */
  site: string;
  title: string;
  /** One line of what the page says, from the passage the agent read. */
  snippet: string;
  /** When it was published, as a person reads it: "Today", "Sep 28"; empty when the page gave no date. */
  age: string;
};

/** What a web step needs to give up its pages: the stream's call or a saved step. */
type WebStep = { name: string; status: string; content: string };

const MONTH_DAY = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  timeZone: "UTC",
});
const MONTH_DAY_YEAR = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  year: "numeric",
  timeZone: "UTC",
});

/** A page's publish day, `YYYY-MM-DD`, as "Today", "Sep 28" or, from another year, "Sep 28, 2025". */
export function sourceAge(published: string, today: Date): string {
  const day = /^(\d{4})-(\d{2})-(\d{2})/u.exec(published);
  if (!day) {
    return "";
  }
  const date = new Date(Date.UTC(Number(day[1]), Number(day[2]) - 1, Number(day[3])));
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const local = new Date(Date.UTC(today.getFullYear(), today.getMonth(), today.getDate()));
  if (date.getTime() === local.getTime()) {
    return "Today";
  }

  return date.getUTCFullYear() === today.getFullYear()
    ? MONTH_DAY.format(date)
    : MONTH_DAY_YEAR.format(date);
}

/**
 * Every page the reply's web steps returned or read, once each by address,
 * in the order they were first returned. A citation's number is a page's
 * place in this list, counted from one.
 */
export function deskWebSources(
  steps: readonly WebStep[],
  today: Date = new Date(),
): DeskWebSource[] {
  const seen = new Set<string>();
  const sources: DeskWebSource[] = [];
  for (const step of steps) {
    if (!isWebTool(step.name)) {
      continue;
    }
    for (const page of sourcesOfStep(step as ToolStep)) {
      const key = sourceKey(page.url);
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      sources.push({
        url: page.url,
        site: page.site || siteOf(page.url),
        title: page.title,
        snippet: page.excerpt,
        age: sourceAge(page.published, today),
      });
    }
  }

  return sources;
}

/** What the reply's first web search asked, for the list under it; empty when it did not search. */
export function webQueryOf(steps: readonly Pick<ToolStep, "name" | "arguments">[]): string {
  const search = steps.find((step) => step.name === WEB_SEARCH_TOOL);
  const query = search?.arguments?.query;

  return typeof query === "string" ? query.trim() : "";
}

/** The site a citation pill names: its domain, a trailing `.com`, `.gov` or `.org` left off. */
export function citeLabel(site: string): string {
  return site.replace(/\.(com|gov|org)$/u, "");
}

/** A site's tile colour, the same wherever the site is drawn. */
export function siteHue(site: string): number {
  let hue = 0;
  for (let index = 0; index < site.length; index++) {
    hue = (hue * 31 + site.charCodeAt(index)) % 360;
  }

  return hue;
}

/**
 * A reply cites the web with links to the pages it found. One or more such
 * links side by side become one link whose address is this, followed by the
 * cited pages' numbers, `#dk-web-1.4`, drawn as a single citation pill.
 */
export const WEB_CITE_HREF = "#dk-web-";

/** The page numbers a citation link names, or null for any other link. */
export function webCiteIds(href: string | undefined): number[] | null {
  if (!href?.startsWith(WEB_CITE_HREF)) {
    return null;
  }
  const ids = href
    .slice(WEB_CITE_HREF.length)
    .split(".")
    .map(Number)
    .filter((id) => Number.isInteger(id) && id > 0);

  return ids.length > 0 ? ids : null;
}

/** The pages a citation names, in its order; numbers with no page are dropped. */
export function citedSources(
  ids: readonly number[],
  sources: readonly DeskWebSource[],
): DeskWebSource[] {
  return ids.map((id) => sources[id - 1]).filter((source) => source !== undefined);
}

const PAGE_LINK = /\[[^\]\n]*\]\((https?:\/\/[^\s)]+)\)|<(https?:\/\/[^\s>]+)>/gu;
const BETWEEN_LINKS = /^[\s,;]*$/u;

/** Links to found pages in one stretch of prose, merged where they sit side by side. */
function citeProse(prose: string, numberOf: ReadonlyMap<string, number>): string {
  let out = "";
  let at = 0;
  let run: { start: number; end: number; ids: number[] } | null = null;

  const closeRun = () => {
    if (!run) {
      return;
    }
    out +=
      prose.slice(at, run.start) + `[${run.ids.join(", ")}](${WEB_CITE_HREF}${run.ids.join(".")})`;
    at = run.end;
    run = null;
  };

  for (const match of prose.matchAll(PAGE_LINK)) {
    const start = match.index;
    const end = start + match[0].length;
    const id = numberOf.get(sourceKey(match[1] ?? match[2] ?? ""));
    if (id === undefined) {
      closeRun();
      continue;
    }
    if (run && BETWEEN_LINKS.test(prose.slice(run.end, start))) {
      if (!run.ids.includes(id)) {
        run.ids.push(id);
      }
      run.end = end;
      continue;
    }
    closeRun();
    run = { start, end, ids: [id] };
  }
  closeRun();

  return out + prose.slice(at);
}

/**
 * The reply with each citation of a found page as a citation link. A link to
 * an address the agent never found stays a plain link: a model can write one
 * it made up, and that should not look like a source. Code is left as written.
 */
export function withWebCites(content: string, sources: readonly DeskWebSource[]): string {
  if (sources.length === 0 || content === "") {
    return content;
  }
  const numberOf = new Map(sources.map((source, index) => [sourceKey(source.url), index + 1]));

  return content
    .split(/(^\s*(?:```|~~~)[^\n]*\n[\s\S]*?^\s*(?:```|~~~)\s*$|`[^`\n]*`)/mu)
    .map((part, index) => (index % 2 === 1 ? part : citeProse(part, numberOf)))
    .join("");
}

/** The search the turn is on, drawn in place of the words that have not started yet. */
export type LiveWebSearch = {
  query: string;
  /** The pages the search has reported so far. */
  sources: DeskWebSource[];
};

/**
 * The web search the turn is on: its latest step is a search and nothing has
 * been said or done after it. It stays while the agent weighs what came back
 * and goes the moment it takes another step or starts to write.
 */
export function liveWebSearch(turn: TurnState | null): LiveWebSearch | null {
  if (!turn || (turn.status !== "streaming" && turn.status !== "working")) {
    return null;
  }
  const latest = turn.segments.filter((segment) => segment.kind !== "reasoning").at(-1);
  if (
    latest?.kind !== "tool" ||
    latest.name !== WEB_SEARCH_TOOL ||
    latest.status === "failed" ||
    latest.status === "proposed"
  ) {
    return null;
  }
  const query = latest.arguments.query;

  return {
    query: typeof query === "string" ? query.trim() : "",
    sources: deskWebSources([latest]),
  };
}

/** The pages the turn being written has found so far. */
export function turnWebSources(turn: TurnState | null): DeskWebSource[] {
  if (!turn) {
    return [];
  }

  return deskWebSources(turn.segments.filter((segment) => segment.kind === "tool"));
}
