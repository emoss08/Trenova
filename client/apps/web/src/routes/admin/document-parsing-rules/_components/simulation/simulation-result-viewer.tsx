import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Separator } from "@trenova/shared/components/ui/separator";
import type { DocumentParsingAnalysis, SimulationResult } from "@/types/document-parsing-rule";
import { AlertTriangleIcon, CheckCircle2Icon, MapPinIcon, XCircleIcon } from "lucide-react";
import { useMemo, useState } from "react";
import ReactDiffViewer, { DiffMethod } from "react-diff-viewer-continued";

type SortKey = "key" | "confidence" | "source";
type SortDir = "asc" | "desc";

export function SimulationResultViewer({ result }: { result: SimulationResult }) {
  const t = useT();

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <Badge
          variant={result.matched ? "active" : "inactive"}
          className="gap-1.5 px-3 py-1 text-sm"
        >
          {result.matched ? (
            <CheckCircle2Icon className="size-3.5" />
          ) : (
            <XCircleIcon className="size-3.5" />
          )}
          {result.matched ? "Matched" : "Not Matched"}
        </Badge>
        <Badge
          variant={result.validationPassed ? "active" : "inactive"}
          className="gap-1.5 px-3 py-1 text-sm"
        >
          {result.validationPassed ? (
            <CheckCircle2Icon className="size-3.5" />
          ) : (
            <XCircleIcon className="size-3.5" />
          )}
          {result.validationPassed ? "Validation Passed" : "Validation Failed"}
        </Badge>
        {result.candidate?.overallConfidence != null && (
          <Badge variant="info" className="gap-1.5 px-3 py-1 text-sm">
            {t("{0}% overall confidence", (result.candidate.overallConfidence * 100).toFixed(1))}
          </Badge>
        )}
      </div>

      {(result.validationErrors?.length ?? 0) > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-destructive flex items-center gap-2">
              <AlertTriangleIcon className="size-4" />
              {t("Validation Errors")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="text-destructive list-inside list-disc space-y-1 text-sm">
              {result.validationErrors?.map((err, i) => (
                <li key={i}>{err}</li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}

      {result.metadata && (
        <Card>
          <CardHeader>
            <CardTitle>{t("Rule Metadata")}</CardTitle>
            <CardDescription>{t("Details about which rule version matched and how.")}</CardDescription>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-3">
              <div>
                <dt className="text-muted-foreground text-xs">{t("Rule Set")}</dt>
                <dd className="font-medium">{result.metadata.ruleSetName}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground text-xs">{t("Version")}</dt>
                <dd className="font-medium">{result.metadata.versionNumber}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground text-xs">{t("Parser Mode")}</dt>
                <dd className="font-medium">{result.metadata.parserMode}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground text-xs">{t("Provider Matched")}</dt>
                <dd className="font-medium">{result.metadata.providerMatched || "\u2014"}</dd>
              </div>
              <div>
                <dt className="text-muted-foreground text-xs">{t("Match Specificity")}</dt>
                <dd className="font-medium">{result.metadata.matchSpecificity}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      )}

      {result.candidate && <AnalysisCard title={t("Candidate Analysis")} analysis={result.candidate} />}

      {result.diff && (
        <DiffCard diff={result.diff} baseline={result.baseline} candidate={result.candidate} />
      )}
    </div>
  );
}

function confidenceVariant(confidence: number) {
  if (confidence >= 0.9) return "active";
  if (confidence >= 0.7) return "warning";
  return "inactive";
}

function AnalysisCard({ title, analysis }: { title: string; analysis: DocumentParsingAnalysis }) {
  const t = useT();

  const fieldEntries = Object.entries(analysis.fields ?? {});

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>
          {t("Confidence: {0}% {1}", ((analysis.overallConfidence ?? 0) * 100).toFixed(1), analysis.reviewStatus ? ` \u00b7 Status: ${analysis.reviewStatus}` : "")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {fieldEntries.length > 0 && (
          <div>
            <h4 className="mb-2 text-sm font-medium">{t("Fields ({0})", fieldEntries.length)}</h4>
            <FieldsTable fieldEntries={fieldEntries} />
          </div>
        )}

        {(analysis.stops?.length ?? 0) > 0 && (
          <>
            <Separator />
            <div>
              <h4 className="mb-2 text-sm font-medium">{t("Stops ({0})", analysis.stops?.length ?? 0)}</h4>
              <div className="space-y-2">
                {analysis.stops?.map((stop, i) => (
                  <div key={i} className="rounded-md border p-3">
                    <div className="mb-2 flex items-center gap-2">
                      <MapPinIcon className="text-muted-foreground size-3.5" />
                      <Badge variant="info" className="capitalize">
                        {stop.role}
                      </Badge>
                      <span className="text-muted-foreground text-xs">{t("Seq {0}", stop.sequence)}</span>
                      <Badge variant={confidenceVariant(stop.confidence)}>
                        {(stop.confidence * 100).toFixed(0)}%
                      </Badge>
                      {stop.reviewRequired && <Badge variant="warning">{t("Review")}</Badge>}
                    </div>
                    <div className="space-y-0.5 text-sm">
                      {stop.name && <p className="font-medium">{stop.name}</p>}
                      {stop.addressLine1 && <p>{stop.addressLine1}</p>}
                      {stop.addressLine2 && <p>{stop.addressLine2}</p>}
                      <p>
                        {[stop.city, stop.state].filter(Boolean).join(", ")}
                        {stop.postalCode ? ` ${stop.postalCode}` : ""}
                      </p>
                      {(stop.date || stop.timeWindow) && (
                        <p className="text-muted-foreground text-xs">
                          {[stop.date, stop.timeWindow].filter(Boolean).join(" \u00b7 ")}
                        </p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </>
        )}

        {(analysis.conflicts?.length ?? 0) > 0 && (
          <>
            <Separator />
            <div>
              <h4 className="text-destructive mb-2 text-sm font-medium">
                {t("Conflicts ({0})", analysis.conflicts?.length ?? 0)}
              </h4>
              <ul className="space-y-1 text-sm">
                {analysis.conflicts?.map((c, i) => (
                  <li key={i} className="text-destructive">
                    <span className="font-medium">{t(c.label)}:</span> {c.values.join(" vs ")}
                  </li>
                ))}
              </ul>
            </div>
          </>
        )}

        {(analysis.missingFields?.length ?? 0) > 0 && (
          <>
            <Separator />
            <div>
              <h4 className="text-warning mb-2 text-sm font-medium">{t("Missing Fields")}</h4>
              <div className="flex flex-wrap gap-1">
                {analysis.missingFields?.map((f) => (
                  <Badge key={f} variant="warning">
                    {f}
                  </Badge>
                ))}
              </div>
            </div>
          </>
        )}

        {(analysis.signals?.length ?? 0) > 0 && (
          <>
            <Separator />
            <div>
              <h4 className="mb-2 text-sm font-medium">{t("Signals")}</h4>
              <ul className="text-muted-foreground list-inside list-disc text-sm">
                {analysis.signals?.map((s, i) => (
                  <li key={i}>{s}</li>
                ))}
              </ul>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function FieldsTable({
  fieldEntries,
}: {
  fieldEntries: [
    string,
    {
      key: string;
      label: string;
      value: string;
      confidence: number;
      source: string;
      reviewRequired: boolean;
    },
  ][];
}) {
  const t = useT();

  const [sortKey, setSortKey] = useState<SortKey>("key");
  const [sortDir, setSortDir] = useState<SortDir>("asc");

  const sorted = useMemo(() => {
    const copy = [...fieldEntries];
    copy.sort(([, a], [, b]) => {
      let cmp = 0;
      if (sortKey === "key") {
        cmp = a.key.localeCompare(b.key);
      } else if (sortKey === "confidence") {
        cmp = a.confidence - b.confidence;
      } else if (sortKey === "source") {
        cmp = (a.source ?? "").localeCompare(b.source ?? "");
      }
      return sortDir === "asc" ? cmp : -cmp;
    });
    return copy;
  }, [fieldEntries, sortKey, sortDir]);

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const sortIndicator = (key: SortKey) => {
    if (sortKey !== key) return "";
    return sortDir === "asc" ? " \u2191" : " \u2193";
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-muted-foreground border-b text-left">
            <th className="cursor-pointer pr-4 pb-2 select-none" onClick={() => toggleSort("key")}>
              {t("Key{0}", sortIndicator("key"))}
            </th>
            <th className="pr-4 pb-2">{t("Value")}</th>
            <th
              className="cursor-pointer pr-4 pb-2 select-none"
              onClick={() => toggleSort("confidence")}
            >
              {t("Confidence{0}", sortIndicator("confidence"))}
            </th>
            <th
              className="cursor-pointer pr-4 pb-2 select-none"
              onClick={() => toggleSort("source")}
            >
              {t("Source{0}", sortIndicator("source"))}
            </th>
            <th className="pb-2">{t("Review")}</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map(([entryKey, field]) => (
            <tr key={entryKey} className="border-b last:border-0">
              <td className="py-1.5 pr-4 font-mono text-xs">{field.key}</td>
              <td className="py-1.5 pr-4">{field.value}</td>
              <td className="py-1.5 pr-4">
                <Badge variant={confidenceVariant(field.confidence)} className="font-mono">
                  {(field.confidence * 100).toFixed(0)}%
                </Badge>
              </td>
              <td className="py-1.5 pr-4">
                <Badge variant="secondary">{field.source}</Badge>
              </td>
              <td className="py-1.5">
                {field.reviewRequired && <Badge variant="warning">{t("Review")}</Badge>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function DiffCard({
  diff,
  baseline,
  candidate,
}: {
  diff: SimulationResult["diff"];
  baseline?: DocumentParsingAnalysis | null;
  candidate?: DocumentParsingAnalysis | null;
}) {
  const t = useT();

  const hasChanges =
    (diff.addedFields?.length ?? 0) > 0 ||
    (diff.changedFields?.length ?? 0) > 0 ||
    (diff.addedStopRoles?.length ?? 0) > 0 ||
    (diff.changedStopRoles?.length ?? 0) > 0;

  if (!hasChanges && !baseline) return null;

  const totalChanges =
    (diff.addedFields?.length ?? 0) +
    (diff.changedFields?.length ?? 0) +
    (diff.addedStopRoles?.length ?? 0) +
    (diff.changedStopRoles?.length ?? 0);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>{t("Diff")}</span>
          {totalChanges > 0 && (
            <Badge variant="info" className="font-normal">
              {t("{0} change{1}", totalChanges, totalChanges !== 1 ? "s" : "")}
            </Badge>
          )}
        </CardTitle>
        <CardDescription>
          {t("Comparison between baseline and candidate extraction results.")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {hasChanges && (
          <div className="flex flex-wrap gap-2">
            {diff.addedFields?.map((f) => (
              <Badge key={`af-${f}`} variant="active" className="gap-1">
                + {f}
              </Badge>
            ))}
            {diff.changedFields?.map((f) => (
              <Badge key={`cf-${f}`} variant="warning" className="gap-1">
                ~ {f}
              </Badge>
            ))}
            {diff.addedStopRoles?.map((r) => (
              <Badge key={`ar-${r}`} variant="active" className="gap-1">
                {t("+ stop: {0}", r)}
              </Badge>
            ))}
            {diff.changedStopRoles?.map((r) => (
              <Badge key={`cr-${r}`} variant="warning" className="gap-1">
                {t("~ stop: {0}", r)}
              </Badge>
            ))}
          </div>
        )}

        {baseline && candidate && (
          <>
            <Separator />
            <div className="overflow-hidden rounded-md border">
              <ReactDiffViewer
                oldValue={JSON.stringify(baseline, null, 2)}
                newValue={JSON.stringify(candidate, null, 2)}
                splitView={false}
                compareMethod={DiffMethod.LINES}
                useDarkTheme
                hideLineNumbers={false}
                leftTitle="Baseline"
                rightTitle="Candidate"
              />
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
