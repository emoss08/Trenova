import type {
  WorkerConcernCode,
  WorkerConcernSeverity,
  WorkerPageRecord,
  WorkerPageTab,
  WorkerPageView,
} from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { withoutRecordIds } from "@/lib/record-ids";
import {
  isWorkerViewOf,
  workerFormTarget,
  type WorkerForm,
  type WorkerRecordParam,
  type WorkerTab,
  type WorkerTarget,
  type WorkerView,
} from "@/routes/worker/use-worker-state";

/** What the server says about one concern; the words are built here, never sent. */
export type WorkerConcernFacts = {
  severity: WorkerConcernSeverity;
  code: WorkerConcernCode;
  subject: string | null;
  count: number;
  occurredAt: number | null;
  dueAt: number | null;
  daysUntil: number | null;
  stopsAssignment: boolean;
};

/** A place on the worker record page, as the server names it. */
export type WorkerDestination = {
  tab: WorkerPageTab;
  view: WorkerPageView | null;
  record: WorkerPageRecord | null;
  recordId: string | null;
};

export type WorkerConcernWords = {
  title: string;
  /** Null when there is nothing worth a second line. */
  detail: string | null;
  /** The one step that moves the concern, as its button reads. */
  action: string;
};

const TAB_SLUGS: Readonly<Record<WorkerPageTab, WorkerTab>> = {
  Overview: "overview",
  Profile: "profile",
  Employment: "employment",
  Qualification: "qualification",
  Development: "development",
  Safety: "safety",
  Time: "time",
  Pay: "pay",
  Files: "files",
  Activity: "activity",
};

const VIEW_SLUGS: Readonly<Record<WorkerPageView, WorkerView>> = {
  History: "history",
  Checklists: "checklists",
  Credentials: "credentials",
  DqFile: "dq-file",
  DrugAndAlcohol: "drug-and-alcohol",
  Training: "training",
  Reviews: "reviews",
  HoursOfService: "hours-of-service",
  Schedule: "schedule",
  TimeOff: "time-off",
  Leave: "leave",
  PayProfile: "pay-profile",
  Benefits: "benefits",
  Documents: "documents",
  DriverPortal: "driver-portal",
  AgentMemory: "agent-memory",
};

/**
 * The page's record parameter for a destination's record. An event is an employment event on
 * Employment and a safety event on Safety; a request is a time off request on Time and a
 * profile change request anywhere else.
 */
function recordParam(record: WorkerPageRecord, tab: WorkerPageTab): WorkerRecordParam {
  switch (record) {
    case "Credential":
      return "credential";
    case "Training":
      return "training";
    case "Review":
      return "review";
    case "Event":
      return tab === "Safety" ? "safetyEvent" : "event";
    case "Checklist":
      return "checklist";
    case "Leave":
      return "leave";
    case "Test":
      return "test";
    case "Employer":
      return "employer";
    case "Injury":
      return "injury";
    case "Request":
      return tab === "Time" ? "timeOff" : "request";
  }
}

/** Where a server destination opens on the worker page: its tab, its view and its record. */
export function workerDestinationTarget(destination: WorkerDestination): WorkerTarget {
  const tab = TAB_SLUGS[destination.tab];
  const view = destination.view ? VIEW_SLUGS[destination.view] : null;
  const target: Record<string, string> = { tab };
  if (view && isWorkerViewOf(tab, view)) target.view = view;
  if (destination.record && destination.recordId) {
    target[recordParam(destination.record, destination.tab)] = destination.recordId;
  }
  return target as WorkerTarget;
}

/** A form opened on its home tab, carrying the record the destination names. */
function formTarget(form: WorkerForm, destination: WorkerDestination): WorkerTarget {
  const refs =
    destination.record && destination.recordId
      ? { [recordParam(destination.record, destination.tab)]: destination.recordId }
      : {};
  return { ...workerFormTarget(form), ...refs } as WorkerTarget;
}

/** The form a concern's action opens instead of its destination, when there is one. */
function concernForm(
  concern: Pick<WorkerConcernFacts, "code">,
  destination: WorkerDestination,
): WorkerForm | null {
  switch (concern.code) {
    case "CredentialExpired":
    case "CredentialExpiring":
      return destination.recordId ? "renew-credential" : null;
    case "CredentialMissing":
      return "add-credential";
    case "TrainingMissing":
      return "assign-training";
    case "ClearinghouseQueryDue":
    case "ClearinghouseQueryOverdue":
      return "log-query";
    case "ReviewDue":
    case "ReviewOverdue":
      return destination.recordId ? null : "start-review";
    case "NotEmployed":
    case "NotAssignable":
    case "TrainingExpired":
    case "TrainingOverdue":
    case "TrainingDue":
    case "SafetyRating":
    case "OutOfService":
    case "ActiveDiscipline":
    case "OpenSafetyEvent":
    case "ChecklistOverdue":
    case "ChecklistOpen":
    case "DrugAlcoholProhibited":
    case "EmploymentVerificationOutstanding":
      return null;
  }
}

/** Where a concern's action goes: the form that fixes it, or the record it is about. */
export function workerConcernTarget(
  concern: Pick<WorkerConcernFacts, "code"> & { destination: WorkerDestination },
): WorkerTarget {
  const form = concernForm(concern, concern.destination);
  return form
    ? formTarget(form, concern.destination)
    : workerDestinationTarget(concern.destination);
}

/** The one record a concern names, when it is about exactly one and the server named it. */
function soleSubject(concern: WorkerConcernFacts): string | null {
  if (concern.count !== 1 || !concern.subject) return null;
  const subject = withoutRecordIds(concern.subject).trim();
  return subject === "" ? null : subject;
}

function date(at: number | null): string | null {
  return at ? formatUnixDateMedium(at) : null;
}

function lapsedTitle(subject: string, daysUntil: number | null, t: TranslateFn): string {
  if (daysUntil === null) return t("{0} has expired", subject);
  if (daysUntil >= 0) return t("{0} expired today", subject);
  return t("{0} expired {1, plural, one {# day} other {# days}} ago", subject, -daysUntil);
}

function credentialWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const subject = soleSubject(concern);
  const due = date(concern.dueAt);
  const days = concern.daysUntil;
  switch (concern.code) {
    case "CredentialExpired":
      return {
        title: subject
          ? lapsedTitle(subject, days, t)
          : t(
              "{0, plural, one {# credential has} other {# credentials have}} expired",
              concern.count,
            ),
        detail: subject
          ? due
            ? concern.stopsAssignment
              ? t("Lapsed {0}. Dispatch will not assign the driver until it is renewed.", due)
              : t("Lapsed {0}.", due)
            : concern.stopsAssignment
              ? t("Dispatch will not assign the driver until it is renewed.")
              : null
          : concern.stopsAssignment
            ? t("Dispatch will not assign the driver until they are renewed.")
            : null,
        action: subject ? t("Renew") : t("Open credentials"),
      };
    case "CredentialMissing":
      return {
        title: subject
          ? t("{0} is not on file", subject)
          : t(
              "{0, plural, one {# required credential is} other {# required credentials are}} not on file",
              concern.count,
            ),
        detail: concern.stopsAssignment
          ? subject
            ? t("Dispatch will not assign the driver until it is recorded.")
            : t("Dispatch will not assign the driver until they are recorded.")
          : subject
            ? t("It is required for the file.")
            : t("They are required for the file."),
        action: subject ? t("Add credential") : t("Open credentials"),
      };
    default:
      return {
        title: subject
          ? days === null
            ? t("{0} expires soon", subject)
            : days <= 0
              ? t("{0} expires today", subject)
              : t("{0} expires in {1, plural, one {# day} other {# days}}", subject, days)
          : t(
              "{0, plural, one {# credential expires} other {# credentials expire}} soon",
              concern.count,
            ),
        detail: subject
          ? due
            ? concern.stopsAssignment
              ? t("{0}. Dispatch stops assigning the driver when it lapses.", due)
              : t("Expires {0}.", due)
            : null
          : due
            ? t("The first expires {0}.", due)
            : null,
        action: subject ? t("Renew") : t("Open credentials"),
      };
  }
}

function trainingWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const subject = soleSubject(concern);
  const due = date(concern.dueAt);
  const assigned = date(concern.occurredAt);
  const days = concern.daysUntil;
  switch (concern.code) {
    case "TrainingExpired":
      return {
        title: subject
          ? lapsedTitle(subject, days, t)
          : t("{0, plural, one {# course has} other {# courses have}} expired", concern.count),
        detail: due
          ? subject
            ? concern.stopsAssignment
              ? t("Lapsed {0}. Dispatch will not assign the driver until it is retaken.", due)
              : t("Lapsed {0}.", due)
            : t("The first lapsed {0}.", due)
          : null,
        action: t("Open training"),
      };
    case "TrainingMissing":
      return {
        title: subject
          ? t("{0} has never been assigned", subject)
          : t(
              "{0, plural, one {# required course has} other {# required courses have}} never been assigned",
              concern.count,
            ),
        detail: subject
          ? t("Assign it so the driver can complete it.")
          : t("Assign them so the driver can complete them."),
        action: t("Assign training"),
      };
    case "TrainingOverdue":
      return {
        title: subject
          ? days !== null && days < 0
            ? t("{0} is {1, plural, one {# day} other {# days}} overdue", subject, -days)
            : t("{0} is overdue", subject)
          : t("{0, plural, one {# course is} other {# courses are}} overdue", concern.count),
        detail: subject
          ? assigned && due
            ? t("Assigned {0}, due {1}.", assigned, due)
            : due
              ? t("It was due {0}.", due)
              : null
          : due
            ? t("The oldest was due {0}.", due)
            : null,
        action: t("Open training"),
      };
    default:
      return {
        title: subject
          ? days === null
            ? t("{0} is due soon", subject)
            : days <= 0
              ? t("{0} is due today", subject)
              : t("{0} is due in {1, plural, one {# day} other {# days}}", subject, days)
          : t("{0, plural, one {# course is} other {# courses are}} due soon", concern.count),
        detail: subject
          ? assigned
            ? t("Assigned {0}.", assigned)
            : due
              ? t("Due {0}.", due)
              : null
          : due
            ? t("The first is due {0}.", due)
            : null,
        action: t("Open training"),
      };
  }
}

function safetyWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const subject = soleSubject(concern);
  const when = date(concern.occurredAt);
  switch (concern.code) {
    case "SafetyRating":
      return {
        title:
          concern.severity === "Critical"
            ? t("The safety record is at risk")
            : t("The safety record is on watch"),
        detail: t(
          "{0, plural, one {# active point} other {# active points}} on the record.",
          concern.count,
        ),
        action: t("Open safety"),
      };
    case "OutOfService":
      return {
        title: t(
          "{0, plural, one {# out-of-service order} other {# out-of-service orders}} on the record",
          concern.count,
        ),
        detail: when ? t("The latest on {0}.", when) : null,
        action: t("Open safety"),
      };
    case "ActiveDiscipline":
      return {
        title: t(
          "{0, plural, one {# disciplinary action is} other {# disciplinary actions are}} still standing",
          concern.count,
        ),
        detail: when ? t("The latest was issued {0}.", when) : null,
        action: t("Open safety"),
      };
    default:
      return {
        title: subject
          ? t("{0} is under review", subject)
          : t(
              "{0, plural, one {# safety event is} other {# safety events are}} under review",
              concern.count,
            ),
        detail: when ? t("It happened {0}.", when) : null,
        action: subject ? t("Open event") : t("Open safety"),
      };
  }
}

function checklistWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const subject = concern.subject ? withoutRecordIds(concern.subject).trim() || null : null;
  const started = date(concern.occurredAt);
  const due = date(concern.dueAt);
  const detail =
    started && due
      ? t("Started {0}, due {1}.", started, due)
      : started
        ? t("Started {0}.", started)
        : due
          ? t("Due {0}.", due)
          : null;
  if (concern.code === "ChecklistOverdue") {
    return {
      title: subject
        ? t(
            "{0} has {1, plural, one {# overdue item} other {# overdue items}}",
            subject,
            concern.count,
          )
        : t(
            "{0, plural, one {# checklist item is} other {# checklist items are}} overdue",
            concern.count,
          ),
      detail,
      action: t("Open checklist"),
    };
  }
  return {
    title: subject
      ? t("{0} is open", subject)
      : t("{0, plural, one {# checklist is} other {# checklists are}} open", concern.count),
    detail,
    action: t("Open checklist"),
  };
}

function reviewWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const due = date(concern.dueAt);
  const last = date(concern.occurredAt);
  const days = concern.daysUntil;
  if (concern.code === "ReviewOverdue") {
    return {
      title:
        days !== null && days < 0
          ? t("The review is {0, plural, one {# day} other {# days}} overdue", -days)
          : t("The review is overdue"),
      detail: due ? t("It was due {0}.", due) : null,
      action: t("Start review"),
    };
  }
  return {
    title: due ? t("A review is due {0}", due) : t("A review is due"),
    detail: last ? t("Last reviewed {0}.", last) : null,
    action: t("Start review"),
  };
}

function clearinghouseWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const last = date(concern.occurredAt);
  const days = concern.daysUntil;
  if (concern.code === "ClearinghouseQueryOverdue") {
    return {
      title:
        days !== null && days < 0
          ? t(
              "The Clearinghouse annual query is {0, plural, one {# day} other {# days}} overdue",
              -days,
            )
          : t("The Clearinghouse annual query is overdue"),
      detail: last ? t("Last run {0}.", last) : t("No query is on file."),
      action: t("Log a query"),
    };
  }
  return {
    title:
      days === null
        ? t("The Clearinghouse annual query is due soon")
        : days <= 0
          ? t("The Clearinghouse annual query is due today")
          : t(
              "The Clearinghouse annual query is due in {0, plural, one {# day} other {# days}}",
              days,
            ),
    detail: last ? t("Last run {0}.", last) : null,
    action: t("Log a query"),
  };
}

function verificationWords(concern: WorkerConcernFacts, t: TranslateFn): WorkerConcernWords {
  const subject = soleSubject(concern);
  const asked = date(concern.occurredAt);
  return {
    title: subject
      ? t("{0} has not answered the safety history request", subject)
      : t(
          "{0, plural, one {# previous employer has} other {# previous employers have}} not answered the safety history request",
          concern.count,
        ),
    detail: subject
      ? asked
        ? t("Asked {0}. The file stays incomplete until it is answered or closed out.", asked)
        : t("The file stays incomplete until it is answered or closed out.")
      : t("The file stays incomplete until they are answered or closed out."),
    action: subject ? t("Open request") : t("Open DQ file"),
  };
}

/**
 * A concern in words: its title, a second line from its dates, and what its button says. One
 * record is named ("Medical card expires in 84 days"); several are counted ("3 credentials
 * expire soon"). Every code has a case, so a code the server adds fails the typecheck here.
 */
export function workerConcernWords(
  concern: WorkerConcernFacts,
  t: TranslateFn,
): WorkerConcernWords {
  const left = date(concern.occurredAt);
  switch (concern.code) {
    case "NotEmployed":
      return {
        title: t("Not currently employed"),
        detail: left
          ? t("Left {0}. Only an employment event brings a worker back.", left)
          : t("Only an employment event brings a worker back."),
        action: t("Open history"),
      };
    case "NotAssignable":
      return {
        title: t("Held from dispatch"),
        detail: t("Dispatch will not assign this worker until the hold is lifted."),
        action: t("Open profile"),
      };
    case "CredentialExpired":
    case "CredentialMissing":
    case "CredentialExpiring":
      return credentialWords(concern, t);
    case "TrainingExpired":
    case "TrainingMissing":
    case "TrainingOverdue":
    case "TrainingDue":
      return trainingWords(concern, t);
    case "SafetyRating":
    case "OutOfService":
    case "ActiveDiscipline":
    case "OpenSafetyEvent":
      return safetyWords(concern, t);
    case "ChecklistOverdue":
    case "ChecklistOpen":
      return checklistWords(concern, t);
    case "ReviewOverdue":
    case "ReviewDue":
      return reviewWords(concern, t);
    case "ClearinghouseQueryOverdue":
    case "ClearinghouseQueryDue":
      return clearinghouseWords(concern, t);
    case "DrugAlcoholProhibited":
      return {
        title: t("Prohibited from safety-sensitive work"),
        detail: t("A drug or alcohol violation stands until return to duty is complete."),
        action: t("Open drug and alcohol"),
      };
    case "EmploymentVerificationOutstanding":
      return verificationWords(concern, t);
  }
}
