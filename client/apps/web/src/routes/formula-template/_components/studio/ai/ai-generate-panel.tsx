import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  FormulaTemplateType,
  GenerateFormulaResponse,
  VariableDefinition,
} from "@trenova/shared/types/formula-template";
import { useMutation } from "@tanstack/react-query";
import { AlertTriangleIcon, CheckCircle2Icon, SparklesIcon, WandSparklesIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

const SUGGESTED_PROMPTS = [
  "Per mile rate with an 18% fuel surcharge on top",
  "Charge per hundredweight, rounded up, with a $150 minimum",
  "Flat rate plus $75 for every stop beyond the second",
  "Add a $200 fee when the shipment carries hazmat",
];

type AiGeneratePanelProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  templateType: FormulaTemplateType;
  schemaId: string;
  onInsert: (result: { expression: string; variableDefinitions: VariableDefinition[] }) => void;
};

export function AiGeneratePanel({
  open,
  onOpenChange,
  templateType,
  schemaId,
  onInsert,
}: AiGeneratePanelProps) {
  const [instruction, setInstruction] = useState("");

  const { mutate, data, isPending, reset, error } = useMutation<
    GenerateFormulaResponse,
    Error,
    string
  >({
    mutationFn: (prompt) =>
      apiService.formulaTemplateService.generateFormula({
        instruction: prompt,
        schemaId,
        templateType,
      }),
  });

  const handleGenerate = (prompt: string) => {
    if (!prompt.trim()) return;
    setInstruction(prompt);
    mutate(prompt);
  };

  const handleInsert = () => {
    if (!data) return;
    onInsert({
      expression: data.expression,
      variableDefinitions: data.variableDefinitions,
    });
    toast.success("Formula inserted into the editor", {
      description: "Review and test it before saving.",
    });
    onOpenChange(false);
  };

  const validation = data?.validation;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-0 sm:w-[480px] sm:max-w-[480px]"
      >
        <SheetHeader className="border-b pb-3">
          <SheetTitle className="flex items-center gap-2">
            <WandSparklesIcon className="size-4" />
            Generate Formula
          </SheetTitle>
          <SheetDescription>
            Describe how this template should price a shipment. The generated formula lands in the
            editor for you to review and test — nothing is saved automatically.
          </SheetDescription>
        </SheetHeader>

        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-4 p-4">
            <div className="space-y-2">
              <Textarea
                value={instruction}
                onChange={(event) => setInstruction(event.target.value)}
                placeholder="e.g. Charge $2.85 per mile, add a 20% fuel surcharge, and never bill under $350"
                rows={4}
              />
              <div className="flex items-center justify-between gap-2">
                <p className="text-2xs text-muted-foreground">
                  Reference shipment values like distance, weight, stops, or hazmat.
                </p>
                <Button
                  type="button"
                  size="sm"
                  onClick={() => handleGenerate(instruction)}
                  isLoading={isPending}
                  loadingText="Generating..."
                  disabled={!instruction.trim()}
                  className="gap-1.5"
                >
                  <SparklesIcon className="size-3.5" />
                  Generate
                </Button>
              </div>
            </div>

            {!data && !isPending && (
              <div className="space-y-1.5">
                <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                  Try one of these
                </p>
                {SUGGESTED_PROMPTS.map((prompt) => (
                  <button
                    key={prompt}
                    type="button"
                    onClick={() => handleGenerate(prompt)}
                    className="hover:bg-muted w-full rounded-md border px-3 py-2 text-left text-xs"
                  >
                    {prompt}
                  </button>
                ))}
              </div>
            )}

            {error && (
              <div className="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-3 py-2 text-xs">
                {error.message || "Formula generation failed. Try again."}
              </div>
            )}

            {data && (
              <div className="space-y-3">
                <div className="space-y-1.5">
                  <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                    Generated Expression
                  </p>
                  <pre className="bg-muted overflow-x-auto rounded-md border p-3 font-mono text-xs whitespace-pre-wrap">
                    {data.expression}
                  </pre>
                </div>

                {data.variableDefinitions.length > 0 && (
                  <div className="space-y-1.5">
                    <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                      Custom Variables
                    </p>
                    <div className="overflow-hidden rounded-md border">
                      {data.variableDefinitions.map((variable) => (
                        <div
                          key={variable.name}
                          className="flex items-center justify-between gap-2 border-b px-3 py-1.5 text-xs last:border-b-0"
                        >
                          <span className="font-mono">{variable.name}</span>
                          <span className="flex items-center gap-1.5">
                            {variable.defaultValue !== undefined &&
                              variable.defaultValue !== null && (
                                <span className="text-muted-foreground">
                                  = {String(variable.defaultValue)}
                                </span>
                              )}
                            <Badge variant="outline" className="text-2xs">
                              {variable.type}
                            </Badge>
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {data.explanation && (
                  <div className="space-y-1.5">
                    <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
                      How it works
                    </p>
                    <p className="text-sm leading-relaxed">{data.explanation}</p>
                  </div>
                )}

                {validation && (
                  <div
                    className={cn(
                      "flex items-center gap-2 rounded-md border px-3 py-2 text-xs",
                      validation.valid
                        ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300"
                        : "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300",
                    )}
                  >
                    {validation.valid ? (
                      <CheckCircle2Icon className="size-3.5 shrink-0" />
                    ) : (
                      <AlertTriangleIcon className="size-3.5 shrink-0" />
                    )}
                    {validation.valid
                      ? `Validated against sample data${
                          typeof validation.result === "number"
                            ? ` — result ${formatCurrency(validation.result)}`
                            : ""
                        }`
                      : `Validation warning: ${validation.error || validation.message}`}
                  </div>
                )}

                <div className="flex items-center gap-2">
                  <Button type="button" size="sm" onClick={handleInsert} className="gap-1.5">
                    Insert into editor
                  </Button>
                  <Button type="button" variant="ghost" size="sm" onClick={() => reset()}>
                    Start over
                  </Button>
                </div>
              </div>
            )}
          </div>
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}
