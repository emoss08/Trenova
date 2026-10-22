import { resolveAgentIdentity } from "@/components/agent-identity/agent-identity";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import type { TableReader } from "../../artifact-payloads";
import { ArtIcon } from "../../desk-art-kinds";
import { PAGE_SIZES, railPages, type PageSize } from "./table-pages";
import type { TablePages } from "./use-table-pages";

/**
 * The foot of a table that pages: which rows show of how many, the strip of
 * pages with the agent's and the ones read marked, the way through, and how
 * many rows a page holds. The agent's pages wear its accent.
 */
export function TablePager({ pages, reader }: { pages: TablePages; reader: TableReader | null }) {
  const t = useT();
  const accent = reader ? resolveAgentIdentity(reader).accent : "slate";
  const start = pages.page * pages.size;
  const expected =
    pages.total !== null ? Math.min(pages.size, Math.max(0, pages.total - start)) : pages.size;
  const shown = pages.rows.length > 0 ? pages.rows.length : expected;
  const from = shown === 0 ? 0 : start + 1;
  const to = start + shown;
  const totalText = pages.total !== null ? pages.total : pages.shownRows;
  const lockedFrom = pages.locked ? pages.lastPage + 1 : null;
  const rail = railPages({
    pages: pages.pages,
    current: pages.page,
    size: pages.size,
    agentRows: pages.agentRows,
    readRows: pages.shownRows,
    lockedFrom,
  });

  return (
    <div className="dk-tv-pager">
      <span className="dk-tv-range">
        {pages.exact
          ? t("{0}–{1} of {2}", from, to, totalText)
          : t("{0}–{1} of {2}+", from, to, totalText)}
      </span>
      {rail.length > 1 && (
        <ol
          className={cn("dk-tv-rail", accent === "slate" && "dk-slate")}
          style={{ "--dk-ah": `var(--hue-${accent})` } as CSSProperties}
          aria-label={t("Pages")}
        >
          {rail.map((segment) => (
            <li
              key={segment.index}
              className={cn(
                segment.current && "dk-on",
                segment.agent && "dk-agent",
                segment.read && "dk-seen",
                segment.locked && "dk-locked",
              )}
            >
              <button
                type="button"
                disabled={segment.locked}
                aria-current={segment.current ? "page" : undefined}
                aria-label={
                  !segment.agent
                    ? t("Page {0}", segment.index + 1)
                    : reader
                      ? t("Page {0}, read by {1}", segment.index + 1, reader.name)
                      : t("Page {0}, read in this conversation", segment.index + 1)
                }
                onClick={() => pages.setPage(segment.index)}
              >
                <i />
              </button>
            </li>
          ))}
        </ol>
      )}
      <span className="dk-tv-pgs">
        <button
          type="button"
          className="dk-tv-pb"
          disabled={pages.page === 0}
          aria-label={t("First page")}
          onClick={() => pages.setPage(0)}
        >
          <ArtIcon name="first" size={13} stroke={2.2} />
        </button>
        <button
          type="button"
          className="dk-tv-pb"
          disabled={pages.page === 0}
          aria-label={t("Previous page")}
          onClick={() => pages.setPage(pages.page - 1)}
        >
          <ArtIcon name="left" size={13} stroke={2.2} />
        </button>
        <span className="dk-tv-pn" aria-live="polite">
          {pages.exact
            ? t("Page {0} of {1}", pages.page + 1, pages.pages)
            : t("Page {0} of {1}+", pages.page + 1, pages.pages)}
        </span>
        <button
          type="button"
          className="dk-tv-pb"
          disabled={pages.page >= pages.lastPage}
          aria-label={t("Next page")}
          onClick={() => pages.setPage(pages.page + 1)}
        >
          <ArtIcon name="right" size={13} stroke={2.2} />
        </button>
      </span>
      <span className="dk-tv-size" role="group" aria-label={t("Rows a page")}>
        {PAGE_SIZES.map((size: PageSize) => (
          <button
            key={size}
            type="button"
            aria-pressed={pages.size === size}
            className={cn(pages.size === size && "dk-on")}
            onClick={() => pages.setSize(size)}
          >
            {size}
          </button>
        ))}
      </span>
    </div>
  );
}
