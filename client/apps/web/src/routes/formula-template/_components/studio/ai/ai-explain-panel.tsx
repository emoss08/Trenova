import { apiService } from "@/services/api";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import type { ExplainFormulaResponse } from "@trenova/shared/types/formula-template";
import { useMutation } from "@tanstack/react-query";
import { MessageCircleQuestionIcon, XIcon } from "lucide-react";
import { useState } from "react";

type AiExplainPanelProps = {
  expression: string;
  schemaId: string;
};

export function AiExplainPanel({ expression, schemaId }: AiExplainPanelProps) {
  const [dismissed, setDismissed] = useState(false);

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
  });

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
            <p className="pr-6 text-sm leading-relaxed whitespace-pre-wrap">{data.explanation}</p>
          )}
        </div>
      )}
    </div>
  );
}
