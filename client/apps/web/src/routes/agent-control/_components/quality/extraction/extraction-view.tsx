import {
  EXTRACTION_ACCURACY_KEY,
  fetchExtractionAccuracy,
} from "@/lib/graphql/extraction-accuracy";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import { KpiStripSkeleton } from "@/components/kpi/kpi-strip-skeleton";
import { useQueryState } from "nuqs";
import { useMemo } from "react";
import {
  EXTRACTION_VIEW_PARAM,
  extractionViewParser,
  type ExtractionView as ExtractionSubView,
} from "../../../ai-control-tabs";
import { AccuracyPanel } from "./accuracy-panel";
import { CorrectionsTable } from "./corrections-table";
import { ExtractionFigures } from "./extraction-figures";
import { EXTRACTION_STALE_MS, EXTRACTION_WINDOW_DAYS } from "./extraction-model";
import { AccuracyPanelSkeleton, EXTRACTION_FIGURES } from "./extraction-skeleton";
import { ProviderTrendsPanel } from "./provider-trends-panel";

/**
 * How well Nova reads documents into shipment drafts, measured from the corrections people
 * make when they confirm one.
 */
export default function ExtractionView() {
  const t = useT();
  const [view, setView] = useQueryState(EXTRACTION_VIEW_PARAM, extractionViewParser);

  const accuracy = useQuery({
    queryKey: [EXTRACTION_ACCURACY_KEY, EXTRACTION_WINDOW_DAYS],
    queryFn: ({ signal }) => fetchExtractionAccuracy(EXTRACTION_WINDOW_DAYS, { signal }),
    staleTime: EXTRACTION_STALE_MS,
  });

  const items = useMemo(
    () => [
      { value: "accuracy", label: t("Accuracy") },
      { value: "corrections", label: t("Corrections") },
    ],
    [t],
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      {accuracy.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertCircleIcon />
          <AlertDescription>
            {t("Extraction accuracy could not be loaded. Try again shortly.")}
          </AlertDescription>
        </Alert>
      ) : accuracy.data ? (
        <ExtractionFigures accuracy={accuracy.data} />
      ) : (
        <KpiStripSkeleton count={EXTRACTION_FIGURES} />
      )}

      <div className="w-full max-w-md">
        <SegmentedControl<string>
          fullWidth
          aria-label={t("Document extraction view")}
          value={view}
          onValueChange={(value) => void setView(value as ExtractionSubView)}
          items={items}
        />
      </div>

      {view === "accuracy" && (
        <>
          {accuracy.data ? <AccuracyPanel accuracy={accuracy.data} /> : <AccuracyPanelSkeleton />}
          <ProviderTrendsPanel />
        </>
      )}
      {view === "corrections" && <CorrectionsTable />}
    </div>
  );
}
