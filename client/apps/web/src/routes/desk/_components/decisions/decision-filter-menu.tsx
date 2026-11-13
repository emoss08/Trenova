import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { DeskListFilterLabel, deskListFilterClass } from "../list/desk-list-head";
import {
  NO_DECISION_FILTERS,
  changeLabel,
  decisionFilterCount,
  type DecisionFacets,
  type DecisionFilters,
} from "./decision-filters";

// i18n-ignore: class names, never shown to a person
const COUNT_CLASS = "flex-none text-xs text-muted-foreground tabular-nums";

/**
 * What the Decisions lens can be narrowed to: the agent that asks, and the
 * kind of change it asks for. One of each at most, as the queue reads them;
 * picking the chosen one again takes it off.
 */
export function DecisionFilterMenu({
  filters,
  facets,
  open,
  onOpenChange,
  onChange,
}: {
  filters: DecisionFilters;
  facets: DecisionFacets;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onChange: (filters: DecisionFilters) => void;
}) {
  const t = useT();
  const count = decisionFilterCount(filters);

  return (
    <DropdownMenu open={open} onOpenChange={onOpenChange}>
      <DropdownMenuTrigger
        render={<button type="button" className={deskListFilterClass(count > 0)} />}
        title={t("Press {0}", "F")}
        aria-label={
          count > 0 ? t("{0, plural, one {Filter, # on} other {Filter, # on}}", count) : t("Filter")
        }
      >
        <DeskListFilterLabel count={count} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="w-64" listClassName="max-h-96">
        {facets.agents.length > 0 && (
          <DropdownMenuGroup>
            <DropdownMenuLabel>{t("Agent")}</DropdownMenuLabel>
            {facets.agents.map((agent) => (
              <DropdownMenuCheckboxItem
                key={agent.id}
                checked={filters.agentId === agent.id}
                closeOnClick={false}
                onCheckedChange={() =>
                  onChange({ ...filters, agentId: filters.agentId === agent.id ? "" : agent.id })
                }
              >
                <span className="min-w-0 flex-1 truncate">{agent.name}</span>
                <span className={COUNT_CLASS}>{agent.count}</span>
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuGroup>
        )}
        {facets.agents.length > 0 && facets.tools.length > 0 && <DropdownMenuSeparator />}
        {facets.tools.length > 0 && (
          <DropdownMenuGroup>
            <DropdownMenuLabel>{t("Kind of change")}</DropdownMenuLabel>
            {facets.tools.map((entry) => (
              <DropdownMenuCheckboxItem
                key={entry.tool}
                checked={filters.tool === entry.tool}
                closeOnClick={false}
                onCheckedChange={() =>
                  onChange({ ...filters, tool: filters.tool === entry.tool ? "" : entry.tool })
                }
              >
                <span className="min-w-0 flex-1 truncate">{changeLabel(t, entry.tool)}</span>
                <span className={COUNT_CLASS}>{entry.count}</span>
              </DropdownMenuCheckboxItem>
            ))}
          </DropdownMenuGroup>
        )}
        {facets.agents.length === 0 && facets.tools.length === 0 && (
          <p className="px-1.5 py-1 text-sm text-muted-foreground">
            {t("Nothing waits to filter.")}
          </p>
        )}
        {count > 0 && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              title={t("Clear filters")}
              onClick={() => onChange({ ...NO_DECISION_FILTERS, search: filters.search })}
            />
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
