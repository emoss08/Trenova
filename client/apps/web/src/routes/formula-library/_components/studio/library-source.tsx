import { TariffEditor } from "@/components/formula-editor/tariff-editor";
import { formatTariffDocument } from "@/components/formula-editor/tariff-format";
import { NovaButton } from "@/components/tariff-studio/nova-button";
import { ProblemsList } from "@/components/tariff-studio/problems-list";
import {
  reportFormat,
  sourceStatus,
  SourceStatusBar,
} from "@/components/tariff-studio/source-status";
import { useTariffCheckContext } from "@/hooks/use-tariff-check-context";
import type { EditorView } from "@codemirror/view";
import { Code01Icon, InfoCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaLibraryFormValues } from "@trenova/shared/types/formula-library";
import { useCallback, useRef, type ReactNode } from "react";
import { Controller, useFormContext, useWatch } from "react-hook-form";
import { useLibraryStudio } from "./library-studio-context";
import { headerMismatch, libraryPath, starterLibrary } from "./starter";

/** What saving does to an approved or in-review library. */
function MaterialBanner() {
  const rt = useRichT();
  const { mode, material, library } = useLibraryStudio();
  if (mode !== "edit" || !material || !library) return null;
  if (library.status !== "Active" && library.status !== "InReview") return null;
  const bold = { b: (content: ReactNode) => <b className="font-semibold">{content}</b> };
  return (
    <Alert variant="info" size="sm">
      <InfoCircleIcon />
      <AlertDescription>
        <p>
          {library.status === "Active"
            ? rt(
                "Saving this edit returns the library to <b>Draft</b>. Templates that import it keep the version they pinned, v{0}, until they pin a new one.",
                bold,
                library.approvedVersionNumber ?? library.currentVersionNumber ?? 1,
              )
            : rt(
                "Saving this edit cancels the review in progress and returns the library to <b>Draft</b>.",
                bold,
              )}
        </p>
      </AlertDescription>
    </Alert>
  );
}

/**
 * The library's source: the editor with its status bar and the check's findings. A new,
 * empty library offers a starting file that checks and passes its test.
 */
export function LibrarySource() {
  const t = useT();
  const {
    mode,
    check,
    errors,
    warnings,
    testCount,
    tests,
    context,
    contextError,
    highlight,
    askNova,
    setEditorView,
    reveal,
    applyFix,
    openDefinition,
    rules,
  } = useLibraryStudio();
  const { refetch: retryContext } = useTariffCheckContext();
  const form = useFormContext<FormulaLibraryFormValues>();
  const [source, name] = useWatch({ control: form.control, name: ["source", "name"] });
  const text = source ?? "";
  const viewRef = useRef<EditorView | null>(null);
  const empty = text.trim() === "";
  const mismatch = !empty && headerMismatch(check.header?.name, name ?? "");

  const status = sourceStatus({ empty, errors, warnings }, t);

  const items = check.outline?.items ?? [];
  const functions = items.filter((item) => item.kind === "function").length;
  const declarations = items.filter(
    (item) => item.kind !== "function" && item.kind !== "test" && item.kind !== "header",
  ).length;

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

  const start = () =>
    form.setValue("source", starterLibrary((name ?? "").trim()), {
      shouldDirty: true,
      shouldValidate: true,
    });

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-end justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="flex items-center gap-1.5 text-base font-semibold">
            <Code01Icon className="text-foreground-subtle size-4" />
            {t("Source")}
          </h2>
          <p className="text-foreground-muted text-xs">
            {t(
              "Functions, constants, types and enums, with the tests that pin what they return. Everything is exported unless marked private.",
            )}
          </p>
        </div>
        <NovaButton onClick={() => askNova()}>{t("Ask Nova")}</NovaButton>
      </div>

      <MaterialBanner />

      {contextError && (
        <Alert variant="warning" size="sm" role="status">
          <InfoCircleIcon />
          <AlertDescription>
            {t(
              "Your other libraries could not be loaded, so the check cannot see what this one imports yet.",
            )}
          </AlertDescription>
          <AlertAction>
            <Button type="button" variant="ghost" size="xs" onClick={retryContext}>
              {t("Retry")}
            </Button>
          </AlertAction>
        </Alert>
      )}

      {mode === "create" && empty && (
        <div className="bg-card flex flex-wrap items-center justify-between gap-3 rounded-lg border border-dashed px-4 py-3">
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="text-sm font-medium">{t("Start from a working library")}</span>
            <span className="text-foreground-muted text-xs">
              {t("One function and a test that passes, to rewrite into your own.")}
            </span>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={start}>
            {t("Use the starter")}
          </Button>
        </div>
      )}

      {mismatch && (
        <Alert variant="warning" size="sm">
          <InfoCircleIcon />
          <AlertDescription>
            {t(
              "The header names {0}, but templates import this library as {1}. Name them the same so the source says what it is.",
              check.header?.name ?? "",
              libraryPath((name ?? "").trim()),
            )}
          </AlertDescription>
        </Alert>
      )}

      <div
        className={cn(
          "ui-container-focus-ring bg-card overflow-hidden rounded-lg border",
          status.tone === "bad" && "border-danger-border",
        )}
      >
        <Controller
          name="source"
          control={form.control}
          render={({ field }) => (
            <TariffEditor
              kind="library"
              value={field.value ?? ""}
              onChange={field.onChange}
              context={context}
              highlight={highlight}
              onView={handleView}
              onOpenDefinition={openDefinition}
              onFormat={(result) => reportFormat(result, t)}
              ariaLabel={t("Library source")}
              placeholder={t('library "org.name"')}
              minHeight={mode === "create" && empty ? "96px" : "320px"}
              primary
              embedded
            />
          )}
        />
        <SourceStatusBar
          status={status}
          summary={t(
            "{0, plural, one {# function} other {# functions}} · {1, plural, one {# declaration} other {# declarations}} · {2, plural, one {# test} other {# tests}}",
            functions,
            declarations,
            testCount,
          )}
          tests={tests}
          onFormat={format}
          formatDisabled={empty}
        />
        <ProblemsList
          diagnostics={check.diagnostics}
          source={text}
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
    </section>
  );
}
