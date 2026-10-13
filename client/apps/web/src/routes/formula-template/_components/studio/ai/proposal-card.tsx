import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { useFormulaSchema } from "@/hooks/use-formula-schema";
import { DEFAULT_TEST_VALUES } from "@/components/formula-editor/test-data-editor";
import { formulaTestRequest } from "@/lib/formula-test-request";
import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { apiService } from "@/services/api";
import type { FormulaProposal } from "@/types/page-draft";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  DEFAULT_ROUNDING_MODE,
  DEFAULT_ROUNDING_PRECISION,
  type FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  CheckIcon,
  Code01Icon,
  ReverseLeftIcon,
  Scales01Icon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { useQueries, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useFormContext } from "react-hook-form";
import { analyzeExpression } from "../model/expression-analysis";
import { formatMoney, formatStudioValue, numericAmount, roundingSentence } from "../model/format";
import { lineDiff } from "../model/line-diff";
import {
  applyProposal,
  newProposalVariables,
  policyChanged,
  type ProposalTarget,
} from "./formula-draft";
import { ProposedScenariosCard } from "./proposed-scenarios-card";
import { TickedText } from "./ticked-text";

const PRICED_SHIPMENTS = 3;

/**
 * Where a proposal stands: in the editor (applied the moment it arrived, or
 * by hand), waiting (a reload, or one the engine could not run), or set aside.
 */
export type ProposalState = "pending" | "applied" | "dismissed";

export function proposalTargetOf(values: FormulaTemplateFormValues): ProposalTarget {
  return {
    name: values.name ?? "",
    description: values.description ?? "",
    type: values.type,
    expression: values.expression ?? "",
    variableDefinitions: (values.variableDefinitions ?? []).map((variable) => ({
      name: variable.name,
      type: variable.type,
      description: variable.description ?? "",
      required: variable.required ?? false,
      defaultValue: variable.defaultValue,
    })),
    breakdownDefinitions: (values.breakdownDefinitions ?? []).map((line) => ({
      name: line.name ?? "",
      label: line.label ?? "",
      expression: line.expression ?? "",
    })),
    minCharge: numericAmount(values.minCharge),
    maxCharge: numericAmount(values.maxCharge),
    roundingMode: values.roundingMode ?? DEFAULT_ROUNDING_MODE,
    roundingPrecision: values.roundingPrecision ?? DEFAULT_ROUNDING_PRECISION,
  };
}

function PolicyLine({ before, after }: { before: ProposalTarget; after: ProposalTarget }) {
  const t = useT();
  const bound = (value: number | null, none: string) =>
    value === null ? none : formatMoney(value);
  const parts: string[] = [];
  if (before.minCharge !== after.minCharge) {
    parts.push(
      t("minimum {0} → {1}", bound(before.minCharge, t("none")), bound(after.minCharge, t("none"))),
    );
  }
  if (before.maxCharge !== after.maxCharge) {
    parts.push(
      t("maximum {0} → {1}", bound(before.maxCharge, t("none")), bound(after.maxCharge, t("none"))),
    );
  }
  if (
    before.roundingMode !== after.roundingMode ||
    before.roundingPrecision !== after.roundingPrecision
  ) {
    parts.push(roundingSentence(after.roundingMode, after.roundingPrecision, t).toLowerCase());
  }
  return (
    <div className="flex items-start gap-1.5 border-t px-3 py-2 text-xs">
      <Scales01Icon className="text-foreground-subtle mt-px size-3.5 shrink-0" />
      <span>
        <b className="font-medium">{t("Charge policy")}</b>
        <span className="text-foreground-muted"> · {parts.join(" · ")}</span>
      </span>
    </div>
  );
}

/**
 * A change Nova made, drawn as a diff against the editor as it was: the
 * lines it adds and removes, the inputs it needs, the charge policy it moved,
 * what it says it does, whether every name resolves, and what the engine
 * bills the sample load and three real shipments with it. Applied the moment
 * it arrived; one click undoes it.
 */
export function ProposalCard({
  proposal,
  state,
  before,
  onApply,
  onUndo,
  onDismiss,
}: {
  proposal: FormulaProposal;
  state: ProposalState;
  /** The editor as it was when the proposal was applied. */
  before: ProposalTarget | null;
  onApply: (next: ProposalTarget) => void;
  onUndo: () => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const form = useFormContext<FormulaTemplateFormValues>();
  const values = form.watch();
  const current = proposalTargetOf(values);
  const base = state === "applied" && before ? before : current;
  const next = applyProposal(base, proposal);
  const diff = useMemo(
    () => lineDiff(base.expression, proposal.expression),
    [base.expression, proposal.expression],
  );
  const added = newProposalVariables(base.variableDefinitions, proposal);
  const movedPolicy = policyChanged(base, next);

  const { data: schema } = useFormulaSchema(values.schemaId || "shipment");
  const analysis = useMemo(() => {
    const nextKnown = buildKnownIdentifiers(schema, next.variableDefinitions);
    return analyzeExpression(next.expression, nextKnown);
  }, [next.expression, next.variableDefinitions, schema]);
  const check = !proposal.check.valid
    ? { tone: "bad" as const, text: proposal.check.error || t("The engine could not run it.") }
    : analysis.errorCount > 0
      ? {
          tone: "warn" as const,
          text: analysis.problems.find((problem) => problem.severity === "error")?.message ?? "",
        }
      : {
          tone: "ok" as const,
          text: t("Checked against the shipment schema. Every name resolves."),
        };

  const content = {
    expression: next.expression,
    schemaId: values.schemaId || "shipment",
    variableDefinitions: next.variableDefinitions,
    breakdownDefinitions: next.breakdownDefinitions,
    minCharge: next.minCharge,
    maxCharge: next.maxCharge,
    roundingMode: next.roundingMode,
    roundingPrecision: next.roundingPrecision,
  };

  const shipments = useQuery({
    queryKey: ["formula-proposal-shipments", PRICED_SHIPMENTS],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions(
        { resource: "SHIPMENT", initialLimit: PRICED_SHIPMENTS },
        { signal },
      ),
    staleTime: 5 * 60_000,
  });
  const loads = [
    {
      key: "sample",
      label: t("Sample load"),
      meta: t(
        "{0} mi · {1} stops",
        formatStudioValue(DEFAULT_TEST_VALUES.totalDistance),
        formatStudioValue(DEFAULT_TEST_VALUES.totalStops),
      ),
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
      queryKey: ["formula-proposal-price", content, load.key],
      queryFn: () => {
        const request = formulaTestRequest(content, {
          sampleValues: load.shipmentId ? null : DEFAULT_TEST_VALUES,
          shipmentId: load.shipmentId,
        });
        if (!request) throw new Error("Nothing to price");
        return apiService.formulaTemplateService.test(request);
      },
      enabled: state !== "dismissed" && proposal.check.valid,
      staleTime: 60_000,
      retry: false,
    })),
  });

  if (state === "dismissed") return null;

  const expressionChanged = diff.some((line) => line.kind !== "same");
  const title = !base.expression.trim()
    ? t("Formula")
    : expressionChanged
      ? t("Changed the formula")
      : movedPolicy
        ? t("Changed the charge policy")
        : t("Changed the template");

  return (
    <div className={cn("bg-card flex flex-col overflow-hidden rounded-lg border")}>
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

      {(expressionChanged || !base.expression.trim()) && (
        <div className="bg-sunken overflow-x-auto py-1.5 font-mono text-xs">
          {diff.map((line, index) => (
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
          ))}
        </div>
      )}

      {added.length > 0 && (
        <div className="flex flex-wrap gap-1.5 border-t px-3 py-2">
          {added.map((variable) => (
            <span
              key={variable.name}
              className="bg-surface-hover flex items-center gap-1.5 rounded-md px-2 py-0.5 text-xs"
            >
              <b className="font-mono font-medium">{variable.name}</b>
              <span className="text-foreground-muted">
                {t(variable.type)} = {formatStudioValue(variable.defaultValue)}
              </span>
              <span className="bg-success-subtle text-success-subtle-foreground rounded-full px-1.5 text-2xs">
                {t("new input")}
              </span>
            </span>
          ))}
        </div>
      )}

      {movedPolicy && <PolicyLine before={base} after={next} />}

      {proposal.explanation !== "" && (
        <p className="border-t px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap">
          <TickedText text={proposal.explanation} />
        </p>
      )}

      <div
        className={cn(
          "flex items-start gap-1.5 border-t px-3 py-2 text-xs",
          check.tone === "ok" && "text-success",
          check.tone === "warn" && "text-warning",
          check.tone === "bad" && "text-danger",
        )}
      >
        {check.tone === "ok" ? (
          <CheckCircleIcon className="mt-px size-3.5 shrink-0" />
        ) : check.tone === "warn" ? (
          <AlertTriangleIcon className="mt-px size-3.5 shrink-0" />
        ) : (
          <XCircleIcon className="mt-px size-3.5 shrink-0" />
        )}
        <span>{check.text}</span>
      </div>

      {proposal.check.valid && (
        <div className="flex flex-col border-t">
          <div className="text-foreground-subtle flex justify-between px-3 pt-2 pb-1 text-2xs">
            <span>{t("Priced by the formula engine")}</span>
            <span>{t("Billed")}</span>
          </div>
          {loads.map((load, index) => {
            const query = priced[index];
            const amount = query?.data?.valid ? numericAmount(query.data.result) : null;
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
                ) : amount !== null ? (
                  <span className="font-mono tabular-nums">{formatMoney(amount)}</span>
                ) : (
                  <span
                    className="text-danger max-w-36 truncate"
                    title={query?.data?.error ?? undefined}
                  >
                    {query?.data?.error ?? t("Could not be priced")}
                  </span>
                )}
              </div>
            );
          })}
          <span className="pb-1.5" />
        </div>
      )}

      {proposal.scenarios.length > 0 && (
        <div className="border-t p-2">
          <ProposedScenariosCard scenarios={proposal.scenarios} />
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
              {check.tone === "bad"
                ? t("Not applied: the engine could not run it.")
                : t("Not in the editor yet.")}
            </span>
            <Button
              type="button"
              size="xs"
              disabled={check.tone === "bad"}
              onClick={() => onApply(next)}
            >
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
