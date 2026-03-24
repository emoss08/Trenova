import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { VariableDefinitionInput } from "@/types/formula-template";
import {
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  FlaskConical,
  RotateCcw,
  Sparkles,
  XCircle,
} from "lucide-react";
import { useCallback, useState } from "react";
import { DEFAULT_TEST_VALUES, TestDataEditor } from "./test-data-editor";
import { useExpressionTest } from "./use-expression-test";

type ExpressionTestPanelProps = {
  expression: string;
  schemaId?: string;
  customVariables?: VariableDefinitionInput[];
  className?: string;
};

export function ExpressionTestPanel({
  expression,
  schemaId = "shipment",
  customVariables = [],
  className,
}: ExpressionTestPanelProps) {
  const [isExpanded, setIsExpanded] = useState(true);
  const [testValues, setTestValues] = useState<Record<string, unknown>>({
    ...DEFAULT_TEST_VALUES,
  });
  const { mutate, data, isPending, reset } = useExpressionTest();

  const handleTest = useCallback(() => {
    const variables: Record<string, unknown> = { ...testValues };

    customVariables.forEach((v) => {
      if (v.defaultValue !== undefined) {
        variables[v.name] = v.defaultValue;
      }
    });

    mutate({ expression, schemaId, variables });
    setIsExpanded(true);
  }, [expression, schemaId, customVariables, testValues, mutate]);

  const handleToggle = useCallback(() => {
    setIsExpanded((prev) => !prev);
  }, []);

  const handleClear = useCallback(() => {
    reset();
  }, [reset]);

  const hasResult = data !== undefined;
  const isValid = data?.valid ?? false;

  return (
    <div className={cn("mt-3 space-y-3", className)}>
      <TestDataEditor values={testValues} onChange={setTestValues} />

      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={handleTest}
          isLoading={isPending}
          loadingText="Testing..."
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
                {isValid ? (
                  <CheckCircle2 className="size-3.5" />
                ) : (
                  <XCircle className="size-3.5" />
                )}
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
                    <span className="text-sm text-muted-foreground">
                      (numeric)
                    </span>
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

            {data.message && (
              <p className="mt-3 text-xs text-muted-foreground">
                {data.message}
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
