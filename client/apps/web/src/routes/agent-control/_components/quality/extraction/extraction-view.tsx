import { usePermission } from "@/hooks/use-permission";
import { EXTRACTION_ACCURACY_KEY, fetchExtractionAccuracy } from "@/lib/graphql/extraction-eval";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon, PlayIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { useMemo, useState } from "react";
import {
  EXTRACTION_VIEW_PARAM,
  extractionViewParser,
  type ExtractionView as ExtractionSubView,
} from "../../../ai-control-tabs";
import { AccuracyPanel } from "./accuracy-panel";
import { CasesTable } from "./cases-table";
import { CorrectionsTable } from "./corrections-table";
import { ExtractionFigures } from "./extraction-figures";
import { EXTRACTION_STALE_MS, EXTRACTION_WINDOW_DAYS } from "./extraction-model";
import { NewRunDialog } from "./new-run-dialog";
import { RunsTable } from "./runs-table";

/**
 * How well documents are read into shipment drafts. Production accuracy comes
 * from the corrections people make; the evaluation set freezes the fair ones
 * so any model can be run against the same documents and compared.
 */
export default function ExtractionView() {
  const t = useT();
  const [view, setView] = useQueryState(EXTRACTION_VIEW_PARAM, extractionViewParser);
  const [newRunOpen, setNewRunOpen] = useState(false);
  const { allowed: canCreate } = usePermission(Resource.AgentEvalSuite, Operation.Create);

  const accuracy = useQuery({
    queryKey: [EXTRACTION_ACCURACY_KEY, EXTRACTION_WINDOW_DAYS],
    queryFn: ({ signal }) => fetchExtractionAccuracy(EXTRACTION_WINDOW_DAYS, { signal }),
    staleTime: EXTRACTION_STALE_MS,
  });

  const items = useMemo(
    () => [
      { value: "accuracy", label: t("Accuracy") },
      { value: "corrections", label: t("Corrections") },
      { value: "cases", label: t("Evaluation set") },
      { value: "runs", label: t("Runs") },
    ],
    [t],
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      {accuracy.isError ? (
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {t("Extraction accuracy could not be loaded. Try again shortly.")}
          </AlertDescription>
        </Alert>
      ) : accuracy.data ? (
        <ExtractionFigures accuracy={accuracy.data} />
      ) : (
        <Skeleton className="h-16" aria-busy />
      )}

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="w-full max-w-lg">
          <SegmentedControl<string>
            fullWidth
            aria-label={t("Document extraction view")}
            value={view}
            onValueChange={(value) => void setView(value as ExtractionSubView)}
            items={items}
          />
        </div>
        {canCreate ? (
          <Button type="button" size="sm" onClick={() => setNewRunOpen(true)}>
            <PlayIcon />
            {t("New evaluation run")}
          </Button>
        ) : null}
      </div>

      {view === "accuracy" &&
        (accuracy.data ? (
          <AccuracyPanel accuracy={accuracy.data} />
        ) : (
          <Skeleton className="h-64" aria-busy />
        ))}
      {view === "corrections" && <CorrectionsTable />}
      {view === "cases" && <CasesTable />}
      {view === "runs" && <RunsTable />}

      <NewRunDialog
        open={newRunOpen}
        onOpenChange={setNewRunOpen}
        activeCases={accuracy.data?.cases.active ?? 0}
        onStarted={() => void setView("runs")}
      />
    </div>
  );
}
