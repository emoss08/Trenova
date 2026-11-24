import { Kept } from "@/components/kept";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { RecordPresence, RecordViewers } from "@/components/presence/data-table-viewers";
import { SectionStrip } from "@/components/section-strip";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import { WorkPaneHost, WorkPaneProvider, lazyWorkPane, useWorkPane } from "@/components/work-pane";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { lazyComponent } from "@trenova/shared/lib/lazy-component";
import { whenIdle } from "@/lib/when-idle";
import {
  worker as workerQueries,
  workerOverviewQuery,
  workerRecordQuery,
} from "@/lib/queries/worker";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Resource } from "@trenova/shared/types/permission";
import { useEffect, useState, type ReactNode } from "react";
import { ActivitySkeleton } from "./_components/activity/activity-skeleton";
import { DevelopmentSkeleton } from "./_components/development/development-skeleton";
import {
  ChecklistsSkeletonBody,
  EmploymentSkeleton,
  HistorySkeletonBody,
  LifecycleRouteSkeleton,
} from "./_components/employment/employment-skeleton";
import { FilesSkeleton } from "./_components/files/files-skeleton";
import { OverviewSkeleton } from "./_components/overview/overview-skeleton";
import { workerStripItems, workerTabLabel } from "./_components/overview-model";
import { PaySkeleton } from "./_components/pay/pay-skeleton";
import type { ProfileTabProps } from "./_components/profile/profile-tab";
import type { ProfileDock } from "./_components/profile/profile-dock";
import { ProfileSkeleton } from "./_components/profile/profile-skeleton";
import { QualificationSkeleton } from "./_components/qualification/qualification-skeleton";
import { RosterStepper } from "./_components/roster-stepper";
import { SafetySkeleton } from "./_components/safety/safety-skeleton";
import { TimeSkeleton } from "./_components/time/time-skeleton";
import { WorkerHeader } from "./_components/worker-header";
import { WorkerPageSkeleton } from "./_components/worker-page-skeleton";
import { preloadWorkerForm } from "./_components/worker-form-panes";
import { preloadWorkerView } from "./_components/worker-view-modules";
import { WorkerStripProvider } from "./_components/worker-strip-slot";
import { WorkerFormDropContext } from "./_components/use-worker-form-gate";
import {
  CLOSED_WORKER_OPEN_STATE,
  WORKER_TABS,
  useWorkerState,
  workerFormBlockedByAddress,
  type WorkerRecord,
  type WorkerTab,
  type WorkerTabProps,
  type WorkerTarget,
  type WorkerView,
  type WorkerViewOf,
} from "../use-worker-state";
import { useWorkerAreas } from "../worker-areas";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";

const pick = <P extends object>(module: { default: (props: P) => ReactNode }) => module.default;

const TAB_ENTRIES = {
  overview: lazyComponent(
    () => import("./_components/overview/overview-tab"),
    pick<WorkerTabProps>,
  ),
  employment: lazyComponent(
    () => import("./_components/employment/employment-tab"),
    pick<WorkerTabProps>,
  ),
  qualification: lazyComponent(
    () => import("./_components/qualification/qualification-tab"),
    pick<WorkerTabProps>,
  ),
  development: lazyComponent(
    () => import("./_components/development/development-tab"),
    pick<WorkerTabProps>,
  ),
  safety: lazyComponent(() => import("./_components/safety/safety-tab"), pick<WorkerTabProps>),
  time: lazyComponent(() => import("./_components/time/time-tab"), pick<WorkerTabProps>),
  pay: lazyComponent(() => import("./_components/pay/pay-tab"), pick<WorkerTabProps>),
  files: lazyComponent(() => import("./_components/files/files-tab"), pick<WorkerTabProps>),
  activity: lazyComponent(
    () => import("./_components/activity/activity-tab"),
    pick<WorkerTabProps>,
  ),
};

const ProfileEntry = lazyComponent(
  () => import("./_components/profile/profile-tab"),
  pick<ProfileTabProps>,
);

const ProfileChange = lazyWorkPane(
  () => import("./_components/profile-change/profile-change-pane"),
  (module) => module.ProfileChangePane,
  {
    isOpen: (props) => props.requestId !== null,
    openKey: (props) => props.requestId,
    title: (props, t) => t("Profile change from {0}", props.workerFirstName),
    fields: 3,
  },
);

// The Profile form's save dock arrives with the Profile tab's first save-able form, never before,
// so its code (the dock, react-hook-form's provider) is read then rather than with the page.
const ProfileDockEntry = lazyComponent(
  () => import("./_components/profile/profile-dock"),
  (module) => module.WorkerProfileDock,
);

function preloadTab(tab: WorkerTab) {
  void (tab === "profile" ? ProfileEntry.preload() : TAB_ENTRIES[tab].preload());
}

/** Each tab's own loading state, for the view the page is about to show. */
function TabSkeleton({ tab, view }: { tab: WorkerTab; view: WorkerView | null }) {
  switch (tab) {
    case "overview":
      return <OverviewSkeleton />;
    case "profile":
      return <ProfileSkeleton />;
    case "employment":
      return <EmploymentSkeleton view={(view ?? undefined) as WorkerViewOf<"employment">} />;
    case "qualification":
      return <QualificationSkeleton view={(view ?? undefined) as WorkerViewOf<"qualification">} />;
    case "development":
      return <DevelopmentSkeleton view={(view ?? undefined) as WorkerViewOf<"development">} />;
    case "safety":
      return <SafetySkeleton />;
    case "time":
      return <TimeSkeleton view={(view ?? undefined) as WorkerViewOf<"time">} />;
    case "pay":
      return <PaySkeleton view={(view ?? undefined) as WorkerViewOf<"pay">} />;
    case "files":
      return <FilesSkeleton view={(view ?? undefined) as WorkerViewOf<"files">} />;
    case "activity":
      return <ActivitySkeleton />;
  }
}

/**
 * Tabs that draw something above their section strip and so place it themselves, through
 * `WorkerSectionStrip`: Employment puts its lifecycle route first. The shell still builds the strip;
 * every other tab gets it drawn above its content.
 */
const TABS_PLACING_THEIR_STRIP: ReadonlySet<WorkerTab> = new Set(["employment"]);

/** A strip-placing tab's loading state, with the strip already where the tab will put it. */
function StripPlacingSkeleton({
  tab,
  view,
  strip,
}: {
  tab: WorkerTab;
  view: WorkerView | null;
  strip: ReactNode;
}) {
  const t = useT();
  if (tab !== "employment") return <TabSkeleton tab={tab} view={view} />;
  return (
    <div
      aria-busy
      aria-label={
        view === "checklists" ? t("Loading the checklists") : t("Loading the employment history")
      }
      className="flex flex-col gap-4"
    >
      <LifecycleRouteSkeleton />
      {strip}
      <div aria-hidden className="contents">
        {view === "checklists" ? <ChecklistsSkeletonBody /> : <HistorySkeletonBody />}
      </div>
    </div>
  );
}

export const prefetch: RoutePrefetch = ({ params }) =>
  params.id ? [workerQueries.record(params.id), workerQueries.overview(params.id)] : [];

/**
 * One worker as a page: whether they are cleared to drive and until when, what in their file needs
 * someone, and every part of the file a tab away with how it stands before the click.
 */
export function WorkerPage() {
  const t = useT();
  const { id = "" } = useParams({ strict: false });
  const record = useQuery({ ...workerRecordQuery(id), enabled: id !== "" });
  useBreadcrumbLabel(record.data ? record.data.wholeName || record.data.firstName : undefined);
  const [waitedFor, setWaitedFor] = useState<string | null>(null);
  if (record.isLoading && waitedFor !== id) setWaitedFor(id);

  if (record.isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{ title: t("Worker"), description: t("Loading the worker...") }}
        bleed
      >
        <WorkerPageSkeleton />
      </PageLayout>
    );
  }

  if (record.error || !record.data) {
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Worker"),
          description: t("This worker could not be loaded."),
        }}
      >
        <ErrorState
          layout="sheet"
          error={record.error}
          onRetry={record.error ? () => void record.refetch() : undefined}
          title={record.error ? undefined : t("No worker has this link")}
          description={
            record.error
              ? undefined
              : t("They may have been deleted, or they belong to another organization.")
          }
        />
      </PageLayout>
    );
  }

  return (
    <RecordPresence resource={Resource.Worker} recordId={record.data.id}>
      <WorkerView key={record.data.id} worker={record.data} reveal={waitedFor === id} />
    </RecordPresence>
  );
}

function WorkerView({ worker, reveal }: { worker: WorkerRecord; reveal: boolean }) {
  const t = useT();
  const [paneDirty, setPaneDirty] = useState(false);
  const [profileDirty, setProfileDirty] = useState(false);
  const { state } = useWorkerState();
  const areas = useWorkerAreas();
  const area = areas.resolve({ tab: state.tab, view: state.view });

  return (
    <PageLayout
      pageHeaderProps={{
        title: worker.wholeName || `${worker.firstName} ${worker.lastName}`.trim(),
        description: t(
          "A worker's whole file: whether they are cleared to drive, what needs attention, and every part of their record.",
        ),
        actions: (
          <>
            <RosterStepper workerId={worker.id} tab={area.tab} view={area.view} />
            <RecordViewers />
          </>
        ),
      }}
      bleed
    >
      <WorkPaneProvider onDirtyChange={setPaneDirty}>
        <UnsavedChangesGuard when={profileDirty || paneDirty} />
        <WorkerFile
          worker={worker}
          reveal={reveal}
          profileDirty={profileDirty}
          onProfileDirtyChange={setProfileDirty}
        />
      </WorkPaneProvider>
    </PageLayout>
  );
}

type Direction = "forward" | "back" | null;

function WorkerFile({
  worker,
  reveal,
  profileDirty,
  onProfileDirtyChange,
}: {
  worker: WorkerRecord;
  reveal: boolean;
  profileDirty: boolean;
  onProfileDirtyChange: (dirty: boolean) => void;
}) {
  const t = useT();
  const { closeThen, isOpen: paneOpen } = useWorkPane();
  const { state, openState, open, dropForm } = useWorkerState();
  // The Profile form's save dock, handed up by the Profile tab once it has been shown. It lives in
  // this worker's keyed subtree, so stepping to another worker drops it with the old form.
  const [profileDock, setProfileDock] = useState<ProfileDock | null>(null);
  const areas = useWorkerAreas();
  const overview = useQuery(workerOverviewQuery(worker.id));
  const record = useQuery({ ...workerRecordQuery(worker.id), enabled: false });
  const area = areas.resolve({ tab: state.tab, view: state.view });
  const tab = area.tab;

  // Once the browser is idle, the other views of the tab showing are read, so switching views
  // draws at once; a hidden tab reads nothing.
  const idleViews = areas.viewsOf(tab).join(" ");
  useEffect(() => {
    if (idleViews === "") return undefined;
    return whenIdle(() => {
      for (const view of idleViews.split(" ")) preloadWorkerView(view as WorkerView);
    });
  }, [idleViews]);

  // A form the address alone rules out (its record not named, its view not drawable) leaves the
  // address at once; one whose record turns out missing is refused by its tab
  // (`useWorkerFormGate`). Drawability is known once the reader's permissions are.
  const permissionsKnown = usePermissionStore((store) => store.manifest) !== null;
  const blockedForm =
    permissionsKnown && workerFormBlockedByAddress({ ...state, tab }, areas.isDrawable);
  useEffect(() => {
    if (blockedForm) dropForm();
  }, [blockedForm, dropForm]);

  // A link that opens a form reads the form's pane alongside the tab, not after it.
  const deepLinkedForm = blockedForm ? null : state.form;
  useEffect(() => {
    if (deepLinkedForm) preloadWorkerForm(deepLinkedForm);
  }, [deepLinkedForm]);

  // The view each tab last showed, so coming back to a tab lands where the person left it.
  const [views, setViews] = useState<Partial<Record<WorkerTab, WorkerView>>>(() =>
    area.view ? { [tab]: area.view } : {},
  );
  if (area.view && views[tab] !== area.view) setViews({ ...views, [tab]: area.view });

  // Which way the pane enters: from the right for a later tab, from the left for an earlier one.
  const [shownTab, setShownTab] = useState(tab);
  const [direction, setDirection] = useState<Direction>(null);
  if (shownTab !== tab) {
    setDirection(WORKER_TABS.indexOf(tab) > WORKER_TABS.indexOf(shownTab) ? "forward" : "back");
    setShownTab(tab);
  }

  const go = (target: WorkerTarget) => closeThen(() => open(target));
  const selectTab = (next: WorkerTab) => {
    const remembered = views[next];
    go((remembered ? { tab: next, view: remembered } : { tab: next }) as WorkerTarget);
  };
  const selectView = (owner: WorkerTab, view: WorkerView) =>
    go({ tab: owner, view } as WorkerTarget);
  const viewOf = (key: WorkerTab): WorkerView | null =>
    key === tab ? area.view : (views[key] ?? areas.viewsOf(key)[0] ?? null);

  const tabProps = (key: WorkerTab): WorkerTabProps => ({
    workerId: worker.id,
    worker,
    view: viewOf(key),
    open: key === tab ? openState(key) : CLOSED_WORKER_OPEN_STATE,
    onOpen: open,
    shown: key === tab,
  });

  const pane = (key: WorkerTab, node: ReactNode) => {
    const stripViews = areas.viewsOf(key);
    const shown = key === tab;
    const selected = viewOf(key);
    const placesOwnStrip = TABS_PLACING_THEIR_STRIP.has(key);
    const strip =
      stripViews.length > 1 && selected ? (
        <SectionStrip<WorkerView>
          items={workerStripItems(stripViews, overview.data, t)}
          value={selected}
          onValueChange={(view) => selectView(key, view)}
          aria-label={t("{0} views", workerTabLabel(key, t))}
          loading={overview.isLoading}
          onValueIntent={preloadWorkerView}
        />
      ) : null;
    return (
      <Kept key={key} shown={shown}>
        <div
          className={cn(
            "flex min-w-0 flex-col gap-4 p-4",
            shown && direction === "forward" && "animate-slide-in",
            shown && direction === "back" && "animate-slide-in-back",
            "motion-reduce:animate-none",
          )}
        >
          {placesOwnStrip ? null : strip}
          <LazyComponent
            fallback={
              placesOwnStrip ? (
                <StripPlacingSkeleton tab={key} view={selected} strip={strip} />
              ) : (
                <TabSkeleton tab={key} view={selected} />
              )
            }
          >
            {placesOwnStrip ? (
              <WorkerStripProvider strip={strip}>{node}</WorkerStripProvider>
            ) : (
              node
            )}
          </LazyComponent>
        </div>
      </Kept>
    );
  };

  const entry = (key: Exclude<WorkerTab, "profile">) => {
    const Entry = TAB_ENTRIES[key].Component;
    return pane(key, <Entry {...tabProps(key)} />);
  };

  return (
    <>
      <div
        className={cn(
          "flex min-w-0 flex-1 flex-col",
          reveal && "animate-reveal motion-reduce:animate-none",
        )}
      >
        <WorkerHeader
          worker={worker}
          overview={overview.data}
          overviewUpdatedAt={overview.dataUpdatedAt}
          overviewFetching={overview.isFetching || record.isFetching}
          onRefresh={() => {
            void overview.refetch();
            void record.refetch();
          }}
          tabs={areas.tabs}
          tab={tab}
          onTab={selectTab}
          onTabIntent={preloadTab}
          profileUnsaved={profileDirty}
          onOpen={go}
        />
        <WorkerFormDropContext value={dropForm}>
          <WorkPaneHost className="flex-1 text-sm">
            {areas.tabs.map((key) =>
              key === "profile"
                ? pane(key, <ProfileEntry.Component {...tabProps(key)} onDock={setProfileDock} />)
                : entry(key),
            )}
          </WorkPaneHost>
        </WorkerFormDropContext>
        <ProfileChange.Component
          workerId={worker.id}
          workerFirstName={worker.firstName}
          requestId={state.request}
          onClose={() => open((area.view ? { tab, view: area.view } : { tab }) as WorkerTarget)}
        />
      </div>
      {profileDock ? (
        // Beside the host, never inside it: the host is a containing block (an inline-size
        // container), and the tab panes animate with transforms, so a fixed dock in there would sit
        // at the end of the tab and hide with it. Here it is fixed to the viewport, and its spacer
        // ends whatever tab is showing.
        <LazyComponent fallback={null}>
          <ProfileDockEntry.Component
            dock={profileDock}
            onDirtyChange={onProfileDirtyChange}
            lifted={paneOpen}
          />
        </LazyComponent>
      ) : null}
    </>
  );
}
