import { tableFilterSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  activeRosterView,
  rosterViews,
  type RosterViewId,
} from "@trenova/shared/lib/worker-roster-views";
import { useQueryStates } from "nuqs";
import { useMemo } from "react";

/**
 * One-click answers to the questions HR opens the roster for. Each chip writes
 * the table's own filter state, so a view is shareable as a URL and the filter
 * menu still shows exactly what is applied.
 */
export function RosterViews() {
  const [params, setParams] = useQueryStates(tableFilterSearchParamsParser);

  // Rendering never needs the clock: the chips show labels, and matching the
  // active view compares fields and operators rather than values. The expiry
  // cutoff is read when somebody clicks, from the start of today rather than
  // the current second, so the same view gives the same answer all day.
  const views = useMemo(() => rosterViews(0), []);
  const active = activeRosterView(params.fieldFilters ?? [], views);

  const apply = (id: RosterViewId) => {
    const fresh = rosterViews(getTodayDate()).find((view) => view.id === id);
    if (!fresh) return;
    void setParams({ fieldFilters: fresh.filters });
  };

  return (
    <div className="flex flex-wrap items-center gap-1.5" data-testid="roster-views">
      {views.map((view) => (
        <Tooltip key={view.id}>
          <TooltipTrigger
            render={
              <Button
                type="button"
                size="sm"
                variant={active === view.id ? "default" : "outline"}
                aria-pressed={active === view.id}
                data-testid={`roster-view-${view.id}`}
                className="h-7 rounded-full px-3 text-xs"
                onClick={() => apply(view.id)}
              >
                {view.label}
              </Button>
            }
          />
          <TooltipContent>{view.description}</TooltipContent>
        </Tooltip>
      ))}
    </div>
  );
}
