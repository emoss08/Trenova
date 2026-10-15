import type {
  DeskWorker,
  DeskWorkerCredential,
  DeskWorkerHos,
  DeskWorkerStanding,
} from "@/lib/graphql/worker-card";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  concernState,
  dispatchStanding,
  haulsHazmat,
  hosClocks,
  licenseLine,
  otherCredentialChecks,
  qualificationChecks,
  standingConcerns,
  standingTone,
  workerFacts,
  workerName,
} from "../worker-card";

const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(
      /\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/gu,
      (_, index, one: string, other: string) => {
        const count = Number(args[Number(index)]);
        return (count === 1 ? one : other).replace("#", String(count));
      },
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const DAY = 86_400;
const HOUR_MS = 3_600_000;
const NOW = 1_791_720_000;

type Profile = NonNullable<DeskWorker["profile"]>;

function profile(overrides: Partial<Profile> = {}): Profile {
  return {
    licenseNumber: "D1234567",
    cdlClass: "A",
    cdlRestrictions: "",
    endorsement: "O",
    licenseExpiry: NOW + 400 * DAY,
    hazmatExpiry: null,
    medicalCardExpiry: NOW + 300 * DAY,
    twicExpiry: null,
    physicalDueDate: null,
    mvrDueDate: null,
    hireDate: NOW - 900 * DAY,
    isQualified: true,
    disqualificationReason: "",
    licenseState: { abbreviation: "TX" },
    ...overrides,
  } as Profile;
}

function worker(overrides: Partial<DeskWorker> = {}): DeskWorker {
  return {
    id: "wrk_1",
    status: "Active",
    type: "Employee",
    driverType: "OTR",
    firstName: "Ana",
    lastName: "Ruiz",
    phoneNumber: "214-555-0100",
    email: "ana@example.com",
    city: "Dallas",
    externalId: "",
    assignmentBlocked: "",
    canBeAssigned: true,
    availableForDispatch: true,
    createdAt: NOW - 1000 * DAY,
    state: { abbreviation: "TX" },
    fleetCode: { id: "fc_1", code: "DAL" },
    manager: { id: "usr_1", name: "Mia Chen" },
    profile: profile(),
    ...overrides,
  } as DeskWorker;
}

function credential(
  overrides: Partial<DeskWorkerCredential> & { name?: string; profileField?: string } = {},
): DeskWorkerCredential {
  const { name = "Forklift", profileField = "", ...rest } = overrides;
  return {
    health: "Valid",
    daysUntilExpiry: 200,
    required: true,
    credentialType: { id: `ct_${name}`, name, profileField },
    credential: { id: `cr_${name}`, expiresAt: NOW + 200 * DAY },
    ...rest,
  } as DeskWorkerCredential;
}

type HosState = NonNullable<DeskWorkerHos["state"]>;

function hosState(overrides: Partial<HosState> = {}): HosState {
  return {
    dutyStatus: "driving",
    driveRemainingMs: 6 * HOUR_MS,
    driveLimitMs: 11 * HOUR_MS,
    shiftRemainingMs: 8 * HOUR_MS,
    shiftLimitMs: 14 * HOUR_MS,
    cycleRemainingMs: 40 * HOUR_MS,
    cycleLimitMs: 70 * HOUR_MS,
    breakRemainingMs: 3 * HOUR_MS,
    breakLimitMs: 8 * HOUR_MS,
    currentTractorId: null,
    rulesetCycle: "USA 70 hour / 8 day",
    recordedAt: NOW - 120,
    ...overrides,
  } as HosState;
}

function titles(checks: readonly { title: string }[]) {
  return checks.map((check) => check.title);
}

describe("workerName", () => {
  it("names the worker, and says when no name is on file", () => {
    expect(workerName(worker(), t)).toBe("Ana Ruiz");
    expect(workerName(worker({ firstName: " ", lastName: "" }), t)).toBe("Unnamed worker");
  });
});

describe("dispatchStanding", () => {
  it("reads an active, qualified, assignable worker as available", () => {
    expect(dispatchStanding(worker(), t)).toMatchObject({
      label: "Available for dispatch",
      tone: "success",
    });
  });

  it("puts what keeps the worker off a load first: inactive, a block, then the qualification", () => {
    expect(
      dispatchStanding(worker({ status: "Inactive", assignmentBlocked: "Hold" }), t).label,
    ).toBe("Inactive");
    expect(
      dispatchStanding(
        worker({
          assignmentBlocked: "  Drug test pending  ",
          profile: profile({ isQualified: false }),
        }),
        t,
      ),
    ).toMatchObject({ label: "Blocked from dispatch", tone: "warn", reason: "Drug test pending" });
    expect(
      dispatchStanding(
        worker({ profile: profile({ isQualified: false, disqualificationReason: "CDL expired" }) }),
        t,
      ),
    ).toMatchObject({ label: "Not qualified", reason: "CDL expired" });
  });

  it("reads a worker the dispatch board will not take as unavailable", () => {
    expect(dispatchStanding(worker({ availableForDispatch: false }), t).label).toBe("Unavailable");
    expect(dispatchStanding(worker({ canBeAssigned: false }), t).label).toBe("Unavailable");
  });

  it("does not call a worker with no driver profile unqualified", () => {
    expect(dispatchStanding(worker({ profile: null }), t).label).toBe("Available for dispatch");
  });
});

describe("licenseLine", () => {
  it("reads the class, state, endorsement and restrictions", () => {
    expect(licenseLine(profile({ endorsement: "H", cdlRestrictions: " L " }), t)).toBe(
      "CDL A · TX · Hazmat (H) · Restrictions L",
    );
  });

  it("says when the driver holds no endorsement, and leaves out what is not on file", () => {
    expect(licenseLine(profile({ licenseState: null }), t)).toBe("CDL A · No endorsements");
  });
});

describe("haulsHazmat", () => {
  it("counts the hazmat and tanker-plus-hazmat endorsements only", () => {
    expect(haulsHazmat("H")).toBe(true);
    expect(haulsHazmat("X")).toBe(true);
    expect(haulsHazmat("N")).toBe(false);
    expect(haulsHazmat("O")).toBe(false);
  });
});

describe("qualificationChecks", () => {
  it("checks the CDL and medical card, and nothing that is not on file and not required", () => {
    expect(titles(qualificationChecks(profile(), NOW, t))).toEqual(["CDL", "Medical card"]);
  });

  it("puts a disqualified driver first, then what lapsed, then what is close, each in its order", () => {
    const checks = qualificationChecks(
      profile({
        isQualified: false,
        disqualificationReason: "",
        licenseExpiry: NOW + 20 * DAY,
        medicalCardExpiry: NOW - DAY,
        mvrDueDate: NOW + DAY,
        physicalDueDate: NOW + 100 * DAY,
      }),
      NOW,
      t,
    );
    expect(titles(checks)).toEqual([
      "Not qualified to drive",
      "Medical card expired",
      "CDL expires in 20 days",
      "Motor vehicle record due in 1 day",
      "Physical",
    ]);
    expect(checks[0].detail).toBe("The compliance check failed");
  });

  it("reads a CDL renewing inside 60 days as close, though other dates wait for 30", () => {
    expect(
      qualificationChecks(profile({ licenseExpiry: NOW + 45 * DAY }), NOW, t)[0],
    ).toMatchObject({ state: "warn", title: "CDL expires in 45 days" });
    expect(
      qualificationChecks(profile({ medicalCardExpiry: NOW + 45 * DAY }), NOW, t).map(
        (check) => check.state,
      ),
    ).toEqual(["ok", "ok"]);
  });

  it("asks for the hazmat date only from a driver endorsed to haul it", () => {
    expect(titles(qualificationChecks(profile({ endorsement: "X" }), NOW, t))).toEqual([
      "Hazmat endorsement",
      "CDL",
      "Medical card",
    ]);
    expect(
      qualificationChecks(profile({ endorsement: "X" }), NOW, t).find(
        (check) => check.key === "hazmat",
      ),
    ).toMatchObject({ state: "warn", detail: "No hazmat expiry on file" });
    expect(
      titles(qualificationChecks(profile({ endorsement: "N", hazmatExpiry: NOW - DAY }), NOW, t)),
    ).toEqual(["CDL", "Medical card"]);
  });

  it("says a required date that is missing rather than leaving it out", () => {
    const checks = qualificationChecks(
      profile({ licenseExpiry: 0, medicalCardExpiry: null }),
      NOW,
      t,
    );
    expect(checks.map((check) => [check.title, check.state, check.detail])).toEqual([
      ["CDL", "warn", "No expiry on file"],
      ["Medical card", "warn", "No medical card on file"],
    ]);
  });
});

describe("otherCredentialChecks", () => {
  it("leaves out credentials that mirror a profile date, and unrequired ones not held", () => {
    const checks = otherCredentialChecks(
      [
        credential({ name: "CDL", profileField: "licenseExpiry", health: "Expired" }),
        credential({ name: "Forklift" }),
        credential({
          name: "Defensive driving",
          health: "Missing",
          required: false,
          credential: null,
        }),
      ],
      t,
    );
    expect(titles(checks)).toEqual(["Forklift"]);
  });

  it("reads each credential worst first, with how it stands", () => {
    const checks = otherCredentialChecks(
      [
        credential({ name: "Forklift" }),
        credential({ name: "Port pass", health: "ExpiringSoon", daysUntilExpiry: 1 }),
        credential({ name: "Course", health: "Missing", credential: null }),
        credential({
          name: "Badge",
          health: "Expired",
          credential: { id: "cr_b", expiresAt: NOW - 3 * DAY },
        }),
        credential({ name: "Lifetime", credential: { id: "cr_l", expiresAt: null } }),
      ],
      t,
    );
    expect(checks.map((check) => [check.title, check.state])).toEqual([
      ["Course", "bad"],
      ["Badge", "bad"],
      ["Port pass", "warn"],
      ["Forklift", "ok"],
      ["Lifetime", "ok"],
    ]);
    expect(checks[0].detail).toBe("Required, not on file");
    expect(checks[1].detail).toMatch(/^Expired /);
    expect(checks[2].detail).toBe("Expires in 1 day");
    expect(checks[4].detail).toBe("Does not expire");
  });
});

describe("hosClocks", () => {
  it("reads each clock against its own limit, short under an hour and out at none", () => {
    const clocks = hosClocks(
      hosState({ driveRemainingMs: 0, shiftRemainingMs: HOUR_MS - 1, breakRemainingMs: -5 }),
      t,
    );
    expect(clocks.map((clock) => [clock.key, clock.state])).toEqual([
      ["drive", "bad"],
      ["shift", "warn"],
      ["cycle", "ok"],
      ["break", "bad"],
    ]);
    expect(clocks[2].share).toBeCloseTo(40 / 70);
    expect(clocks[3].remainingMs).toBe(0);
  });

  it("draws an empty meter for a clock with no limit, and never past full", () => {
    const clocks = hosClocks(hosState({ cycleLimitMs: 0, driveRemainingMs: 12 * HOUR_MS }), t);
    expect(clocks[0].share).toBe(1);
    expect(clocks[2].share).toBe(0);
  });
});

describe("standing", () => {
  const concerns = [
    {
      severity: "Critical",
      code: "credentials_expired",
      headline: "1 credential has expired",
      detail: "",
    },
    { severity: "Critical", code: "not_assignable", headline: "Cannot be assigned", detail: "" },
    { severity: "Warning", code: "training_due", headline: "Hazmat training due", detail: "" },
    { severity: "Info", code: "checklist_open", headline: "Onboarding open", detail: "" },
  ] as DeskWorkerStanding["concerns"];

  it("keeps the concerns the card shows nowhere else, in the server's order", () => {
    expect(standingConcerns(concerns).map((concern) => concern.code)).toEqual([
      "training_due",
      "checklist_open",
    ]);
  });

  it("reads each severity and standing as the card draws it", () => {
    expect(["Critical", "Warning", "Info"].map((s) => concernState(s as never))).toEqual([
      "bad",
      "warn",
      "info",
    ]);
    expect(["Good", "Watch", "AtRisk", "Blocked"].map((s) => standingTone(s as never))).toEqual([
      "success",
      "progress",
      "warn",
      "warn",
    ]);
  });
});

describe("workerFacts", () => {
  it("lists how to reach the worker and who they report to, saying what is not on file", () => {
    const facts = workerFacts(
      worker({ email: "", city: "", state: null, manager: null, fleetCode: null, profile: null }),
      t,
    );
    expect(Object.fromEntries(facts.map((fact) => [fact.key, fact.value]))).toEqual({
      phone: "214-555-0100",
      email: "None on file",
      home: "None on file",
      hired: "None on file",
      manager: "None",
      fleet: "None",
      license: "None on file",
      external: "None",
    });
  });

  it("says a license number is hidden when the role withholds personal details", () => {
    const licenseOf = (overrides: Partial<Profile>) =>
      workerFacts(worker({ profile: profile(overrides) }), t).find((fact) => fact.key === "license")
        ?.value;

    expect(licenseOf({ licenseNumber: null })).toBe("Hidden");
    expect(licenseOf({ licenseNumber: "  " })).toBe("None on file");
    expect(licenseOf({ licenseNumber: " D1234567 " })).toBe("D1234567");
  });
});
