import { useT } from "@trenova/shared/i18n/use-t";
import { BillingListEmpty } from "@/components/billing/billing-empty";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import type { OpenStatement } from "@trenova/shared/types/statement";
import { ArrowDownWideNarrowIcon, SearchIcon } from "lucide-react";
import { useDeferredValue, useMemo, useState } from "react";
import { StatementCard } from "./statement-card";

const SORTS = [
  { key: "soonest", label: "Bills soonest" },
  { key: "value", label: "Largest value" },
  { key: "name", label: "Customer name" },
] as const;

type SortKey = (typeof SORTS)[number]["key"];

/**
 * Statements in the order a biller wants them.
 *
 * "Bills soonest" is the default because the work is deadline-shaped: the
 * statement closing tonight is the one worth looking at, not the largest one.
 * Customers with nothing accrued sink below those with freight either way —
 * an empty statement is a fact, not a task.
 */
export function sortStatements(
  statements: readonly OpenStatement[],
  sort: SortKey,
): OpenStatement[] {
  const ranked = [...statements];
  ranked.sort((a, b) => {
    const aEmpty = a.shipmentCount === 0 ? 1 : 0;
    const bEmpty = b.shipmentCount === 0 ? 1 : 0;
    if (aEmpty !== bEmpty) return aEmpty - bEmpty;

    switch (sort) {
      case "value":
        return Number(b.totalAmount ?? 0) - Number(a.totalAmount ?? 0);
      case "name":
        return a.customerName.localeCompare(b.customerName);
      default:
        return a.periodEnd - b.periodEnd;
    }
  });

  return ranked;
}

export function filterStatements(
  statements: readonly OpenStatement[],
  search: string,
): OpenStatement[] {
  const needle = search.trim().toLowerCase();
  if (!needle) return [...statements];

  return statements.filter(
    (statement) =>
      statement.customerName.toLowerCase().includes(needle) ||
      (statement.customerCode ?? "").toLowerCase().includes(needle),
  );
}

export function StatementSidebar({
  statements,
  loading,
  nowSeconds,
  selectedCustomerId,
  onSelect,
}: {
  statements: readonly OpenStatement[];
  loading: boolean;
  nowSeconds: number;
  selectedCustomerId: string | null;
  onSelect: (customerId: string) => void;
}) {
  const t = useT();

  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<SortKey>("soonest");
  const deferredSearch = useDeferredValue(search);

  const visible = useMemo(
    () => sortStatements(filterStatements(statements, deferredSearch), sort),
    [statements, deferredSearch, sort],
  );

  const searching = deferredSearch.trim().length > 0;

  return (
    <div className="flex h-full flex-col">
      <div className="flex flex-col gap-1.5 border-b p-2">
        <Input
          placeholder={t("Search customer...")}
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          className="h-7 flex-1 text-xs"
          inputContainerClassName="w-full"
        />
        <div className="flex items-center gap-1">
          <ArrowDownWideNarrowIcon className="text-muted-foreground size-3" />
          {SORTS.map((option) => (
            <button
              key={option.key}
              type="button"
              onClick={() => setSort(option.key)}
              aria-pressed={sort === option.key}
              className={cn(
                "rounded px-1.5 py-0.5 text-[11px] transition-colors",
                sort === option.key
                  ? "bg-muted text-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {t(option.label)}
            </button>
          ))}
        </div>
      </div>

      <ScrollArea className="flex-1">
        <div className="flex flex-col gap-1.5 p-2">
          {loading && (
            <>
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
            </>
          )}

          {!loading && visible.length === 0 && (
            <BillingListEmpty
              title={searching ? "Nothing matches" : "No statement customers"}
              description={
                searching
                  ? "No customer on a statement schedule matches that search. Clear it to see everyone accruing."
                  : "A customer appears here once their billing profile is set to a consolidated invoice on a recurring cycle. Until then every shipment bills on its own."
              }
              onClearFilters={searching ? () => setSearch("") : undefined}
            />
          )}

          {!loading &&
            visible.map((statement) => (
              <StatementCard
                key={statement.customerId}
                statement={statement}
                nowSeconds={nowSeconds}
                isSelected={statement.customerId === selectedCustomerId}
                onClick={() => onSelect(statement.customerId)}
              />
            ))}
        </div>
      </ScrollArea>

      {!loading && visible.length > 0 && (
        <p className="text-muted-foreground border-t px-3 py-1.5 text-[11px]">
          {t("{0} of {1, plural, one {# statement} other {# statements}} {2}", visible.length, statements.length, selectedCustomerId ? "" : t("· pick one to see what it will bill"))}
        </p>
      )}
    </div>
  );
}
