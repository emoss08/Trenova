import type { BriefingSection } from "@/lib/graphql/briefing";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { CSSProperties } from "react";
import { Link } from "react-router";

/** How many of the morning's figures the strip shows, as the prototype draws it; the briefing holds up to eight. */
const CELLS = 6;

/**
 * The day in figures, the prototype's `.strip`: one joined strip under the
 * box. Each cell is a section of this morning's briefing, what it counts, its
 * first figure and what that figure is, and opens the page that shows it. The
 * numbers are the briefing's own, computed before any wording, so the strip
 * reads the same with Nova away.
 */
export function DayStrip({ sections }: { sections: readonly BriefingSection[] }) {
  const t = useT();
  const cells = sections.filter((section) => section.items.length > 0).slice(0, CELLS);
  if (cells.length === 0) {
    return null;
  }

  return (
    <nav
      className="dk-dy-strip"
      style={{ "--dk-dy-cells": cells.length } as CSSProperties}
      aria-label={t("The day")}
    >
      {cells.map((section) => {
        const item = section.items[0];
        const to = item.path || section.path || null;
        const warn = section.key === "Exceptions";
        const body = (
          <>
            <span className="dk-dy-strip-l">{section.title}</span>
            <span className="dk-dy-strip-v">
              {warn && <span className="dk-dy-dot dk-warn" />}
              {item.value}
            </span>
            <span className={cn("dk-dy-strip-d", warn && "dk-warn")}>{item.label}</span>
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
