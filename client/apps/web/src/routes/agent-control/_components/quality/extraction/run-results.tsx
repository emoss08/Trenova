import { SectionPanelQuiet } from "@/components/section-panel";
import {
  EXTRACTION_EVAL_RESULT_DETAIL_KEY,
  fetchExtractionEvalResult,
  type ExtractionEvalResult,
} from "@/lib/graphql/extraction-eval";
import { useQuery } from "@tanstack/react-query";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChevronDownIcon, ChevronRightIcon } from "lucide-react";
import { useState } from "react";
import { formatShare } from "../quality-model";
import { ResultStatusBadge } from "./extraction-badges";
import { FieldResultsTable } from "./field-results-table";

function ResultFields({ id }: { id: string }) {
  const t = useT();
  const detail = useQuery({
    queryKey: [EXTRACTION_EVAL_RESULT_DETAIL_KEY, id],
    queryFn: ({ signal }) => fetchExtractionEvalResult(id, { signal }),
  });

  if (detail.isLoading) return <ComponentLoader />;
  if (!detail.data || detail.data.fieldResults.length === 0) {
    return <SectionPanelQuiet>{t("Nothing was scored for this case.")}</SectionPanelQuiet>;
  }

  return <FieldResultsTable results={detail.data.fieldResults} />;
}

function ResultRow({ result }: { result: ExtractionEvalResult }) {
  const t = useT();
  const [expanded, setExpanded] = useState(false);
  const scored = result.status === "Completed";

  return (
    <li className="border-border border-b last:border-b-0">
      <div className="flex items-center gap-3 px-3 py-2">
        <Button
          type="button"
          size="icon-sm"
          variant="ghost"
          aria-expanded={expanded}
          aria-label={expanded ? t("Hide fields") : t("Show fields")}
          disabled={!scored}
          onClick={() => setExpanded((value) => !value)}
        >
          {expanded ? <ChevronDownIcon /> : <ChevronRightIcon />}
        </Button>
        <span className="text-muted-foreground w-8 text-xs tabular-nums">{result.ordinal}</span>
        <span className="min-w-0 flex-1 truncate text-sm" title={result.caseTitle}>
          {result.caseTitle}
        </span>
        {result.errorMessage ? (
          <span className="text-danger max-w-72 truncate text-xs" title={result.errorMessage}>
            {result.errorMessage}
          </span>
        ) : null}
        <span className="w-14 text-right text-sm tabular-nums">
          {scored && result.scoredCount > 0 ? formatShare(result.accuracy) : "—"}
        </span>
        <ResultStatusBadge value={result.status} t={t} />
      </div>
      {expanded ? (
        <div className="border-border border-t">
          <ResultFields id={result.id} />
        </div>
      ) : null}
    </li>
  );
}

/** Each case in the run in the order it ran, opened to show its fields. */
export function RunResults({ results }: { results: ExtractionEvalResult[] }) {
  const t = useT();

  if (results.length === 0) {
    return <SectionPanelQuiet>{t("No cases were queued for this run.")}</SectionPanelQuiet>;
  }

  return (
    <ul>
      {results.map((result) => (
        <ResultRow key={result.id} result={result} />
      ))}
    </ul>
  );
}
