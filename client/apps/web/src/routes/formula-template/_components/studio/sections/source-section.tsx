import { TariffEditor } from "@/components/formula-editor/tariff-editor";
import { formatTariffDocument } from "@/components/formula-editor/tariff-format";
import { ProblemsList } from "@/components/tariff-studio/problems-list";
import {
  reportFormat,
  SOURCE_EDITOR_MAX_HEIGHT,
  sourceStatus,
  SourceStatusBar,
} from "@/components/tariff-studio/source-status";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { BookOpen01Icon, Code01Icon, InfoCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import type { TariffInputValues } from "@trenova/shared/types/tariff";
import type { EditorView } from "@codemirror/view";
import { useCallback, useRef, useState, type ReactNode } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { NovaButton } from "@/components/tariff-studio/nova-button";
import { useStudio } from "../studio-context";
import { presentInputs } from "../use-live-preview";
import { ExplainPanel, type ExplainState } from "./explain-panel";
import { SectionHead } from "./section-head";
import { StartCard } from "./start-card";

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
 * The tariff itself: where a new template starts, the editor with its status bar and the
 * check's findings, and Nova's explanation of what it bills on the preview load.
 */
export function SourceSection() {
  const t = useT();
  const {
    mode,
    check,
    errors,
    warnings,
    tests,
    preview,
    context,
    contextError,
    highlight,
    askNova,
    draftContent,
    setEditorView,
    reveal,
    setHighlight,
    applyFix,
    openDefinition,
    previewInputs,
    rules,
  } = useStudio();
  const { refetch: retryContext } = useTariffCheckContext();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const source = useWatch({ control, name: "source" }) ?? "";
  const viewRef = useRef<EditorView | null>(null);
  const [explain, setExplain] = useState<ExplainState | null>(null);
  const empty = source.trim() === "";

  const explainMutation = useMutation({
    mutationFn: () =>
      apiService.formulaTemplateService.explainFormula(
        draftContent(),
        preview.source === "shipment" && preview.shipmentId
          ? { shipmentId: preview.shipmentId }
          : { inputs: presentInputs(previewInputs.getValues("inputs") as TariffInputValues) },
      ),
    onMutate: () => setExplain({ status: "loading" }),
    onSuccess: (data) => setExplain({ status: "ready", data }),
    onError: (error) => setExplain({ status: "error", error }),
  });

  const status = sourceStatus({ empty, errors, warnings }, t);

  const outline = check.outline?.items ?? [];
  const charges = outline.filter((item) => item.kind === "charge").length;
  const testCount = outline.filter((item) => item.kind === "test").length;
  const trace = preview.result && !preview.blocked ? preview.result.trace : null;

  const handleView = useCallback(
    (view: EditorView | null) => {
      viewRef.current = view;
      setEditorView(view);
    },
    [setEditorView],
  );

  const format = () => {
    const view = viewRef.current;
    if (!view) return;
    reportFormat(formatTariffDocument(view, rules), t);
    view.focus();
  };

  return (
    <section className="flex flex-col gap-3">
      <SectionHead
        section="source"
        icon={Code01Icon}
        title={t("Source")}
        description={t(
          "The tariff: its inputs, the charges it bills, its total rules and its tests, in one file.",
        )}
      >
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="gap-1.5"
          disabled={empty || errors > 0 || explainMutation.isPending}
          onClick={() => explainMutation.mutate()}
        >
          <BookOpen01Icon className="size-3" />
          {t("Explain")}
        </Button>
        <NovaButton onClick={() => askNova()}>{t("Ask Nova")}</NovaButton>
      </SectionHead>

      <MaterialBanner />

      {contextError && (
        <Alert variant="warning" size="sm" role="status">
          <InfoCircleIcon />
          <AlertDescription>
            {t(
              "Your rate tables, charge codes and libraries could not be loaded, so the check cannot see them yet.",
            )}
          </AlertDescription>
          <AlertAction>
            <Button type="button" variant="ghost" size="xs" onClick={retryContext}>
              {t("Retry")}
            </Button>
          </AlertAction>
        </Alert>
      )}

      {mode === "create" && empty && <StartCard />}

      <div
        className={cn(
          "ui-container-focus-ring bg-card overflow-hidden rounded-lg border",
          status.tone === "bad" && "border-danger-border",
        )}
      >
        <Controller
          name="source"
          control={control}
          render={({ field }) => (
            <TariffEditor
              kind="tariff"
              value={field.value ?? ""}
              onChange={field.onChange}
              context={context}
              trace={trace}
              highlight={highlight}
              onView={handleView}
              onOpenDefinition={openDefinition}
              onFormat={(result) => reportFormat(result, t)}
              ariaLabel={t("Tariff source")}
              placeholder={t('tariff "Name of the rating method"')}
              minHeight={mode === "create" && empty ? "96px" : "240px"}
              maxHeight={SOURCE_EDITOR_MAX_HEIGHT}
              primary
              embedded
            />
          )}
        />
        <SourceStatusBar
          status={status}
          summary={t(
            "{0, plural, one {# input} other {# inputs}} · {1, plural, one {# charge} other {# charges}} · {2, plural, one {# test} other {# tests}}",
            check.inputs.length,
            charges,
            testCount,
          )}
          tests={tests}
          onFormat={format}
          formatDisabled={empty}
        />
        <ProblemsList
          diagnostics={check.diagnostics}
          source={source}
          onReveal={reveal}
          onApplyFix={applyFix}
          rowActions={(diagnostic) =>
            diagnostic.severity === "error" ? (
              <NovaButton
                size="xs"
                onClick={() =>
                  askNova(t("How do I fix this: {0}", translateMessage(diagnostic.message, t)))
                }
              >
                {t("Ask Nova")}
              </NovaButton>
            ) : null
          }
        />
      </div>

      {explain && (
        <ExplainPanel
          source={source}
          state={explain}
          onHover={setHighlight}
          onRetry={() => explainMutation.mutate()}
          onFollowUp={() => askNova(t("Explain this tariff, statement by statement"))}
          onClose={() => {
            setExplain(null);
            setHighlight(null);
          }}
        />
      )}
    </section>
  );
}
