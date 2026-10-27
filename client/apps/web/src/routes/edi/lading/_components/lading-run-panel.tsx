import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { unlessRefused } from "@/components/lading-editor/lading-service";
import { apiService } from "@/services/api";
import { PlayIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { HighlightedCode } from "@trenova/shared/components/ui/highlighted-code";
import { x12Spans } from "@trenova/shared/lib/highlight/x12";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  ladingRun,
  type LadingCheck,
  type LadingContext,
  type LadingProjectInput,
  type LadingRunInput,
  type LadingRunResult,
  type LadingRunValue,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import type { LadingProject, LadingVersion } from "@trenova/shared/types/lading";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { runValues, sampleInput } from "./studio-model";

type RunOutcome = { result: LadingRunResult | null; where: "browser" | "server" };

/**
 * Runs one map against one sample. Here, it runs the working copy in the browser and answers no
 * lookup; with live data, it runs the saved version on the server, its lookups answered from this
 * organisation's records. The values a run produced feed the editor's hover.
 */
export function LadingRunPanel({
  project,
  version,
  check,
  working,
  context,
  dirty,
  onValues,
}: {
  project: LadingProject;
  version: LadingVersion;
  check: LadingCheck;
  working: LadingProjectInput;
  context: LadingContext;
  dirty: boolean;
  onValues: (values: readonly LadingRunValue[]) => void;
}) {
  const t = useT();
  const edi = useEdi();
  const [mapName, setMapName] = useState(() => check.maps[0]?.name ?? "");
  const [samplePath, setSamplePath] = useState(() => working.samples[0]?.path ?? "");
  const [outcome, setOutcome] = useState<RunOutcome>();

  const map = check.maps.find((item) => item.name === mapName);
  const sample = working.samples.find((item) => item.path === samplePath);
  const ready = map !== undefined && sample !== undefined;

  const show = (next: RunOutcome) => {
    setOutcome(next);
    onValues(next.result ? runValues(next.result) : []);
  };

  const runHere = () => {
    if (!map || !sample) return;
    show({
      result: unlessRefused(
        () =>
          ladingRun(edi, {
            project: working,
            context,
            map: map.name,
            input: sampleInput(sample),
            settings: project.settings,
          }),
        null,
      ),
      where: "browser",
    });
  };

  const runLive = useApiMutation({
    mutationFn: (request: { map: string; input: LadingRunInput }) =>
      apiService.ladingService.run(version.id, request),
    onSuccess: (result: LadingRunResult) => show({ result, where: "server" }),
  });

  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex flex-wrap items-center gap-2">
        <Select
          value={mapName}
          items={check.maps.map((item) => ({ value: item.name, label: item.name }))}
          onValueChange={(next) => setMapName(next ?? "")}
        >
          <SelectTrigger className="w-48" aria-label={t("Map")}>
            <SelectValue placeholder={t("Pick a map")} />
          </SelectTrigger>
          <SelectContent>
            {check.maps.map((item) => (
              <SelectItem key={item.name} value={item.name}>
                {item.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select
          value={samplePath}
          items={working.samples.map((item) => ({ value: item.path, label: item.path }))}
          onValueChange={(next) => setSamplePath(next ?? "")}
        >
          <SelectTrigger className="w-56" aria-label={t("Sample")}>
            <SelectValue placeholder={t("Pick a sample")} />
          </SelectTrigger>
          <SelectContent>
            {working.samples.map((item) => (
              <SelectItem key={item.path} value={item.path}>
                {item.path}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button type="button" size="sm" disabled={!ready} onClick={runHere}>
          <PlayIcon className="size-3.5" />
          {t("Run here")}
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={!ready || dirty}
          title={dirty ? t("Save the draft to run it with live data") : undefined}
          isLoading={runLive.isPending}
          onClick={() => {
            if (map && sample) runLive.mutate({ map: map.name, input: sampleInput(sample) });
          }}
        >
          {t("Run with live data")}
        </Button>
      </div>
      {check.maps.length === 0 || working.samples.length === 0 ? (
        <p className="text-muted-foreground text-sm">
          {check.maps.length === 0
            ? t("The project has no maps to run")
            : t("Add a sample to run a map against it")}
        </p>
      ) : outcome === undefined ? (
        <p className="text-muted-foreground text-sm">
          {t(
            "Run here to try the files as they are now. Run with live data to try the saved version with lookups answered from your records.",
          )}
        </p>
      ) : outcome.result === null ? (
        <p className="text-danger text-sm">
          {t("The map could not run. Fix the project's errors and try again.")}
        </p>
      ) : (
        <RunResult result={outcome.result} where={outcome.where} />
      )}
    </div>
  );
}

function RunResult({ result, where }: { result: LadingRunResult; where: "browser" | "server" }) {
  const t = useT();
  const outcomeLabel = {
    reject: t("Rejects"),
    decline: t("Declines"),
    hold: t("Holds"),
    warn: t("Warns"),
  } as const;

  return (
    <div className="flex flex-col gap-3 text-sm">
      <p className="text-muted-foreground text-xs">
        {where === "browser"
          ? t("{0, plural, one {Ran here in # step} other {Ran here in # steps}}", result.steps)
          : t(
              "{0, plural, one {Ran with live data in # step} other {Ran with live data in # steps}}",
              result.steps,
            )}
      </p>
      {result.error ? <p className="text-danger">{translateMessage(result.error, t)}</p> : null}
      {result.transactions.map((transaction) => (
        <div key={transaction.index} className="flex flex-col gap-2 rounded-md border p-3">
          <div className="flex items-center gap-3">
            <span className="font-medium">
              {transaction.control
                ? t("Transaction {0}", transaction.control)
                : t("Transaction {0}", transaction.index + 1)}
            </span>
            <span
              className={cn(
                "text-xs",
                transaction.status === "accepted" ? "text-success" : "text-danger",
              )}
            >
              {transaction.status === "accepted" ? t("Accepted") : t("Rejected")}
            </span>
          </div>
          {transaction.findings.length > 0 ? (
            <ul className="flex flex-col gap-0.5 text-xs">
              {transaction.findings.map((finding, index) => (
                <li
                  // oxlint-disable-next-line react/no-array-index-key -- findings can repeat at one element
                  key={index}
                  className={finding.severity === "error" ? "text-danger" : "text-warning"}
                >
                  {translateMessage(finding.message, t)}
                </li>
              ))}
            </ul>
          ) : null}
          {transaction.outcomes.length > 0 ? (
            <ul className="flex flex-col gap-0.5 text-xs">
              {transaction.outcomes.map((item, index) => (
                // oxlint-disable-next-line react/no-array-index-key -- one rule can answer twice
                <li key={index}>
                  <span className="font-medium">{outcomeLabel[item.kind]}</span> {item.message}
                </li>
              ))}
            </ul>
          ) : null}
          {transaction.document !== null && transaction.document !== undefined ? (
            <HighlightedCode
              code={JSON.stringify(transaction.document, null, 2)}
              language="json"
              className="max-h-72 overflow-auto rounded-md border p-2 text-xs"
            />
          ) : null}
        </div>
      ))}
      {result.output ? (
        <div className="flex flex-col gap-2 rounded-md border p-3">
          <span className="font-medium">{t("Writes {0}", result.output.transaction)}</span>
          {result.output.findings.map((finding, index) => (
            <p
              // oxlint-disable-next-line react/no-array-index-key -- findings can repeat
              key={index}
              className={cn(
                "text-xs",
                finding.severity === "error" ? "text-danger" : "text-warning",
              )}
            >
              {translateMessage(finding.message, t)}
            </p>
          ))}
          <HighlightedCode
            code={result.output.text}
            language="x12"
            spans={x12Spans(result.output.text)}
            className="max-h-72 overflow-auto rounded-md border p-2 text-xs break-all whitespace-pre-wrap"
          />
        </div>
      ) : null}
      {result.needs.length > 0 ? (
        <div className="flex flex-col gap-1">
          <p className="font-medium">{t("Lookups the run asked for")}</p>
          <ul className="text-muted-foreground flex flex-col gap-0.5 font-mono text-xs">
            {result.needs.map((need) => (
              <li key={`${need.kind}:${need.code}`}>{t("{0} {1}", need.kind, need.code)}</li>
            ))}
          </ul>
        </div>
      ) : null}
      {result.unmapped.length > 0 ? (
        <div className="flex flex-col gap-1">
          <p className="font-medium">{t("Codes with no mapping")}</p>
          <ul className="text-muted-foreground flex flex-col gap-0.5 font-mono text-xs">
            {result.unmapped.map((item) => (
              <li key={`${item.kind}:${item.code}:${item.path}`}>
                {t("{0} {1} at {2}", item.kind, item.code, item.path)}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
