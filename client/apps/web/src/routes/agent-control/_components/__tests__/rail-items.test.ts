import { describe, expect, it } from "vitest";
import { buildRailItems, type RailPermissions } from "../rail-items";

const t = (text: string, ...args: (string | number)[]) =>
  text
    .replace(/\{(\d+), plural, one \{([^}]*)\} other \{([^}]*)\}\}/g, (_m, i, one, other) =>
      String(args[Number(i)]) === "1"
        ? one.replace("#", String(args[Number(i)]))
        : other.replace("#", String(args[Number(i)])),
    )
    .replace(/\{(\d+)\}/g, (_m, i) => String(args[Number(i)]));

const all: RailPermissions = {
  agents: true,
  providers: true,
  runs: true,
  proposals: true,
  exceptions: true,
};

const counts = {
  providersEnabled: 1,
  providersTotal: 2,
  agentsEnabled: 3,
  agentsTotal: 4,
  pendingProposals: 0,
  runsLast24h: 12,
};

describe("buildRailItems", () => {
  it("says what each section holds before it is opened", () => {
    const items = buildRailItems(counts, all, t);

    expect(items.map((item) => item.tab)).toEqual(["overview", "agents", "providers", "activity"]);
    expect(items[1].status).toBe("3 of 4 on");
    expect(items[2].status).toBe("1 of 2 on");
    expect(items[3].status).toBe("12 runs today");
    expect(items[3].children.map((child) => child.view)).toEqual([
      "runs",
      "proposals",
      "exceptions",
    ]);
  });

  // A proposal waiting on a person outranks a run count: it is the one thing
  // on the page somebody has to do.
  it("calls for attention when proposals wait on a person", () => {
    const items = buildRailItems({ ...counts, pendingProposals: 2 }, all, t);

    expect(items[3].status).toBe("2 awaiting decision");
    expect(items[3].attention).toBe(true);
  });

  it("calls for attention when no provider is on", () => {
    const items = buildRailItems({ ...counts, providersEnabled: 0 }, all, t);

    expect(items[2].attention).toBe(true);
    expect(
      buildRailItems({ ...counts, providersTotal: 0, providersEnabled: 0 }, all, t)[2].status,
    ).toBe("None connected");
  });

  it("leaves out what the reader may not open", () => {
    const items = buildRailItems(counts, { ...all, providers: false, exceptions: false }, t);

    expect(items.map((item) => item.tab)).toEqual(["overview", "agents", "activity"]);
    expect(items[2].children.map((child) => child.view)).toEqual(["runs", "proposals"]);
  });

  it("shows nothing under a label until the counts arrive", () => {
    const items = buildRailItems(undefined, all, t);

    expect(items.every((item) => item.status === "")).toBe(true);
  });
});
