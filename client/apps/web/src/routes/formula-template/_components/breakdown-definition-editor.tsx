import type { BreakdownReconciliation } from "@/components/formula-editor/breakdown-reconcile";
import { ExpressionEditor } from "@/components/formula-editor/expression-editor";
import type { DeclareInput } from "@/components/formula-editor/expr-lint";
import type { KnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  MAX_BREAKDOWN_DEFINITIONS,
  type BreakdownDefinitionInput,
  type TestBreakdownItem,
} from "@trenova/shared/types/formula-template";
import {
  AlertTriangleIcon,
  CheckCircleIcon,
  PlusIcon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { useState, type ReactNode } from "react";
import { Controller, useFieldArray, useWatch, type Control } from "react-hook-form";
import { keyFromLabel, sanitizeIdentifier } from "./studio/model/expression-analysis";
import { formatMoney } from "./studio/model/format";

type FormWithBreakdowns = {
  breakdownDefinitions: BreakdownDefinitionInput[];
};

export type BreakdownPricing = {
  /** Each line as the preview priced it, by key. */
  lines: ReadonlyMap<string, TestBreakdownItem>;
  reconciliation: BreakdownReconciliation | null;
  /** The billed amount, when a guardrail moved it away from the formula's result. */
  clamped: { bound: "min" | "max"; billed: number } | null;
};

type BreakdownDefinitionEditorProps = {
  control: Control<FormWithBreakdowns>;
  knownIdentifiers?: KnownIdentifiers;
  pricing: BreakdownPricing | null;
  /** The section's head, handed the editor's own add so both stay one field array. */
  head?: (controls: { add: () => void; atLimit: boolean }) => ReactNode;
  /** Declares an unknown name as an input, from a quick fix in a line's editor. */
  declareInput?: DeclareInput;
};

function Reconcile({ pricing }: { pricing: BreakdownPricing }) {
  const t = useT();
  const reconciliation = pricing.reconciliation;
  if (!reconciliation) return null;
  const ok = reconciliation.balanced && !reconciliation.clampMismatch;
  const message = reconciliation.clampMismatch
    ? pricing.clamped?.bound === "max"
      ? t(
          "Lines add up to the formula's {0}, but the maximum bills {1}. An invoice built from these lines won't match the charge.",
          formatMoney(reconciliation.sum),
          formatMoney(pricing.clamped.billed),
        )
      : t(
          "Lines add up to the formula's {0}, but the minimum bills {1}. An invoice built from these lines won't match the charge.",
          formatMoney(reconciliation.sum),
          formatMoney(pricing.clamped?.billed ?? 0),
        )
    : reconciliation.balanced
      ? t("Lines explain the total")
      : reconciliation.failedCount > 0
        ? t(
            "{0, plural, one {# line fails} other {# lines fail}} on the preview load",
            reconciliation.failedCount,
          )
        : reconciliation.residual > 0
          ? t("{0} of the total isn't on any line", formatMoney(reconciliation.residual))
          : t("Lines exceed the total by {0}", formatMoney(-reconciliation.residual));

  return (
    <div
      className={cn(
        "flex items-start gap-2 border-t px-3 py-2 text-xs",
        ok ? "text-foreground-muted" : "bg-warning-subtle text-warning-subtle-foreground",
      )}
    >
      {ok ? (
        <CheckCircleIcon className="text-success mt-px size-3.5 shrink-0" />
      ) : (
        <AlertTriangleIcon className="text-warning mt-px size-3.5 shrink-0" />
      )}
      <span className="min-w-0 flex-1">{message}</span>
      <span className="font-mono tabular-nums">{formatMoney(reconciliation.sum)}</span>
    </div>
  );
}

/**
 * The total itemized into named lines for invoices and disputes: a label, a
 * short expression and what it comes to on the preview load, with the key an
 * invoice line carries underneath, followed from the label until it is set by
 * hand. The footer says whether the lines explain the total.
 */
export function BreakdownDefinitionEditor({
  control,
  knownIdentifiers,
  pricing,
  head,
  declareInput,
}: BreakdownDefinitionEditorProps) {
  const t = useT();
  const { fields, append, remove } = useFieldArray({ control, name: "breakdownDefinitions" });
  const values = useWatch({ control, name: "breakdownDefinitions" }) ?? [];
  const [touchedKeys, setTouchedKeys] = useState<ReadonlySet<string>>(() => new Set());
  const atLimit = fields.length >= MAX_BREAKDOWN_DEFINITIONS;

  const add = () => append({ name: "", label: "", expression: "" });

  const heading = head?.({ add, atLimit });

  if (fields.length === 0) {
    return (
      <>
        {heading}
        <div className="text-foreground-muted flex items-center justify-between gap-3 rounded-lg border border-dashed px-3 py-2.5 text-sm">
          <span>{t("No lines. The invoice shows one amount for this charge.")}</span>
          <Button type="button" variant="ghost" size="xs" className="gap-1" onClick={add}>
            <PlusIcon className="size-3" />
            {t("Add line")}
          </Button>
        </div>
      </>
    );
  }

  return (
    <>
      {heading}
      <div className="bg-card overflow-hidden rounded-lg border">
        {fields.map((field, index) => {
          const current = values[index] ?? field;
          const line = pricing?.lines.get(current.name ?? "");
          const keyFollowsLabel =
            !touchedKeys.has(field.id) &&
            (current.name === "" || current.name === keyFromLabel(current.label ?? ""));
          return (
            <div
              key={field.id}
              className="grid grid-cols-[minmax(0,10rem)_minmax(0,1fr)_5.5rem_1.75rem] items-start gap-2 border-b px-3 py-2 last:border-b-0"
            >
              <div className="flex min-w-0 flex-col gap-1">
                <Controller
                  control={control}
                  name={`breakdownDefinitions.${index}.label`}
                  render={({ field: labelField }) => (
                    <Controller
                      control={control}
                      name={`breakdownDefinitions.${index}.name`}
                      render={({ field: nameField }) => (
                        <Input
                          ref={labelField.ref}
                          value={labelField.value ?? ""}
                          onBlur={labelField.onBlur}
                          onChange={(event) => {
                            labelField.onChange(event.target.value);
                            if (keyFollowsLabel) {
                              nameField.onChange(keyFromLabel(event.target.value));
                            }
                          }}
                          aria-label={t("Line label")}
                          placeholder={t("Fuel surcharge")}
                        />
                      )}
                    />
                  )}
                />
                <Controller
                  control={control}
                  name={`breakdownDefinitions.${index}.name`}
                  render={({ field: nameField, fieldState }) => (
                    <Input
                      ref={nameField.ref}
                      value={nameField.value ?? ""}
                      onBlur={nameField.onBlur}
                      onChange={(event) => {
                        nameField.onChange(sanitizeIdentifier(event.target.value));
                        setTouchedKeys((previous) => new Set(previous).add(field.id));
                      }}
                      aria-label={t("Line key")}
                      aria-invalid={fieldState.invalid || undefined}
                      placeholder={t("key")}
                      title={fieldState.error?.message}
                      className="text-foreground-subtle h-5.5 border-transparent bg-transparent px-1 font-mono text-2xs md:text-2xs"
                    />
                  )}
                />
              </div>
              <ExpressionEditor
                name={`breakdownDefinitions.${index}.expression`}
                control={control as never}
                variant="mini"
                knownIdentifiers={knownIdentifiers}
                declareInput={declareInput}
                placeholder="baseRate * totalDistance"
                aria-label={t("Line expression")}
              />
              <div className="flex h-7 items-center justify-end">
                {line?.error ? (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <span className="text-danger cursor-default font-mono text-sm">
                          {t("Fails")}
                        </span>
                      }
                    />
                    <TooltipContent>{line.error}</TooltipContent>
                  </Tooltip>
                ) : (
                  <span className="font-mono text-sm tabular-nums">
                    {line ? formatMoney(line.amount) : "—"}
                  </span>
                )}
              </div>
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                className="mt-0.5"
                aria-label={t("Remove line {0}", current.label || current.name || t("untitled"))}
                onClick={() => remove(index)}
              >
                <Trash01Icon className="size-3.5" />
              </Button>
            </div>
          );
        })}
        {pricing && <Reconcile pricing={pricing} />}
        {atLimit && (
          <p className="text-foreground-muted border-t px-3 py-2 text-xs">
            {t("Maximum of {0} breakdown items reached.", MAX_BREAKDOWN_DEFINITIONS)}
          </p>
        )}
      </div>
    </>
  );
}
