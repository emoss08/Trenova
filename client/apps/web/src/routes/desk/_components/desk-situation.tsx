import type { BriefingSection } from "@/lib/graphql/briefing";
import { briefing as briefingQueries } from "@/lib/queries/briefing";
import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useNavigate } from "react-router";

/** How many of the morning's figures the line carries before the live ones. */
const BRIEFING_CELLS = 3;

type Cell = {
  key: string;
  label: string;
  value: string;
  to: string;
  tone?: "warn";
  narrow?: boolean;
};

function briefingCells(sections: readonly BriefingSection[]): Cell[] {
  return sections
    .filter((section) => section.items.length > 0)
    .slice(0, BRIEFING_CELLS)
    .map((section, index) => {
      const item = section.items[0];
      return {
        key: section.key,
        label: item.label || section.title,
        value: item.value,
        to: item.path || section.path || "/desk",
        tone: section.key === "Exceptions" ? "warn" : undefined,
        narrow: index > 0,
      };
    });
}

/**
 * The operation now, in one joined strip across the top of the Desk: the
 * morning's first figures, what waits on people and what agents have in hand.
 * Each cell opens the place that shows it. It reads only what the person may
 * read, and draws nothing until the figures are in.
 */
export function DeskSituation({ canWatch, needsYou }: { canWatch: boolean; needsYou: number }) {
  const t = useT();
  const navigate = useNavigate();
  const briefingQuery = useQuery({ ...briefingQueries.today(), retry: false });
  const countsQuery = useQuery({ ...watchtower.counts(), enabled: canWatch });
  const cells = briefingCells(briefingQuery.data?.sections ?? []);
  const withAgents = countsQuery.data?.withAgents ?? 0;

  if (cells.length === 0 && !canWatch) {
    return null;
  }

  return (
    <div className="dk-w-sit" role="group" aria-label={t("The operation now")}>
      {cells.map((cell) => (
        <button
          key={cell.key}
          type="button"
          className={cn("ui-focus-ring", cell.narrow && "dk-w-sit-md")}
          onClick={() => void navigate(cell.to)}
        >
          {cell.tone && <span className="dk-w-dot dk-warn" />}
          <span className="dk-w-sit-l">{cell.label}</span>
          <span className="dk-w-sit-v">{cell.value}</span>
        </button>
      ))}
      {canWatch && (
        <>
          <button type="button" className="ui-focus-ring" onClick={() => void navigate("/desk/needs")}>
            <span className="dk-w-sit-l">{t("Needs you")}</span>
            <span className="dk-w-sit-v">{needsYou}</span>
          </button>
          <button
            type="button"
            className="ui-focus-ring dk-w-sit-md"
            onClick={() => void navigate("/desk#dk-w-working")}
          >
            <NovaBadge size="xs" spin={withAgents > 0 ? "always" : "none"} />
            <span className="dk-w-sit-l">{t("Working")}</span>
            <span className="dk-w-sit-v">{withAgents}</span>
          </button>
        </>
      )}
    </div>
  );
}
