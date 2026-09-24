import { describe, expect, it } from "vitest";
import { buildRailItems, resolveRailView, type RailPermissions } from "../rail-items";
import type { RetrievalRailState } from "../retrieval/retrieval-model";

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
  extensions: true,
  runs: true,
  proposals: true,
  exceptions: true,
  memory: true,
  retrieval: true,
  safety: true,
  quality: true,
  ratings: true,
};

const counts = {
  providersEnabled: 1,
  providersTotal: 2,
  agentsEnabled: 3,
  agentsTotal: 4,
  pendingProposals: 0,
  runsLast24h: 12,
  memoriesActive: 3,
  extensionsOn: 1,
  extensionsTotal: 1,
  qualityRegressions: 0,
  retrieval: null,
};

describe("buildRailItems", () => {
  it("says what each section holds before it is opened", () => {
    const items = buildRailItems(counts, all, t);

    expect(items.map((item) => item.tab)).toEqual([
      "overview",
      "agents",
      "providers",
      "extensions",
      "memory",
      "retrieval",
      "safety",
      "quality",
      "activity",
    ]);
    expect(items[1].status).toBe("3 of 4 on");
    expect(items[2].status).toBe("1 of 2 on");
    expect(items[3].status).toBe("1 of 1 on");
    expect(items[4].status).toBe("3 active");
    expect(items[5].status).toBe("");
    expect(items[6].status).toBe("");
    expect(items[7].status).toBe("");
    expect(items[8].status).toBe("12 runs today");
    expect(items[8].children.map((child) => child.view)).toEqual([
      "runs",
      "proposals",
      "plans",
      "evaluations",
      "exceptions",
    ]);
  });

  // A proposal waiting on a person outranks a run count: it is the one thing
  // on the page somebody has to do.
  it("calls for attention when proposals wait on a person", () => {
    const items = buildRailItems({ ...counts, pendingProposals: 2 }, all, t);

    expect(items[8].status).toBe("2 awaiting decision");
    expect(items[8].attention).toBe(true);
  });

  it("calls for attention when no provider is on", () => {
    const items = buildRailItems({ ...counts, providersEnabled: 0 }, all, t);

    expect(items[2].attention).toBe(true);
    expect(
      buildRailItems({ ...counts, providersTotal: 0, providersEnabled: 0 }, all, t)[2].status,
    ).toBe("None connected");
  });

  it("leaves out what the reader may not open", () => {
    const items = buildRailItems(
      counts,
      {
        ...all,
        providers: false,
        extensions: false,
        exceptions: false,
        memory: false,
        retrieval: false,
        safety: false,
        quality: false,
      },
      t,
    );

    expect(items.map((item) => item.tab)).toEqual(["overview", "agents", "activity"]);
    expect(items[2].children.map((child) => child.view)).toEqual([
      "runs",
      "proposals",
      "plans",
      "evaluations",
    ]);
  });

  // A plan is decided under the same right as the proposals it groups, so
  // the two views come and go together.
  it("lists plans only where proposals may be read", () => {
    const items = buildRailItems(counts, { ...all, proposals: false }, t);

    expect(items[8].children.map((child) => child.view)).toEqual([
      "runs",
      "evaluations",
      "exceptions",
    ]);
  });

  // An empty memory is worth saying: the section exists so someone records
  // the first instruction, and "0 active" reads like a count nobody set.
  it("says when nothing has been recorded for agents yet", () => {
    const items = buildRailItems({ ...counts, memoriesActive: 0 }, all, t);

    expect(items[4].status).toBe("Nothing recorded");
    expect(items[4].attention).toBe(false);
  });

  // What agents may do without a person is read under the right to read
  // agents, so a reader without it never sees the section.
  it("lists safety only where agents may be read", () => {
    const items = buildRailItems(counts, { ...all, safety: false }, t);

    expect(items.map((item) => item.tab)).not.toContain("safety");
    expect(buildRailItems(counts, all, t).find((item) => item.tab === "safety")).toEqual({
      tab: "safety",
      status: "",
      attention: false,
      children: [
        { view: "rules", label: "Tool rules" },
        { view: "agents", label: "By agent" },
      ],
    });
  });

  it("shows nothing under a label until the counts arrive", () => {
    const items = buildRailItems(undefined, all, t);

    expect(items.every((item) => item.status === "")).toBe(true);
  });

  it("says when no extension is on, and leaves the section out without access", () => {
    const none = buildRailItems({ ...counts, extensionsOn: 0 }, all, t);
    expect(none.find((item) => item.tab === "extensions")?.status).toBe("None on");

    const hidden = buildRailItems(counts, { ...all, extensions: false }, t);
    expect(hidden.some((item) => item.tab === "extensions")).toBe(false);
  });

  // A regression is something to look at: the section says how many agents
  // scored worse after they changed, and draws the warning dot.
  it("calls for attention when an agent's quality regressed", () => {
    const quiet = buildRailItems(counts, all, t).find((item) => item.tab === "quality");
    expect(quiet?.status).toBe("");
    expect(quiet?.attention).toBe(false);

    const one = buildRailItems({ ...counts, qualityRegressions: 1 }, all, t).find(
      (item) => item.tab === "quality",
    );
    expect(one?.status).toBe("1 agent regressed");
    expect(one?.attention).toBe(true);

    const two = buildRailItems({ ...counts, qualityRegressions: 2 }, all, t).find(
      (item) => item.tab === "quality",
    );
    expect(two?.status).toBe("2 agents regressed");
  });

  // Each quality table is its own view, so the section lists them; the
  // answers people rated down are read under the right to read feedback.
  it("lists the quality views, worst-rated answers only where feedback may be read", () => {
    const views = (permissions: RailPermissions) =>
      buildRailItems(counts, permissions, t)
        .find((item) => item.tab === "quality")
        ?.children.map((child) => child.view);

    expect(views(all)).toEqual(["agents", "runs", "ratings", "golden", "settings"]);
    expect(views({ ...all, ratings: false })).toEqual(["agents", "runs", "golden", "settings"]);
  });

  it("lists quality only where the golden set may be read", () => {
    const items = buildRailItems(counts, { ...all, quality: false }, t);

    expect(items.map((item) => item.tab)).not.toContain("quality");
    expect(items.at(-1)?.tab).toBe("activity");
  });

  // Retrieval sits beside Memory: both are what agents read back, and the
  // index is built from the memories among other things.
  it("lists retrieval after memory, under the right to read providers", () => {
    const tabs = buildRailItems(counts, all, t).map((item) => item.tab);
    expect(tabs.indexOf("retrieval")).toBe(tabs.indexOf("memory") + 1);

    const hidden = buildRailItems(counts, { ...all, retrieval: false }, t);
    expect(hidden.map((item) => item.tab)).not.toContain("retrieval");
  });

  it("says nothing about retrieval until its status arrives", () => {
    const item = buildRailItems(counts, all, t).find((entry) => entry.tab === "retrieval");

    expect(item).toEqual({ tab: "retrieval", status: "", attention: false, children: [] });
  });

  it("says how far the index has come while search by meaning works", () => {
    const retrieval = (failed: number, waiting: number) =>
      buildRailItems(
        { ...counts, retrieval: { available: true, reason: null, failed, waiting } },
        all,
        t,
      ).find((item) => item.tab === "retrieval");

    expect(retrieval(0, 0)).toMatchObject({ status: "On", attention: false });
    expect(retrieval(0, 1)).toMatchObject({ status: "1 to index", attention: false });
    expect(retrieval(0, 40)).toMatchObject({ status: "40 to index", attention: false });
    // A failure outranks what is waiting: it is the thing someone has to look at.
    expect(retrieval(3, 40)).toMatchObject({ status: "3 failed", attention: true });
  });

  // Keyword-only is where every installation starts, so an organization that
  // has not set retrieval up is not told something is wrong; one whose index
  // stopped working is.
  it("calls for attention only when something set up has stopped", () => {
    const retrieval = (reason: NonNullable<RetrievalRailState["reason"]>) =>
      buildRailItems(
        { ...counts, retrieval: { available: false, reason, failed: 5, waiting: 5 } },
        all,
        t,
      ).find((item) => item.tab === "retrieval");

    expect(retrieval("ExtensionMissing")).toMatchObject({
      status: "Words only",
      attention: false,
    });
    expect(retrieval("NoProvider")).toMatchObject({ status: "Words only", attention: false });
    expect(retrieval("NotIndexed")).toMatchObject({ status: "Starting", attention: false });
    expect(retrieval("Disabled")).toMatchObject({ status: "Paused", attention: false });
    expect(retrieval("SchemaMissing")).toMatchObject({ status: "Words only", attention: true });
    expect(retrieval("TooOld")).toMatchObject({ status: "Words only", attention: true });
    expect(retrieval("BudgetPaused")).toMatchObject({ status: "Budget spent", attention: true });
    expect(retrieval("QueryTimeout")).toMatchObject({
      status: "Provider failing",
      attention: true,
    });
    expect(retrieval("ProviderFailed")).toMatchObject({
      status: "Provider failing",
      attention: true,
    });
  });
});

describe("resolveRailView", () => {
  const items = buildRailItems(counts, { ...all, ratings: false }, t);
  const item = (tab: string) => items.find((entry) => entry.tab === tab);

  it("opens the view asked for when the row offers it", () => {
    expect(resolveRailView(item("safety"), "agents")).toBe("agents");
    expect(resolveRailView(item("quality"), "runs")).toBe("runs");
  });

  // A link can name a view the reader may not open; the row's first view is
  // shown instead of an empty section.
  it("falls back to the row's first view", () => {
    expect(resolveRailView(item("quality"), "ratings")).toBe("agents");
    expect(resolveRailView(item("safety"), null)).toBe("rules");
  });

  it("has no view for a row without views", () => {
    expect(resolveRailView(item("memory"), "runs")).toBeNull();
    expect(resolveRailView(undefined, "runs")).toBeNull();
  });
});
