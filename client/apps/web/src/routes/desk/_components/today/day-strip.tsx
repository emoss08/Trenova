import type { BriefingSection } from "@/lib/graphql/briefing";
import { cn } from "@trenova/shared/lib/utils";
import { useT } from "@trenova/shared/i18n/use-t";
import { Link } from "react-router";

/** How many of the morning's figures the strip shows; the briefing holds up to eight. */
const CELLS = 6;

/**
 * The day in figures, one joined strip under the box: each cell is a section
 * of this morning's briefing — what it counts, its first figure and what that
 * figure is — and opens the page that shows it. The numbers are the
 * briefing's own, computed before any wording, so the strip reads the same
 * with Nova away.
 */
export function DeskDayStrip({ sections }: { sections: readonly BriefingSection[] }) {
  const t = useT();
  const cells = sections.filter((section) => section.items.length > 0).slice(0, CELLS);
  if (cells.length === 0) {
    return null;
  }

  return (
    <nav
      className="dk-w-strip"
      style={{ gridTemplateColumns: `repeat(${cells.length}, minmax(0, 1fr))` }}
      aria-label={t("The day")}
    >
      {cells.map((section) => {
        const item = section.items[0];
        const to = item.path || section.path || null;
        const warn = section.key === "Exceptions";
        const body = (
          <>
            <span className="dk-w-strip-l">{section.title}</span>
            <span className="dk-w-strip-v">
              {warn && <span className="dk-w-dot dk-warn" />}
              {item.value}
            </span>
            <span className={cn("dk-w-strip-d", warn && "dk-warn")}>{item.label}</span>
          </>
        );
        return to ? (
          <Link key={section.key} to={to} className="ui-focus-ring" title={section.read}>
            {body}
          </Link>
        ) : (
          <div key={section.key} title={section.read}>
            {body}
          </div>
        );
      })}
    </nav>
  );
}
