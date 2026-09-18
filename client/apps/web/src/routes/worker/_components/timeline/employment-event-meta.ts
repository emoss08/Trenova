import type { WorkerEmploymentEventRow } from "@/lib/graphql/worker-employment";
import {
  EMPLOYMENT_EVENT_LABELS,
  employmentEventKindSchema,
  type EmploymentEventKind,
} from "@trenova/shared/types/worker-employment";
import {
  ArrowLeftRightIcon,
  BadgeCheckIcon,
  BanIcon,
  BriefcaseIcon,
  CalendarOffIcon,
  CalendarCheckIcon,
  CircleDollarSignIcon,
  DoorOpenIcon,
  RotateCcwIcon,
  TrendingUpIcon,
  UserCheckIcon,
  type LucideIcon,
} from "lucide-react";

export type EmploymentEventMeta = {
  label: string;
  icon: LucideIcon;
  /** Ring/icon colour on the timeline rail. */
  toneClass: string;
  /** Short helper shown under the kind picker. */
  hint: string;
};

const META: Record<EmploymentEventKind, EmploymentEventMeta> = {
  Hired: {
    label: EMPLOYMENT_EVENT_LABELS.Hired,
    icon: BriefcaseIcon,
    toneClass: "bg-success/15 text-success-foreground ring-success/30",
    hint: "Opens the timeline. Recorded automatically when a worker is created.",
  },
  ProbationEnded: {
    label: EMPLOYMENT_EVENT_LABELS.ProbationEnded,
    icon: BadgeCheckIcon,
    toneClass: "bg-accent-sky/15 text-accent-sky-on-subtle ring-accent-sky/30",
    hint: "Informational — nothing on the worker changes.",
  },
  Promoted: {
    label: EMPLOYMENT_EVENT_LABELS.Promoted,
    icon: TrendingUpIcon,
    toneClass: "bg-accent-violet/15 text-accent-violet-on-subtle ring-accent-violet/30",
    hint: "Moves the worker to a new driver type or worker type.",
  },
  Transferred: {
    label: EMPLOYMENT_EVENT_LABELS.Transferred,
    icon: ArrowLeftRightIcon,
    toneClass: "bg-accent-indigo/15 text-accent-indigo-on-subtle ring-accent-indigo/30",
    hint: "Moves the worker to another fleet or manager.",
  },
  LeaveStarted: {
    label: EMPLOYMENT_EVENT_LABELS.LeaveStarted,
    icon: CalendarOffIcon,
    toneClass: "bg-warning/15 text-warning-foreground ring-warning/30",
    hint: "Takes the worker off the dispatch board until the leave ends.",
  },
  LeaveEnded: {
    label: EMPLOYMENT_EVENT_LABELS.LeaveEnded,
    icon: CalendarCheckIcon,
    toneClass: "bg-success/15 text-success-foreground ring-success/30",
    hint: "Returns the worker to the dispatch board.",
  },
  Suspended: {
    label: EMPLOYMENT_EVENT_LABELS.Suspended,
    icon: BanIcon,
    toneClass: "bg-warning/15 text-warning-foreground ring-warning/30",
    hint: "Blocks dispatch without ending employment.",
  },
  Reinstated: {
    label: EMPLOYMENT_EVENT_LABELS.Reinstated,
    icon: UserCheckIcon,
    toneClass: "bg-success/15 text-success-foreground ring-success/30",
    hint: "Lifts a suspension.",
  },
  Terminated: {
    label: EMPLOYMENT_EVENT_LABELS.Terminated,
    icon: DoorOpenIcon,
    toneClass: "bg-danger/15 text-danger-foreground ring-danger/30",
    hint: "Ends employment: closes PTO and pay assignments and cancels upcoming time off.",
  },
  Rehired: {
    label: EMPLOYMENT_EVENT_LABELS.Rehired,
    icon: RotateCcwIcon,
    toneClass: "bg-success/15 text-success-foreground ring-success/30",
    hint: "Reopens employment and enrols the worker in the default PTO policy.",
  },
  RateChanged: {
    label: EMPLOYMENT_EVENT_LABELS.RateChanged,
    icon: CircleDollarSignIcon,
    toneClass: "bg-accent-teal/15 text-accent-teal-on-subtle ring-accent-teal/30",
    hint: "Notes a pay change. Pay profiles still drive settlement.",
  },
};

export function employmentEventMeta(kind: string): EmploymentEventMeta {
  return META[kind as EmploymentEventKind] ?? META.ProbationEnded;
}

export const ALL_EMPLOYMENT_EVENT_KINDS = employmentEventKindSchema.options;

export type EmploymentSnapshot = {
  status: string;
  openLeave: boolean;
  openSuspension: boolean;
  hasHire: boolean;
};

/** Mirrors the server's EmploymentStateOf so the picker only offers what will save. */
export function employmentSnapshot(
  status: string,
  history: readonly Pick<WorkerEmploymentEventRow, "kind">[],
): EmploymentSnapshot {
  const snapshot: EmploymentSnapshot = {
    status,
    openLeave: false,
    openSuspension: false,
    hasHire: false,
  };
  for (const event of history) {
    switch (event.kind) {
      case "Hired":
      case "Rehired":
        snapshot.hasHire = true;
        break;
      case "LeaveStarted":
        snapshot.openLeave = true;
        break;
      case "LeaveEnded":
        snapshot.openLeave = false;
        break;
      case "Suspended":
        snapshot.openSuspension = true;
        break;
      case "Reinstated":
        snapshot.openSuspension = false;
        break;
      case "Terminated":
        snapshot.openLeave = false;
        snapshot.openSuspension = false;
        break;
      default:
        break;
    }
  }
  return snapshot;
}

/** Kinds the server's CanRecord will accept for this snapshot, in picker order. */
export function recordableKinds(snapshot: EmploymentSnapshot): EmploymentEventKind[] {
  const employed = snapshot.status === "Active";
  if (!employed) {
    return snapshot.hasHire ? ["Rehired"] : ["Hired", "Rehired"];
  }
  const kinds: EmploymentEventKind[] = [];
  if (!snapshot.hasHire) kinds.push("Hired");
  kinds.push("ProbationEnded", "Promoted", "Transferred", "RateChanged");
  kinds.push(snapshot.openLeave ? "LeaveEnded" : "LeaveStarted");
  kinds.push(snapshot.openSuspension ? "Reinstated" : "Suspended");
  kinds.push("Terminated");
  return kinds;
}
