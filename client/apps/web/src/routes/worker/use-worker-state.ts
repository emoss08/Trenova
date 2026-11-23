import { appPath } from "@/lib/app-path";
import type { WorkerRecord } from "@/lib/graphql/worker-record";
import { useSearchState } from "@trenova/shared/hooks/use-search-state";
import { searchLiteral, searchString } from "@trenova/shared/lib/router/search-codec";

export type { WorkerRecord } from "@/lib/graphql/worker-record";

export const WORKERS_PATH = appPath("/hr/workers");

export const NEW_WORKER_PATH = `${WORKERS_PATH}/new`;

export const WORKER_TABS = [
  "overview",
  "profile",
  "employment",
  "qualification",
  "development",
  "safety",
  "time",
  "pay",
  "files",
  "activity",
] as const;

export type WorkerTab = (typeof WORKER_TABS)[number];

export const WORKER_TAB_VIEWS = {
  overview: [],
  profile: [],
  employment: ["history", "checklists"],
  qualification: ["credentials", "dq-file", "drug-and-alcohol"],
  development: ["training", "reviews"],
  safety: [],
  time: ["hours-of-service", "schedule", "time-off", "leave"],
  pay: ["pay-profile", "benefits"],
  files: ["documents", "driver-portal", "agent-memory"],
  activity: [],
} as const satisfies Record<WorkerTab, readonly string[]>;

export type WorkerViewOf<T extends WorkerTab> = (typeof WORKER_TAB_VIEWS)[T][number];

export type WorkerView = WorkerViewOf<WorkerTab>;

export const WORKER_VIEWS: readonly WorkerView[] = WORKER_TABS.flatMap(
  (tab) => WORKER_TAB_VIEWS[tab] as readonly WorkerView[],
);

export const WORKER_RECORD_PARAMS = [
  "event",
  "checklist",
  "credential",
  "employer",
  "test",
  "query",
  "training",
  "course",
  "review",
  "safetyEvent",
  "action",
  "injury",
  "timeOff",
  "leave",
  "request",
] as const;

export type WorkerRecordParam = (typeof WORKER_RECORD_PARAMS)[number];

export const WORKER_FORM_HOMES = {
  "record-event": { tab: "employment", view: "history" },
  "amend-event": { tab: "employment", view: "history" },
  "add-credential": { tab: "qualification", view: "credentials" },
  "renew-credential": { tab: "qualification", view: "credentials" },
  "edit-credential": { tab: "qualification", view: "credentials" },
  "add-employer": { tab: "qualification", view: "dq-file" },
  "edit-employer": { tab: "qualification", view: "dq-file" },
  "record-collection": { tab: "qualification", view: "drug-and-alcohol" },
  "record-test-result": { tab: "qualification", view: "drug-and-alcohol" },
  "log-query": { tab: "qualification", view: "drug-and-alcohol" },
  "answer-query": { tab: "qualification", view: "drug-and-alcohol" },
  "assign-training": { tab: "development", view: "training" },
  "record-training-result": { tab: "development", view: "training" },
  "start-review": { tab: "development", view: "reviews" },
  "edit-review": { tab: "development", view: "reviews" },
  "record-safety-event": { tab: "safety", view: null },
  "issue-action": { tab: "safety", view: null },
  "add-recognition": { tab: "safety", view: null },
  "record-injury": { tab: "safety", view: null },
  "assign-shift": { tab: "time", view: "schedule" },
  "request-time-off": { tab: "time", view: "time-off" },
  "change-policy": { tab: "time", view: "time-off" },
  "adjust-balance": { tab: "time", view: "time-off" },
  "open-leave-case": { tab: "time", view: "leave" },
  "assign-pay-profile": { tab: "pay", view: "pay-profile" },
  "enroll-benefit": { tab: "pay", view: "benefits" },
  "upload-document": { tab: "files", view: "documents" },
} as const satisfies Record<string, WorkerArea>;

export type WorkerForm = keyof typeof WORKER_FORM_HOMES;

/**
 * The record a form edits, for the forms that edit one: the address must name it (`event=` for
 * `form=amend-event`), or the form cannot open. A form not listed needs no record.
 */
export const WORKER_FORM_RECORDS = {
  "amend-event": "event",
  "renew-credential": "credential",
  "edit-credential": "credential",
  "edit-employer": "employer",
  "record-test-result": "test",
  "answer-query": "query",
  "edit-review": "review",
} as const satisfies Partial<Record<WorkerForm, WorkerRecordParam>>;

/**
 * Whether the address names a form that cannot open whatever the data says: one that edits a
 * record the address does not name, one away from its own tab, or one whose view the reader may
 * not draw. The page drops such a form from the address; a form that names a record which turns
 * out missing or not editable is refused by its tab through `useWorkerFormGate`.
 */
export function workerFormBlockedByAddress(
  state: { tab: WorkerTab; form: WorkerForm | null } & WorkerRecordRefsState,
  isDrawable: (area: WorkerArea) => boolean,
): boolean {
  const { form } = state;
  if (form === null) return false;
  const home: WorkerArea = WORKER_FORM_HOMES[form];
  if (home.tab !== state.tab || !isDrawable(home)) return true;
  const record = (WORKER_FORM_RECORDS as Partial<Record<WorkerForm, WorkerRecordParam>>)[form];
  return record !== undefined && !state[record];
}

export const WORKER_FORMS = Object.keys(WORKER_FORM_HOMES) as WorkerForm[];

export type WorkerArea = { tab: WorkerTab; view: WorkerView | null };

export type WorkerRecordRefs = Partial<Record<WorkerRecordParam, string>>;

type WorkerRecordRefsState = { [K in WorkerRecordParam]: string | null };

type TargetFor<T extends WorkerTab> = {
  tab: T;
  view?: WorkerViewOf<T>;
  form?: WorkerForm;
  kind?: string;
} & WorkerRecordRefs;

export type WorkerTarget = { [T in WorkerTab]: TargetFor<T> }[WorkerTab];

export type WorkerOpenState = {
  form: WorkerForm | null;
  kind: string | null;
} & { [K in WorkerRecordParam]: string | null };

export type WorkerTabProps = {
  workerId: string;
  worker: WorkerRecord;
  view: WorkerView | null;
  open: WorkerOpenState;
  onOpen: (target: WorkerTarget) => void;
  shown: boolean;
};

const recordParsers = Object.fromEntries(
  WORKER_RECORD_PARAMS.map((param) => [param, searchString]),
) as { [K in WorkerRecordParam]: typeof searchString };

export const workerPageParsers = {
  tab: searchLiteral(WORKER_TABS).withDefault("overview"),
  view: searchLiteral(WORKER_VIEWS),
  form: searchLiteral(WORKER_FORMS),
  kind: searchString,
  ...recordParsers,
};

export const CLOSED_WORKER_OPEN_STATE: WorkerOpenState = Object.freeze({
  form: null,
  kind: null,
  ...(Object.fromEntries(WORKER_RECORD_PARAMS.map((param) => [param, null])) as {
    [K in WorkerRecordParam]: null;
  }),
});

export function workerViewsOf(tab: WorkerTab): readonly WorkerView[] {
  return WORKER_TAB_VIEWS[tab];
}

export function isWorkerViewOf(
  tab: WorkerTab,
  view: string | null | undefined,
): view is WorkerView {
  return view != null && (WORKER_TAB_VIEWS[tab] as readonly string[]).includes(view);
}

export function resolveWorkerView(
  tab: WorkerTab,
  requested: string | null | undefined,
): WorkerView | null {
  if (isWorkerViewOf(tab, requested)) return requested;
  return WORKER_TAB_VIEWS[tab][0] ?? null;
}

export function workerTabOfView(view: WorkerView): WorkerTab {
  return WORKER_TABS.find((tab) => isWorkerViewOf(tab, view)) ?? "overview";
}

export function workerFormTarget(form: WorkerForm, kind?: string): WorkerTarget {
  const home: WorkerArea = WORKER_FORM_HOMES[form];
  return {
    tab: home.tab,
    ...(home.view ? { view: home.view } : {}),
    form,
    ...(kind ? { kind } : {}),
  } as WorkerTarget;
}

export function workerPath(workerId: string, target?: WorkerTarget): string {
  const path = `${WORKERS_PATH}/${encodeURIComponent(workerId)}`;
  if (!target) return path;
  const params = new URLSearchParams();
  if (target.tab !== "overview") params.set("tab", target.tab);
  if (target.view) params.set("view", target.view);
  for (const param of WORKER_RECORD_PARAMS) {
    const value = target[param];
    if (value) params.set(param, value);
  }
  if (target.form) params.set("form", target.form);
  if (target.form && target.kind) params.set("kind", target.kind);
  const query = params.toString();
  return query === "" ? path : `${path}?${query}`;
}

export function workerOpenStateFor(
  state: WorkerOpenState & { tab: WorkerTab },
  tab: WorkerTab,
): WorkerOpenState {
  if (state.tab !== tab) return CLOSED_WORKER_OPEN_STATE;
  const open: WorkerOpenState = { ...CLOSED_WORKER_OPEN_STATE, form: state.form };
  if (state.form) open.kind = state.kind;
  for (const param of WORKER_RECORD_PARAMS) open[param] = state[param];
  return open;
}

function targetPatch(target: WorkerTarget) {
  return {
    ...CLOSED_WORKER_OPEN_STATE,
    tab: target.tab,
    view: target.view ?? null,
    form: target.form ?? null,
    kind: target.form ? (target.kind ?? null) : null,
    ...Object.fromEntries(WORKER_RECORD_PARAMS.map((param) => [param, target[param] ?? null])),
  };
}

export function useWorkerState() {
  const [state, setState] = useSearchState(workerPageParsers);

  const selectTab = (tab: WorkerTab) =>
    setState({ ...CLOSED_WORKER_OPEN_STATE, tab, view: null }, { history: "push" });

  const selectView = (view: WorkerView) =>
    setState(
      { ...CLOSED_WORKER_OPEN_STATE, tab: workerTabOfView(view), view },
      { history: "push" },
    );

  const open = (target: WorkerTarget) => setState(targetPatch(target), { history: "push" });

  const closeWork = () => setState({ ...CLOSED_WORKER_OPEN_STATE });

  /** Takes a form that cannot open out of the address, in place, keeping what else it names. */
  const dropForm = () => setState({ form: null, kind: null }, { history: "replace" });

  return {
    state,
    openState: (tab: WorkerTab) => workerOpenStateFor(state, tab),
    selectTab,
    selectView,
    open,
    closeWork,
    dropForm,
  };
}
