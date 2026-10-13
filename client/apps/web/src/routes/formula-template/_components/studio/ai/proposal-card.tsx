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
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  CheckIcon,
  Code01Icon,
  ReverseLeftIcon,
  XCircleIcon,
} from "@trenova/shared/components/icons";
import { useQueries, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useFormContext } from "react-hook-form";
import { analyzeExpression } from "../model/expression-analysis";
import { formatMoney, formatStudioValue, numericAmount } from "../model/format";
import { lineDiff } from "../model/line-diff";
import { applyProposal, newProposalVariables, type ProposalTarget } from "./formula-draft";
import { ProposedScenariosCard } from "./proposed-scenarios-card";
import { TickedText } from "./ticked-text";

const PRICED_SHIPMENTS = 3;

export type ProposalState = "pending" | "inserted" | "dismissed";

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
  };
}

/**
 * A change Nova proposed, drawn as a diff against the editor: the lines it
 * adds and removes, the inputs it needs, what it says it does, whether every
 * name resolves, and what the engine bills the sample load and three real
 * shipments with it. Nothing reaches the editor until it is inserted, and an
 * insert can be undone.
 */
export function ProposalCard({
  proposal,
  state,
  before,
  onInsert,
  onUndo,
  onDismiss,
}: {
  proposal: FormulaProposal;
  state: ProposalState;
  /** The expression the editor held when the proposal was inserted. */
  before: string | null;
  onInsert: (next: ProposalTarget) => void;
  onUndo: () => void;
  onDismiss: () => void;
}) {
  const t = useT();
  const form = useFormContext<FormulaTemplateFormValues>();
  const values = form.watch();
  const current = proposalTargetOf(values);
  const next = applyProposal(current, proposal);
  const baseExpression = state === "inserted" ? (before ?? "") : current.expression;
  const diff = useMemo(
    () => lineDiff(baseExpression, proposal.expression),
    [baseExpression, proposal.expression],
  );
  const added =
    state === "inserted"
      ? proposal.variables
      : newProposalVariables(current.variableDefinitions, proposal);

  const { data: schema } = useFormulaSchema(values.schemaId || "shipment");
  const analysis = useMemo(() => {
    const nextKnown = buildKnownIdentifiers(schema, next.variableDefinitions);
    return analyzeExpression(
      next.expression,
      nextKnown,
      new Set(next.variableDefinitions.map((variable) => variable.name)),
    );
  }, [next.expression, next.variableDefinitions, schema]);
  const check = !proposal.check.valid
    ? { tone: "bad" as const, text: proposal.check.error || t("The engine could not run it.") }
    : analysis.problems.length > 0
      ? { tone: "warn" as const, text: analysis.problems[0]?.message ?? "" }
      : {
          tone: "ok" as const,
          text: t("Checked against the shipment schema. Every name resolves."),
        };

  const content = {
    expression: next.expression,
    schemaId: values.schemaId || "shipment",
    variableDefinitions: next.variableDefinitions,
    breakdownDefinitions: next.breakdownDefinitions,
    minCharge: values.minCharge,
    maxCharge: values.maxCharge,
    roundingMode: values.roundingMode,
    roundingPrecision: values.roundingPrecision,
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

  return (
    <div
      className={cn(
        "bg-card flex flex-col overflow-hidden rounded-lg border",
        state === "inserted" && "opacity-90",
      )}
    >
      <div className="flex items-center gap-2 border-b px-3 py-2">
        <Code01Icon className="text-foreground-subtle size-3.5" />
        <span className="text-sm font-semibold">
          {baseExpression.trim() ? t("Proposed change") : t("Proposed formula")}
        </span>
        {proposal.name && !current.name.trim() && state !== "inserted" && (
          <span className="text-foreground-muted truncate text-xs">{proposal.name}</span>
        )}
        <span className="flex-1" />
        {state === "inserted" && (
          <Badge variant="success" className="gap-1">
            <CheckIcon className="size-3" />
            {t("In the editor")}
          </Badge>
        )}
      </div>

      <div className="bg-sunken overflow-x-auto py-1.5 font-mono text-xs">
        {diff.map((line, index) => (
          <div
            // oxlint-disable-next-line react/no-array-index-key -- a diff is positional
            key={index}
            className={cn(
              "flex gap-2 px-3 whitespace-pre",
              line.kind === "add" && "bg-success-subtle text-success-subtle-foreground",
              line.kind === "del" && "bg-danger-subtle text-danger-subtle-foreground line-through",
            )}
          >
            <span className="w-2 shrink-0 select-none" aria-hidden>
              {line.kind === "add" ? "+" : line.kind === "del" ? "−" : ""}
            </span>
            <span>{line.text || " "}</span>
          </div>
        ))}
      </div>

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

      {proposal.scenarios.length > 0 && state !== "inserted" && (
        <div className="border-t p-2">
          <ProposedScenariosCard scenarios={proposal.scenarios} />
        </div>
      )}

      <div className="flex items-center gap-1.5 border-t px-3 py-2">
        {state === "inserted" ? (
          <>
            <span className="text-foreground-muted flex-1 text-xs">
              {t("Review and test it, then save. Nothing is saved yet.")}
            </span>
            <Button type="button" variant="ghost" size="xs" className="gap-1" onClick={onUndo}>
              <ReverseLeftIcon className="size-3" />
              {t("Undo")}
            </Button>
          </>
        ) : (
          <>
            <Button
              type="button"
              size="sm"
              disabled={check.tone === "bad"}
              onClick={() => onInsert(next)}
            >
              {t("Insert into editor")}
            </Button>
            <Button type="button" variant="quiet" size="sm" onClick={onDismiss}>
              {t("Dismiss")}
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
