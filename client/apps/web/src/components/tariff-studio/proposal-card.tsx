import { checkTariffSource } from "@/components/formula-editor/tariff-service";
import { isTestItem } from "@/lib/tariff-outline";
import { refusalText, tariffMessageText } from "@/components/tariff-bill/refusal-text";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { apiService } from "@/services/api";
import type { SourceProposal } from "@/types/page-draft";
import { useQueries, useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  CheckIcon,
  Code01Icon,
  ReverseLeftIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatTariffValue } from "@trenova/shared/lib/tariff-value";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaPreviewResponse } from "@trenova/shared/types/formula-template";
import { useMemo } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { applyProposal, type ProposalTarget } from "./formula-draft";
import { diffHunks, lineDiff } from "./line-diff";
import { proposalCheck } from "./proposal-check";
import { TickedText } from "./ticked-text";
import { useTariffTests } from "./use-tariff-tests";

const PRICED_SHIPMENTS = 3;

/**
 * Where a proposal stands: in the editor (applied the moment it arrived, or by hand), waiting
 * (a reload, or one the checks refused), or set aside.
 */
export type ProposalState = "pending" | "applied" | "dismissed";

/** The fields of a Studio's form a proposal can change; templates and libraries share them. */
export type ProposalFormValues = { name?: string; description?: string; source?: string };

export function proposalTargetOf(values: ProposalFormValues): ProposalTarget {
  return {
    name: values.name ?? "",
    description: values.description ?? "",
    source: values.source ?? "",
  };
}

/** What a preview of a proposal billed, in a word or an amount. */
function pricedText(
  result: FormulaPreviewResponse,
  t: ReturnType<typeof useT>,
): {
  ok: boolean;
  text: string;
} {
  if (result.bill) return { ok: true, text: formatTariffValue(result.bill.total, t) };
  if (result.refusal) return { ok: false, text: refusalText(result.refusal, t) };
  if (result.error) return { ok: false, text: tariffMessageText(result.error.message, t) };
  const problem = result.diagnostics.find((finding) => finding.severity === "error");
  return {
    ok: false,
    text: problem ? translateMessage(problem.message, t) : t("Could not be rated"),
  };
}

/**
 * A change Nova made, drawn as a line diff against the file as it was, what Nova says it does,
 * and where it stands with this Studio's own check and tests (the server offered it only once it
 * checked clean and its tests passed). A tariff's proposal is also rated on the sample load and
 * three real shipments. Applied the moment it arrived; one click undoes it.
 */
export function ProposalCard({
  kind,
  proposal,
  state,
  before,
  onApply,
  onUndo,
  onDismiss,
}: {
  kind: "tariff" | "library";
  proposal: SourceProposal;
  state: ProposalState;
  /** The file as it was when the proposal was applied. */
  before: ProposalTarget | null;
  onApply: (next: ProposalTarget) => void;
  onUndo: () => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const rules = useRules();
  const { context, tablesStamp } = useTariffCheckContext();
  const { control } = useFormContext<ProposalFormValues>();
  const [name, description, source] = useWatch({
    control,
    name: ["name", "description", "source"],
  });
  const current = useMemo(
    () => proposalTargetOf({ name, description, source }),
    [name, description, source],
  );
  const base = state === "applied" && before ? before : current;
  const next = useMemo(() => applyProposal(rules, base, proposal), [rules, base, proposal]);
  const diff = useMemo(() => lineDiff(base.source, next.source), [base.source, next.source]);
  const hunks = useMemo(() => diffHunks(diff), [diff]);

  const check = useMemo(
    () => checkTariffSource(rules, next.source, kind, context),
    [rules, next.source, kind, context],
  );
  const errors = check.diagnostics.filter((finding) => finding.severity === "error").length;
  const testCount = check.outline?.items.filter(isTestItem).length ?? 0;
  const tests = useTariffTests({
    rules,
    source: next.source,
    context,
    tablesStamp,
    errors,
    testCount,
  });
  const verdict = proposalCheck(check, tests);
  const status =
    verdict.kind === "problem"
      ? { tone: "bad" as const, text: translateMessage(verdict.problem.message, t) }
      : verdict.kind === "failing"
        ? {
            tone: "warn" as const,
            text: t("Checks clean, but {0} of {1} tests fail.", verdict.failed, verdict.total),
          }
        : {
            tone: "ok" as const,
            text:
              testCount > 0
                ? t("Checks clean and every test passes.")
                : t("Checks clean. It has no tests yet."),
          };

  const priceable = kind === "tariff" && state !== "dismissed";
  const shipments = useQuery({
    queryKey: ["formula-proposal-shipments", PRICED_SHIPMENTS],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions(
        { resource: "SHIPMENT", initialLimit: PRICED_SHIPMENTS },
        { signal },
      ),
    enabled: priceable,
    staleTime: 5 * 60_000,
  });
  const loads = [
    {
      key: "sample",
      label: t("Sample load"),
      meta: "",
      shipmentId: undefined as string | undefined,
    },
    ...(shipments.data?.results ?? []).slice(0, PRICED_SHIPMENTS).map((option) => ({
      key: option.id,
      label: option.label,
      meta: option.description ?? "",
      shipmentId: option.id,
    })),
  ];
  const priced = useQueries({
    queries: loads.map((load) => ({
      queryKey: ["formula-proposal-price", next.source, load.key],
      queryFn: ({ signal }: { signal?: AbortSignal }) =>
        apiService.formulaTemplateService.preview(
          { source: next.source, ...(load.shipmentId ? { shipmentId: load.shipmentId } : {}) },
          { signal },
        ),
      enabled: priceable && errors === 0,
      staleTime: 60_000,
      retry: false,
    })),
  });

  if (state === "dismissed") return null;

  const changed = diff.some((line) => line.kind !== "same");
  const title = !base.source.trim()
    ? kind === "library"
      ? t("Library")
      : t("Tariff")
    : changed
      ? t("Changed the source")
      : t("Changed the details");
  const blocked = status.tone === "bad";

  return (
    <div className="bg-card flex flex-col overflow-hidden rounded-lg border">
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <Code01Icon className="text-foreground-subtle size-3.5" />
        <span className="text-sm font-semibold">{title}</span>
        {proposal.name && !base.name.trim() && (
          <span className="text-foreground-muted truncate text-xs">{proposal.name}</span>
        )}
        <span className="flex-1" />
        {state === "applied" && (
          <Badge variant="success" className="gap-1">
            <CheckIcon className="size-3" />
            {t("In the editor")}
          </Badge>
        )}
      </div>

      {(changed || !base.source.trim()) && (
        <div className="bg-sunken max-h-80 overflow-auto py-1.5 font-mono text-xs">
          {hunks.map((line, index) =>
            line.kind === "gap" ? (
              <div
                // oxlint-disable-next-line react/no-array-index-key -- a diff is positional
                key={index}
                className="text-foreground-subtle px-3 py-0.5 font-sans text-2xs select-none"
              >
                {t("{0} unchanged lines", line.count)}
              </div>
            ) : (
              <div
                // oxlint-disable-next-line react/no-array-index-key -- a diff is positional
                key={index}
                className={cn(
                  "flex gap-2 px-3 whitespace-pre",
                  line.kind === "add" && "bg-success-subtle text-success-subtle-foreground",
                  line.kind === "del" &&
                    "bg-danger-subtle text-danger-subtle-foreground line-through",
                )}
              >
                <span className="w-2 shrink-0 select-none" aria-hidden>
                  {line.kind === "add" ? "+" : line.kind === "del" ? "−" : ""}
                </span>
                <span>{line.text || " "}</span>
              </div>
            ),
          )}
        </div>
      )}

      {proposal.explanation !== "" && (
        <p className="border-t px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap">
          <TickedText text={proposal.explanation} />
        </p>
      )}

      <div
        className={cn(
          "flex items-start gap-1.5 border-t px-3 py-2 text-xs",
          status.tone === "ok" && "text-success",
          status.tone === "warn" && "text-warning",
          status.tone === "bad" && "text-danger",
        )}
      >
        {status.tone === "ok" ? (
          <CheckCircleIcon className="mt-px size-3.5 shrink-0" />
        ) : status.tone === "warn" ? (
          <AlertTriangleIcon className="mt-px size-3.5 shrink-0" />
        ) : (
          <XCircleIcon className="mt-px size-3.5 shrink-0" />
        )}
        <span>{status.text}</span>
      </div>

      {priceable && errors === 0 && (
        <div className="flex flex-col border-t">
          <div className="text-foreground-subtle flex justify-between px-3 pt-2 pb-1 text-2xs">
            <span>{t("Rated on the server")}</span>
            <span>{t("Total")}</span>
          </div>
          {loads.map((load, index) => {
            const query = priced[index];
            const result = query?.data ? pricedText(query.data, t) : null;
            return (
              <div key={load.key} className="flex items-center gap-2 px-3 py-1 text-xs">
                <span className="flex min-w-0 flex-1 items-baseline gap-1.5">
                  <b className={cn("font-medium", load.shipmentId && "font-mono")}>{load.label}</b>
                  {load.meta && (
                    <span className="text-foreground-subtle truncate">{load.meta}</span>
                  )}
                </span>
                {query?.isPending ? (
                  <span className="ui-shimmer h-3 w-14 rounded-sm" />
                ) : result ? (
                  <span
                    className={cn(
                      "max-w-40 truncate",
                      result.ok ? "font-mono tabular-nums" : "text-danger",
                    )}
                    title={result.text}
                  >
                    {result.text}
                  </span>
                ) : (
                  <span className="text-danger">{t("Could not be rated")}</span>
                )}
              </div>
            );
          })}
          <span className="pb-1.5" />
        </div>
      )}

      <div className="flex items-center gap-1.5 border-t px-3 py-2">
        {state === "applied" ? (
          <>
            <span className="text-foreground-muted flex-1 text-xs">
              {t("In the editor. Test it, then save — nothing is saved yet.")}
            </span>
            <Button type="button" variant="ghost" size="xs" className="gap-1" onClick={onUndo}>
              <ReverseLeftIcon className="size-3" />
              {t("Undo")}
            </Button>
          </>
        ) : (
          <>
            <span className="text-foreground-muted flex-1 text-xs">
              {blocked ? t("Not applied: it does not check.") : t("Not in the editor yet.")}
            </span>
            <Button type="button" size="xs" onClick={() => onApply(next)}>
              {t("Apply to editor")}
            </Button>
            <Button type="button" variant="quiet" size="xs" onClick={onDismiss}>
              {t("Dismiss")}
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
