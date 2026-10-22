import { ActiveEditorProvider } from "@/components/formula-editor/active-editor";
import { DefinitionDialog } from "@/components/tariff-studio/definition-dialog";
import { OutlinePanel } from "@/components/tariff-studio/outline-panel";
import { REFERENCE_SEARCH_ID } from "@/components/tariff-studio/reference-search";
import { newTestBlock } from "@/components/tariff-studio/source-edits";
import { useStudioShortcuts } from "@/components/tariff-studio/studio-shortcuts";
import {
  NovaPaneSkeleton,
  ReferencePaneSkeleton,
  SourceSectionSkeleton,
  StudioSkeleton,
  TestsPaneSkeleton,
} from "@/components/tariff-studio/studio-skeletons";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import { archiveRefusal } from "@/lib/archive-refusal";
import { formulaLibraryRoutes } from "@/lib/formula-library-routes";
import { invalidateFormulaLibrary } from "@/lib/queries/formula-library";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { XCloseIcon } from "@trenova/shared/components/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type {
  FormulaLibrary,
  FormulaLibraryFormValues,
} from "@trenova/shared/types/formula-library";
import { Suspense, lazy, useCallback, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useNavigate } from "react-router";
import "@/components/tariff-studio/tariff-studio.css";
import { LibraryDetails } from "./library-details";
import { LibraryHistoryPane } from "./library-history-pane";
import type { LibraryDecision } from "./library-review-dialogs";
import {
  LibraryStudioProvider,
  useLibraryStudio,
  type LibraryStudioProps,
  type LibraryStudioTab,
} from "./library-studio-context";
import { LibraryTopBar } from "./library-top-bar";
import { LibraryUsagePane, LibraryUsagePaneSkeleton } from "./library-usage-pane";

const LibrarySource = lazy(() =>
  import("./library-source").then((module) => ({ default: module.LibrarySource })),
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
const LibraryNovaPane = lazy(() =>
  import("./library-nova-pane").then((module) => ({ default: module.LibraryNovaPane })),
);
const LibrarySubmitDialog = lazy(() =>
  import("./library-review-dialogs").then((module) => ({
    default: module.LibrarySubmitDialog,
  })),
);
const LibraryDecisionDialog = lazy(() =>
  import("./library-review-dialogs").then((module) => ({
    default: module.LibraryDecisionDialog,
  })),
);

export type LibraryStudioPageProps = LibraryStudioProps & {
  /** A rollback or a decision replaced the saved library; the page reseats its form on it. */
  onLibraryChanged?: (library: FormulaLibrary) => void;
};

/**
 * A formula library's create and edit page: a top bar over the outline, the library's name,
 * description and source, and a panel of its tests, the reference, who imports it, its history
 * and Nova.
 */
export function LibraryStudio({ onLibraryChanged, ...props }: LibraryStudioPageProps) {
  const t = useT();
  return (
    <ActiveEditorProvider>
      <Suspense fallback={<StudioSkeleton inputs={false} label={t("Loading the library")} />}>
        <LibraryStudioProvider {...props}>
          <LibraryStudioLayout onLibraryChanged={onLibraryChanged} />
        </LibraryStudioProvider>
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
  const { tab, setTab, setRailOpen, tests, novaBusy, library } = useLibraryStudio();
  const tabs: { value: LibraryStudioTab; label: string }[] = [
    { value: "tests", label: t("Tests") },
    { value: "reference", label: t("Reference") },
    ...(library
      ? [
          { value: "usage" as const, label: t("Imported by") },
          { value: "history" as const, label: t("History") },
        ]
      : []),
  ];
  const total = tests.status === "ran" ? tests.passed + tests.failed : 0;
  const allPass = tests.status === "ran" && total > 0 && tests.failed === 0;

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
        aria-label={t("Library panes")}
        className="flex min-w-0 flex-1 items-center gap-1 overflow-x-auto"
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
            {item.value === "tests" && tests.status === "ran" && total > 0 && (
              <span
                className={cn(
                  "rounded-badge text-2xs px-1.5 font-mono tabular-nums",
                  allPass
                    ? "bg-success-subtle text-success-subtle-foreground"
                    : "bg-danger-subtle text-danger-subtle-foreground",
                )}
              >
                {tests.passed}/{total}
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

function RailBody({ onRestored }: { onRestored: (library: FormulaLibrary) => void }) {
  const t = useT();
  const { tab, tests, check, context, reveal, appendToSource, library } = useLibraryStudio();
  const { control } = useFormContext<FormulaLibraryFormValues>();
  const source = useWatch({ control, name: "source" }) ?? "";
  const novaOpened = useOnceOpened(tab === "nova");
  // The usage and history tabs exist only once the library is saved.
  const shown = !library && (tab === "usage" || tab === "history") ? "tests" : tab;
  const addTest = () =>
    appendToSource(
      newTestBlock(
        "library",
        (check.outline?.items ?? [])
          .filter((item) => item.kind === "test")
          .map((item) => item.name),
        t("a new test"),
      ),
    );

  return (
    <>
      {shown !== "reference" && shown !== "nova" && (
        <ScrollArea maskVariant="card" className="min-h-0 flex-1">
          {shown === "tests" && (
            <Suspense fallback={<TestsPaneSkeleton />}>
              <TestsPanel state={tests} source={source} onReveal={reveal} onAddTest={addTest} />
            </Suspense>
          )}
          {shown === "usage" && (
            <Suspense fallback={<LibraryUsagePaneSkeleton />}>
              <LibraryUsagePane />
            </Suspense>
          )}
          {shown === "history" && <LibraryHistoryPane onRestored={onRestored} />}
        </ScrollArea>
      )}
      {shown === "reference" && (
        <Suspense fallback={<ReferencePaneSkeleton />}>
          <ReferencePane context={context} />
        </Suspense>
      )}
      {novaOpened && (
        <div className={cn("flex min-h-0 flex-1 flex-col", shown !== "nova" && "hidden")}>
          <Suspense fallback={<NovaPaneSkeleton />}>
            <LibraryNovaPane />
          </Suspense>
        </div>
      )}
    </>
  );
}

function FormColumn() {
  return (
    <div className="mx-auto flex w-full max-w-200 flex-col gap-7 px-8 pt-5 pb-32 max-[900px]:px-4">
      <LibraryDetails />
      <Suspense fallback={<SourceSectionSkeleton />}>
        <LibrarySource />
      </Suspense>
    </div>
  );
}

function ArchiveDialog({
  open,
  onOpenChange,
  libraryId,
  onArchived,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  libraryId: string;
  onArchived: (library: FormulaLibrary) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const [held, setHeld] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const close = (next: boolean) => {
    if (!next) setHeld(null);
    onOpenChange(next);
  };

  // The server asks for a confirmation, in its own words, when templates still import the
  // library; the dialog shows it and offers "Archive anyway".
  const archive = async () => {
    setBusy(true);
    try {
      const archived = await apiService.formulaLibraryService.archive(libraryId, {
        confirmInUse: held !== null,
      });
      toast.success(t("Library archived"));
      onArchived(archived);
      close(false);
    } catch (error) {
      const refusal = archiveRefusal(error);
      if (refusal?.kind === "needsConfirmation") {
        setHeld(refusal.message);
      } else {
        toast.error(refusal?.message ?? t("The library could not be archived"));
      }
    } finally {
      setBusy(false);
      await invalidateFormulaLibrary(queryClient);
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={close}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="text-lg font-semibold">
            {t("Archive this library?")}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {held ??
              t(
                "New template versions can no longer import it. Template versions that already import it keep the version they pinned.",
              )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="outline" size="default">
            {t("Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="default"
            onClick={(event) => {
              event.preventDefault();
              void archive();
            }}
            disabled={busy}
            isLoading={busy}
          >
            {held === null ? t("Archive") : t("Archive anyway")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function LibraryStudioLayout({
  onLibraryChanged,
}: {
  onLibraryChanged?: (library: FormulaLibrary) => void;
}) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const studio = useLibraryStudio();
  const {
    mode,
    library,
    dirty,
    isSubmitting,
    onSave,
    setTab,
    tab,
    stacked,
    railOpen,
    check,
    testStatus,
    reveal,
    definition,
    context,
    closeDefinition,
  } = studio;

  const [submitOpen, setSubmitOpen] = useState(false);
  const [decision, setDecision] = useState<LibraryDecision | null>(null);
  const [archiveOpen, setArchiveOpen] = useState(false);
  const submitOpened = useOnceOpened(submitOpen);
  const decisionOpened = useOnceOpened(decision !== null);
  const railOpened = useOnceOpened(railOpen);

  const changed = useCallback(
    (next: FormulaLibrary) => onLibraryChanged?.(next),
    [onLibraryChanged],
  );

  const duplicate = useCallback(async () => {
    if (!library) return;
    try {
      const [copy] = await apiService.formulaLibraryService.duplicate([library.id]);
      await invalidateFormulaLibrary(queryClient);
      toast.success(t("Duplicated as a draft"));
      if (copy?.id) void navigate(formulaLibraryRoutes.edit(copy.id));
    } catch {
      toast.error(t("The library could not be duplicated"));
    }
  }, [library, navigate, queryClient, t]);

  const shortcuts = useMemo(
    () => ({
      save: () => {
        if (isSubmitting) return;
        if (mode === "create" || dirty) onSave();
      },
      search: () => {
        setTab("reference");
        window.setTimeout(() => document.getElementById(REFERENCE_SEARCH_ID)?.focus(), 30);
      },
      tests: () => setTab("tests"),
      nova: () => setTab(tab === "nova" && railOpen ? "tests" : "nova"),
    }),
    [isSubmitting, mode, dirty, onSave, setTab, tab, railOpen],
  );
  useStudioShortcuts(shortcuts);

  const rail = railOpened && (
    <aside
      aria-label={t("Library panel")}
      className={cn(
        "bg-card border-border-strong animate-panel-in flex min-h-0 flex-col overflow-hidden rounded-lg border",
        stacked ? "ui-lift absolute inset-x-2 top-2 bottom-2 z-30" : "my-2 mr-2",
        !railOpen && "hidden",
      )}
    >
      <RailTabs />
      <RailBody onRestored={changed} />
    </aside>
  );

  return (
    <>
      <div className="fs-studio bg-canvas flex h-full flex-col overflow-hidden">
        <UnsavedChangesGuard when={dirty} />
        <LibraryTopBar
          onSubmitForReview={() => setSubmitOpen(true)}
          onDecision={setDecision}
          onDuplicate={() => void duplicate()}
          onArchive={() => setArchiveOpen(true)}
        />

        {stacked ? (
          <div className="relative flex min-h-0 flex-1 flex-col">
            <ScrollArea maskVariant="background" className="min-h-0 flex-1">
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
            <ScrollArea maskVariant="background" className="min-h-0">
              <FormColumn />
            </ScrollArea>
            {rail}
          </div>
        )}

        {library && (
          <Suspense fallback={null}>
            {submitOpened && (
              <LibrarySubmitDialog
                open={submitOpen}
                onOpenChange={setSubmitOpen}
                library={library}
              />
            )}
            {decisionOpened && (
              <LibraryDecisionDialog
                decision={decision}
                onClose={() => setDecision(null)}
                library={library}
              />
            )}
          </Suspense>
        )}
        {library && (
          <ArchiveDialog
            open={archiveOpen}
            onOpenChange={setArchiveOpen}
            libraryId={library.id}
            onArchived={changed}
          />
        )}
      </div>
      <DefinitionDialog target={definition} context={context} onClose={closeDefinition} />
    </>
  );
}
