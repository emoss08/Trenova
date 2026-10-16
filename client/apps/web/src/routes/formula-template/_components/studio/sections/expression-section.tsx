import { formatEditorDocument, type FormatResult } from "@/components/formula-editor/expr-format";
import type { InlineValue } from "@/components/formula-editor/expr-inline-values";
import { ExpressionEditor, selectEditorRange } from "@/components/formula-editor/expression-editor";
import type { EditorRange } from "@/components/formula-editor/highlight-range";
import { apiService } from "@/services/api";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatAltShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaStatement,
  FormulaTemplateFormValues,
} from "@trenova/shared/types/formula-template";
import {
  AlignLeftIcon,
  BookOpen01Icon,
  Code01Icon,
  EyeIcon,
  EyeOffIcon,
  InfoCircleIcon,
} from "@trenova/shared/components/icons";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { NovaButton } from "../ai/nova-button";
import { engineErrorText } from "@/components/formula-editor/expr-check";
import { formatMoney, formatStudioValue, numericAmount } from "../model/format";
import { useStudio } from "../studio-context";
import { ExplainPanel, type ExplainState } from "./explain-panel";
import { ExpressionProblems } from "./expression-problems";
import { SectionHead } from "./section-head";
import { StartCard } from "./start-card";

function StatusDot({ tone }: { tone: "ok" | "bad" | "warn" | "idle" }) {
  return (
    <span
      aria-hidden
      className={cn(
        "size-1.5 rounded-full",
        tone === "ok" && "bg-success",
        tone === "bad" && "bg-danger",
        tone === "warn" && "bg-warning",
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

/** What each statement came to on the preview load, as the editor draws it at the line's end. */
function inlineValuesOf(
  statements: readonly FormulaStatement[] | null | undefined,
  expression: string,
): InlineValue[] {
  if (!statements) return [];
  const values: InlineValue[] = [];
  for (const statement of statements) {
    if (statement.end > expression.length) continue;
    if (statement.error) {
      values.push({
        at: statement.end,
        text: `✕ ${engineErrorText(statement.error)}`,
        tone: "bad",
      });
      break;
    }
    const amount = numericAmount(statement.value);
    const text =
      statement.kind === "result" && amount !== null
        ? formatMoney(amount)
        : amount !== null
          ? formatStudioValue(amount)
          : formatStudioValue(statement.value);
    values.push({
      at: statement.end,
      text: `= ${text}`,
      tone: statement.kind === "result" ? "ok" : "muted",
    });
  }
  return values;
}

/**
 * The formula itself: where a new template starts, the editor with its
 * status bar and problems, and Nova's explanation of what it charges.
 */
export function ExpressionSection() {
  const t = useT();
  const { mode, known, analysis, preview, askNova, draftContent, declareInput } = useStudio();
  const { control, getValues } = useFormContext<FormulaTemplateFormValues>();
  const editorRef = useRef<ReactCodeMirrorRef>(null);
  const [cursor, setCursor] = useState({ line: 1, column: 1 });
  const [focusRange, setFocusRange] = useState<EditorRange | null>(null);
  const [explain, setExplain] = useState<ExplainState | null>(null);
  const [showValues, setShowValues] = useState(true);

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
  const serverWarnings = result?.valid ? (result.warnings?.length ?? 0) : 0;
  const warnings = analysis.warningCount + serverWarnings;
  const status = empty
    ? { tone: "idle" as const, label: t("Empty") }
    : analysis.errorCount > 0
      ? {
          tone: "bad" as const,
          label: t("{0, plural, one {# error} other {# errors}}", analysis.errorCount),
        }
      : result && !result.valid
        ? { tone: "bad" as const, label: t("Fails on the preview load") }
        : warnings > 0
          ? {
              tone: "warn" as const,
              label: t("Valid · {0, plural, one {# warning} other {# warnings}}", warnings),
            }
          : { tone: "ok" as const, label: t("Valid") };

  const inlineValues = useMemo(
    () =>
      showValues && !empty && analysis.errorCount === 0
        ? inlineValuesOf(result?.statements, getValues("expression") ?? "")
        : [],
    [showValues, empty, analysis.errorCount, result?.statements, getValues],
  );

  const jump = useCallback((range: EditorRange) => {
    const view = editorRef.current?.view;
    if (view) selectEditorRange(view, range);
  }, []);

  // Formatting is refused, never approximated: a formula that does not parse
  // would lose what the parser could not place, and a comment would be dropped.
  const formatted = useCallback(
    (outcome: FormatResult) => {
      if (outcome.ok) return;
      if (outcome.reason === "syntax") {
        toast.warning(t("Fix the syntax first, then format"));
      } else if (outcome.reason === "comments") {
        toast.warning(t("Formatting would drop a comment; take it out first"));
      }
    },
    [t],
  );
  const format = useCallback(() => {
    const view = editorRef.current?.view;
    if (!view) return;
    formatted(formatEditorDocument(view, { operators: known.operatorNames }));
    view.focus();
  }, [formatted, known.operatorNames]);

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
          declareInput={declareInput}
          inlineValues={inlineValues}
          onFormat={formatted}
        />
        <div className="bg-sunken text-foreground-muted flex flex-wrap items-center gap-x-3 gap-y-1 border-t px-3 py-1 text-xs">
          <span
            className={cn(
              "flex items-center gap-1.5 font-medium",
              status.tone === "ok" && "text-success",
              status.tone === "warn" && "text-warning",
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
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-label={t("Format the formula")}
                  disabled={empty}
                  onClick={format}
                  className="text-foreground-muted"
                />
              }
            >
              <AlignLeftIcon className="size-3.5" />
            </TooltipTrigger>
            <TooltipContent>
              {t("Format")} <Kbd className="ml-1">{formatAltShortcut("⇧F")}</Kbd>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  aria-pressed={showValues}
                  aria-label={t("Show each line's value on the preview load")}
                  onClick={() => setShowValues((current) => !current)}
                  className="text-foreground-muted"
                />
              }
            >
              {showValues ? <EyeIcon className="size-3.5" /> : <EyeOffIcon className="size-3.5" />}
            </TooltipTrigger>
            <TooltipContent>
              {showValues ? t("Hide inline values") : t("Show inline values")}
            </TooltipContent>
          </Tooltip>
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
        <ExpressionProblems onJump={jump} />
      </div>

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
