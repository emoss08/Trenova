import { translate } from "@trenova/shared/i18n/runtime";
import { defineLabels } from "@trenova/shared/i18n/labels";
import {
  employmentEventKindSchema,
  employmentEventLabel,
  type EmploymentEventKind,
} from "@trenova/shared/types/worker-employment";
import {
  CalendarCheck01Icon,
  CalendarRangeIcon,
  CurrencyDollarIcon,
  type IconComponent,
  LogOut01Icon,
  RefreshCcw01Icon,
  ShieldTickIcon,
  SlashCircle01Icon,
  SwitchHorizontal01Icon,
  TrendUp01Icon,
  UserCheck01Icon,
  UserPlus01Icon,
} from "@trenova/shared/components/icons";

/** The kind filter's groups, in the order the filter lists them. */
export const EMPLOYMENT_EVENT_GROUPS = ["status", "pay", "move", "leave"] as const;
export type EmploymentEventGroup = (typeof EMPLOYMENT_EVENT_GROUPS)[number];

export const EMPLOYMENT_EVENT_GROUP_LABELS: Record<EmploymentEventGroup, string> = defineLabels({
  status: "Hiring and status",
  pay: "Rate changes",
  move: "Transfers and promotions",
  leave: "Leave",
});

/** The tone of a kind's mark on the history rail. Tone is severity: leaving is danger. */
export type EmploymentEventTone = "success" | "warning" | "danger" | "info" | "neutral";

export type EmploymentEventMeta = {
  icon: IconComponent;
  tone: EmploymentEventTone;
  group: EmploymentEventGroup;
};

const META: Record<EmploymentEventKind, EmploymentEventMeta> = {
  Hired: { icon: UserPlus01Icon, tone: "success", group: "status" },
  Rehired: { icon: RefreshCcw01Icon, tone: "success", group: "status" },
  ProbationEnded: { icon: ShieldTickIcon, tone: "success", group: "status" },
  Promoted: { icon: TrendUp01Icon, tone: "info", group: "move" },
  Transferred: { icon: SwitchHorizontal01Icon, tone: "info", group: "move" },
  RateChanged: { icon: CurrencyDollarIcon, tone: "neutral", group: "pay" },
  LeaveStarted: { icon: CalendarRangeIcon, tone: "warning", group: "leave" },
  LeaveEnded: { icon: CalendarCheck01Icon, tone: "neutral", group: "leave" },
  Suspended: { icon: SlashCircle01Icon, tone: "warning", group: "status" },
  Reinstated: { icon: UserCheck01Icon, tone: "success", group: "status" },
  Terminated: { icon: LogOut01Icon, tone: "danger", group: "status" },
};

export const EMPLOYMENT_EVENT_TONE_CLASS: Record<EmploymentEventTone, string> = {
  success: "text-success-foreground",
  warning: "text-warning-foreground",
  danger: "text-danger-foreground",
  info: "text-info-foreground",
  neutral: "text-muted-foreground",
};

export const ALL_EMPLOYMENT_EVENT_KINDS = employmentEventKindSchema.options;

export function isEmploymentEventKind(value: string): value is EmploymentEventKind {
  return (ALL_EMPLOYMENT_EVENT_KINDS as readonly string[]).includes(value);
}

export function employmentEventMeta(kind: string): EmploymentEventMeta {
  return isEmploymentEventKind(kind) ? META[kind] : META.ProbationEnded;
}

/** The kind's name in the reader's language; a kind the client does not know reads as itself. */
export function employmentEventName(kind: string): string {
  return isEmploymentEventKind(kind) ? employmentEventLabel(kind) : translate("Employment event");
}

/** One line under the kind picker saying what the kind is for. */
export function employmentEventHint(kind: EmploymentEventKind): string {
  switch (kind) {
    case "Hired":
      return translate("Opens the employment. Recorded on its own when a worker is created.");
    case "Rehired":
      return translate("Reopens employment after a termination.");
    case "ProbationEnded":
      return translate("Records the end of probation.");
    case "Promoted":
      return translate("Moves the worker to a new driver type or worker type.");
    case "Transferred":
      return translate("Moves the worker to another fleet or manager.");
    case "RateChanged":
      return translate("Notes a pay change. Pay profiles still drive settlement.");
    case "LeaveStarted":
      return translate("Takes the worker off the dispatch board until the leave ends.");
    case "LeaveEnded":
      return translate("Returns the worker to the dispatch board.");
    case "Suspended":
      return translate("Takes the worker off dispatch without ending employment.");
    case "Reinstated":
      return translate("Lifts a suspension.");
    case "Terminated":
      return translate("Ends employment.");
  }
}

/** The kinds whose own fields change what saving does, so the preview reads again for them. */
export const KINDS_WITH_FIELDS: ReadonlySet<EmploymentEventKind> = new Set([
  "Transferred",
  "Promoted",
  "RateChanged",
  "LeaveStarted",
]);
