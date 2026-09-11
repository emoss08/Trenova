import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelDotTest,
  fetchWorkerDrugAlcoholFile,
  WORKER_DRUG_ALCOHOL_KEY,
  type ClearinghouseQueryRow,
  type DotTestRow,
  type DotViolationRow,
} from "@/lib/graphql/worker-drug-alcohol";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDate } from "@trenova/shared/lib/date";
import {
  clearinghouseQueryTypeLabel,
  clearinghouseResultLabel,
  clearinghouseResultTone,
  dotResultTone,
  dotTestResultLabel,
  dotTestStatusLabel,
  dotTestTypeLabel,
  drugAlcoholStatusMeta,
  dotViolationStatusLabel,
  dotViolationTypeLabel,
  randomEntryStatusLabel,
  returnToDutyLabel,
} from "@trenova/shared/lib/drug-alcohol";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { AnswerQueryDialog, RecordQueryDialog } from "./clearinghouse-dialogs";
import { RecordResultDialog } from "./record-result-dialog";
import { RecordTestDialog } from "./record-test-dialog";
import { useTestingInvalidation } from "./use-testing-invalidation";
import { ViolationProgressDialog } from "./violation-progress-dialog";

type DialogState =
  | { kind: "test"; drawEntryId?: string | null; substance?: "Drug" | "Alcohol" }
  | { kind: "result"; test: DotTestRow }
  | { kind: "query" }
  | { kind: "answer"; query: ClearinghouseQueryRow }
  | { kind: "violation"; violation: DotViolationRow };

export default function WorkerTestingTab({ workerId }: { workerId: string }) {
  const t = useT();

  const { allowed: canRecord } = usePermission(Resource.WorkerDOTTest, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerDOTTest, Operation.Update);
  const { allowed: canCancel } = usePermission(Resource.WorkerDOTTest, Operation.Cancel);
  const { allowed: canManage } = usePermission(Resource.WorkerDOTTest, Operation.Manage);
  const invalidate = useTestingInvalidation(workerId);
  const [dialog, setDialog] = useState<DialogState | null>(null);

  const fileQuery = useQuery({
    queryKey: [WORKER_DRUG_ALCOHOL_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerDrugAlcoholFile(workerId, { signal }),
  });

  const cancelMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) => cancelDotTest(id, reason),
    onSuccess: () => {
      toast.success(t("Collection voided"), {
        description: t("The record stays on file with the reason it was voided."),
      });
      void invalidate();
    },
    onError: (error: Error) =>
      toast.error(t("Could not void the collection"), {
        description: error.message,
      }),
  });

  // Open collections first: they are the ones the office is waiting on, and a
  // list ordered purely by date buries them under years of history.
  const tests = useMemo(() => {
    const rows = fileQuery.data?.tests ?? [];
    return [...rows].sort((a, b) => {
      const aOpen = a.status !== "Completed" && a.status !== "Cancelled";
      const bOpen = b.status !== "Completed" && b.status !== "Cancelled";
      if (aOpen !== bOpen) return aOpen ? -1 : 1;
      return (b.collectedAt ?? b.createdAt) - (a.collectedAt ?? a.createdAt);
    });
  }, [fileQuery.data?.tests]);

  if (fileQuery.isLoading) {
    return (
      <div className="flex flex-col gap-3 p-4">
        {[0, 1, 2].map((row) => (
          <Skeleton key={row} className="h-24 w-full" />
        ))}
      </div>
    );
  }

  const file = fileQuery.data;
  if (!file) return null;

  const standing = drugAlcoholStatusMeta(file.standing.status);
  const openViolation = file.violations.find((violation) => violation.status !== "Resolved");
  const pendingQuery = file.queries.find((query) => query.result === "Pending");

  return (
    <div className="flex flex-col gap-4 p-4">
      <section
        className={cn(
          "rounded-lg border p-4",
          file.standing.status === "Prohibited" && "border-red-500/60 bg-red-500/5",
        )}
      >
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <Badge variant={standing.tone}>{t(standing.label)}</Badge>
              {file.standing.returnToDuty !== "NotRequired" ? (
                <Badge variant="secondary">{returnToDutyLabel(file.standing.returnToDuty)}</Badge>
              ) : null}
              <InfoPopover title={t("Testing standing")}>
                <p>
                  {t("Prohibited while an open violation is still short of return to duty, or a Clearinghouse query found violations. Awaiting result while a collection is at the lab. Clear once a negative result is on file with nothing open. Not on file when no test has been recorded.")}
                </p>
                <p>
                  {t("Only Prohibited bars dispatch. Follow-up testing runs after the driver is back at work and is not a bar. The next Clearinghouse query falls due twelve months after the last answered one.")}
                </p>
              </InfoPopover>
            </div>
            <p className="text-muted-foreground mt-2 max-w-prose text-xs">{standing.detail}</p>
          </div>
          <div className="flex shrink-0 gap-2">
            {canRecord ? (
              <>
                <Button size="sm" onClick={() => setDialog({ kind: "test" })}>
                  {t("Record a collection")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => setDialog({ kind: "query" })}>
                  {t("Log a query")}
                </Button>
              </>
            ) : null}
          </div>
        </div>

        <dl className="mt-4 grid grid-cols-2 gap-3 text-xs sm:grid-cols-4">
          <Figure
            label={t("Pre-employment test")}
            value={file.standing.hasPreEmploymentTest ? "On file" : "Missing"}
            warn={!file.standing.hasPreEmploymentTest}
          />
          <Figure
            label={t("Pre-employment query")}
            value={file.standing.hasPreEmploymentQuery ? "On file" : "Missing"}
            warn={!file.standing.hasPreEmploymentQuery}
          />
          <Figure
            label={t("Last query")}
            value={
              file.standing.lastClearinghouseQueryAt
                ? formatUnixDate(file.standing.lastClearinghouseQueryAt)
                : "Never"
            }
            warn={!file.standing.lastClearinghouseQueryAt}
          />
          <Figure
            label={t("Next query due")}
            value={
              file.standing.nextClearinghouseQueryDue
                ? formatUnixDate(file.standing.nextClearinghouseQueryDue)
                : "—"
            }
            warn={Boolean(
              file.standing.nextClearinghouseQueryDue &&
              file.standing.nextClearinghouseQueryDue * 1000 <
                Date.parse(new Date().toDateString()),
            )}
          />
        </dl>
      </section>

      {openViolation ? (
        <Section
          title={t("Return-to-duty process")}
          action={
            canManage ? (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setDialog({ kind: "violation", violation: openViolation })}
              >
                {t("Update")}
              </Button>
            ) : null
          }
        >
          {openViolation.status === "FollowUp" ? (
            <Alert className="mb-2">
              <AlertDescription>
                {t("The driver is back on duty. Follow-up testing does not bar dispatch on its own; the violation stays open here until the last follow-up test is recorded.")}
              </AlertDescription>
            </Alert>
          ) : null}
          <div className="rounded-md border p-3 text-xs">
            <div className="flex flex-wrap items-center gap-2">
              <Badge variant="inactive">{dotViolationTypeLabel(openViolation.violationType)}</Badge>
              <Badge variant="secondary">{dotViolationStatusLabel(openViolation.status)}</Badge>
              <span className="text-muted-foreground">
                {formatUnixDate(openViolation.occurredAt)}
              </span>
            </div>
            <dl className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
              <Figure
                label={t("SAP referred")}
                value={
                  openViolation.sapReferredAt
                    ? formatUnixDate(openViolation.sapReferredAt)
                    : "Not yet"
                }
                warn={!openViolation.sapReferredAt}
              />
              <Figure
                label={t("Evaluation done")}
                value={
                  openViolation.sapEvaluationCompletedAt
                    ? formatUnixDate(openViolation.sapEvaluationCompletedAt)
                    : "Not yet"
                }
                warn={!openViolation.sapEvaluationCompletedAt}
              />
              <Figure
                label={t("Returned to duty")}
                value={
                  openViolation.rtdCompletedAt
                    ? formatUnixDate(openViolation.rtdCompletedAt)
                    : "Not yet"
                }
                warn={!openViolation.rtdCompletedAt}
              />
              <Figure
                label={t("Follow-up tests")}
                value={`${openViolation.followUpTestsCompleted} of ${openViolation.followUpTestCount}`}
              />
            </dl>
          </div>
        </Section>
      ) : null}

      {file.selections.length > 0 ? (
        <Section title={t("Random selections outstanding")}>
          <ul className="flex flex-col gap-1.5">
            {file.selections.map((entry) => (
              <li
                key={entry.id}
                className="flex items-center justify-between rounded-md border px-3 py-2 text-xs"
              >
                <span className="flex items-center gap-2">
                  <Badge variant="warning">{randomEntryStatusLabel(entry.status)}</Badge>
                  <span>{entry.substance === "Alcohol" ? "Alcohol" : "Controlled substances"}</span>
                </span>
                {canRecord ? (
                  <Button
                    size="xs"
                    variant="outline"
                    onClick={() =>
                      setDialog({
                        kind: "test",
                        drawEntryId: entry.id,
                        substance: entry.substance as "Drug" | "Alcohol",
                      })
                    }
                  >
                    {t("Record collection")}
                  </Button>
                ) : null}
              </li>
            ))}
          </ul>
        </Section>
      ) : null}

      <Section title={t("Tests")}>
        {tests.length === 0 ? (
          <Empty>{t("No test is on file for this worker.")}</Empty>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {tests.map((test) => (
              <li key={test.id} className="rounded-md border px-3 py-2 text-xs">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="flex flex-wrap items-center gap-2">
                    <span className="font-medium">{dotTestTypeLabel(test.testType)}</span>
                    <span className="text-muted-foreground">
                      {test.substance === "Alcohol" ? "Alcohol" : "Controlled substances"}
                    </span>
                    <Badge variant={dotResultTone(test.result)}>
                      {test.result === "Pending"
                        ? dotTestStatusLabel(test.status)
                        : dotTestResultLabel(test.result)}
                    </Badge>
                    {test.isDot ? null : <Badge variant="secondary">{t("Non-DOT")}</Badge>}
                  </span>
                  <span className="flex items-center gap-2">
                    <span className="text-muted-foreground tabular-nums">
                      {test.collectedAt
                        ? formatUnixDate(test.collectedAt)
                        : test.scheduledAt
                          ? `scheduled ${formatUnixDate(test.scheduledAt)}`
                          : "—"}
                    </span>
                    {canUpdate && test.status !== "Completed" && test.status !== "Cancelled" ? (
                      <Button
                        size="xs"
                        variant="outline"
                        onClick={() => setDialog({ kind: "result", test })}
                      >
                        {t("Record result")}
                      </Button>
                    ) : null}
                    {canCancel && test.status !== "Completed" && test.status !== "Cancelled" ? (
                      <Button
                        size="xs"
                        variant="ghost"
                        isLoading={cancelMutation.isPending}
                        onClick={() =>
                          cancelMutation.mutate({
                            id: test.id,
                            reason: "Collection did not take place",
                          })
                        }
                      >
                        {t("Void")}
                      </Button>
                    ) : null}
                  </span>
                </div>
                {test.alcoholConcentration ? (
                  <p className="text-muted-foreground mt-1 tabular-nums">
                    {t("Concentration {0}", test.alcoholConcentration)}
                  </p>
                ) : null}
                {test.reason ? <p className="text-muted-foreground mt-1">{test.reason}</p> : null}
              </li>
            ))}
          </ul>
        )}
      </Section>

      <Section title={t("Clearinghouse queries")}>
        {file.queries.length === 0 ? (
          <Empty>{t("No Clearinghouse query has been run for this worker.")}</Empty>
        ) : (
          <ul className="flex flex-col gap-1.5">
            {file.queries.map((query) => (
              <li
                key={query.id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-md border px-3 py-2 text-xs"
              >
                <span className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">
                    {clearinghouseQueryTypeLabel(query.queryType)}
                  </span>
                  <Badge variant={clearinghouseResultTone(query.result)}>
                    {clearinghouseResultLabel(query.result)}
                  </Badge>
                  {query.violationCount > 0 ? (
                    <span className="text-muted-foreground">
                      {t("{0} violation{1}", query.violationCount, query.violationCount === 1 ? "" : "s")}
                    </span>
                  ) : null}
                </span>
                <span className="flex items-center gap-2">
                  <span className="text-muted-foreground tabular-nums">
                    {formatUnixDate(query.completedAt ?? query.requestedAt)}
                  </span>
                  {canUpdate && query.result === "Pending" ? (
                    <Button
                      size="xs"
                      variant="outline"
                      onClick={() => setDialog({ kind: "answer", query })}
                    >
                      {t("Record answer")}
                    </Button>
                  ) : null}
                </span>
              </li>
            ))}
          </ul>
        )}
        {pendingQuery ? (
          <p className="text-muted-foreground mt-2 text-[11px]">
            {t("A query logged but not answered does not restart the twelve-month clock.")}
          </p>
        ) : null}
      </Section>

      <RecordTestDialog
        open={dialog?.kind === "test"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        drawEntryId={dialog?.kind === "test" ? (dialog.drawEntryId ?? null) : null}
        defaultSubstance={dialog?.kind === "test" ? (dialog.substance ?? "Drug") : "Drug"}
      />
      <RecordResultDialog
        open={dialog?.kind === "result"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        test={dialog?.kind === "result" ? dialog.test : null}
      />
      <RecordQueryDialog
        open={dialog?.kind === "query"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
      />
      <AnswerQueryDialog
        open={dialog?.kind === "answer"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        query={dialog?.kind === "answer" ? dialog.query : null}
      />
      <ViolationProgressDialog
        open={dialog?.kind === "violation"}
        onOpenChange={(open) => !open && setDialog(null)}
        workerId={workerId}
        violation={dialog?.kind === "violation" ? dialog.violation : null}
      />
    </div>
  );
}

function Section({
  title,
  action,
  children,
}: {
  title: string;
  action?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section>
      <div className="mb-2 flex items-center justify-between">
        <h3 className="cc-label text-foreground">{title}</h3>
        {action}
      </div>
      {children}
    </section>
  );
}

function Figure({ label, value, warn }: { label: string; value: string; warn?: boolean }) {
  return (
    <div>
      <dt className="text-muted-foreground text-[11px]">{label}</dt>
      <dd className={cn("font-medium tabular-nums", warn && "text-amber-600 dark:text-amber-400")}>
        {value}
      </dd>
    </div>
  );
}

function Empty({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-muted-foreground rounded-md border border-dashed p-3 text-xs">{children}</p>
  );
}
