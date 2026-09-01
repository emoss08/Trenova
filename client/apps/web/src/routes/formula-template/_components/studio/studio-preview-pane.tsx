import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { TestDataEditor } from "@/components/formula-editor/test-data-editor";
import { Button } from "@trenova/shared/components/ui/button";
import { Label } from "@trenova/shared/components/ui/label";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Switch } from "@trenova/shared/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { guardNullableField, scopeToFormPath } from "@/components/formula-editor/guard-nullable";
import { flattenResolvedVariables } from "@/components/formula-editor/resolved-variables";
import type {
  ExpressionWarning,
  FormulaTemplateFormValues,
  GuardrailResult,
  RoundingResult,
  TestBreakdownItem,
} from "@trenova/shared/types/formula-template";
import {
  AlertTriangleIcon,
  Braces,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  FlaskConical,
  ListTree,
  PinIcon,
  PlayIcon,
  ShieldIcon,
  Truck,
  XCircle,
} from "lucide-react";
import { useState } from "react";
import { useFormContext, type Path } from "react-hook-form";
import type { LivePreviewState } from "./use-live-preview";

function GuardrailNotice({ guardrail }: { guardrail: GuardrailResult }) {
  if (!guardrail.applied) {
    return (
      <div className="mt-2 flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400">
        <ShieldIcon className="size-3" />
        Within guardrails
      </div>
    );
  }

  const bound = guardrail.bound === "min" ? "minimum" : "maximum";
  const limit = guardrail.bound === "min" ? guardrail.minCharge : guardrail.maxCharge;

  return (
    <div className="mt-2 flex items-start gap-1.5 rounded-md border border-amber-500/40 bg-amber-500/10 px-2 py-1.5 text-xs text-amber-700 dark:text-amber-300">
      <ShieldIcon className="mt-0.5 size-3 shrink-0" />
      <span>
        The formula produced {formatCurrency(guardrail.rawAmount)} and was clamped to the {bound}{" "}
        charge{limit != null ? ` of ${formatCurrency(limit)}` : ""}.
      </span>
    </div>
  );
}

const ROUNDING_MODE_LABELS: Record<string, string> = {
  HalfUp: "half up",
  HalfEven: "half even",
  Up: "up",
  Down: "down",
  None: "not rounded",
};

function RoundingNotice({ rounding }: { rounding: RoundingResult }) {
  const modeLabel = ROUNDING_MODE_LABELS[rounding.mode] ?? rounding.mode;
  const places = rounding.precision === 1 ? "1 decimal" : `${rounding.precision} decimals`;

  if (!rounding.applied) {
    return (
      <div className="text-muted-foreground text-2xs mt-1">
        Rounding ({modeLabel}, {places}) made no change.
      </div>
    );
  }

  return (
    <div className="text-muted-foreground text-2xs mt-1">
      Rounded {modeLabel} to {places} from{" "}
      <span className="font-mono tabular-nums">{rounding.unroundedAmount.toFixed(6)}</span>.
    </div>
  );
}

function NullableWarnings({ warnings }: { warnings: ExpressionWarning[] }) {
  const { getValues, setValue } = useFormContext<FormulaTemplateFormValues>();

  const applyFix = (warning: ExpressionWarning) => {
    const path = scopeToFormPath(warning.scope) as Path<FormulaTemplateFormValues>;
    const current = getValues(path);
    if (typeof current !== "string") return;
    const guarded = guardNullableField(current, warning.field, warning.suggestion);
    if (guarded === current) return;
    setValue(path, guarded as never, { shouldDirty: true, shouldValidate: true });
  };

  return (
    <div className="mt-4 space-y-2">
      <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase">
        <AlertTriangleIcon className="size-3" />
        Would fail on some shipments
      </div>
      <div className="space-y-1.5">
        {warnings.map((warning) => (
          <div
            key={`${warning.scope}:${warning.field}`}
            className="flex items-start justify-between gap-3 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-200"
          >
            <div className="min-w-0 space-y-0.5">
              <p>{warning.message}</p>
              {warning.scope !== "expression" && (
                <p className="text-2xs opacity-80">In {warning.scope}</p>
              )}
            </div>
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="shrink-0 font-mono"
              onClick={() => applyFix(warning)}
            >
              Use {warning.suggestion}
            </Button>
          </div>
        ))}
      </div>
    </div>
  );
}

const RUN_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
});

function formatRunTime(timestamp: number): string {
  return RUN_TIME_FORMAT.format(new Date(timestamp));
}

function BreakdownResultTable({ items }: { items: TestBreakdownItem[] }) {
  return (
    <div className="mt-4 space-y-2">
      <div className="text-muted-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase">
        <ListTree className="size-3" />
        Breakdown
      </div>
      <div className="bg-background/50 overflow-hidden rounded-md border">
        {items.map((item) => (
          <div
            key={item.name}
            className="flex items-center justify-between gap-3 border-b px-3 py-1.5 text-sm last:border-b-0"
          >
            <div className="min-w-0">
              <span className="font-mono text-xs">{item.name}</span>
              {item.label && (
                <span className="text-muted-foreground ml-2 text-xs">{item.label}</span>
              )}
            </div>
            {item.error ? (
              <span className="text-destructive flex items-center gap-1 text-xs">
                <AlertTriangleIcon className="size-3" />
                {item.error}
              </span>
            ) : (
              <span className="font-mono text-xs font-medium tabular-nums">
                {formatCurrency(item.amount)}
              </span>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function ResolvedVariablesView({
  variables,
  onUseValues,
}: {
  variables: Record<string, unknown>;
  onUseValues?: (values: Record<string, unknown>) => void;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const count = Object.keys(variables).length;

  return (
    <div className="mt-4 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <button
          type="button"
          onClick={() => setIsOpen((prev) => !prev)}
          className="text-muted-foreground hover:text-foreground flex items-center gap-2 text-xs font-medium tracking-wide uppercase"
        >
          {isOpen ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
          <Braces className="size-3" />
          Resolved Variables ({count})
        </button>
        {onUseValues && (
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={() => onUseValues(flattenResolvedVariables(variables))}
          >
            Use these values
          </Button>
        )}
      </div>
      {isOpen && (
        <pre className="bg-background/50 max-h-64 overflow-auto rounded-md border p-3 font-mono text-xs whitespace-pre-wrap">
          {JSON.stringify(variables, null, 2)}
        </pre>
      )}
    </div>
  );
}

type StudioPreviewPaneProps = {
  preview: LivePreviewState;
  /** Offered when a template exists to pin the current inputs and result as a scenario. */
  onPinScenario?: (sample: { variables: Record<string, unknown>; result: number }) => void;
};

export function StudioPreviewPane({ preview, onPinScenario }: StudioPreviewPaneProps) {
  const {
    result,
    isPending,
    requestError,
    lastRunAt,
    testValues,
    setTestValues,
    useRealShipment,
    setUseRealShipment,
    shipmentId,
    setShipmentId,
    runNow,
  } = preview;

  const hasResult = result !== undefined;
  const isValid = result?.valid ?? false;
  const numericResult = typeof result?.result === "number" ? result.result : null;

  const sampleForPin = (): Record<string, unknown> =>
    useRealShipment && result?.resolvedVariables
      ? flattenResolvedVariables(result.resolvedVariables)
      : { ...testValues };

  const adoptResolvedValues = (values: Record<string, unknown>) => {
    setTestValues(values);
    setUseRealShipment(false);
    setShipmentId("");
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between gap-2 border-b px-3 py-2">
        <div className="flex items-center gap-2">
          <FlaskConical className="text-muted-foreground size-4" />
          <span className="text-sm font-semibold">Live Preview</span>
          {isPending ? (
            <span className="text-muted-foreground text-2xs flex items-center gap-1">
              <Spinner className="size-3" />
              Updating
            </span>
          ) : (
            lastRunAt && (
              <span className="text-muted-foreground text-2xs tabular-nums">
                Ran {formatRunTime(lastRunAt)}
              </span>
            )
          )}
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5">
            <Truck className="text-muted-foreground size-3.5" />
            <Label htmlFor="preview-real-shipment" className="text-xs">
              Real shipment
            </Label>
            <Switch
              id="preview-real-shipment"
              size="sm"
              checked={useRealShipment}
              onCheckedChange={(checked) => {
                setUseRealShipment(checked);
                if (!checked) setShipmentId("");
              }}
            />
          </div>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="outline"
                  size="icon-xs"
                  onClick={runNow}
                  disabled={useRealShipment && !shipmentId}
                >
                  <PlayIcon className="size-3" />
                </Button>
              }
            />
            <TooltipContent>Run now</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1">
        <div className="space-y-3 p-3">
          {useRealShipment && (
            <div className="space-y-1.5">
              <ControlledShipmentAutocompleteField
                value={shipmentId}
                onValueChange={setShipmentId}
                clearable
              />
              <p className="text-2xs text-muted-foreground">
                Variables resolve from the selected shipment; sample data is ignored. Custom
                variable defaults still apply.
              </p>
            </div>
          )}

          {!useRealShipment && <TestDataEditor values={testValues} onChange={setTestValues} />}

          {requestError && (
            <div
              role="alert"
              className="border-destructive/40 bg-destructive/10 flex items-start justify-between gap-3 rounded-lg border px-3 py-2 text-xs"
            >
              <div className="space-y-0.5">
                <p className="text-destructive font-medium">Preview could not run</p>
                <p className="text-muted-foreground">{requestError}</p>
                {hasResult && (
                  <p className="text-muted-foreground text-2xs">
                    The result below is from the last successful run.
                  </p>
                )}
              </div>
              <Button type="button" variant="outline" size="xs" onClick={runNow}>
                Retry
              </Button>
            </div>
          )}

          {!hasResult && !requestError && (
            <div className="text-muted-foreground flex flex-col items-center gap-2 py-8 text-center text-sm">
              <FlaskConical className="size-8 opacity-40" />
              <span>Start typing an expression and the result appears here.</span>
            </div>
          )}

          {hasResult && (
            <div
              aria-busy={isPending}
              className={cn(
                "overflow-hidden rounded-lg border transition-opacity",
                isValid
                  ? "border-emerald-500/30 bg-emerald-500/5"
                  : "border-destructive/30 bg-destructive/5",
                (isPending || requestError) && "opacity-60",
              )}
            >
              <div
                className={cn(
                  "flex items-center gap-2 border-b px-4 py-2",
                  isValid
                    ? "border-emerald-500/20 bg-emerald-500/10"
                    : "border-destructive/20 bg-destructive/10",
                )}
              >
                {isValid ? (
                  <CheckCircle2 className="size-4 text-emerald-600 dark:text-emerald-400" />
                ) : (
                  <XCircle className="text-destructive size-4" />
                )}
                <span
                  className={cn(
                    "text-sm font-medium",
                    isValid
                      ? "text-emerald-700 dark:text-emerald-300"
                      : "text-red-700 dark:text-red-300",
                  )}
                >
                  {isValid ? "Expression Valid" : "Expression Invalid"}
                </span>
                {isValid && numericResult !== null && onPinScenario && (
                  <Tooltip>
                    <TooltipTrigger
                      render={
                        <Button
                          type="button"
                          variant="outline"
                          size="xs"
                          className="ml-auto gap-1"
                          onClick={() =>
                            onPinScenario({ variables: sampleForPin(), result: numericResult })
                          }
                        >
                          <PinIcon className="size-3" />
                          Pin as scenario
                        </Button>
                      }
                    />
                    <TooltipContent>
                      Save these inputs and this result as a scenario that must keep passing
                    </TooltipContent>
                  </Tooltip>
                )}
              </div>

              <div className="p-4">
                {isValid && result.result !== undefined && (
                  <div className="space-y-1">
                    <div className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                      Computed Charge
                    </div>
                    <span className="text-foreground font-mono text-3xl font-semibold tabular-nums">
                      {typeof result.result === "number"
                        ? formatCurrency(result.result)
                        : String(result.result)}
                    </span>
                    {result.guardrail && <GuardrailNotice guardrail={result.guardrail} />}
                    {result.rounding && <RoundingNotice rounding={result.rounding} />}
                  </div>
                )}

                {!isValid && result.error && (
                  <div className="space-y-2">
                    <div className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                      Error Details
                    </div>
                    <pre className="border-destructive/10 bg-background/50 text-destructive overflow-x-auto rounded-md border p-3 font-mono text-sm wrap-break-word whitespace-pre-wrap">
                      {result.error}
                    </pre>
                  </div>
                )}

                {isValid && result.breakdown && result.breakdown.length > 0 && (
                  <BreakdownResultTable items={result.breakdown} />
                )}

                {isValid && result.warnings && result.warnings.length > 0 && (
                  <NullableWarnings warnings={result.warnings} />
                )}

                {result.resolvedVariables && Object.keys(result.resolvedVariables).length > 0 && (
                  <ResolvedVariablesView
                    variables={result.resolvedVariables}
                    onUseValues={useRealShipment ? adoptResolvedValues : undefined}
                  />
                )}
              </div>
            </div>
          )}
        </div>
      </ScrollArea>
    </div>
  );
}
