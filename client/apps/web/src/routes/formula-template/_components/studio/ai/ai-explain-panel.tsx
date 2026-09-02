import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import type { ExplainFormulaResponse } from "@trenova/shared/types/formula-template";
import { useMutation } from "@tanstack/react-query";
import { MessageCircleQuestionIcon, XIcon } from "lucide-react";
import { useState } from "react";
import { explanationStatus } from "./explanation-status";

type AiExplainPanelProps = {
  expression: string;
  schemaId: string;
};

export function AiExplainPanel({ expression, schemaId }: AiExplainPanelProps) {
  const [dismissed, setDismissed] = useState(false);
  const [explainedFor, setExplainedFor] = useState<string | null>(null);

  const { mutate, data, isPending, error, reset } = useMutation<
    ExplainFormulaResponse,
    Error,
    string
  >({
    mutationFn: (currentExpression) =>
      apiService.formulaTemplateService.explainFormula({
        expression: currentExpression,
        schemaId,
      }),
    onSuccess: (_result, currentExpression) => setExplainedFor(currentExpression),
  });

  const status = explanationStatus({ expression, explainedFor, hasExplanation: !!data });

  const handleExplain = () => {
    if (!expression.trim()) return;
    setDismissed(false);
    mutate(expression);
  };

  const showResult = !dismissed && (isPending || data || error);

  return (
    <div className="space-y-2">
      <Button
        type="button"
        variant="outline"
        size="xs"
        onClick={handleExplain}
        disabled={!expression.trim() || isPending}
        className="gap-1.5"
      >
        <MessageCircleQuestionIcon className="size-3" />
        Explain formula
      </Button>

      {showResult && (
        <div className="bg-muted/40 relative rounded-md border p-3">
          <button
            type="button"
            aria-label="Dismiss explanation"
            onClick={() => {
              setDismissed(true);
              reset();
            }}
            className="text-muted-foreground hover:text-foreground absolute top-2 right-2"
          >
            <XIcon className="size-3.5" />
          </button>

          {isPending && (
            <div className="text-muted-foreground flex items-center gap-2 text-xs">
              <Spinner className="size-3.5" />
              Explaining this formula...
            </div>
          )}

          {error && (
            <p className="text-destructive pr-6 text-xs">
              {error.message || "Could not explain this formula. Try again."}
            </p>
          )}

          {data && (
            <div className="space-y-2 pr-6">
              {status === "stale" && (
                <div className="flex items-center justify-between gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-2 py-1 text-xs text-amber-800 dark:text-amber-200">
                  <span>The formula changed since this explanation was written.</span>
                  <Button type="button" variant="ghost" size="xs" onClick={handleExplain}>
                    Explain again
                  </Button>
                </div>
              )}
              <p
                className={cn(
                  "text-sm leading-relaxed whitespace-pre-wrap",
                  status === "stale" && "text-muted-foreground",
                )}
              >
                {data.explanation}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
