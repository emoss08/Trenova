import { DisplayValue } from "@/components/assistant/display-value";
import {
  isDetailType,
  isFigureType,
  type DisplayColumn,
} from "@/components/assistant/readable-values";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowDownIcon, ArrowUpIcon, ChevronRightIcon } from "lucide-react";
import { Fragment, useMemo, useState, type MouseEvent } from "react";
import { Link, useNavigate } from "react-router";
import { sortTableRows, type TableSort, type TableViewRow } from "./artifact-payloads";

/** The next sort a header click asks for: descending, ascending, then as found. */
function nextSort(current: TableSort | null, key: string): TableSort | null {
  if (current?.key !== key) return { key, direction: "desc" };
  if (current.direction === "desc") return { key, direction: "asc" };
  return null;
}

function hasDetail(row: TableViewRow, detail: readonly DisplayColumn[]): boolean {
  return detail.some((column) => row.values[column.key] !== undefined);
}

/**
 * A list result as a table that reads like the rest of the product.
 *
 * The cells hold what can be scanned down a column — names, statuses, dates,
 * amounts — in compact rows on the row tokens. Prose and measurements would
 * turn a row into a paragraph, so they wait in the row's detail, a click
 * away. The first column stays put while the rest scroll sideways, so a
 * narrow pane never loses which record a value belongs to. A row whose
 * record has a page opens it; its name is the link.
 */
export function ArtifactTable({
  columns,
  rows,
}: {
  columns: readonly DisplayColumn[];
  rows: readonly TableViewRow[];
}) {
  const t = useT();
  const navigate = useNavigate();
  const [sort, setSort] = useState<TableSort | null>(null);
  // The body rises into its new order when a person sorts it, and only then:
  // a table arriving already moves in with the pane.
  const [sortTouched, setSortTouched] = useState(false);
  const [expanded, setExpanded] = useState<ReadonlySet<string>>(() => new Set());

  const { cells, detail } = useMemo(() => {
    const inCells = columns.filter((column) => !isDetailType(column.type));
    // A result that is all prose still needs a first column to hang the
    // detail from.
    return inCells.length > 0
      ? { cells: inCells, detail: columns.filter((column) => isDetailType(column.type)) }
      : { cells: columns.slice(0, 1), detail: columns.slice(1) };
  }, [columns]);
  const sorted = useMemo(() => sortTableRows(rows, cells, sort), [rows, cells, sort]);
  const expandable = useMemo(
    () => detail.length > 0 && rows.some((row) => hasDetail(row, detail)),
    [detail, rows],
  );

  const toggle = (key: string) =>
    setExpanded((current) => {
      const next = new Set(current);
      if (!next.delete(key)) {
        next.add(key);
      }
      return next;
    });

  // A click anywhere on a row does what its first cell offers: opens the
  // record, or its detail. Links and buttons inside it keep their own click.
  const onRowClick = (event: MouseEvent<HTMLTableRowElement>, row: TableViewRow) => {
    if ((event.target as HTMLElement).closest("a, button")) {
      return;
    }
    if (row.path !== "") {
      void navigate(row.path);
    } else if (hasDetail(row, detail)) {
      toggle(row.key);
    }
  };

  return (
    <div className="scrollbar-overlay @container min-h-0 flex-1 overflow-auto">
      <table className="w-full caption-bottom text-xs [--row-h:var(--row-h-compact)]">
        <TableHeader className="sticky top-0 z-20">
          <TableRow className="hover:bg-transparent">
            {cells.map((column, index) => {
              const active = sort?.key === column.key;
              const figure = isFigureType(column.type);
              return (
                <TableHead
                  key={column.key}
                  aria-sort={
                    active ? (sort.direction === "asc" ? "ascending" : "descending") : undefined
                  }
                  className={cn(
                    figure && "text-right",
                    index === 0 && "border-border-subtle sticky left-0 z-10 border-r",
                  )}
                >
                  <button
                    type="button"
                    onClick={() => {
                      setSortTouched(true);
                      setSort((current) => nextSort(current, column.key));
                    }}
                    className={cn(
                      "ui-focus-ring inline-flex max-w-56 items-center gap-1 rounded-control transition-colors",
                      "hover:text-foreground",
                      figure && "flex-row-reverse",
                      active && "text-foreground",
                      index === 0 && expandable && "pl-5",
                    )}
                  >
                    <span className="truncate">{column.label}</span>
                    {active && (
                      <span key={sort.direction} className="animate-confirm flex shrink-0">
                        {sort.direction === "asc" ? (
                          <ArrowUpIcon aria-hidden className="size-3" />
                        ) : (
                          <ArrowDownIcon aria-hidden className="size-3" />
                        )}
                      </span>
                    )}
                  </button>
                </TableHead>
              );
            })}
          </TableRow>
        </TableHeader>
        <TableBody
          key={sort ? `${sort.key}:${sort.direction}` : "found"}
          className={cn(sortTouched && "animate-rise")}
        >
          {sorted.map((row) => {
            const detailed = hasDetail(row, detail);
            const open = detailed && expanded.has(row.key);
            const actionable = row.path !== "" || detailed;
            return (
              <Fragment key={row.key}>
                <TableRow
                  onClick={actionable ? (event) => onRowClick(event, row) : undefined}
                  data-expanded={open || undefined}
                  className={cn(
                    "group/row border-border-subtle data-expanded:bg-surface-hover",
                    actionable && "cursor-pointer",
                    open && "border-b-0",
                  )}
                >
                  {cells.map((column, index) => {
                    const figure = isFigureType(column.type);
                    const value = row.values[column.key];
                    return (
                      <TableCell
                        key={column.key}
                        className={cn(
                          "max-w-64",
                          figure && "text-right font-mono tabular-nums",
                          index === 0 &&
                            "bg-card group-hover/row:bg-surface-hover group-data-expanded/row:bg-surface-hover border-border-subtle sticky left-0 z-10 border-r transition-colors",
                        )}
                      >
                        {index === 0 ? (
                          <span className="flex min-w-0 items-center gap-1">
                            {expandable &&
                              (detailed ? (
                                <button
                                  type="button"
                                  aria-expanded={open}
                                  aria-label={open ? t("Hide details") : t("Show details")}
                                  onClick={() => toggle(row.key)}
                                  className="ui-focus-ring text-foreground-subtle hover:text-foreground flex size-4 shrink-0 items-center justify-center rounded-control transition-colors"
                                >
                                  <ChevronRightIcon
                                    aria-hidden
                                    className={cn(
                                      "size-3 transition-[rotate] duration-200 ease-settle",
                                      open && "rotate-90",
                                    )}
                                  />
                                </button>
                              ) : (
                                <span aria-hidden className="size-4 shrink-0" />
                              ))}
                            {row.path !== "" && value !== undefined ? (
                              <Link
                                to={row.path}
                                className="ui-focus-ring text-brand min-w-0 truncate rounded-control underline-offset-2 hover:underline"
                              >
                                <DisplayValue
                                  type={column.type}
                                  value={value}
                                  label={column.label}
                                  inline
                                />
                              </Link>
                            ) : (
                              <span className="min-w-0 truncate">
                                <DisplayValue
                                  type={column.type}
                                  value={value}
                                  label={column.label}
                                  inline
                                />
                              </span>
                            )}
                          </span>
                        ) : (
                          <span className="block truncate">
                            <DisplayValue
                              type={column.type}
                              value={value}
                              label={column.label}
                              inline
                            />
                          </span>
                        )}
                      </TableCell>
                    );
                  })}
                </TableRow>
                {open && (
                  <tr className="border-border-subtle border-b">
                    <td colSpan={cells.length} className="bg-sunken p-0">
                      <div className="animate-rise sticky left-0 w-[100cqw] py-3 pr-(--cell-px) pl-7.5">
                        <DescriptionList layout="stacked" columns={1} className="gap-y-2.5">
                          {detail.map((column) =>
                            row.values[column.key] === undefined ? null : (
                              <DescriptionItem
                                key={column.key}
                                label={column.label}
                                valueClassName="text-xs"
                              >
                                <DisplayValue
                                  type={column.type}
                                  value={row.values[column.key]}
                                  label={column.label}
                                />
                              </DescriptionItem>
                            ),
                          )}
                        </DescriptionList>
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            );
          })}
        </TableBody>
      </table>
    </div>
  );
}
