import { guardNullableField, scopeToFormPath } from "@/components/formula-editor/guard-nullable";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  ExpressionWarning,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import { AlertTriangleIcon, PlusIcon, XCircleIcon } from "@trenova/shared/components/icons";
import type { ReactNode } from "react";
import { useFormContext, type Path } from "react-hook-form";
import { toast } from "sonner";
import { NovaButton } from "../ai/nova-button";
import { inferInputType } from "../model/expression-analysis";
import { useStudio } from "../studio-context";

function ProblemRow({
  tone,
  children,
  action,
}: {
  tone: "error" | "warn";
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 border-b px-3 py-1.5 text-xs last:border-b-0",
        tone === "error" ? "text-danger-subtle-foreground" : "text-warning-subtle-foreground",
      )}
    >
      {tone === "error" ? (
        <XCircleIcon className="text-danger size-3.5 shrink-0" />
      ) : (
        <AlertTriangleIcon className="text-warning size-3.5 shrink-0" />
      )}
      <span className="text-foreground min-w-0 flex-1">{children}</span>
      {action}
    </div>
  );
}

/**
 * Everything wrong with the expression, one row each, with the fix beside
 * it: an unknown name becomes an input, an optional field gets its guard, and
 * anything else, or a failure on the preview load, goes to Nova.
 */
const nameTag = {
  name: (content: ReactNode) => <span className="font-mono">{content}</span>,
};

export function ExpressionProblems() {
  const t = useT();
  const rt = useRichT();
  const { analysis, preview, askNova } = useStudio();
  const form = useFormContext<FormulaTemplateFormValues>();
  const result = preview.result;
  const warnings = result?.valid ? (result.warnings ?? []) : [];
  const runtimeError =
    analysis.errorCount === 0 && result && !result.valid ? (result.error ?? result.message) : null;

  if (analysis.problems.length === 0 && warnings.length === 0 && !runtimeError) {
    return null;
  }

  const addInput = (name: string) => {
    const type = inferInputType(name);
    const current = form.getValues("variableDefinitions") ?? [];
    form.setValue(
      "variableDefinitions",
      [
        ...current,
        {
          name,
          type,
          description: "",
          required: true,
          defaultValue: type === "Boolean" ? false : undefined,
        },
      ],
      { shouldDirty: true, shouldValidate: true },
    );
    toast(t("Added {0} as an input. Set its default below.", name));
  };

  const guard = (warning: ExpressionWarning) => {
    const path = scopeToFormPath(warning.scope) as Path<FormulaTemplateFormValues>;
    const current = form.getValues(path);
    if (typeof current !== "string") return;
    const guarded = guardNullableField(current, warning.field, warning.suggestion);
    if (guarded !== current) {
      form.setValue(path, guarded as never, { shouldDirty: true, shouldValidate: true });
    }
  };

  const fix = (message: string) => askNova(t("Fix this problem in the formula: {0}", message));

  return (
    <div className="bg-card overflow-hidden rounded-lg border">
      {analysis.problems.map((problem) => (
        <ProblemRow
          key={`${problem.kind}:${problem.from}:${problem.to}`}
          tone="error"
          action={
            problem.kind === "unknown-variable" && problem.name ? (
              <Button
                type="button"
                variant="outline"
                size="xs"
                className="gap-1"
                onClick={() => addInput(problem.name ?? "")}
              >
                <PlusIcon className="size-3" />
                {rt("Add <name>{0}</name> as input", nameTag, problem.name)}
              </Button>
            ) : (
              <NovaButton onClick={() => fix(problem.message)}>{t("Fix with Nova")}</NovaButton>
            )
          }
        >
          {problem.message}
        </ProblemRow>
      ))}
      {warnings.map((warning) => (
        <ProblemRow
          key={`${warning.scope}:${warning.field}`}
          tone="warn"
          action={
            <Button
              type="button"
              variant="outline"
              size="xs"
              className="font-mono"
              onClick={() => guard(warning)}
            >
              {t("Use {0}", warning.suggestion)}
            </Button>
          }
        >
          {warning.message}
          {warning.scope !== "expression" && (
            <span className="text-foreground-subtle"> · {t("in a breakdown line")}</span>
          )}
        </ProblemRow>
      ))}
      {runtimeError && (
        <ProblemRow
          tone="error"
          action={<NovaButton onClick={() => fix(runtimeError)}>{t("Fix with Nova")}</NovaButton>}
        >
          {t("On the preview load: {0}", runtimeError)}
        </ProblemRow>
      )}
    </div>
  );
}
