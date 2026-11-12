import type { BriefingSection } from "@/lib/graphql/briefing";
import { briefing as briefingQueries } from "@/lib/queries/briefing";
import { usePermission } from "@/hooks/use-permission";
import { aiControl } from "@/lib/queries/ai-control";
import { watchtower } from "@/lib/queries/watchtower";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useNavigate } from "react-router";

/** How many of the morning's figures the line carries before the live ones. */
const BRIEFING_CELLS = 3;

type Cell = { key: string; label: string; value: string; to: string; warn: boolean; md: boolean };

function briefingCells(sections: readonly BriefingSection[]): Cell[] {
  return sections
    .filter((section) => section.items.length > 0)
    .slice(0, BRIEFING_CELLS)
    .map((section, index) => ({
      key: section.key,
      label: section.title,
      value: section.items[0].value,
      to: section.items[0].path || section.path || "/desk",
      warn: section.key === "Exceptions",
      md: index > 0,
    }));
}

/**
 * The operation now, in one joined strip (the prototype's `.sit`): the
 * morning's first figures, what waits on people and what agents have in hand.
 * Each cell opens the place that shows it; the quieter cells give way first
 * as the room narrows.
 */
export function DeskSituation({ canWatch, needsYou }: { canWatch: boolean; needsYou: number }) {
  const t = useT();
  const navigate = useNavigate();
  const briefingQuery = useQuery({ ...briefingQueries.today(), retry: false });
  const countsQuery = useQuery({ ...watchtower.counts(), enabled: canWatch });
  const cells = briefingCells(briefingQuery.data?.sections ?? []);
  // How many runs are working, the number AI control shows: the runs
  // themselves for whoever may read them, else the items a run is working.
  const { allowed: canSeeRuns } = usePermission(Resource.AgentRun, Operation.Read);
  const runsQuery = useQuery({ ...aiControl.workingRuns(), enabled: canWatch && canSeeRuns });
  const working = canSeeRuns ? (runsQuery.data?.length ?? 0) : (countsQuery.data?.withAgents ?? 0);

  if (cells.length === 0 && !canWatch) {
    return null;
  }

  return (
    <div className="dk-tb-sit" role="group" aria-label={t("The operation now")}>
      {cells.map((cell) => (
        <button
          key={cell.key}
          type="button"
          className={cn("ui-focus-ring", cell.md && "dk-tb-md")}
          onClick={() => void navigate(cell.to)}
        >
          {cell.warn && <span className="dk-tb-dot dk-warn" />}
          <span className="dk-tb-sit-l">{cell.label}</span>
          <span className="dk-tb-sit-v">{cell.value}</span>
        </button>
      ))}
      {canWatch && (
        <>
          <button
            type="button"
            className="ui-focus-ring"
            onClick={() => void navigate("/desk/needs")}
          >
            <span className="dk-tb-sit-l">{t("Needs you")}</span>
            <span className="dk-tb-sit-v">{needsYou}</span>
          </button>
          <button
            type="button"
            className="dk-tb-md ui-focus-ring"
            onClick={() => void navigate("/desk#dk-dy-working")}
          >
            {/* Agents are the organization's own; Nova's mark is kept for Nova. */}
            <span
              aria-hidden
              className={cn(
                "size-1.5 flex-none rounded-full",
                working > 0 ? "animate-pulse bg-dsk-agent" : "bg-dsk-b-strong",
              )}
            />
            <span className="dk-tb-sit-l">{t("Agents working")}</span>
            <span className="dk-tb-sit-v">{working}</span>
          </button>
        </>
      )}
    </div>
  );
}
