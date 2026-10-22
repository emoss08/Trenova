import { ActiveEditorProvider } from "@/components/formula-editor/active-editor";
import { PreviewValuesProvider } from "@/components/formula-editor/preview-values";
import { useStudioShortcuts } from "@/components/formula-editor/studio-shortcuts";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import {
  buildTemplateExport,
  downloadJson,
  getExportFilename,
} from "@/lib/formula-template-export";
import { formulaTemplateRoutes, importLandingRoute } from "@/lib/formula-template-routes";
import { apiService } from "@/services/api";
import { XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FormulaTemplateFormValues } from "@trenova/shared/types/formula-template";
import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useNavigate, useSearchParams } from "react-router";
import { toast } from "@trenova/shared/components/ui/toast";
import type { ApprovalAction } from "../approval-action-dialog";
import "./formula-studio.css";
import { RateTrace } from "./rate-trace";
import { REFERENCE_SEARCH_ID } from "./rail/reference-search";
import { DetailsHero } from "./sections/details-hero";
import { InputsSection } from "./sections/inputs-section";
import { PolicySection } from "./sections/policy-section";
import {
  STUDIO_SECTIONS,
  StudioProvider,
  sectionDomId,
  useStudio,
  useStudioActions,
  type StudioProps,
  type StudioTab,
} from "./studio-context";
import {
  BreakdownSectionSkeleton,
  ExpressionSectionSkeleton,
  NovaPaneSkeleton,
  PreviewPaneSkeleton,
  ReferencePaneSkeleton,
  ScenariosPaneSkeleton,
  StudioSkeleton,
} from "./studio-skeletons";
import { StudioTopBar } from "./studio-top-bar";

export type { StudioSaveExtras } from "./studio-context";

/*
 * The heavy parts of the studio arrive on their own: the editor (CodeMirror
 * and the language service), each rail pane, and every dialog, so the page
 * paints its frame, name and inputs first and the rest lands in place behind
 * a skeleton of its own shape. A dialog's module is read the first time it
 * opens and kept mounted after, so it closes with its animation.
 */
const ExpressionSection = lazy(() =>
  import("./sections/expression-section").then((module) => ({
    default: module.ExpressionSection,
  })),
);
const BreakdownSection = lazy(() =>
  import("./sections/breakdown-section").then((module) => ({
    default: module.BreakdownSection,
  })),
);
const PreviewPane = lazy(() =>
  import("./rail/preview-pane").then((module) => ({ default: module.PreviewPane })),
);
const ScenariosPane = lazy(() =>
  import("./rail/scenarios-pane").then((module) => ({ default: module.ScenariosPane })),
);
const ReferencePane = lazy(() =>
  import("./rail/reference-pane").then((module) => ({ default: module.ReferencePane })),
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
 * and the browser is idle, so the first switch to Scenarios, Reference or
 * Nova and the first Submit land without a skeleton. A failed read is left
 * to the real open, which reports it.
 */
function warmStudioModules(): () => void {
  const warm = () => {
    void Promise.allSettled([
      import("./rail/scenarios-pane"),
      import("./rail/reference-pane"),
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
 * The formula template's create and edit form: a top bar over the rate
 * trace, the form, and a floating panel of preview, scenarios, reference and
 * Nova that sits beside the form, or over it on a narrow screen.
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

/** True from the first time a flag is raised, so a piece opened once stays mounted. */
function useOnceOpened(open: boolean): boolean {
  const [opened, setOpened] = useState(open);
  if (open && !opened) setOpened(true);
  return opened;
}

function RailTabs() {
  const t = useT();
  const { tab, setTab, setRailOpen, scenarios, novaBusy } = useStudio();
  const tabs: { value: StudioTab; label: string }[] = [
    { value: "preview", label: t("Preview") },
    { value: "scenarios", label: t("Scenarios") },
    { value: "reference", label: t("Reference") },
  ];
  const allPass = scenarios.total > 0 && scenarios.passed === scenarios.total;

  const tabClass = (active: boolean) =>
    cn(
      "ui-focus-ring relative flex h-10 items-center gap-1.5 rounded-sm px-2 text-sm transition-colors",
      active
        ? "text-foreground after:bg-foreground font-medium after:absolute after:inset-x-2 after:-bottom-px after:h-0.5 after:rounded-full"
        : "text-foreground-muted hover:text-foreground",
    );

  return (
    <div className="flex shrink-0 items-center gap-1 border-b pr-1.5 pl-2">
      <div
        role="tablist"
        aria-label={t("Studio panes")}
        className="flex min-w-0 flex-1 items-center gap-1"
      >
        {tabs.map((item) => (
          <button
            key={item.value}
            type="button"
            role="tab"
            aria-selected={tab === item.value}
            onClick={() => setTab(item.value)}
            className={tabClass(tab === item.value)}
          >
            {item.label}
            {item.value === "scenarios" && scenarios.total > 0 && (
              <span
                className={cn(
                  "rounded-badge px-1.5 font-mono text-2xs tabular-nums",
                  allPass
                    ? "bg-success-subtle text-success-subtle-foreground"
                    : "bg-danger-subtle text-danger-subtle-foreground",
                )}
              >
                {scenarios.passed}/{scenarios.total}
              </span>
            )}
          </button>
        ))}
        <span className="flex-1" />
        <button
          type="button"
          role="tab"
          aria-selected={tab === "nova"}
          onClick={() => setTab("nova")}
          className={cn(tabClass(tab === "nova"), "fs-nova-tab")}
        >
          <NovaBadge size="xs" className="fs-nova-badge" />
          {t("Nova")}
          {novaBusy && <Spinner className="size-3" />}
        </button>
      </div>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label={t("Close panel")}
        onClick={() => setRailOpen(false)}
      >
        <XCloseIcon className="size-4" />
      </Button>
    </div>
  );
}

/**
 * The pane the tab shows. Nova's stays mounted once it has been opened and
 * is only hidden behind another tab, so the conversation, a reply under
 * way and what it applied are all there when the person comes back to it.
 * The reference pane scrolls itself: its list is virtualized.
 */
function RailBody() {
  const { tab } = useStudio();
  const novaOpened = useOnceOpened(tab === "nova");
  return (
    <>
      {(tab === "preview" || tab === "scenarios") && (
        <ScrollArea maskVariant="card" className="min-h-0 flex-1">
          {tab === "preview" && (
            <Suspense fallback={<PreviewPaneSkeleton />}>
              <PreviewPane />
            </Suspense>
          )}
          {tab === "scenarios" && (
            <Suspense fallback={<ScenariosPaneSkeleton />}>
              <ScenariosPane />
            </Suspense>
          )}
        </ScrollArea>
      )}
      {tab === "reference" && (
        <Suspense fallback={<ReferencePaneSkeleton />}>
          <ReferencePane />
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
      <Suspense fallback={<ExpressionSectionSkeleton />}>
        <ExpressionSection />
      </Suspense>
      <InputsSection />
      <PolicySection />
      <Suspense fallback={<BreakdownSectionSkeleton />}>
        <BreakdownSection />
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
  } = studio;
  const form = useFormContext<FormulaTemplateFormValues>();
  const expression = useWatch({ control: form.control, name: "expression" }) ?? "";

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
      scenarios: () => setTab("scenarios"),
      nova: () => setTab(studio.tab === "nova" && railOpen ? "preview" : "nova"),
    }),
    [isSubmitting, mode, dirty, save, preview.runNow, setTab, studio.tab, railOpen],
  );
  useStudioShortcuts(shortcuts);

  const handleExport = useCallback(async () => {
    if (!template) return;
    try {
      const testCases = await apiService.formulaTemplateService.listTestCases(template.id);
      const exportData = buildTemplateExport(template, { testCases });
      const filename = getExportFilename(template, false);
      downloadJson(exportData, filename);
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
    <PreviewValuesProvider receipt={preview.result?.receipt}>
      <div className="fs-studio bg-canvas flex h-full flex-col overflow-hidden">
        <UnsavedChangesGuard when={dirty} />
        <StudioTopBar
          onApprovalAction={setApprovalAction}
          onSubmitForReview={() => setSubmitOpen(true)}
          onVersionHistory={() => setVersionHistoryOpen(true)}
          onBacktest={() => setBacktestOpen(true)}
          onFork={() => setForkOpen(true)}
          onLineage={() => setLineageOpen(true)}
          onExport={() => void handleExport()}
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
            <RateTrace className="hidden min-h-0 border-r min-[1180px]:flex" />
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
            />
          )}

          {template && backtestOpened && (
            <BacktestDialog
              open={backtestOpen}
              onOpenChange={setBacktestOpen}
              template={template}
              expression={expression}
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
              onRollback={(updated) => {
                form.reset(updated as unknown as FormulaTemplateFormValues);
                onTemplateChanged?.(updated);
              }}
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
    </PreviewValuesProvider>
  );
}
