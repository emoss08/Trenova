import { useT } from "@trenova/shared/i18n/use-t";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { OshaLogCase } from "@/lib/graphql/worker-injury";
import {
  CASE_FILTERS,
  caseFilterCounts,
  caseLabel,
  filterCases,
  illnessTypeNumber,
  logColumn,
  type CaseFilter,
} from "@/lib/osha-log";
import { workerRecordHref } from "@/lib/route-utils";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Table,
  TableBody,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  caseClassificationLabel,
  claimStatusLabel,
  illnessTypeLabel,
} from "@trenova/shared/lib/injury";
import { cn } from "@trenova/shared/lib/utils";
import {
  ClipboardListIcon,
  LockIcon,
  PencilIcon,
  SearchIcon,
  Trash2Icon,
  UserRoundIcon,
} from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { OshaEmptyLog } from "./osha-empty-log";
import { FormMark } from "./osha-form-marks";

const STAGGER_LIMIT = 12;

type OshaCaseTableProps = {
  year: number;
  cases: readonly OshaLogCase[];
  filter: CaseFilter;
  onFilterChange: (filter: CaseFilter) => void;
  query: string;
  onQueryChange: (query: string) => void;
  canUpdate: boolean;
  canDelete: boolean;
  onOpen: (entry: OshaLogCase) => void;
  onEdit: (entry: OshaLogCase) => void;
  onDelete: (entry: OshaLogCase) => void;
};

/**
 * The 300 log itself, one row per case in case-number order, with the columns
 * the paper form has. Cases the office decided not to record are shown dimmed
 * and off the totals, because the decision not to record is worth seeing.
 */
export function OshaCaseTable({
  year,
  cases,
  filter,
  onFilterChange,
  query,
  onQueryChange,
  canUpdate,
  canDelete,
  onOpen,
  onEdit,
  onDelete,
}: OshaCaseTableProps) {
  const t = useT();

  const navigate = useNavigate();
  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);
  const counts = useMemo(() => caseFilterCounts(cases), [cases]);
  const rows = useMemo(() => filterCases(cases, filter, query), [cases, filter, query]);
  const filterItems = useMemo(
    () =>
      CASE_FILTERS.map((item) => ({
        value: item.value,
        label: item.label,
        caption: counts[item.value],
      })),
    [counts],
  );

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 700);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <section aria-label={t("Form 300")} className="flex min-w-0 flex-col gap-3">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="min-w-0">
          <h2 className="text-sm font-medium">
            {t("Log of work-related injuries and illnesses, {0}", year)}
          </h2>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {cases.length === 1
              ? t(
                  "Form 300. One case recorded; case numbers restart each January. A case is recorded from the worker's safety tab.",
                )
              : t(
                  "Form 300. {0} cases recorded; case numbers restart each January. A case is recorded from the worker's safety tab.",
                  cases.length,
                )}
          </p>
        </div>
        {cases.length > 0 ? (
          <div className="flex flex-wrap items-center gap-2">
            <SegmentedControl<CaseFilter>
              items={filterItems}
              value={filter}
              onValueChange={onFilterChange}
              aria-label={t("Which cases")}
            />
            <Input
              type="search"
              value={query}
              onChange={(event) => onQueryChange(event.target.value)}
              placeholder={t("Name, injury or place")}
              aria-label={t("Search the log")}
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
              inputContainerClassName="w-52"
            />
          </div>
        ) : null}
      </div>

      {cases.length === 0 ? (
        <OshaEmptyLog
          title={`Nothing recorded for ${year}`}
          description={
            "A year with no recordable case still posts a 300A with zeros in it.\n" +
            "A case is recorded from the worker's safety tab and lands here."
          }
          action={{ label: t("Open the workers list"), to: "/hr/workers" }}
        />
      ) : rows.length === 0 ? (
        <OshaEmptyLog
          title={t("No case matches")}
          description={t("Nothing in the log fits that search and filter together.")}
          action={{
            label: t("Show every case"),
            onClick: () => {
              onQueryChange("");
              onFilterChange("all");
            },
          }}
        />
      ) : (
        <div className="bg-card overflow-hidden rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-20">{t("Case")}</TableHead>
                <TableHead>{t("Employee")}</TableHead>
                <TableHead className="w-24">{t("Date")}</TableHead>
                <TableHead>{t("Where it happened")}</TableHead>
                <TableHead>{t("What happened")}</TableHead>
                <TableHead className="w-16 text-center">{t("Column")}</TableHead>
                <TableHead className="w-16 text-right">
                  <span className="inline-flex items-center gap-1">
                    <FormMark>K</FormMark>
                    {t("Away")}
                  </span>
                </TableHead>
                <TableHead className="w-20 text-right">
                  <span className="inline-flex items-center gap-1">
                    <FormMark>L</FormMark>
                    {t("Restricted")}
                  </span>
                </TableHead>
                <TableHead className="w-36">{t("Type")}</TableHead>
                <TableHead className="w-24">{t("Status")}</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((entry, index) => {
                const column = logColumn(entry.classification);
                const typeNumber = illnessTypeNumber(entry.illnessType);
                const label = caseLabel(entry);
                const actions: RowAction[] = [
                  {
                    id: "open",
                    label: t("Open case"),
                    icon: ClipboardListIcon,
                    onSelect: () => onOpen(entry),
                  },
                  {
                    id: "worker",
                    label: t("Open worker"),
                    icon: UserRoundIcon,
                    onSelect: () => void navigate(workerRecordHref(entry.workerId, "safety")),
                  },
                ];
                if (canUpdate) {
                  actions.push({
                    id: "edit",
                    label: t("Edit case"),
                    icon: PencilIcon,
                    onSelect: () => onEdit(entry),
                  });
                }
                if (canDelete) {
                  actions.push({
                    id: "delete",
                    label: t("Delete case"),
                    icon: Trash2Icon,
                    destructive: true,
                    onSelect: () => onDelete(entry),
                  });
                }
                return (
                  <m.tr
                    key={entry.id}
                    data-slot="table-row"
                    data-recordable={entry.recordable}
                    aria-label={`Case ${label}`}
                    initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.22, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
                    onClick={() => onOpen(entry)}
                    className={cn(
                      "hover:bg-muted/50 cursor-pointer border-b text-xs transition-colors last:border-0",
                      !entry.recordable && "text-muted-foreground",
                    )}
                  >
                    <td className="px-2 py-2 align-middle">
                      <button
                        type="button"
                        onClick={(event) => {
                          event.stopPropagation();
                          onOpen(entry);
                        }}
                        className="focus-visible:ring-ring/60 rounded-sm font-medium tabular-nums outline-none focus-visible:ring-2"
                      >
                        {label}
                      </button>
                    </td>
                    <td className="px-2 py-2 align-middle">
                      <span className="flex items-center gap-1.5">
                        <span className={cn("truncate", entry.recordable && "font-medium")}>
                          {entry.logName}
                        </span>
                        {entry.privacyCase ? (
                          <Tooltip>
                            <TooltipTrigger
                              render={
                                <span
                                  className="text-muted-foreground inline-flex"
                                  aria-label={t("Privacy case")}
                                />
                              }
                            >
                              <LockIcon className="size-3" />
                            </TooltipTrigger>
                            <TooltipContent>
                              {t("Privacy case: the name is withheld from the posted log.")}
                            </TooltipContent>
                          </Tooltip>
                        ) : null}
                      </span>
                    </td>
                    <td className="px-2 py-2 align-middle tabular-nums whitespace-nowrap">
                      {formatUnixDate(entry.occurredAt)}
                    </td>
                    <td className="max-w-40 truncate px-2 py-2 align-middle">
                      {entry.location?.trim() || (
                        <span className="text-muted-foreground/70">{t("Not given")}</span>
                      )}
                    </td>
                    <td className="max-w-64 px-2 py-2 align-middle">
                      <span className="block truncate">{t(entry.description)}</span>
                      {entry.bodyPart?.trim() ? (
                        <span className="text-muted-foreground block truncate text-2xs">
                          {entry.bodyPart.trim()}
                        </span>
                      ) : null}
                    </td>
                    <td className="px-2 py-2 text-center align-middle">
                      {column ? (
                        <FormMark
                          title={caseClassificationLabel(entry.classification)}
                          className="text-foreground"
                        >
                          {column}
                        </FormMark>
                      ) : (
                        <Badge
                          variant="secondary"
                          title={caseClassificationLabel(entry.classification)}
                        >
                          {t("Off the log")}
                        </Badge>
                      )}
                    </td>
                    <td
                      className={cn(
                        "px-2 py-2 text-right align-middle tabular-nums",
                        entry.daysAway === 0 && "text-muted-foreground/60",
                      )}
                    >
                      {entry.daysAway}
                    </td>
                    <td
                      className={cn(
                        "px-2 py-2 text-right align-middle tabular-nums",
                        entry.daysRestricted === 0 && "text-muted-foreground/60",
                      )}
                    >
                      {entry.daysRestricted}
                    </td>
                    <td className="px-2 py-2 align-middle whitespace-nowrap">
                      <span className="inline-flex items-center gap-1.5">
                        {typeNumber !== null ? <FormMark>({typeNumber})</FormMark> : null}
                        {illnessTypeLabel(entry.illnessType)}
                      </span>
                    </td>
                    <td className="px-2 py-2 align-middle">
                      <span className="flex flex-col items-start gap-0.5">
                        {entry.status === "Open" ? (
                          <Badge variant="warning">{t("Open")}</Badge>
                        ) : (
                          <span className="text-muted-foreground">{t("Closed")}</span>
                        )}
                        {entry.claimStatus !== "NotFiled" ? (
                          <span className="text-muted-foreground text-2xs">
                            {t("Claim {0}", claimStatusLabel(entry.claimStatus).toLowerCase())}
                          </span>
                        ) : null}
                      </span>
                    </td>
                    <td
                      className="px-1 py-1 text-right align-middle"
                      onClick={(event) => event.stopPropagation()}
                    >
                      <RowActionsMenu label={`Actions for case ${label}`} actions={actions} />
                    </td>
                  </m.tr>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}
    </section>
  );
}
