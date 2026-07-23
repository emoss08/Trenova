import { ControlledShipmentAutocompleteField } from "@/components/autocomplete-fields";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { cn, formatCurrency } from "@/lib/utils";
import type {
  BreakdownDefinitionInput,
  TestBreakdownItem,
  TestExpressionRequest,
  VariableDefinitionInput,
} from "@/types/formula-template";
import {
  AlertTriangleIcon,
  Braces,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ChevronUp,
  FlaskConical,
  ListTree,
  RotateCcw,
  Sparkles,
  Truck,
  XCircle,
} from "lucide-react";
import { useCallback, useState } from "react";
import { DEFAULT_TEST_VALUES, TestDataEditor } from "./test-data-editor";
import { useExpressionTest } from "./use-expression-test";

type ExpressionTestPanelProps = {
  expression: string;
  schemaId?: string;
  customVariables?: VariableDefinitionInput[];
  breakdowns?: BreakdownDefinitionInput[];
  className?: string;
};

function BreakdownResultTable({ items }: { items: TestBreakdownItem[] }) {
  return (
    <div className="mt-4 space-y-2">
      <div className="flex items-center gap-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
        <ListTree className="size-3" />
        Breakdown
      </div>
      <div className="overflow-hidden rounded-md border bg-background/50">
        {items.map((item) => (
          <div
            key={item.name}
            className="flex items-center justify-between gap-3 border-b px-3 py-1.5 text-sm last:border-b-0"
          >
            <div className="min-w-0">
              <span className="font-mono text-xs">{item.name}</span>
              {item.label && (
                <span className="ml-2 text-xs text-muted-foreground">{item.label}</span>
              )}
            </div>
            {item.error ? (
              <span className="flex items-center gap-1 text-xs text-destructive">
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

function ResolvedVariablesView({ variables }: { variables: Record<string, unknown> }) {
  const [isOpen, setIsOpen] = useState(false);
  const count = Object.keys(variables).length;

  return (
    <div className="mt-4 space-y-2">
      <button
        type="button"
        onClick={() => setIsOpen((prev) => !prev)}
        className="flex items-center gap-2 text-xs font-medium tracking-wide text-muted-foreground uppercase hover:text-foreground"
      >
        {isOpen ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
        <Braces className="size-3" />
        Resolved Variables ({count})
      </button>
      {isOpen && (
        <pre className="max-h-64 overflow-auto rounded-md border bg-background/50 p-3 font-mono text-xs whitespace-pre-wrap">
          {JSON.stringify(variables, null, 2)}
        </pre>
      )}
    </div>
  );
}

export function ExpressionTestPanel({
  expression,
  schemaId = "shipment",
  customVariables = [],
  breakdowns = [],
  className,
}: ExpressionTestPanelProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [testValues, setTestValues] = useState<Record<string, unknown>>({
    ...DEFAULT_TEST_VALUES,
  });
  const [useRealShipment, setUseRealShipment] = useState(false);
  const [shipmentId, setShipmentId] = useState("");
  const { mutate, data, isPending, reset } = useExpressionTest();

  const handleTest = useCallback(() => {
    const usingShipment = useRealShipment && !!shipmentId;
    const variables: Record<string, unknown> = usingShipment ? {} : { ...testValues };

    customVariables.forEach((v) => {
      if (v.defaultValue !== undefined) {
        variables[v.name] = v.defaultValue;
      }
    });

    const validBreakdowns = breakdowns.filter((b) => b.name.trim() && b.expression.trim());

    const request: TestExpressionRequest = {
      expression,
      schemaId,
      variables,
      ...(usingShipment ? { shipmentId } : {}),
      ...(validBreakdowns.length > 0 ? { breakdowns: validBreakdowns } : {}),
    };

    mutate(request);
    setIsExpanded(true);
  }, [
    expression,
    schemaId,
    customVariables,
    breakdowns,
    testValues,
    useRealShipment,
    shipmentId,
    mutate,
  ]);

  const handleToggle = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  const handleClear = useCallback(() => {
    reset();
  }, [reset]);

  const handleUseRealShipmentChange = useCallback(
    (checked: boolean) => {
      setUseRealShipment(checked);
      if (!checked) {
        setShipmentId("");
      }
      reset();
    },
    [reset],
  );

  const hasResult = data !== undefined;
  const isValid = data?.valid ?? false;

  return (
    <div className={cn("mt-3 space-y-3", className)}>
      <div className="rounded-lg border bg-muted/30 px-3 py-2">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <Truck className="size-3.5 text-muted-foreground" />
            <Label htmlFor="use-real-shipment" className="text-xs font-medium">
              Use real shipment
            </Label>
          </div>
          <Switch
            id="use-real-shipment"
            size="sm"
            checked={useRealShipment}
            onCheckedChange={handleUseRealShipmentChange}
          />
        </div>
        {useRealShipment && (
          <div className="mt-2 space-y-1.5">
            <ControlledShipmentAutocompleteField
              value={shipmentId}
              onValueChange={setShipmentId}
              clearable
            />
            <p className="text-2xs text-muted-foreground">
              Variables are resolved from the selected shipment; manual test data is ignored. Custom
              variable defaults still apply.
            </p>
          </div>
        )}
      </div>

      {!useRealShipment && <TestDataEditor values={testValues} onChange={setTestValues} />}

      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleTest}
          isLoading={isPending}
          loadingText="Testing..."
          disabled={useRealShipment && !shipmentId}
          className="gap-2"
        >
          <FlaskConical className="size-3.5" />
          Test Expression
        </Button>

        {hasResult && (
          <>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleToggle}
              className="gap-1.5 text-muted-foreground hover:text-foreground"
            >
              {isExpanded ? (
                <ChevronUp className="size-3.5" />
              ) : (
                <ChevronDown className="size-3.5" />
              )}
              {isExpanded ? "Hide" : "Show"}
            </Button>

            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={handleClear}
              className="gap-1.5 text-muted-foreground hover:text-foreground"
            >
              <RotateCcw className="size-3" />
              Clear
            </Button>

            {!isExpanded && (
              <div
                className={cn(
                  "ml-auto flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium",
                  isValid
                    ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                    : "bg-destructive/10 text-destructive",
                )}
              >
                {isValid ? <CheckCircle2 className="size-3.5" /> : <XCircle className="size-3.5" />}
                {isValid ? "Valid" : "Invalid"}
              </div>
            )}
          </>
        )}
      </div>

      {hasResult && isExpanded && (
        <div
          className={cn(
            "overflow-hidden rounded-lg border",
            isValid
              ? "border-emerald-500/30 bg-emerald-500/5"
              : "border-destructive/30 bg-destructive/5",
          )}
        >
          <div
            className={cn(
              "flex items-center gap-2 border-b px-4 py-2.5",
              isValid
                ? "border-emerald-500/20 bg-emerald-500/10"
                : "border-destructive/20 bg-destructive/10",
            )}
          >
            {isValid ? (
              <CheckCircle2 className="size-4 text-emerald-600 dark:text-emerald-400" />
            ) : (
              <XCircle className="size-4 text-destructive" />
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
          </div>

          <div className="p-4">
            {isValid && data.result !== undefined && (
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
                  <Sparkles className="size-3" />
                  Result
                </div>
                <div className="flex items-baseline gap-2">
                  <span className="font-mono text-2xl font-semibold text-foreground">
                    {typeof data.result === "number"
                      ? data.result.toLocaleString()
                      : String(data.result)}
                  </span>
                  {typeof data.result === "number" && (
                    <span className="text-sm text-muted-foreground">(numeric)</span>
                  )}
                </div>
              </div>
            )}

            {!isValid && data.error && (
              <div className="space-y-2">
                <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                  Error Details
                </div>
                <pre className="overflow-x-auto rounded-md border border-destructive/10 bg-background/50 p-3 font-mono text-sm wrap-break-word whitespace-pre-wrap text-destructive">
                  {data.error}
                </pre>
              </div>
            )}

            {isValid && data.breakdown && data.breakdown.length > 0 && (
              <BreakdownResultTable items={data.breakdown} />
            )}

            {data.resolvedVariables && Object.keys(data.resolvedVariables).length > 0 && (
              <ResolvedVariablesView variables={data.resolvedVariables} />
            )}

            {data.message && <p className="mt-3 text-xs text-muted-foreground">{data.message}</p>}
          </div>
        </div>
      )}
    </div>
  );
}
