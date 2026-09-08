import { describe, expect, it } from "vitest";
import {
  DQF_GOOD_FAITH_FOLLOW_UPS,
  dqfItemBlocks,
  dqfItemTone,
  dqfNextSteps,
  dqfSectionLabel,
  dqfSectionProgress,
  dqfSectionTab,
  verificationNextStep,
  verificationSettled,
  verificationTone,
} from "../dqf";

describe("dqfItemBlocks", () => {
  // Expiring soon warns. The document on file is still valid today, and
  // treating it as a gap would make every file incomplete for a month before
  // every renewal.
  it("stops short of expiring-soon", () => {
    expect(dqfItemBlocks("Missing")).toBe(true);
    expect(dqfItemBlocks("Expired")).toBe(true);
    expect(dqfItemBlocks("Outstanding")).toBe(true);
    expect(dqfItemBlocks("ExpiringSoon")).toBe(false);
    expect(dqfItemBlocks("Satisfied")).toBe(false);
    expect(dqfItemBlocks("NotApplicable")).toBe(false);
  });
});

describe("verificationSettled", () => {
  // An employer who never answers still settles the obligation: the rule asks
  // for a good-faith effort and a record of it, not an answer nobody can
  // compel.
  it("counts a silent employer as settled", () => {
    expect(verificationSettled("NoResponse")).toBe(true);
    expect(verificationSettled("Received")).toBe(true);
    expect(verificationSettled("NotApplicable")).toBe(true);
    expect(verificationSettled("Pending")).toBe(false);
    expect(verificationSettled("Requested")).toBe(false);
  });
});

describe("tones", () => {
  it("grades items and investigations", () => {
    expect(dqfItemTone("Satisfied")).toBe("active");
    expect(dqfItemTone("ExpiringSoon")).toBe("warning");
    expect(dqfItemTone("Missing")).toBe("inactive");
    expect(verificationTone("Received")).toBe("active");
    expect(verificationTone("Requested")).toBe("warning");
    expect(verificationTone("NoResponse")).toBe("secondary");
  });
});

describe("dqfSectionLabel", () => {
  it("names each section", () => {
    expect(dqfSectionLabel("SafetyHistory")).toBe("Previous employers");
  });

  // A section the client does not recognise still has to render as something.
  it("falls back to the raw value", () => {
    expect(dqfSectionLabel("SomethingNew")).toBe("SomethingNew");
  });
});

describe("dqfSectionProgress", () => {
  const item = (section: string, status: string) => ({ section, status });

  // The file spine: how much of each section is on file. Not-applicable rows
  // are neither a gap nor a success, so they leave the count alone.
  it("counts each section by what is settled, warning and blocking", () => {
    const progress = dqfSectionProgress([
      item("Credentials", "Satisfied"),
      item("Credentials", "ExpiringSoon"),
      item("Credentials", "Expired"),
      item("Documents", "Missing"),
      item("SafetyHistory", "Outstanding"),
      item("DrugAlcohol", "NotApplicable"),
    ]);

    expect(progress.map((row) => row.section)).toEqual([
      "Credentials",
      "Documents",
      "SafetyHistory",
      "DrugAlcohol",
    ]);
    expect(progress[0]).toMatchObject({ total: 3, satisfied: 1, warning: 1, blocking: 1 });
    expect(progress[1]).toMatchObject({ total: 1, satisfied: 0, warning: 0, blocking: 1 });
    expect(progress[3]).toMatchObject({ total: 0, satisfied: 0, warning: 0, blocking: 0 });
  });
});

describe("dqfSectionTab", () => {
  it("names the panel tab where each section is fixed", () => {
    expect(dqfSectionTab("Credentials")).toBe("credentials");
    expect(dqfSectionTab("Documents")).toBe("documents");
    expect(dqfSectionTab("DrugAlcohol")).toBe("testing");
    expect(dqfSectionTab("SafetyHistory")).toBeNull();
  });
});

describe("verificationNextStep", () => {
  const day = 86400;
  const now = 1_800_000_000;
  const base = {
    status: "Pending",
    requestedAt: null as number | null,
    lastFollowUpAt: null as number | null,
    followUpCount: 0,
    wasDotRegulated: true,
    drugAlcoholResponseReceivedAt: null as number | null,
  };

  it("asks for the request to go out first", () => {
    expect(verificationNextStep(base, now)).toMatchObject({ action: "request" });
  });

  // A request that just went out is not something to chase tomorrow; the
  // interval is what makes a chase a good-faith effort rather than noise.
  it("waits the interval before asking for a chase", () => {
    const sent = { ...base, status: "Requested", requestedAt: now - 3 * day };
    expect(verificationNextStep(sent, now)).toMatchObject({ action: "wait" });
    const stale = { ...base, status: "Requested", requestedAt: now - 20 * day };
    expect(verificationNextStep(stale, now)).toMatchObject({ action: "chase" });
  });

  it("measures the interval from the last chase, not the first request", () => {
    const chased = {
      ...base,
      status: "Requested",
      requestedAt: now - 40 * day,
      lastFollowUpAt: now - 2 * day,
      followUpCount: 1,
    };
    expect(verificationNextStep(chased, now)).toMatchObject({ action: "wait" });
  });

  // 391.23 asks for a good-faith effort and a record of it. After the
  // documented chases and another silent interval, the honest answer is that
  // the employer will not reply, and the file should say so.
  it("offers to close a silent employer once the good-faith chases are on record", () => {
    const silent = {
      ...base,
      status: "Requested",
      requestedAt: now - 60 * day,
      lastFollowUpAt: now - 20 * day,
      followUpCount: DQF_GOOD_FAITH_FOLLOW_UPS,
    };
    expect(verificationNextStep(silent, now)).toMatchObject({ action: "close" });
  });

  it("asks for the drug and alcohol half when a regulated employer answered without it", () => {
    const partial = { ...base, status: "Received" };
    expect(verificationNextStep(partial, now)).toMatchObject({ action: "drugAlcohol" });
    expect(verificationNextStep({ ...partial, wasDotRegulated: false }, now)).toMatchObject({
      action: "done",
    });
    expect(
      verificationNextStep({ ...partial, drugAlcoholResponseReceivedAt: now }, now),
    ).toMatchObject({ action: "done" });
  });

  it("has nothing to do for a settled employer", () => {
    expect(verificationNextStep({ ...base, status: "NoResponse" }, now)).toMatchObject({
      action: "done",
    });
    expect(verificationNextStep({ ...base, status: "NotApplicable" }, now)).toMatchObject({
      action: "done",
    });
  });
});

describe("dqfNextSteps", () => {
  const now = 1_800_000_000;
  const file = {
    complete: false,
    purgeEligible: false,
    items: [
      { section: "Credentials", code: "MEDICAL", name: "Medical card", status: "Expired" },
      { section: "Credentials", code: "CDL", name: "CDL", status: "ExpiringSoon" },
      { section: "Documents", code: "APPLICATION", name: "Application", status: "Missing" },
      { section: "SafetyHistory", code: "SAFETY_HISTORY", name: "History", status: "Outstanding" },
      { section: "DrugAlcohol", code: "CLEARINGHOUSE", name: "Clearinghouse", status: "Satisfied" },
    ],
    verifications: [
      {
        id: "ev_1",
        employerName: "Acme Freight",
        status: "Pending",
        requestedAt: null,
        lastFollowUpAt: null,
        followUpCount: 0,
        wasDotRegulated: true,
        drugAlcoholResponseReceivedAt: null,
      },
    ],
  };

  // The office reads the top of this list and does that. Gaps that stop the
  // file come first, then the chasing, then the things that merely warn.
  it("orders the work: blocking gaps, then employer chasing, then warnings", () => {
    const steps = dqfNextSteps(file, now);

    expect(steps.map((step) => step.id)).toEqual([
      "item:Credentials:MEDICAL",
      "item:Documents:APPLICATION",
      "verification:ev_1",
      "item:Credentials:CDL",
    ]);
    expect(steps[0]).toMatchObject({ tab: "credentials" });
    expect(steps[2]).toMatchObject({ verificationId: "ev_1", action: "request" });
  });

  // No employers recorded is a gap the office must fill by hand, not a chase.
  it("asks for a previous employer when none is recorded", () => {
    const steps = dqfNextSteps({ ...file, verifications: [], items: [file.items[3]] }, now);

    expect(steps).toHaveLength(1);
    expect(steps[0]).toMatchObject({ id: "employers", action: "addEmployer" });
  });

  it("flags a file held past retention as a step of its own", () => {
    const steps = dqfNextSteps({ ...file, items: [], verifications: [], purgeEligible: true }, now);

    expect(steps.map((step) => step.id)).toEqual(["purge"]);
  });

  it("is empty for a complete file", () => {
    expect(
      dqfNextSteps({ complete: true, purgeEligible: false, items: [], verifications: [] }, now),
    ).toEqual([]);
  });
});
