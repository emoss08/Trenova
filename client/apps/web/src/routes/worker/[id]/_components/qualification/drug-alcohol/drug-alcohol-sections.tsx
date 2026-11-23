import { InfoTooltip } from "@/components/info-tooltip";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import type { PreloadIntent } from "@/components/work-pane";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type {
  ClearinghouseQueryRow,
  DotTestRow,
  DotViolationRow,
  RandomDrawEntryRow,
  WorkerDrugAlcoholFile,
} from "@/lib/graphql/worker-drug-alcohol";
import {
  CheckIcon,
  FileCheck02Icon,
  InfoCircleIcon,
  SlashCircle01Icon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { useNow } from "@trenova/shared/hooks/use-now";
import { daysUntil, formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  clearinghouseQueryTypeLabel,
  clearinghouseResultLabel,
  clearinghouseResultTone,
  dotResultTone,
  dotTestResultLabel,
  dotTestStatusLabel,
  dotTestTypeLabel,
  dotViolationStatusLabel,
  dotViolationTypeLabel,
  drugAlcoholStatusMeta,
  randomEntryStatusLabel,
  returnToDutyLabel,
} from "@trenova/shared/lib/drug-alcohol";
import { cn } from "@trenova/shared/lib/utils";
import type { Ref } from "react";
import {
  BOUNDED_CELL_CLASS,
  BoundedTable,
  type BoundedColumn,
  type BoundedRowProps,
  type BoundedTableHandle,
} from "../../bounded-table";
import { ROW_CHANGED_CLASS, type FlashPhase } from "../../use-row-flash";
import { substanceLabel } from "./drug-alcohol-labels";

/** Whether a test is still open: neither completed nor voided. */
export function testIsOpen(test: Pick<DotTestRow, "status">): boolean {
  return test.status !== "Completed" && test.status !== "Cancelled";
}

function dueSub(t: TranslateFn, days: number): string {
  if (days < 0) return t("{0, plural, one {# day overdue} other {# days overdue}}", -days);
  if (days === 0) return t("today");
  return t("{0, plural, one {in # day} other {in # days}}", days);
}

/**
 * The testing standing in one bordered strip: the state with what it means, the four facts an
 * auditor asks about, and the two things the office records most.
 */
export function DrugAlcoholBand({
  file,
  canRecord,
  collectionIntent,
  queryIntent,
  onRecordCollection,
  onLogQuery,
}: {
  file: WorkerDrugAlcoholFile;
  canRecord: boolean;
  /** Reads the collection form's code as the pointer or focus reaches its button. */
  collectionIntent: PreloadIntent;
  /** Reads the query form's code as the pointer or focus reaches its button. */
  queryIntent: PreloadIntent;
  onRecordCollection: () => void;
  onLogQuery: () => void;
}) {
  const t = useT();
  const { standing } = file;
  const meta = drugAlcoholStatusMeta(standing.status);
  const openViolations = file.violations.filter((violation) => violation.status !== "Resolved");
  const sentence =
    standing.status === "Clear" && standing.openTestCount === 0 && openViolations.length === 0
      ? t("No open tests, no violations on record.")
      : t(meta.detail);
  const nextDue = standing.nextClearinghouseQueryDue;

  return (
    <section
      aria-label={t("Drug and alcohol standing")}
      className="grid min-w-0 overflow-hidden rounded-lg border bg-card xl:grid-cols-[15rem_minmax(0,1fr)_auto]"
    >
      <div
        className={cn(
          "flex flex-wrap items-center gap-2 border-b border-border-subtle px-4 py-3 xl:flex-col xl:items-start xl:border-r xl:border-b-0",
          standing.status === "Prohibited" && "bg-danger-subtle",
        )}
      >
        <span className="flex items-center gap-1.5">
          <Badge variant={meta.tone} dot>
            {t(meta.label)}
          </Badge>
          {standing.returnToDuty !== "NotRequired" ? (
            <Badge variant="neutral" dot>
              {returnToDutyLabel(standing.returnToDuty)}
            </Badge>
          ) : null}
          <InfoTooltip label={t("Testing standing")}>
            {t(
              "Only Prohibited bars dispatch. Follow-up testing runs after the driver is back at work and is not a bar. The next Clearinghouse query falls due twelve months after the last clear answer.",
            )}
          </InfoTooltip>
        </span>
        <p className="text-sm text-muted-foreground">{sentence}</p>
      </div>
      <DescriptionList layout="ruled" columns={4} className="rounded-none border-0">
        <DescriptionItem label={t("Pre-employment test")}>
          {standing.hasPreEmploymentTest ? (
            t("On file")
          ) : (
            <span className="text-warning-foreground">{t("Missing")}</span>
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Pre-employment query")}>
          {standing.hasPreEmploymentQuery ? (
            t("On file")
          ) : (
            <span className="text-warning-foreground">{t("Missing")}</span>
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Last query")} numeric>
          {standing.lastClearinghouseQueryAt ? (
            formatUnixDateMedium(standing.lastClearinghouseQueryAt)
          ) : (
            <span className="text-warning-foreground">{t("Never")}</span>
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Next query due")} numeric>
          {nextDue ? (
            <NextQueryDue dueAt={nextDue} />
          ) : (
            <span className="text-muted-foreground">—</span>
          )}
        </DescriptionItem>
      </DescriptionList>
      {canRecord ? (
        <div className="flex items-center gap-2 border-t border-border-subtle p-3 xl:border-t-0 xl:border-l">
          <Button size="sm" variant="secondary" {...collectionIntent} onClick={onRecordCollection}>
            {t("Record collection")}
          </Button>
          <Button size="sm" {...queryIntent} onClick={onLogQuery}>
            {t("Log query")}
          </Button>
        </div>
      ) : null}
    </section>
  );
}

/**
 * When the next Clearinghouse query falls due and how far off that is. It reads the minute here,
 * not in the band, so the clock re-draws only this line.
 */
function NextQueryDue({ dueAt }: { dueAt: number }) {
  const t = useT();
  const now = useNow("minute");
  const days = daysUntil(dueAt, { now });
  return (
    <span data-testid="next-query-due">
      {formatUnixDateMedium(dueAt)}{" "}
      <span
        className={cn(
          "text-xs",
          days < 0
            ? "text-danger-foreground"
            : days <= 30
              ? "text-warning-foreground"
              : "text-muted-foreground",
        )}
      >
        {dueSub(t, days)}
      </span>
    </span>
  );
}

/** The open violation's return-to-duty process; drawn only while one is open. */
export function ReturnToDutyPanel({
  violation,
  canManage,
  updateIntent,
  onUpdate,
}: {
  violation: DotViolationRow;
  canManage: boolean;
  /** Reads the dialog's code as the pointer or focus reaches Update. */
  updateIntent: PreloadIntent;
  onUpdate: () => void;
}) {
  const t = useT();
  const notYet = <span className="text-warning-foreground">{t("Not yet")}</span>;
  return (
    <SectionPanel
      title={t("Return to duty")}
      action={
        canManage ? (
          <Button size="sm" variant="secondary" {...updateIntent} onClick={onUpdate}>
            {t("Update")}
          </Button>
        ) : undefined
      }
    >
      <div className="flex flex-col gap-3 p-3">
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <Badge variant="danger" dot>
            {dotViolationTypeLabel(violation.violationType)}
          </Badge>
          <Badge variant="neutral" dot>
            {dotViolationStatusLabel(violation.status)}
          </Badge>
          <span className="text-muted-foreground tabular-nums">
            {formatUnixDateMedium(violation.occurredAt)}
          </span>
        </div>
        {violation.status === "FollowUp" ? (
          <p className="text-xs text-muted-foreground">
            {t(
              "The driver is back on duty. Follow-up testing does not bar dispatch on its own; the violation stays open until the last follow-up test is recorded.",
            )}
          </p>
        ) : null}
        <DescriptionList layout="ruled" columns={4}>
          <DescriptionItem label={t("SAP referred")} numeric>
            {violation.sapReferredAt ? formatUnixDateMedium(violation.sapReferredAt) : notYet}
          </DescriptionItem>
          <DescriptionItem label={t("Evaluation done")} numeric>
            {violation.sapEvaluationCompletedAt
              ? formatUnixDateMedium(violation.sapEvaluationCompletedAt)
              : notYet}
          </DescriptionItem>
          <DescriptionItem label={t("Returned to duty")} numeric>
            {violation.rtdCompletedAt ? formatUnixDateMedium(violation.rtdCompletedAt) : notYet}
          </DescriptionItem>
          <DescriptionItem label={t("Follow-up tests")} numeric>
            {t("{0} of {1}", violation.followUpTestsCompleted, violation.followUpTestCount)}
          </DescriptionItem>
        </DescriptionList>
      </div>
    </SectionPanel>
  );
}

/** Random selections still waiting on a collection; drawn only while there are some. */
export function RandomSelectionsPanel({
  selections,
  canRecord,
  recordIntent,
  onRecord,
}: {
  selections: readonly RandomDrawEntryRow[];
  canRecord: boolean;
  /** Reads the collection form's code as the pointer or focus reaches Record collection. */
  recordIntent: PreloadIntent;
  onRecord: (entry: RandomDrawEntryRow) => void;
}) {
  const t = useT();
  return (
    <SectionPanel title={t("Random selections outstanding")} count={selections.length}>
      <ul>
        {selections.map((entry) => (
          <li
            key={entry.id}
            className="flex items-center justify-between gap-3 px-3 py-2 text-sm not-first:border-t not-first:border-border-subtle"
          >
            <span className="flex items-center gap-2">
              <Badge variant="warning" dot>
                {randomEntryStatusLabel(entry.status)}
              </Badge>
              {substanceLabel(t, entry.substance)}
            </span>
            {canRecord ? (
              <Button
                size="xs"
                variant="secondary"
                {...recordIntent}
                onClick={() => onRecord(entry)}
              >
                {t("Record collection")}
              </Button>
            ) : null}
          </li>
        ))}
      </ul>
    </SectionPanel>
  );
}

/** Every test, open ones first, in a bounded table; voided ones stay, struck through. */
export function TestsPanel({
  tests,
  canUpdate,
  canCancel,
  flashOf,
  handleRef,
  resultIntent,
  onRecordResult,
  onVoid,
}: {
  tests: readonly DotTestRow[];
  canUpdate: boolean;
  canCancel: boolean;
  flashOf: (key: string) => FlashPhase | undefined;
  /** Brings a test's row into view by its id. */
  handleRef: Ref<BoundedTableHandle>;
  /** Reads the result form's code as the pointer or focus reaches an open test's menu. */
  resultIntent: PreloadIntent;
  onRecordResult: (test: DotTestRow) => void;
  onVoid: (test: DotTestRow) => void;
}) {
  const t = useT();
  const columns: BoundedColumn[] = [
    { key: "reason", label: t("Reason") },
    { key: "substance", label: t("Substance") },
    { key: "collected", label: t("Collected") },
    { key: "result", label: t("Result") },
    { key: "dot", label: t("DOT"), align: "center", shrink: true },
    { key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true },
  ];

  return (
    <SectionPanel title={t("Tests")} count={tests.length} hint={t("DOT and company")}>
      {tests.length === 0 ? (
        <SectionPanelQuiet>{t("No test is on file for this worker.")}</SectionPanelQuiet>
      ) : (
        <BoundedTable
          columns={columns}
          label={t("Tests")}
          maxHeightClassName="max-h-88"
          handleRef={handleRef}
          rows={tests.map((test) => ({
            key: test.id,
            render: (props: BoundedRowProps) => {
              const voided = test.status === "Cancelled";
              const open = testIsOpen(test);
              const date = test.collectedAt ?? test.scheduledAt;
              const actions: RowAction[] = [];
              if (open && canUpdate) {
                actions.push({
                  id: "result",
                  label: t("Record result"),
                  icon: FileCheck02Icon,
                  preload: resultIntent.onFocus,
                  onSelect: () => onRecordResult(test),
                });
              }
              if (open && canCancel) {
                actions.push({
                  id: "void",
                  label: t("Void test"),
                  icon: SlashCircle01Icon,
                  destructive: true,
                  onSelect: () => onVoid(test),
                });
              }
              const struck =
                voided && "text-muted-foreground line-through decoration-foreground-subtle";
              return (
                <tr
                  {...props}
                  data-glide-target
                  data-testid={`dot-test-${test.id}`}
                  data-voided={voided || undefined}
                  data-changed={flashOf(test.id)}
                  className={ROW_CHANGED_CLASS}
                >
                  <td className={cn(BOUNDED_CELL_CLASS, struck)}>
                    {dotTestTypeLabel(test.testType)}
                  </td>
                  <td className={cn(BOUNDED_CELL_CLASS, struck)}>
                    {substanceLabel(t, test.substance)}
                  </td>
                  <td
                    className={cn(
                      BOUNDED_CELL_CLASS,
                      "font-mono text-xs whitespace-nowrap",
                      struck,
                    )}
                  >
                    {date
                      ? test.collectedAt
                        ? formatUnixDateMedium(date)
                        : t("Scheduled {0}", formatUnixDateMedium(date))
                      : "—"}
                  </td>
                  <td className={cn(BOUNDED_CELL_CLASS, "whitespace-nowrap")}>
                    <Badge variant={dotResultTone(test.result)} dot>
                      {test.result === "Pending"
                        ? dotTestStatusLabel(test.status)
                        : dotTestResultLabel(test.result)}
                    </Badge>
                    {test.alcoholConcentration ? (
                      <span className="ml-1.5 font-mono text-xs text-muted-foreground">
                        {test.alcoholConcentration}
                      </span>
                    ) : null}
                  </td>
                  <td className={cn(BOUNDED_CELL_CLASS, "text-center")}>
                    {test.isDot ? (
                      <CheckIcon aria-label={t("DOT test")} className="mx-auto size-3.5" />
                    ) : (
                      <span className="text-xs text-muted-foreground">{t("Company")}</span>
                    )}
                  </td>
                  <td className={BOUNDED_CELL_CLASS}>
                    <RowActionsMenu
                      label={
                        date
                          ? t(
                              "Actions for {0}, {1}",
                              dotTestTypeLabel(test.testType),
                              formatUnixDateMedium(date),
                            )
                          : t("Actions for {0}", dotTestTypeLabel(test.testType))
                      }
                      actions={actions}
                    />
                  </td>
                </tr>
              );
            },
          }))}
        />
      )}
    </SectionPanel>
  );
}

/** Every Clearinghouse query in a bounded table; a pending one can take its answer. */
export function QueriesPanel({
  queries,
  canUpdate,
  flashOf,
  handleRef,
  answerIntent,
  onAnswer,
}: {
  queries: readonly ClearinghouseQueryRow[];
  canUpdate: boolean;
  flashOf: (key: string) => FlashPhase | undefined;
  /** Brings a query's row into view by its id. */
  handleRef: Ref<BoundedTableHandle>;
  /** Reads the answer form's code as the pointer or focus reaches a pending query's menu. */
  answerIntent: PreloadIntent;
  onAnswer: (query: ClearinghouseQueryRow) => void;
}) {
  const t = useT();
  const columns: BoundedColumn[] = [
    { key: "query", label: t("Query") },
    { key: "requested", label: t("Requested") },
    { key: "answer", label: t("Answer") },
    { key: "reference", label: t("Reference") },
    { key: "actions", label: <span className="sr-only">{t("Actions")}</span>, shrink: true },
  ];

  return (
    <SectionPanel title={t("Clearinghouse queries")} count={queries.length}>
      {queries.length === 0 ? (
        <SectionPanelQuiet>
          {t("No Clearinghouse query has been run for this worker.")}
        </SectionPanelQuiet>
      ) : (
        <BoundedTable
          columns={columns}
          label={t("Clearinghouse queries")}
          maxHeightClassName="max-h-88"
          handleRef={handleRef}
          rows={queries.map((query) => ({
            key: query.id,
            render: (props: BoundedRowProps) => {
              const actions: RowAction[] =
                canUpdate && query.result === "Pending"
                  ? [
                      {
                        id: "answer",
                        label: t("Record answer"),
                        icon: FileCheck02Icon,
                        preload: answerIntent.onFocus,
                        onSelect: () => onAnswer(query),
                      },
                    ]
                  : [];
              return (
                <tr
                  {...props}
                  data-glide-target
                  data-testid={`clearinghouse-query-${query.id}`}
                  data-changed={flashOf(query.id)}
                  className={ROW_CHANGED_CLASS}
                >
                  <td className={BOUNDED_CELL_CLASS}>
                    {clearinghouseQueryTypeLabel(query.queryType)}
                  </td>
                  <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs whitespace-nowrap")}>
                    {formatUnixDateMedium(query.requestedAt)}
                  </td>
                  <td className={cn(BOUNDED_CELL_CLASS, "whitespace-nowrap")}>
                    <Badge variant={clearinghouseResultTone(query.result)} dot>
                      {clearinghouseResultLabel(query.result)}
                    </Badge>
                    {query.violationCount > 0 ? (
                      <span className="ml-1.5 text-xs text-muted-foreground">
                        {t(
                          "{0, plural, one {# violation} other {# violations}}",
                          query.violationCount,
                        )}
                      </span>
                    ) : null}
                  </td>
                  <td className={cn(BOUNDED_CELL_CLASS, "font-mono text-xs text-muted-foreground")}>
                    {query.reference || "—"}
                  </td>
                  <td className={BOUNDED_CELL_CLASS}>
                    <RowActionsMenu
                      label={t(
                        "Actions for {0}, {1}",
                        clearinghouseQueryTypeLabel(query.queryType),
                        formatUnixDateMedium(query.requestedAt),
                      )}
                      actions={actions}
                    />
                  </td>
                </tr>
              );
            },
          }))}
        />
      )}
      <p className="flex items-center gap-1.5 border-t border-border-subtle px-3 py-2 text-xs text-muted-foreground">
        <InfoCircleIcon aria-hidden className="size-3 text-foreground-subtle" />
        {t("A pending query does not restart the 12-month clock.")}
      </p>
    </SectionPanel>
  );
}
