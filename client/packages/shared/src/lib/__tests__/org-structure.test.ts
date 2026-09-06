import { describe, expect, it } from "vitest";
import {
  delegationState,
  delegationStateTone,
  headcountShare,
  jobDepartmentLabel,
} from "../org-structure";

const now = 1_800_000_000;
const day = 86_400;

describe("jobDepartmentLabel", () => {
  it("spaces the compound department out", () => {
    expect(jobDepartmentLabel("HumanResources")).toBe("Human Resources");
  });

  it("passes an unknown department through rather than blanking it", () => {
    expect(jobDepartmentLabel("Warehouse")).toBe("Warehouse");
  });
});

describe("delegationState", () => {
  it("is in force inside its window", () => {
    expect(delegationState({ startsAt: now - day, endsAt: now + day }, now)).toBe("active");
  });

  it("has not started before its window", () => {
    expect(delegationState({ startsAt: now + day }, now)).toBe("scheduled");
  });

  it("has finished after its window", () => {
    expect(delegationState({ startsAt: now - 2 * day, endsAt: now - day }, now)).toBe("ended");
  });

  // An open end date runs until somebody revokes it, which is what a manager
  // handing over an area rather than a fortnight actually wants.
  it("runs on when there is no end date", () => {
    expect(delegationState({ startsAt: now - 365 * day }, now)).toBe("active");
  });

  // A delegation called back mid-window is not "in force until Friday", it is
  // over — so revoked has to win over the window.
  it("reads as called back even inside its window", () => {
    expect(
      delegationState({ startsAt: now - day, endsAt: now + day, revokedAt: now - 3600 }, now),
    ).toBe("revoked");
  });

  it("ignores a revocation that has not happened yet", () => {
    expect(
      delegationState({ startsAt: now - day, endsAt: now + day, revokedAt: now + 3600 }, now),
    ).toBe("active");
  });
});

describe("delegationStateTone", () => {
  it("grades the four states", () => {
    expect(delegationStateTone("active")).toBe("active");
    expect(delegationStateTone("revoked")).toBe("inactive");
    expect(delegationStateTone("scheduled")).toBe("warning");
    expect(delegationStateTone("ended")).toBe("secondary");
  });
});

describe("headcountShare", () => {
  it("is the share of the roster", () => {
    expect(headcountShare(25, 100)).toBe(25);
    expect(headcountShare(100, 100)).toBe(100);
  });

  // A terminal with one driver drawn as nothing reads as a terminal with none.
  it("keeps a visible sliver for a small group", () => {
    expect(headcountShare(1, 1000)).toBe(2);
  });

  // An empty roster has no distribution, not an even one.
  it("is nothing when there is nobody to divide by", () => {
    expect(headcountShare(0, 100)).toBe(0);
    expect(headcountShare(5, 0)).toBe(0);
  });
});
