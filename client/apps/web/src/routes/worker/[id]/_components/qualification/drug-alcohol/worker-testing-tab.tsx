import { ReasonDialog } from "@/components/reason-dialog";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  cancelDotTest,
  fetchWorkerDrugAlcoholFile,
  WORKER_DRUG_ALCOHOL_KEY,
  type DotTestRow,
} from "@/lib/graphql/worker-drug-alcohol";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptyTableSketch } from "@trenova/shared/components/ui/empty-table";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { QualificationSkeletonBody } from "../qualification-skeleton";
import { useQualificationRoute, type QualificationViewProps } from "../qualification-route";
import { revealElement, useRowFlash } from "../use-row-flash";
import { AnswerQueryPane, LogQueryPane } from "./clearinghouse-panes";
import {
  DrugAlcoholBand,
  QueriesPanel,
  RandomSelectionsPanel,
  ReturnToDutyPanel,
  TestsPanel,
  testIsOpen,
} from "./drug-alcohol-sections";
import { RecordCollectionPane } from "./record-collection-pane";
import { RecordResultPane } from "./record-result-pane";
import { useTestingInvalidation } from "./use-testing-invalidation";
import { ViolationProgressDialog } from "./violation-progress-dialog";

const VIEW = "drug-and-alcohol" as const;
/** The longest void reason kept; the server stores it in the test's notes. */
const VOID_REASON_MAX = 500;

/**
 * The drug and alcohol programme for one worker: the standing band, the return-to-duty process
 * and random selections while there are any, and the tests and Clearinghouse queries. Recording
 * a collection or a result, and logging or answering a query, open in the work pane from the
 * address (`form=`, `test=`, `query=`); voiding a test asks why.
 */
export default function WorkerTestingTab({ workerId, open, onOpen }: QualificationViewProps) {
  const t = useT();
  const { go, closeWhen } = useQualificationRoute(VIEW, onOpen);
  const { flash, flashOf } = useRowFlash();
  const rows = useRef(new Map<string, HTMLTableRowElement>());
  const now = useNow();

  const { allowed: canRecord } = usePermission(Resource.WorkerDOTTest, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.WorkerDOTTest, Operation.Update);
  const { allowed: canCancel } = usePermission(Resource.WorkerDOTTest, Operation.Cancel);
  const { allowed: canManage } = usePermission(Resource.WorkerDOTTest, Operation.Manage);
  const invalidate = useTestingInvalidation(workerId);
  const [progressOpen, setProgressOpen] = useState(false);
  const [voiding, setVoiding] = useState<DotTestRow | null>(null);

  const fileQuery = useQuery({
    queryKey: [WORKER_DRUG_ALCOHOL_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerDrugAlcoholFile(workerId, { signal }),
  });

  const voidTest = useApiMutation<DotTestRow, { test: DotTestRow; reason: string }>({
    scope: { id: `worker-dot-test-void:${workerId}` },
    resourceName: "Test",
    mutationFn: ({ test, reason }) => cancelDotTest(test.id, reason),
    onSuccess: (saved) => {
      setVoiding(null);
      flash(saved.id);
      void invalidate();
    },
  });

  const file = fileQuery.data;
  const lastShown = useRef<string | null>(null);
  useEffect(() => {
    const id = open.form == null ? (open.test ?? open.query) : null;
    if (!id || id === lastShown.current || !rows.current.has(id)) return;
    lastShown.current = id;
    flash(id);
    revealElement(rows.current.get(id));
  });

  if (fileQuery.isPending) return <QualificationSkeletonBody view={VIEW} />;
  if (fileQuery.isError || !file) {
    return (
      <ErrorState
        layout="sheet"
        error={fileQuery.error ?? new Error("No drug and alcohol file")}
        title={t("The drug and alcohol record could not be loaded")}
        description={t("This isn't an empty record. Try again in a moment.")}
        onRetry={() => void fileQuery.refetch()}
        sketch={<EmptyTableSketch columns={[{ label: t("Reason") }, { label: t("Result") }]} />}
      />
    );
  }

  // Open collections first: they are the ones the office is waiting on.
  const tests = [...file.tests].sort((a, b) => {
    const aOpen = testIsOpen(a);
    const bOpen = testIsOpen(b);
    if (aOpen !== bOpen) return aOpen ? -1 : 1;
    return (b.collectedAt ?? b.createdAt) - (a.collectedAt ?? a.createdAt);
  });
  const openViolation = file.violations.find((violation) => violation.status !== "Resolved");
  const resultTest =
    open.form === "record-test-result" && open.test
      ? (file.tests.find((test) => test.id === open.test && testIsOpen(test)) ?? null)
      : null;
  const answerQuery =
    open.form === "answer-query" && open.query
      ? (file.queries.find((query) => query.id === open.query && query.result === "Pending") ??
        null)
      : null;
  const selection =
    open.form === "record-collection" && open.kind
      ? (file.selections.find((entry) => entry.id === open.kind) ?? null)
      : null;
  const rowRef = (key: string, element: HTMLTableRowElement | null) => {
    if (element) rows.current.set(key, element);
    else rows.current.delete(key);
  };

  return (
    <div className="flex flex-col gap-4">
      <DrugAlcoholBand
        file={file}
        now={now}
        canRecord={canRecord}
        onRecordCollection={() =>
          go({ tab: "qualification", view: VIEW, form: "record-collection" })
        }
        onLogQuery={() => go({ tab: "qualification", view: VIEW, form: "log-query" })}
      />

      {openViolation ? (
        <ReturnToDutyPanel
          violation={openViolation}
          canManage={canManage}
          onUpdate={() => setProgressOpen(true)}
        />
      ) : null}

      {file.selections.length > 0 ? (
        <RandomSelectionsPanel
          selections={file.selections}
          canRecord={canRecord}
          onRecord={(entry) =>
            go({ tab: "qualification", view: VIEW, form: "record-collection", kind: entry.id })
          }
        />
      ) : null}

      <div className="grid items-start gap-4 xl:grid-cols-2">
        <TestsPanel
          tests={tests}
          canUpdate={canUpdate}
          canCancel={canCancel}
          flashOf={flashOf}
          rowRef={rowRef}
          onRecordResult={(test) =>
            go({ tab: "qualification", view: VIEW, form: "record-test-result", test: test.id })
          }
          onVoid={setVoiding}
        />
        <QueriesPanel
          queries={file.queries}
          canUpdate={canUpdate}
          flashOf={flashOf}
          rowRef={rowRef}
          onAnswer={(query) =>
            go({ tab: "qualification", view: VIEW, form: "answer-query", query: query.id })
          }
        />
      </div>

      <RecordCollectionPane
        open={open.form === "record-collection" && canRecord}
        onOpenChange={closeWhen(open.form === "record-collection")}
        workerId={workerId}
        drawEntryId={selection?.id ?? null}
        defaultSubstance={selection?.substance === "Alcohol" ? "Alcohol" : "Drug"}
        onSaved={flash}
      />
      <RecordResultPane
        open={open.form === "record-test-result" && canUpdate && resultTest != null}
        onOpenChange={closeWhen(open.form === "record-test-result")}
        workerId={workerId}
        test={resultTest}
        onSaved={flash}
      />
      <LogQueryPane
        open={open.form === "log-query" && canRecord}
        onOpenChange={closeWhen(open.form === "log-query")}
        workerId={workerId}
        onSaved={flash}
      />
      <AnswerQueryPane
        open={open.form === "answer-query" && canUpdate && answerQuery != null}
        onOpenChange={closeWhen(open.form === "answer-query")}
        workerId={workerId}
        query={answerQuery}
        onSaved={flash}
      />
      <ViolationProgressDialog
        open={progressOpen && openViolation != null}
        onOpenChange={setProgressOpen}
        workerId={workerId}
        violation={openViolation ?? null}
      />
      <ReasonDialog
        open={voiding !== null}
        title={t("Void this test?")}
        description={t(
          "It stays on the record, struck through, and stops counting toward the testing history.",
        )}
        placeholder={t("e.g. Recorded against the wrong driver")}
        confirmLabel={t("Void test")}
        destructive
        maxLength={VOID_REASON_MAX}
        pending={voidTest.isPending}
        onConfirm={(reason) => {
          if (voiding && !voidTest.isPending) {
            void voidTest.mutateAsync({ test: voiding, reason }).catch(() => undefined);
          }
        }}
        onOpenChange={(next) => !next && !voidTest.isPending && setVoiding(null)}
      />
    </div>
  );
}
