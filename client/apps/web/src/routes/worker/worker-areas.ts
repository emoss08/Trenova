import { PlanFeature, type PlanFeatureKey } from "@/lib/plan-feature";
import { isFeatureOnPlan, useUnavailableFeatures } from "@trenova/shared/hooks/use-plan-feature";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type UnavailableFeature } from "@trenova/shared/types/permission";
import {
  WORKER_TABS,
  WORKER_TAB_VIEWS,
  type WorkerArea,
  type WorkerTab,
  type WorkerView,
} from "./use-worker-state";

type AreaGate = {
  feature?: PlanFeatureKey;
  resource?: string;
};

const TAB_GATES: Readonly<Record<WorkerTab, AreaGate>> = {
  overview: { resource: Resource.Worker },
  profile: { resource: Resource.Worker },
  employment: {},
  qualification: {},
  development: {},
  safety: { feature: PlanFeature.WorkforceSafety, resource: Resource.WorkerSafetyEvent },
  time: {},
  pay: {},
  files: {},
  activity: { resource: Resource.AuditLog },
};

const VIEW_GATES: Readonly<Record<WorkerView, AreaGate>> = {
  history: { resource: Resource.WorkerEmploymentEvent },
  checklists: { resource: Resource.WorkerChecklist },
  credentials: { feature: PlanFeature.WorkforceCompliance, resource: Resource.WorkerCredential },
  "dq-file": { feature: PlanFeature.WorkforceCompliance, resource: Resource.Qualification },
  "drug-and-alcohol": {
    feature: PlanFeature.WorkforceCompliance,
    resource: Resource.WorkerDOTTest,
  },
  training: { feature: PlanFeature.WorkforceTalent, resource: Resource.WorkerTraining },
  reviews: { feature: PlanFeature.WorkforceTalent, resource: Resource.PerformanceReview },
  "hours-of-service": { resource: Resource.Worker },
  schedule: { feature: PlanFeature.WorkforceTimeTracking, resource: Resource.WorkerSchedule },
  "time-off": { feature: PlanFeature.WorkforceTimeOff, resource: Resource.WorkerPTO },
  leave: { feature: PlanFeature.WorkforceTimeOff, resource: Resource.WorkerLeave },
  "pay-profile": { feature: PlanFeature.DriverPay, resource: Resource.DriverPayProfile },
  benefits: { feature: PlanFeature.WorkforceBenefits, resource: Resource.BenefitPlan },
  documents: { resource: Resource.Document },
  "driver-portal": { feature: PlanFeature.DriverPortal, resource: Resource.DriverPortal },
  "agent-memory": { feature: PlanFeature.AIDesk, resource: Resource.Assistant },
};

export type WorkerAreaAccess = {
  unavailable: readonly UnavailableFeature[];
  canRead: (resource: string) => boolean;
};

function passes(gate: AreaGate, access: WorkerAreaAccess): boolean {
  if (!isFeatureOnPlan(access.unavailable, gate.feature)) return false;
  if (gate.resource && !access.canRead(gate.resource)) return false;
  return true;
}

export function isWorkerViewDrawable(view: WorkerView, access: WorkerAreaAccess): boolean {
  return passes(VIEW_GATES[view], access);
}

export function drawableWorkerViews(
  tab: WorkerTab,
  access: WorkerAreaAccess,
): readonly WorkerView[] {
  if (!passes(TAB_GATES[tab], access)) return [];
  return (WORKER_TAB_VIEWS[tab] as readonly WorkerView[]).filter((view) =>
    isWorkerViewDrawable(view, access),
  );
}

export function isWorkerTabDrawable(tab: WorkerTab, access: WorkerAreaAccess): boolean {
  if (!passes(TAB_GATES[tab], access)) return false;
  return WORKER_TAB_VIEWS[tab].length === 0 || drawableWorkerViews(tab, access).length > 0;
}

export function drawableWorkerTabs(access: WorkerAreaAccess): readonly WorkerTab[] {
  return WORKER_TABS.filter((tab) => isWorkerTabDrawable(tab, access));
}

export function firstDrawableWorkerView(
  tab: WorkerTab,
  access: WorkerAreaAccess,
): WorkerView | null {
  return drawableWorkerViews(tab, access)[0] ?? null;
}

export function resolveWorkerArea(
  requested: { tab: string | null | undefined; view: string | null | undefined },
  access: WorkerAreaAccess,
): WorkerArea {
  const tab =
    (WORKER_TABS as readonly string[]).includes(requested.tab ?? "") &&
    isWorkerTabDrawable(requested.tab as WorkerTab, access)
      ? (requested.tab as WorkerTab)
      : "overview";
  const views = drawableWorkerViews(tab, access);
  if (views.length === 0) return { tab, view: null };
  const view = views.find((candidate) => candidate === requested.view) ?? views[0] ?? null;
  return { tab, view };
}

export type WorkerAreas = {
  tabs: readonly WorkerTab[];
  viewsOf: (tab: WorkerTab) => readonly WorkerView[];
  isDrawable: (area: WorkerArea) => boolean;
  resolve: (requested: { tab: string | null; view: string | null }) => WorkerArea;
};

export function workerAreas(access: WorkerAreaAccess): WorkerAreas {
  const views = Object.fromEntries(
    WORKER_TABS.map((tab) => [tab, drawableWorkerViews(tab, access)]),
  ) as Record<WorkerTab, readonly WorkerView[]>;
  const tabs = drawableWorkerTabs(access);
  return {
    tabs,
    viewsOf: (tab) => views[tab],
    isDrawable: ({ tab, view }) =>
      tabs.includes(tab) && (view === null ? views[tab].length === 0 : views[tab].includes(view)),
    resolve: (requested) => resolveWorkerArea(requested, access),
  };
}

export function useWorkerAreas(): WorkerAreas {
  const unavailable = useUnavailableFeatures();
  const manifest = usePermissionStore((state) => state.manifest);
  const hasPermission = usePermissionStore((state) => state.hasPermission);
  return workerAreas({
    unavailable,
    canRead: (resource) => (manifest ? hasPermission(resource, Operation.Read) : false),
  });
}
