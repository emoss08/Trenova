import { ActiveEditorProvider } from "@/components/formula-editor/active-editor";
import { isTestItem } from "@/lib/tariff-outline";
import { DefinitionDialog } from "@/components/tariff-studio/definition-dialog";
import { READINESS_KEYS } from "@/components/tariff-studio/readiness";
import { OutlinePanel } from "@/components/tariff-studio/outline-panel";
import { REFERENCE_SEARCH_ID } from "@/components/tariff-studio/reference-search";
import { ExplainTest } from "./rail/explain-test";
import { useStudioShortcuts } from "@/components/tariff-studio/studio-shortcuts";
import { StudioRailTabs, useOnceOpened } from "@/components/tariff-studio/studio-rail-tabs";
import {
  ListSectionSkeleton,
  NovaPaneSkeleton,
  PreviewPaneSkeleton,
  ReferencePaneSkeleton,
  SourceSectionSkeleton,
  StudioSkeleton,
  TestsPaneSkeleton,
} from "@/components/tariff-studio/studio-skeletons";
import { newTestBlock } from "@/components/tariff-studio/source-edits";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import {
  buildTemplateExport,
  downloadJson,
  getExportFilename,
} from "@/lib/formula-template-export";
import { formulaTemplateRoutes, importLandingRoute } from "@/lib/formula-template-routes";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import type { ApprovalAction } from "../approval-action-dialog";
import "@/components/tariff-studio/tariff-studio.css";
import { DetailsHero } from "./sections/details-hero";
import { InputsSection } from "./sections/inputs-section";
import {
  STUDIO_SECTIONS,
  StudioProvider,
  sectionDomId,
  useStudio,
  useStudioActions,
  type StudioProps,
  type StudioTab,
} from "./studio-context";
import { StudioTopBar } from "./studio-top-bar";

/*
 * The heavy parts of the studio arrive on their own: the editor (CodeMirror
 * and the language service), each rail pane, and every dialog, so the page
 * paints its frame, name and inputs first and the rest lands in place behind
 * a skeleton of its own shape. A dialog's module is read the first time it
 * opens and kept mounted after, so it closes with its animation.
 */
const SourceSection = lazy(() =>
  import("./sections/source-section").then((module) => ({
    default: module.SourceSection,
  })),
);
const TestsSection = lazy(() =>
  import("./sections/tests-section").then((module) => ({ default: module.TestsSection })),
);
const PreviewPane = lazy(() =>
  import("./rail/preview-pane").then((module) => ({ default: module.PreviewPane })),
);
const TestsPanel = lazy(() =>
  import("@/components/tariff-studio/tests-panel").then((module) => ({
    default: module.TestsPanel,
  })),
);
const ReferencePane = lazy(() =>
  import("@/components/tariff-studio/reference-pane").then((module) => ({
    default: module.ReferencePane,
  })),
);
const NovaPane = lazy(() =>
  import("./rail/nova-pane").then((module) => ({ default: module.NovaPane })),
);
const ApprovalActionDialog = lazy(() =>
  import("../approval-action-dialog").then((module) => ({
    default: module.ApprovalActionDialog,
  })),
);
const SubmitReviewDialog = lazy(() =>
  import("./dialogs/submit-review-dialog").then((module) => ({
    default: module.SubmitReviewDialog,
  })),
);
const BacktestDialog = lazy(() =>
  import("./dialogs/backtest-dialog").then((module) => ({ default: module.BacktestDialog })),
);
const BacktestSheet = lazy(() =>
  import("./backtest-sheet").then((module) => ({ default: module.BacktestSheet })),
);
const VersionHistoryPanel = lazy(() =>
  import("../version/version-history-panel").then((module) => ({
    default: module.VersionHistoryPanel,
  })),
);
const ForkTemplateDialog = lazy(() =>
  import("../fork-template-dialog").then((module) => ({ default: module.ForkTemplateDialog })),
);
const ForkLineageDialog = lazy(() =>
  import("../fork-lineage-dialog").then((module) => ({ default: module.ForkLineageDialog })),
);
const ImportTemplateDialog = lazy(() =>
  import("./import-template-dialog").then((module) => ({
    default: module.ImportTemplateDialog,
  })),
);

const SPY_OFFSET = 140;
const WARM_AFTER_MS = 1500;

/**
 * Reads the modules a tab or a button will need once the page has painted
 * and the browser is idle, so the first switch to Tests, Reference or
 * Nova and the first Submit land without a skeleton. A failed read is left
 * to the real open, which reports it.
 */
function warmStudioModules(): () => void {
  const warm = () => {
    void Promise.allSettled([
      import("@/components/tariff-studio/tests-panel"),
      import("@/components/tariff-studio/reference-pane"),
      import("./rail/nova-pane"),
      import("./dialogs/submit-review-dialog"),
    ]);
  };
  if (typeof window.requestIdleCallback === "function") {
    const handle = window.requestIdleCallback(warm, { timeout: WARM_AFTER_MS * 2 });
    return () => window.cancelIdleCallback(handle);
  }
  const timer = window.setTimeout(warm, WARM_AFTER_MS);
  return () => window.clearTimeout(timer);
}

/**
 * The formula template's create and edit form: a top bar over the outline,
 * the form, and a floating panel of preview, tests, reference and Nova that
 * sits beside the form, or over it on a narrow screen.
 */
export function FormulaStudio(props: StudioProps) {
  // The provider suspends until the rules module has loaded (useRules), so
  // the studio's own outline holds the page meanwhile.
  return (
    <ActiveEditorProvider>
      <Suspense fallback={<StudioSkeleton />}>
        <StudioProvider {...props}>
          <StudioLayout />
        </StudioProvider>
      </Suspense>
    </ActiveEditorProvider>
  );
}

function RailTabs() {
  const t = useT();
  const { tab, setTab, setRailOpen, tests, novaBusy } = useStudio();
  const tabs: { value: Exclude<StudioTab, "nova">; label: string }[] = [
    { value: "preview", label: t("Preview") },
    { value: "tests", label: t("Tests") },
    { value: "reference", label: t("Reference") },
  ];

  return (
    <StudioRailTabs
      label={t("Studio panes")}
      tabs={tabs}
      tab={tab}
      onTab={setTab}
      tests={tests}
      novaBusy={novaBusy}
      onClose={() => setRailOpen(false)}
    />
  );
}

/**
 * The pane the tab shows. Nova's stays mounted once it has been opened and
 * is only hidden behind another tab, so the conversation, a reply under
 * way and what it applied are all there when the person comes back to it.
 * The reference pane scrolls itself: its list is virtualized.
 */
function RailBody() {
  const t = useT();
  const { tab, tests, check, context, reveal, appendToSource } = useStudio();
  const { control } = useFormContext<FormulaTemplateFormValues>();
  const source = useWatch({ control, name: "source" }) ?? "";
  const novaOpened = useOnceOpened(tab === "nova");
  const addTest = () =>
    appendToSource(
      newTestBlock(
        "tariff",
        (check.outline?.items ?? []).filter(isTestItem).map((item) => item.name),
        t("a new test"),
      ),
    );
  return (
    <>
      {(tab === "preview" || tab === "tests") && (
        <ScrollArea maskVariant="card" className="min-h-0 flex-1">
          {tab === "preview" && (
            <Suspense fallback={<PreviewPaneSkeleton />}>
              <PreviewPane />
            </Suspense>
          )}
          {tab === "tests" && (
            <Suspense fallback={<TestsPaneSkeleton />}>
              <TestsPanel
                state={tests}
                source={source}
                onReveal={reveal}
                onAddTest={addTest}
                onSaveAsTest={appendToSource}
                failureAside={(test) => <ExplainTest key={test.name} name={test.name} />}
              />
            </Suspense>
          )}
        </ScrollArea>
      )}
      {tab === "reference" && (
        <Suspense fallback={<ReferencePaneSkeleton />}>
          <ReferencePane context={context} />
        </Suspense>
      )}
      {novaOpened && (
        <div className={cn("flex min-h-0 flex-1 flex-col", tab !== "nova" && "hidden")}>
          <Suspense fallback={<NovaPaneSkeleton />}>
            <NovaPane />
          </Suspense>
        </div>
      )}
    </>
  );
}

function FormColumn() {
  return (
    <div className="mx-auto flex w-full max-w-200 flex-col gap-7 px-8 pt-5 pb-32 max-[900px]:px-4">
      <DetailsHero />
      <Suspense fallback={<SourceSectionSkeleton />}>
        <SourceSection />
      </Suspense>
      <InputsSection />
      <Suspense fallback={<ListSectionSkeleton />}>
        <TestsSection />
      </Suspense>
    </div>
  );
}

function useSectionSpy() {
  const { formViewport, setActiveSection } = useStudioActions();
  useEffect(() => {
    const viewport = formViewport.current;
    if (!viewport) return;
    const update = () => {
      const top = viewport.getBoundingClientRect().top;
      let current = STUDIO_SECTIONS[0];
      for (const section of STUDIO_SECTIONS) {
        const element = document.getElementById(sectionDomId(section));
        if (element && element.getBoundingClientRect().top - top <= SPY_OFFSET) {
          current = section;
        }
      }
      setActiveSection(current);
    };
    update();
    viewport.addEventListener("scroll", update, { passive: true });
    return () => viewport.removeEventListener("scroll", update);
  }, [formViewport, setActiveSection]);
}

function StudioLayout() {
  const t = useT();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const studio = useStudio();
  const {
    mode,
    template,
    dirty,
    isSubmitting,
    save,
    setTab,
    preview,
    formViewport,
    onTemplateChanged,
    stacked,
    railOpen,
    check,
    testStatus,
    reveal,
    definition,
    context,
    closeDefinition,
    readiness,
  } = studio;
  const form = useFormContext<FormulaTemplateFormValues>();
  const source = useWatch({ control: form.control, name: "source" }) ?? "";

  const [approvalAction, setApprovalAction] = useState<ApprovalAction | null>(null);
  const [submitOpen, setSubmitOpen] = useState(false);
  const [versionHistoryOpen, setVersionHistoryOpen] = useState(false);
  const [forkOpen, setForkOpen] = useState(false);
  const [lineageOpen, setLineageOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [backtestOpen, setBacktestOpen] = useState(false);
  const [backtestDetailsOpen, setBacktestDetailsOpen] = useState(false);

  const submitOpened = useOnceOpened(submitOpen);
  const versionHistoryOpened = useOnceOpened(versionHistoryOpen);
  const forkOpened = useOnceOpened(forkOpen);
  const lineageOpened = useOnceOpened(lineageOpen);
  const importOpened = useOnceOpened(importOpen);
  const backtestOpened = useOnceOpened(backtestOpen);
  const backtestDetailsOpened = useOnceOpened(backtestDetailsOpen);
  const railOpened = useOnceOpened(railOpen);

  useSectionSpy();
  useEffect(() => warmStudioModules(), []);

  // A reviewer following a notification lands on the decision, not the
  // editor: the approve dialog opens itself, then the marker leaves the URL so
  // a refresh does not reopen it.
  const reviewStep = searchParams.get("review");
  const opensApproval = reviewStep === "approve" && template?.status === "InReview";
  const [seenOpensApproval, setSeenOpensApproval] = useState(false);
  if (opensApproval !== seenOpensApproval) {
    setSeenOpensApproval(opensApproval);
    if (opensApproval) setApprovalAction("approve");
  }
  useEffect(() => {
    if (opensApproval) {
      setSearchParams(
        (params) => {
          params.delete("review");
          return params;
        },
        { replace: true },
      );
    }
  }, [opensApproval, setSearchParams]);

  const shortcuts = useMemo(
    () => ({
      save: () => {
        if (isSubmitting) return;
        if (mode === "create" || dirty) save();
      },
      run: preview.runNow,
      search: () => {
        setTab("reference");
        window.setTimeout(() => document.getElementById(REFERENCE_SEARCH_ID)?.focus(), 30);
      },
      preview: () => setTab("preview"),
      tests: () => setTab("tests"),
      nova: () => setTab(studio.tab === "nova" && railOpen ? "preview" : "nova"),
    }),
    [isSubmitting, mode, dirty, save, preview.runNow, setTab, studio.tab, railOpen],
  );
  useStudioShortcuts(shortcuts);

  const handleExport = useCallback(() => {
    if (!template) return;
    try {
      downloadJson(buildTemplateExport(template), getExportFilename(template, false));
    } catch {
      toast.error(t("Export failed"), {
        description: t("Could not export the template. Please try again."),
      });
    }
  }, [template, t]);

  // Kept mounted once opened and only hidden when closed, so Nova's
  // conversation and a reply under way survive closing the panel.
  const rail = railOpened && (
    <aside
      aria-label={t("Studio rail")}
      className={cn(
        "bg-card border-border-strong animate-panel-in flex min-h-0 flex-col overflow-hidden rounded-lg border",
        stacked ? "ui-lift absolute inset-x-2 top-2 bottom-2 z-30" : "my-2 mr-2",
        !railOpen && "hidden",
      )}
    >
      <RailTabs />
      <RailBody />
    </aside>
  );

  return (
    <>
      <div className="fs-studio bg-canvas flex h-full flex-col overflow-hidden">
        <UnsavedChangesGuard when={dirty} />
        <StudioTopBar
          onApprovalAction={setApprovalAction}
          onSubmitForReview={() => setSubmitOpen(true)}
          onVersionHistory={() => setVersionHistoryOpen(true)}
          onBacktest={() => setBacktestOpen(true)}
          onFork={() => setForkOpen(true)}
          onLineage={() => setLineageOpen(true)}
          onExport={handleExport}
          onImport={() => setImportOpen(true)}
        />

        {stacked ? (
          <div className="relative flex min-h-0 flex-1 flex-col">
            <ScrollArea
              maskVariant="background"
              viewportRef={formViewport}
              className="min-h-0 flex-1"
            >
              <FormColumn />
            </ScrollArea>
            {rail}
          </div>
        ) : (
          <div
            className={cn(
              "grid min-h-0 flex-1",
              railOpen
                ? "grid-cols-[minmax(0,1fr)_minmax(360px,420px)] min-[1180px]:grid-cols-[248px_minmax(0,1fr)_minmax(360px,420px)]"
                : "grid-cols-[minmax(0,1fr)] min-[1180px]:grid-cols-[248px_minmax(0,1fr)]",
            )}
          >
            <ScrollArea
              maskVariant="background"
              className="hidden min-h-0 border-r min-[1180px]:flex"
            >
              <OutlinePanel outline={check.outline} onReveal={reveal} testStatus={testStatus} />
            </ScrollArea>
            <ScrollArea maskVariant="background" viewportRef={formViewport} className="min-h-0">
              <FormColumn />
            </ScrollArea>
            {rail}
          </div>
        )}

        <Suspense fallback={null}>
          {approvalAction && (
            <ApprovalActionDialog
              open={approvalAction !== null}
              onOpenChange={(open) => {
                if (!open) setApprovalAction(null);
              }}
              action={approvalAction}
              template={template}
            />
          )}

          {template && submitOpened && (
            <SubmitReviewDialog
              open={submitOpen}
              onOpenChange={setSubmitOpen}
              template={template}
              liveChecks={readiness.filter((check) => check.key === READINESS_KEYS.coverage)}
              link={{
                source,
                onReveal: (range) => {
                  setSubmitOpen(false);
                  reveal(range);
                },
              }}
            />
          )}

          {template && backtestOpened && (
            <BacktestDialog
              open={backtestOpen}
              onOpenChange={setBacktestOpen}
              template={template}
              source={source}
              onOpenDetails={() => setBacktestDetailsOpen(true)}
            />
          )}

          {backtestDetailsOpened && (
            <BacktestSheet
              open={backtestDetailsOpen}
              onOpenChange={setBacktestDetailsOpen}
              form={form}
              template={template}
            />
          )}

          {versionHistoryOpened && (
            <VersionHistoryPanel
              open={versionHistoryOpen}
              onOpenChange={setVersionHistoryOpen}
              template={template}
              onRollback={onTemplateChanged}
            />
          )}

          {forkOpened && (
            <ForkTemplateDialog
              open={forkOpen}
              onOpenChange={setForkOpen}
              template={template}
              onForkSuccess={(forked) => {
                setForkOpen(false);
                if (forked.id) void navigate(formulaTemplateRoutes.edit(forked.id));
              }}
            />
          )}

          {lineageOpened && (
            <ForkLineageDialog
              open={lineageOpen}
              onOpenChange={setLineageOpen}
              templateId={template?.id}
              currentTemplateId={template?.id}
              onNavigateToTemplate={(templateId) => {
                setLineageOpen(false);
                if (templateId) void navigate(formulaTemplateRoutes.edit(templateId));
              }}
            />
          )}

          {importOpened && (
            <ImportTemplateDialog
              open={importOpen}
              onOpenChange={setImportOpen}
              onImported={(response) => void navigate(importLandingRoute(response))}
            />
          )}
        </Suspense>
      </div>
      <DefinitionDialog target={definition} context={context} onClose={closeDefinition} />
    </>
  );
}
