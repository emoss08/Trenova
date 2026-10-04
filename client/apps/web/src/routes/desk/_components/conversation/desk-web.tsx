import { MarkdownLinkContext, type MarkdownLinkRenderer } from "@/components/elements/ai-markdown";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { DeskIcon } from "../desk-icons";
import { placePopover } from "./desk-citations";
import { citedSources, citeLabel, siteHue, webCiteIds, type DeskWebSource } from "./web-cites";

/**
 * A site's tile: its first letter on a colour drawn from its name, so the
 * same site wears the same colour in a citation, a live result and the list
 * under the reply. Nothing is fetched from the site.
 */
export function DeskFavicon({ site, size = 14 }: { site: string; size?: number }) {
  return (
    <span
      className="dk-wf"
      aria-hidden
      style={
        {
          width: size,
          height: size,
          fontSize: size * 0.6,
          "--dk-wh": siteHue(site),
        } as CSSProperties
      }
    >
      {site.charAt(0).toUpperCase()}
    </span>
  );
}

/**
 * A citation in the reply: the first cited site as a pill, with a count of
 * the others. Resting on it lists every cited page; the pill opens the first,
 * an entry opens its own. A citation whose pages are all unknown draws
 * nothing.
 */
export function DeskWebCite({
  ids,
  sources,
  live = false,
}: {
  ids: readonly number[];
  sources: readonly DeskWebSource[];
  /** Drawn while the reply is still being written, so it arrives as artifact badges do. */
  live?: boolean;
}) {
  const t = useT();
  const [place, setPlace] = useState<{ below: boolean; shift: number } | null>(null);
  const anchorRef = useRef<HTMLSpanElement>(null);
  const list = citedSources(ids, sources);
  if (list.length === 0) {
    return <></>;
  }
  const first = list[0];
  const show = () => {
    if (anchorRef.current) setPlace(placePopover(anchorRef.current, CITE_POPOVER_WIDTH));
  };
  const hide = () => setPlace(null);

  return (
    <span
      ref={anchorRef}
      className="dk-wc-w"
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      <a
        className={cn("dk-wc", live && "dk-w")}
        href={first.url}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={
          list.length > 1
            ? t("{0} and {1} more sources, opens in a new tab", first.site, list.length - 1)
            : t("{0} on {1}, opens in a new tab", first.title, first.site)
        }
      >
        {citeLabel(first.site)}
        {list.length > 1 && <em>+{list.length - 1}</em>}
      </a>
      {place && (
        <span
          className={cn("dk-wc-pop", place.below && "dk-below")}
          role="tooltip"
          style={place.shift ? { left: `calc(50% + ${place.shift}px)` } : undefined}
        >
          <span className="dk-wc-in">
            {list.map((source) => (
              <a
                key={source.url}
                className="dk-wc-it"
                href={source.url}
                target="_blank"
                rel="noopener noreferrer"
                tabIndex={-1}
              >
                <span className="dk-wc-d">
                  <DeskFavicon site={source.site} size={12} />
                  {source.site}
                  <i>{source.age}</i>
                </span>
                <b>{source.title}</b>
                {source.snippet !== "" && <span className="dk-wc-s">{source.snippet}</span>}
              </a>
            ))}
          </span>
        </span>
      )}
    </span>
  );
}

const CITE_POPOVER_WIDTH = 280;

/** Lets the reply's citation links draw as citation pills over the reply's pages. */
export function DeskWebCites({
  sources,
  live = false,
  children,
}: {
  sources: readonly DeskWebSource[];
  live?: boolean;
  children: ReactNode;
}) {
  const renderLink = useCallback<MarkdownLinkRenderer>(
    (href) => {
      const ids = webCiteIds(href);
      return ids ? <DeskWebCite ids={ids} sources={sources} live={live} /> : null;
    },
    [live, sources],
  );

  if (sources.length === 0) {
    return children;
  }

  return <MarkdownLinkContext value={renderLink}>{children}</MarkdownLinkContext>;
}

/**
 * The search while it runs: what the agent is doing, the query it sent, and
 * a chip for each page the search reports, each arriving as it is reported.
 */
export function DeskWebLive({
  label,
  query,
  sources,
}: {
  label: string;
  query: string;
  sources: readonly DeskWebSource[];
}) {
  return (
    <div className="dk-wl" role="status">
      <div className="dk-wl-h">
        <DeskIcon name="globe" size={13} />
        <span className="dk-wl-l">{label}</span>
      </div>
      {query !== "" && <div className="dk-wl-q">{query}</div>}
      {sources.length > 0 && (
        <div className="dk-wl-s">
          {sources.map((source) => (
            <span key={source.url} className="dk-wl-it">
              <DeskFavicon site={source.site} size={12} />
              {source.site}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

/**
 * Every page the reply found, under it once it is done: a stack of the first
 * sites and a count, opening to the query and the numbered pages.
 */
export function DeskWebSources({
  sources,
  query,
}: {
  sources: readonly DeskWebSource[];
  query: string;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  if (sources.length === 0) {
    return null;
  }

  return (
    <div className={cn("dk-ws", open && "dk-open")}>
      <button
        type="button"
        className="dk-ws-b"
        aria-expanded={open}
        onClick={() => setOpen((was) => !was)}
      >
        <span className="dk-ws-st">
          {sources.slice(0, 4).map((source) => (
            <DeskFavicon key={source.url} site={source.site} size={14} />
          ))}
        </span>
        <span>{t("{0, plural, one {# source} other {# sources}}", sources.length)}</span>
        <DeskIcon name="chevR" size={10} />
      </button>
      <div className="dk-ws-x" inert={!open}>
        <div>
          {query !== "" && (
            <div className="dk-ws-q">
              <DeskIcon name="search" size={11} />
              {query}
            </div>
          )}
          <ol className="dk-ws-l">
            {sources.map((source, index) => (
              <li key={source.url}>
                <a href={source.url} target="_blank" rel="noopener noreferrer">
                  <span className="dk-ws-n">{index + 1}</span>
                  <DeskFavicon site={source.site} size={14} />
                  <span className="dk-ws-t">{source.title}</span>
                  <span className="dk-ws-d">
                    {source.age !== "" ? `${source.site} · ${source.age}` : source.site}
                  </span>
                </a>
              </li>
            ))}
          </ol>
        </div>
      </div>
    </div>
  );
}
