import type { DeskAttentionKind } from "@/lib/graphql/desk-attention";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DeskListFilterLabel, deskListFilterClass } from "../list/desk-list-head";
import { SEVERITIES, severityLabel, severityMarkClass } from "../watchtower/severity";
import { NO_FILTERS, filterCount, toggled, type NeedsFilters } from "./needs-filters";

// i18n-ignore: class names, never shown to a person
const MARK_SLOT = "flex size-4 flex-none items-center justify-center";

/**
 * What Needs you can be narrowed to, behind one button: how loudly a row
 * asks, what has not been opened, what an agent is on, and what kind of
 * thing it is. A choice narrows the list as it is made and the menu stays
 * open, so several are picked in one visit; each also shows as a token under
 * the search, where it is taken off again. Kinds sit one level in, because
 * there can be twenty of them and the rest should stay a glance.
 */
export function NeedsFilterMenu({
  filters,
  kinds,
  open,
  onOpenChange,
  onChange,
}: {
  filters: NeedsFilters;
  /** The kinds on offer under the lens, each with how much of it is open. */
  kinds: readonly DeskAttentionKind[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onChange: (filters: NeedsFilters) => void;
}) {
  const t = useT();
  const count = filterCount(filters);

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
      <DropdownMenuContent align="end" sideOffset={6} className="w-60">
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("Severity")}</DropdownMenuLabel>
          {SEVERITIES.map((severity) => (
            <DropdownMenuCheckboxItem
              key={severity}
              checked={filters.severities.includes(severity)}
              closeOnClick={false}
              onCheckedChange={() =>
                onChange({ ...filters, severities: toggled(filters.severities, severity) })
              }
            >
              <span className={MARK_SLOT}>
                <i className={cn("size-2 rounded-[2px]", severityMarkClass(severity))} />
              </span>
              {severityLabel(t, severity)}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t("Show only")}</DropdownMenuLabel>
          <DropdownMenuCheckboxItem
            checked={filters.unseen}
            closeOnClick={false}
            onCheckedChange={() => onChange({ ...filters, unseen: !filters.unseen })}
          >
            <span className={MARK_SLOT}>
              <i className="size-1.5 rounded-full bg-dsk-blue" />
            </span>
            {t("Not opened yet")}
          </DropdownMenuCheckboxItem>
          <DropdownMenuCheckboxItem
            checked={filters.working}
            closeOnClick={false}
            onCheckedChange={() => onChange({ ...filters, working: !filters.working })}
          >
            <span className={MARK_SLOT}>
              <i className="size-1.5 rounded-full bg-dsk-agent" />
            </span>
            {t("An agent is working")}
          </DropdownMenuCheckboxItem>
        </DropdownMenuGroup>
        {kinds.length > 0 && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <span className="flex-1">{t("Kind")}</span>
                {filters.kinds.length > 0 && (
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {filters.kinds.length}
                  </span>
                )}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent className="w-64" listClassName="max-h-80">
                {kinds.map((entry) => (
                  <DropdownMenuCheckboxItem
                    key={entry.kind}
                    checked={filters.kinds.includes(entry.kind)}
                    closeOnClick={false}
                    onCheckedChange={() =>
                      onChange({ ...filters, kinds: toggled(filters.kinds, entry.kind) })
                    }
                  >
                    <span className="min-w-0 flex-1 truncate">{entry.label}</span>
                    <span className="flex-none text-xs text-muted-foreground tabular-nums">
                      {entry.count}
                    </span>
                  </DropdownMenuCheckboxItem>
                ))}
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </>
        )}
        {count > 0 && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              title={t("Clear filters")}
              onClick={() => onChange({ ...NO_FILTERS, search: filters.search })}
            />
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
