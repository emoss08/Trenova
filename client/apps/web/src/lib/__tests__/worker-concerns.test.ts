import type { WorkerConcernCode } from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describe, expect, it } from "vitest";
import {
  workerConcernTarget,
  workerConcernWords,
  workerDestinationTarget,
  type WorkerConcernFacts,
  type WorkerDestination,
} from "../worker-concerns";

const DAY = 86_400;
const AS_OF = Date.UTC(2026, 10, 22, 12) / 1000;
const t = translate;

function facts(
  code: WorkerConcernCode,
  overrides: Partial<WorkerConcernFacts> = {},
): WorkerConcernFacts {
  return {
    severity: "Warning",
    code,
    subject: null,
    count: 1,
    occurredAt: null,
    dueAt: null,
    daysUntil: null,
    stopsAssignment: false,
    ...overrides,
  };
}

const date = (at: number) => formatUnixDateMedium(at);

describe("workerConcernWords: one record is named", () => {
  const cases: [
    string,
    WorkerConcernFacts,
    { title: string; detail: string | null; action: string },
  ][] = [
    [
      "NotEmployed",
      facts("NotEmployed", { occurredAt: AS_OF - 30 * DAY }),
      {
        title: "Not currently employed",
        detail: `Left ${date(AS_OF - 30 * DAY)}. Only an employment event brings a worker back.`,
        action: "Open history",
      },
    ],
    [
      "NotAssignable",
      facts("NotAssignable", { severity: "Critical", stopsAssignment: true }),
      {
        title: "Held from dispatch",
        detail: "Dispatch will not assign this worker until the hold is lifted.",
        action: "Open profile",
      },
    ],
    [
      "CredentialExpired",
      facts("CredentialExpired", {
        subject: "Medical card",
        dueAt: AS_OF - 3 * DAY,
        daysUntil: -3,
        stopsAssignment: true,
      }),
      {
        title: "Medical card expired 3 days ago",
        detail: `Lapsed ${date(AS_OF - 3 * DAY)}. Dispatch will not assign the driver until it is renewed.`,
        action: "Renew",
      },
    ],
    [
      "CredentialMissing",
      facts("CredentialMissing", { subject: "TWIC card" }),
      {
        title: "TWIC card is not on file",
        detail: "It is required for the file.",
        action: "Add credential",
      },
    ],
    [
      "CredentialExpiring",
      facts("CredentialExpiring", {
        subject: "Medical card",
        dueAt: AS_OF + 84 * DAY,
        daysUntil: 84,
        stopsAssignment: true,
      }),
      {
        title: "Medical card expires in 84 days",
        detail: `${date(AS_OF + 84 * DAY)}. Dispatch stops assigning the driver when it lapses.`,
        action: "Renew",
      },
    ],
    [
      "TrainingExpired",
      facts("TrainingExpired", { subject: "Hazmat awareness", dueAt: AS_OF - DAY, daysUntil: -1 }),
      {
        title: "Hazmat awareness expired 1 day ago",
        detail: `Lapsed ${date(AS_OF - DAY)}.`,
        action: "Open training",
      },
    ],
    [
      "TrainingMissing",
      facts("TrainingMissing", { subject: "Defensive driving" }),
      {
        title: "Defensive driving has never been assigned",
        detail: "Assign it so the driver can complete it.",
        action: "Assign training",
      },
    ],
    [
      "TrainingOverdue",
      facts("TrainingOverdue", {
        subject: "Hazmat awareness",
        occurredAt: AS_OF - 40 * DAY,
        dueAt: AS_OF - 10 * DAY,
        daysUntil: -10,
      }),
      {
        title: "Hazmat awareness is 10 days overdue",
        detail: `Assigned ${date(AS_OF - 40 * DAY)}, due ${date(AS_OF - 10 * DAY)}.`,
        action: "Open training",
      },
    ],
    [
      "TrainingDue",
      facts("TrainingDue", {
        subject: "Hazmat awareness refresher",
        occurredAt: AS_OF - 19 * DAY,
        dueAt: AS_OF + 26 * DAY,
        daysUntil: 26,
      }),
      {
        title: "Hazmat awareness refresher is due in 26 days",
        detail: `Assigned ${date(AS_OF - 19 * DAY)}.`,
        action: "Open training",
      },
    ],
    [
      "SafetyRating",
      facts("SafetyRating", { count: 1 }),
      {
        title: "The safety record is on watch",
        detail: "1 active point on the record.",
        action: "Open safety",
      },
    ],
    [
      "OutOfService",
      facts("OutOfService", { severity: "Critical", occurredAt: AS_OF - 5 * DAY }),
      {
        title: "1 out-of-service order on the record",
        detail: `The latest on ${date(AS_OF - 5 * DAY)}.`,
        action: "Open safety",
      },
    ],
    [
      "ActiveDiscipline",
      facts("ActiveDiscipline", { occurredAt: AS_OF - 5 * DAY }),
      {
        title: "1 disciplinary action is still standing",
        detail: `The latest was issued ${date(AS_OF - 5 * DAY)}.`,
        action: "Open safety",
      },
    ],
    [
      "OpenSafetyEvent",
      facts("OpenSafetyEvent", {
        severity: "Info",
        subject: "A roadside inspection",
        occurredAt: AS_OF - 41 * DAY,
      }),
      {
        title: "A roadside inspection is under review",
        detail: `It happened ${date(AS_OF - 41 * DAY)}.`,
        action: "Open event",
      },
    ],
    [
      "ChecklistOverdue",
      facts("ChecklistOverdue", {
        subject: "Annual file review 2026",
        count: 1,
        occurredAt: AS_OF - 20 * DAY,
      }),
      {
        title: "Annual file review 2026 has 1 overdue item",
        detail: `Started ${date(AS_OF - 20 * DAY)}.`,
        action: "Open checklist",
      },
    ],
    [
      "ChecklistOpen",
      facts("ChecklistOpen", {
        severity: "Info",
        subject: "Onboarding",
        occurredAt: AS_OF - 2 * DAY,
        dueAt: AS_OF + 12 * DAY,
      }),
      {
        title: "Onboarding is open",
        detail: `Started ${date(AS_OF - 2 * DAY)}, due ${date(AS_OF + 12 * DAY)}.`,
        action: "Open checklist",
      },
    ],
    [
      "ReviewOverdue",
      facts("ReviewOverdue", { dueAt: AS_OF - 4 * DAY, daysUntil: -4 }),
      {
        title: "The review is 4 days overdue",
        detail: `It was due ${date(AS_OF - 4 * DAY)}.`,
        action: "Start review",
      },
    ],
    [
      "ReviewDue",
      facts("ReviewDue", {
        severity: "Info",
        dueAt: AS_OF + 54 * DAY,
        occurredAt: AS_OF - 314 * DAY,
      }),
      {
        title: `A review is due ${date(AS_OF + 54 * DAY)}`,
        detail: `Last reviewed ${date(AS_OF - 314 * DAY)}.`,
        action: "Start review",
      },
    ],
    [
      "ClearinghouseQueryOverdue",
      facts("ClearinghouseQueryOverdue", { severity: "Critical", daysUntil: -2 }),
      {
        title: "The Clearinghouse annual query is 2 days overdue",
        detail: "No query is on file.",
        action: "Log a query",
      },
    ],
    [
      "ClearinghouseQueryDue",
      facts("ClearinghouseQueryDue", { daysUntil: 17, occurredAt: AS_OF - 348 * DAY }),
      {
        title: "The Clearinghouse annual query is due in 17 days",
        detail: `Last run ${date(AS_OF - 348 * DAY)}.`,
        action: "Log a query",
      },
    ],
    [
      "DrugAlcoholProhibited",
      facts("DrugAlcoholProhibited", { severity: "Critical", stopsAssignment: true }),
      {
        title: "Prohibited from safety-sensitive work",
        detail: "A drug or alcohol violation stands until return to duty is complete.",
        action: "Open drug and alcohol",
      },
    ],
    [
      "EmploymentVerificationOutstanding",
      facts("EmploymentVerificationOutstanding", {
        severity: "Critical",
        subject: "Ridgeline Freight",
        occurredAt: AS_OF - 800 * DAY,
      }),
      {
        title: "Ridgeline Freight has not answered the safety history request",
        detail: `Asked ${date(AS_OF - 800 * DAY)}. The file stays incomplete until it is answered or closed out.`,
        action: "Open request",
      },
    ],
  ];

  it.each(cases)("words %s", (_, concern, expected) => {
    expect(workerConcernWords(concern, t)).toEqual(expected);
  });

  it("covers every concern code", () => {
    const codes: Record<WorkerConcernCode, true> = {
      NotEmployed: true,
      NotAssignable: true,
      CredentialExpired: true,
      CredentialMissing: true,
      CredentialExpiring: true,
      TrainingExpired: true,
      TrainingMissing: true,
      TrainingOverdue: true,
      TrainingDue: true,
      SafetyRating: true,
      OutOfService: true,
      ActiveDiscipline: true,
      OpenSafetyEvent: true,
      ChecklistOverdue: true,
      ChecklistOpen: true,
      ReviewOverdue: true,
      ReviewDue: true,
      ClearinghouseQueryOverdue: true,
      ClearinghouseQueryDue: true,
      DrugAlcoholProhibited: true,
      EmploymentVerificationOutstanding: true,
    };
    expect(cases.map(([code]) => code).sort()).toEqual(Object.keys(codes).sort());
  });
});

describe("workerConcernWords: several records are counted", () => {
  const cases: [WorkerConcernFacts, string][] = [
    [facts("CredentialExpired", { count: 2 }), "2 credentials have expired"],
    [facts("CredentialMissing", { count: 2 }), "2 required credentials are not on file"],
    [
      facts("CredentialExpiring", { count: 3, dueAt: AS_OF + 12 * DAY }),
      "3 credentials expire soon",
    ],
    [facts("TrainingExpired", { count: 2 }), "2 courses have expired"],
    [facts("TrainingMissing", { count: 4 }), "4 required courses have never been assigned"],
    [facts("TrainingOverdue", { count: 2 }), "2 courses are overdue"],
    [facts("TrainingDue", { count: 3 }), "3 courses are due soon"],
    [facts("OutOfService", { count: 2 }), "2 out-of-service orders on the record"],
    [facts("ActiveDiscipline", { count: 3 }), "3 disciplinary actions are still standing"],
    [facts("OpenSafetyEvent", { count: 2 }), "2 safety events are under review"],
    [
      facts("ChecklistOverdue", { subject: "Onboarding", count: 3 }),
      "Onboarding has 3 overdue items",
    ],
    [facts("ChecklistOpen", { count: 2 }), "2 checklists are open"],
    [
      facts("EmploymentVerificationOutstanding", { count: 2 }),
      "2 previous employers have not answered the safety history request",
    ],
  ];

  it.each(cases)("counts %#", (concern, title) => {
    expect(workerConcernWords(concern, t).title).toBe(title);
  });

  it("does not name a subject when the concern covers several records", () => {
    const words = workerConcernWords(
      facts("CredentialExpiring", { subject: "Medical card", count: 3, dueAt: AS_OF + 12 * DAY }),
      t,
    );
    expect(words).toEqual({
      title: "3 credentials expire soon",
      detail: `The first expires ${date(AS_OF + 12 * DAY)}.`,
      action: "Open credentials",
    });
  });

  it("says the points when the safety rating is at risk", () => {
    expect(
      workerConcernWords(facts("SafetyRating", { severity: "Critical", count: 12 }), t),
    ).toMatchObject({
      title: "The safety record is at risk",
      detail: "12 active points on the record.",
    });
  });

  it("never shows a record id the server put in a subject", () => {
    const words = workerConcernWords(
      facts("CredentialMissing", { subject: "cus_01JW3K18A8TPC7T7SPX7745V1H" }),
      t,
    );
    expect(words.title).not.toContain("cus_01JW3K18A8TPC7T7SPX7745V1H");
  });
});

describe("workerDestinationTarget and workerConcernTarget", () => {
  const credential: WorkerDestination = {
    tab: "Qualification",
    view: "Credentials",
    record: "Credential",
    recordId: "cred_1",
  };

  it("turns a destination into the page's tab, view and record", () => {
    expect(workerDestinationTarget(credential)).toEqual({
      tab: "qualification",
      view: "credentials",
      credential: "cred_1",
    });
    expect(
      workerDestinationTarget({ tab: "Safety", view: null, record: "Event", recordId: "se_1" }),
    ).toEqual({
      tab: "safety",
      safetyEvent: "se_1",
    });
    expect(
      workerDestinationTarget({
        tab: "Employment",
        view: "History",
        record: "Event",
        recordId: "ev_1",
      }),
    ).toEqual({
      tab: "employment",
      view: "history",
      event: "ev_1",
    });
    expect(
      workerDestinationTarget({
        tab: "Time",
        view: "TimeOff",
        record: "Request",
        recordId: "pto_1",
      }),
    ).toEqual({
      tab: "time",
      view: "time-off",
      timeOff: "pto_1",
    });
  });

  it("drops a view that does not belong to the tab", () => {
    expect(
      workerDestinationTarget({
        tab: "Profile",
        view: "Credentials",
        record: null,
        recordId: null,
      }),
    ).toEqual({
      tab: "profile",
    });
  });

  it("opens the form that fixes a concern, with the record it names", () => {
    expect(workerConcernTarget({ code: "CredentialExpiring", destination: credential })).toEqual({
      tab: "qualification",
      view: "credentials",
      form: "renew-credential",
      credential: "cred_1",
    });
    expect(
      workerConcernTarget({
        code: "ClearinghouseQueryDue",
        destination: { tab: "Qualification", view: "DrugAndAlcohol", record: null, recordId: null },
      }),
    ).toEqual({ tab: "qualification", view: "drug-and-alcohol", form: "log-query" });
    expect(
      workerConcernTarget({
        code: "OpenSafetyEvent",
        destination: { tab: "Safety", view: null, record: "Event", recordId: "se_9" },
      }),
    ).toEqual({ tab: "safety", safetyEvent: "se_9" });
  });
});
