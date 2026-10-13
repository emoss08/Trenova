import { ExpressionEditor } from "@/components/formula-editor/expression-editor";
import type { EditorRange } from "@/components/formula-editor/highlight-range";
import { apiService } from "@/services/api";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { BookOpen01Icon, Code01Icon, InfoCircleIcon } from "@trenova/shared/components/icons";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useMutation } from "@tanstack/react-query";
import { useRef, useState, type ReactNode } from "react";
import { useFormContext } from "react-hook-form";
import { NovaButton } from "../ai/nova-button";
import { useStudio } from "../studio-context";
import { ExplainPanel, type ExplainState } from "./explain-panel";
import { ExpressionProblems } from "./expression-problems";
import { SectionHead } from "./section-head";
import { StartCard } from "./start-card";

function StatusDot({ tone }: { tone: "ok" | "bad" | "idle" }) {
  return (
    <span
      aria-hidden
      className={cn(
        "size-1.5 rounded-full",
        tone === "ok" && "bg-success",
        tone === "bad" && "bg-danger",
        tone === "idle" && "bg-foreground-subtle",
      )}
    />
  );
}

function MaterialBanner() {
  const rt = useRichT();
  const { mode, material, template, usageTotal } = useStudio();
  if (mode !== "edit" || !material || !template) return null;
  if (template.status !== "Active" && template.status !== "InReview") return null;
  const bold = { b: (content: ReactNode) => <b className="font-semibold">{content}</b> };
  return (
    <Alert variant="info" size="sm">
      <InfoCircleIcon />
      <AlertDescription>
        <p>
        {template.status === "Active"
          ? rt(
              "Saving this edit returns the template to <b>Draft</b>. The {0, plural, one {# place} other {# places}} that use it keep rating with v{1} until a new version is approved.",
              bold,
              usageTotal,
              template.currentVersionNumber || 1,
            )
          : rt(
              "Saving this edit cancels the review in progress and returns the template to <b>Draft</b>.",
              bold,
            )}
        </p>
      </AlertDescription>
    </Alert>
  );
}

/**
 * The formula itself: where a new template starts, the editor with a status
 * bar, what is wrong with it and how to fix it, and Nova's explanation of
 * what it charges.
 */
export function ExpressionSection() {
  const t = useT();
  const { mode, known, analysis, preview, askNova, draftContent } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const [cursor, setCursor] = useState({ line: 1, column: 1 });
  const [focusRange, setFocusRange] = useState<EditorRange | null>(null);
  const [explain, setExplain] = useState<ExplainState | null>(null);

  const explainMutation = useMutation({
    mutationFn: () =>
      apiService.formulaTemplateService.explainFormula(
        draftContent(),
        preview.source === "shipment" && preview.shipmentId
          ? { shipmentId: preview.shipmentId }
          : { sampleValues: preview.loadValues() },
      ),
    onMutate: () => setExplain({ status: "loading" }),
    onSuccess: (data) => setExplain({ status: "ready", data }),
    onError: (error) => setExplain({ status: "error", error }),
  });

  const empty = analysis.empty;
  const result = preview.result;
  const warningCount = result?.valid ? (result.warnings?.length ?? 0) : 0;
  const status = empty
    ? { tone: "idle" as const, label: t("Empty") }
    : analysis.errorCount > 0
      ? {
          tone: "bad" as const,
          label: t("{0, plural, one {# error} other {# errors}}", analysis.errorCount),
        }
      : result && !result.valid
        ? { tone: "bad" as const, label: t("Fails on the preview load") }
        : {
            tone: "ok" as const,
            label:
              warningCount > 0
                ? t("Valid · {0, plural, one {# warning} other {# warnings}}", warningCount)
                : t("Valid"),
          };

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="expression"
        icon={Code01Icon}
        title={t("Expression")}
        description={t(
          "The formula that turns shipment facts into the charge. Charge policy applies after it.",
        )}
      >
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="gap-1.5"
          disabled={empty || explainMutation.isPending}
          onClick={() => explainMutation.mutate()}
        >
          <BookOpen01Icon className="size-3" />
          {t("Explain")}
        </Button>
        <NovaButton onClick={() => askNova()}>{t("Ask Nova")}</NovaButton>
      </SectionHead>

      <MaterialBanner />

      {mode === "create" && empty && <StartCard />}

      <div
        className={cn(
          "ui-container-focus-ring bg-card overflow-hidden rounded-lg border",
          status.tone === "bad" && "border-danger-border",
        )}
      >
        <ExpressionEditor
          name="expression"
          control={control}
          placeholder={t("e.g. baseRate * totalDistance")}
          minHeight={mode === "create" && empty ? "72px" : "120px"}
          knownIdentifiers={known}
          editorRef={editorRef}
          primary
          embedded
          highlight={focusRange}
          onCursorChange={setCursor}
        />
        <div className="bg-sunken text-foreground-muted flex flex-wrap items-center gap-x-3 gap-y-1 border-t px-3 py-1.5 text-xs">
          <span
            className={cn(
              "flex items-center gap-1.5 font-medium",
              status.tone === "ok" && "text-success",
              status.tone === "bad" && "text-danger",
            )}
          >
            <StatusDot tone={status.tone} />
            {status.label}
          </span>
          <span>
            {t(
              "{0} schema · {1} inputs · {2} functions",
              analysis.schemaReads.size,
              analysis.inputReads.size,
              analysis.functionCalls.size,
            )}
          </span>
          <span className="flex-1" />
          <span className="flex items-center gap-1 max-[700px]:hidden">
            <KbdGroup>
              <Kbd>Ctrl</Kbd>
              <Kbd>Space</Kbd>
            </KbdGroup>
            {t("suggest")}
          </span>
          <span className="font-mono tabular-nums">
            {t("Ln {0}, Col {1}", cursor.line, cursor.column)}
          </span>
        </div>
      </div>

      <ExpressionProblems />

      {explain && (
        <ExplainPanel
          state={explain}
          onHover={setFocusRange}
          onRetry={() => explainMutation.mutate()}
          onFollowUp={() => askNova(t("Explain this formula, term by term"))}
          onClose={() => {
            setExplain(null);
            setFocusRange(null);
          }}
        />
      )}
    </section>
  );
}
