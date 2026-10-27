import { useT } from "@trenova/shared/i18n/use-t";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { unlessRefused } from "@/components/lading-editor/lading-service";
import { apiService } from "@/services/api";
import {
  ArrowNarrowRightIcon,
  Database01Icon,
  Dataflow03Icon,
  File06Icon,
  PlayIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
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
  type LadingOutcome,
  type LadingProjectInput,
  type LadingRunInput,
  type LadingRunResult,
  type LadingRunValue,
  type LadingTransactionResult,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import type { LadingProject, LadingVersion } from "@trenova/shared/types/lading";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import {
  CodeBlock,
  DiffList,
  MetaList,
  PanelEmpty,
  PanelHeading,
  PanelToolbar,
  SeverityIcon,
  StatusIcon,
} from "./lading-panel-kit";
import { runValues, sampleInput } from "./studio-model";

type RunOutcome = { result: LadingRunResult | null; where: "browser" | "server" };

/** A run as shown: its outcome, numbered so each new run opens on its first transaction. */
type ShownRun = RunOutcome & { run: number };

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
  const [outcome, setOutcome] = useState<ShownRun>();

  const map = check.maps.find((item) => item.name === mapName);
  const sample = working.samples.find((item) => item.path === samplePath);
  const ready = map !== undefined && sample !== undefined;

  const show = (next: RunOutcome) => {
    setOutcome((current) => ({ ...next, run: (current?.run ?? 0) + 1 }));
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

  if (check.maps.length === 0 || working.samples.length === 0) {
    return (
      <PanelEmpty
        icon={check.maps.length === 0 ? Dataflow03Icon : File06Icon}
        title={
          check.maps.length === 0
            ? t("The project has no maps to run")
            : t("Add a sample to run a map against it")
        }
        description={t(
          "A run reads one sample through one map and shows the document, the X12 it writes, and every finding along the way.",
        )}
      />
    );
  }

  return (
    <div className="flex flex-col">
      <PanelToolbar
        end={
          <>
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
              <Database01Icon className="size-3.5" />
              {t("Run with live data")}
            </Button>
          </>
        }
      >
        <Select
          value={mapName}
          items={check.maps.map((item) => ({ value: item.name, label: item.name }))}
          onValueChange={(next) => setMapName(next ?? "")}
        >
          <SelectTrigger className="w-52" aria-label={t("Map")}>
            <SelectValue placeholder={t("Pick a map")} />
          </SelectTrigger>
          <SelectContent>
            {check.maps.map((item) => (
              <SelectItem key={item.name} value={item.name}>
                <span className="font-mono text-xs">{item.name}</span>
                <span className="text-muted-foreground ml-auto pl-3 text-xs">
                  {item.direction === "inbound"
                    ? t("{0} in", item.set ?? item.source.toUpperCase())
                    : t("{0} out", item.set ?? item.document)}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <ArrowNarrowRightIcon aria-hidden className="text-muted-foreground size-4" />
        <Select
          value={samplePath}
          items={working.samples.map((item) => ({ value: item.path, label: item.path }))}
          onValueChange={(next) => setSamplePath(next ?? "")}
        >
          <SelectTrigger className="w-64" aria-label={t("Sample")}>
            <SelectValue placeholder={t("Pick a sample")} />
          </SelectTrigger>
          <SelectContent>
            {working.samples.map((item) => (
              <SelectItem key={item.path} value={item.path}>
                <span className="font-mono text-xs">{item.path}</span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </PanelToolbar>
      {outcome === undefined ? (
        <PanelEmpty
          icon={PlayIcon}
          title={t("Run a map on a sample")}
          description={t(
            "Run here to try the files as they are now. Run with live data to try the saved version with lookups answered from your records.",
          )}
        />
      ) : outcome.result === null ? (
        <PanelEmpty
          icon={XCircleIcon}
          tone="danger"
          title={t("The map could not run")}
          description={t("Fix the project's errors and try again.")}
        />
      ) : (
        <RunResult key={outcome.run} result={outcome.result} where={outcome.where} />
      )}
    </div>
  );
}

function RunResult({ result, where }: { result: LadingRunResult; where: "browser" | "server" }) {
  const t = useT();
  const [picked, setPicked] = useState(0);
  const rejected = result.transactions.filter((item) => item.status === "rejected").length;
  const transaction = result.transactions[picked] ?? result.transactions[0];
  const failed = result.error !== undefined || rejected > 0;

  return (
    <div className="flex flex-col gap-3 p-3">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <span className="flex items-center gap-2 text-sm font-medium">
          <StatusIcon status={failed ? "failed" : "passed"} className="size-4" />
          {result.error
            ? t("The run stopped")
            : rejected > 0
              ? t("{0} of {1} transactions rejected", rejected, result.transactions.length)
              : result.output
                ? t("Wrote {0}", result.output.transaction)
                : t("Every transaction accepted")}
        </span>
        <Badge variant={where === "server" ? "info" : "neutral"}>
          {where === "server" ? t("Live data") : t("In the browser")}
        </Badge>
        <MetaList
          items={[
            { label: t("Steps"), value: result.steps },
            {
              label: t("Transactions"),
              value: result.transactions.length,
              hidden: result.transactions.length === 0,
            },
            { label: t("Lookups"), value: result.needs.length, hidden: result.needs.length === 0 },
            {
              label: t("Unmapped codes"),
              value: result.unmapped.length,
              hidden: result.unmapped.length === 0,
            },
          ]}
        />
      </div>

      {result.error ? (
        <Alert variant="destructive" size="sm">
          <AlertDescription>{translateMessage(result.error, t)}</AlertDescription>
        </Alert>
      ) : null}

      {transaction ? (
        <div
          className={cn(
            "grid min-w-0 overflow-hidden rounded-md border",
            result.transactions.length > 1 && "md:grid-cols-[13rem_minmax(0,1fr)]",
          )}
        >
          {result.transactions.length > 1 ? (
            <ul
              aria-label={t("Transactions")}
              className="flex flex-col border-b md:border-r md:border-b-0"
            >
              {result.transactions.map((item, index) => (
                <li key={item.index}>
                  <button
                    type="button"
                    aria-current={index === picked ? "true" : undefined}
                    onClick={() => setPicked(index)}
                    className={cn(
                      "ui-focus-ring flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm",
                      index === picked ? "bg-surface-active" : "hover:bg-surface-hover",
                    )}
                  >
                    <StatusIcon status={item.status === "accepted" ? "passed" : "failed"} />
                    <span className="min-w-0 flex-1 truncate">{transactionName(item, t)}</span>
                    {item.findings.length + item.outcomes.length > 0 ? (
                      <span className="text-muted-foreground text-xs tabular-nums">
                        {item.findings.length + item.outcomes.length}
                      </span>
                    ) : null}
                  </button>
                </li>
              ))}
            </ul>
          ) : null}
          <TransactionDetail transaction={transaction} />
        </div>
      ) : null}

      {result.output ? (
        <section className="flex flex-col gap-2">
          {result.output.findings.length > 0 ? (
            <ul className="flex flex-col gap-1">
              {result.output.findings.map((finding, index) => (
                <li
                  // oxlint-disable-next-line react/no-array-index-key -- findings can repeat
                  key={index}
                  className="flex items-start gap-2 text-sm"
                >
                  <SeverityIcon severity={finding.severity} className="mt-0.5" />
                  <span className="min-w-0 flex-1">{translateMessage(finding.message, t)}</span>
                  <span className="text-muted-foreground font-mono text-xs">{finding.code}</span>
                </li>
              ))}
            </ul>
          ) : null}
          <Outcomes outcomes={result.output.outcomes} />
          <CodeBlock title={t("Writes {0}", result.output.transaction)} code={result.output.text} />
        </section>
      ) : null}

      {result.needs.length > 0 || result.unmapped.length > 0 ? (
        <div className="grid gap-3 md:grid-cols-2">
          {result.needs.length > 0 ? (
            <section className="flex flex-col gap-2">
              <PanelHeading
                title={t("Lookups the run asked for")}
                count={result.needs.length}
                help={t("Run with live data to answer them from your records")}
              />
              <ul className="flex flex-col divide-y rounded-md border text-xs">
                {result.needs.map((need) => (
                  <li
                    key={`${need.kind}:${need.code}`}
                    className="flex items-center gap-2 px-3 py-1.5"
                  >
                    <Badge variant="neutral" appearance="outline">
                      {need.kind}
                    </Badge>
                    <span className="font-mono">{need.code}</span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
          {result.unmapped.length > 0 ? (
            <section className="flex flex-col gap-2">
              <PanelHeading
                title={t("Codes with no mapping")}
                count={result.unmapped.length}
                help={t("A message with these codes waits until someone maps them")}
              />
              <ul className="flex flex-col divide-y rounded-md border text-xs">
                {result.unmapped.map((item) => (
                  <li
                    key={`${item.kind}:${item.code}:${item.path}`}
                    className="flex items-center gap-2 px-3 py-1.5"
                  >
                    <Badge variant="warning">{item.kind}</Badge>
                    <span className="font-mono">{item.code}</span>
                    <span className="text-muted-foreground ml-auto truncate font-mono">
                      {item.path}
                    </span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function transactionName(
  transaction: LadingTransactionResult,
  t: ReturnType<typeof useT>,
): string {
  return t("Transaction {0}", transaction.control ?? String(transaction.index + 1));
}

function TransactionDetail({ transaction }: { transaction: LadingTransactionResult }) {
  const t = useT();
  return (
    <div className="flex min-w-0 flex-col gap-3 p-3">
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">{transactionName(transaction, t)}</span>
        <Badge variant={transaction.status === "accepted" ? "success" : "danger"}>
          {transaction.status === "accepted" ? t("Accepted") : t("Rejected")}
        </Badge>
      </div>
      {transaction.findings.length > 0 ? (
        <ul className="flex flex-col gap-1">
          {transaction.findings.map((finding, index) => (
            <li
              // oxlint-disable-next-line react/no-array-index-key -- findings can repeat at one element
              key={index}
              className="flex items-start gap-2 text-sm"
            >
              <SeverityIcon severity={finding.severity} className="mt-0.5" />
              <span className="min-w-0 flex-1">{translateMessage(finding.message, t)}</span>
              {finding.segmentId ? (
                <span className="text-muted-foreground font-mono text-xs">
                  {finding.element !== undefined
                    ? `${finding.segmentId}${String(finding.element).padStart(2, "0")}`
                    : finding.segmentId}
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
      <Outcomes outcomes={transaction.outcomes} />
      {transaction.fixes.length > 0 ? (
        <section className="flex flex-col gap-1.5">
          <PanelHeading title={t("Fixed by rules")} count={transaction.fixes.length} />
          <DiffList differences={transaction.fixes} />
        </section>
      ) : null}
      {transaction.document !== null && transaction.document !== undefined ? (
        <CodeBlock
          title={t("Document")}
          language="json"
          code={JSON.stringify(transaction.document, null, 2)}
        />
      ) : null}
    </div>
  );
}

const OUTCOME_TONE: Record<LadingOutcome["kind"], "danger" | "warning" | "info"> = {
  reject: "danger",
  decline: "warning",
  hold: "warning",
  warn: "info",
};

/** What the rules decided, each as its kind, its message and the rule that said it. */
function Outcomes({ outcomes }: { outcomes: readonly LadingOutcome[] }) {
  const t = useT();
  if (outcomes.length === 0) return null;
  const label: Record<LadingOutcome["kind"], string> = {
    reject: t("Rejects"),
    decline: t("Declines"),
    hold: t("Holds"),
    warn: t("Warns"),
  };
  return (
    <ul className="flex flex-col gap-1">
      {outcomes.map((item, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- one rule can answer twice
        <li key={index} className="flex items-start gap-2 text-sm">
          <Badge variant={OUTCOME_TONE[item.kind]} className="mt-px">
            {label[item.kind]}
          </Badge>
          <span className="min-w-0 flex-1">{item.message}</span>
          {item.rule ? (
            <span className="text-muted-foreground truncate text-xs">{item.rule}</span>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
