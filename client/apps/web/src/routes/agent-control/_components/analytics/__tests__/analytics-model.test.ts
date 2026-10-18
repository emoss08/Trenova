import type { AgentAnalytics, AIAnalyticsSeries } from "@/lib/graphql/ai-analytics";
import { describe, expect, it } from "vitest";
import {
  analyticsCsv,
  board,
  boardSentence,
  completionIntensity,
  display,
  failureIntensity,
  failureShare,
  formatLatency,
  formatRunTime,
  linePath,
  METRICS,
  moveAhead,
  placeLabels,
  runTimeOrder,
  runTimePosition,
  sameModelPair,
  seriesColors,
  stackLabels,
  bucketPosition,
  emptyScopeMessage,
  hiddenAmong,
  knownTask,
  trendLines,
  whoseRows,
} from "../analytics-model";

const t = (text: string | null | undefined, ...args: unknown[]) =>
  (text ?? "").replace(/\{(\d+)\}/g, (_match, index) => String(args[Number(index)]));

const rt = (message: string, _tags: unknown, ...args: unknown[]) =>
  t(message.replace(/<b>(.*?)<\/b>/g, "[$1]"), ...args);

function series(overrides: Partial<AIAnalyticsSeries> & { key: string }): AIAnalyticsSeries {
  return {
    label: overrides.key,
    shortLabel: overrides.key,
    modelKey: null,
    providerId: null,
    few: false,
    colorSlot: 0,
    models: [],
    providers: [],
    runs: 100,
    completed: 90,
    handedOff: 4,
    stopped: 2,
    failed: 4,
    completedRate: 0.9,
    medianRunMs: 4000,
    p95RunMs: 20000,
    modelCalls: 300,
    toolCalls: 400,
    toolCallsFailed: 8,
    toolFailuresModel: 6,
    toolFailureRate: 0.02,
    modelMistakeRate: 0.75,
    toolFailuresByCause: [],
    costUsd: "4.5",
    pricedRuns: 100,
    costPerCompletedRun: "0.05",
    firstTokenMedianMs: 420,
    lastRanAt: 1_790_000_000,
    quiet: false,
    runTimes: [],
    decided: 0,
    accepted: 0,
    modified: 0,
    rejected: 0,
    approvalRate: null,
    rated: 0,
    helpful: 0,
    helpfulRate: null,
    ...overrides,
  };
}

describe("seriesColors", () => {
  it("draws each series in the palette slot the server holds for it", () => {
    const colors = seriesColors([
      { key: "llama-3.3-70b", colorSlot: 0 },
      { key: "gpt-5.1", colorSlot: 11 },
    ]);

    expect(colors.get("llama-3.3-70b")).toBe("var(--series-1)");
    expect(colors.get("gpt-5.1")).toBe("var(--series-12)");
  });
});

describe("formatting", () => {
  it("reads run times the way a person counts them", () => {
    expect(formatRunTime(4100)).toBe("4.1s");
    expect(formatRunTime(13_400)).toBe("13s");
    expect(formatRunTime(69_000)).toBe("1m 09s");
    expect(formatRunTime(null)).toBe("—");
  });

  it("reads a first token in milliseconds under a second", () => {
    expect(formatLatency(687)).toBe("687ms");
    expect(formatLatency(1420)).toBe("1.42s");
  });

  it("gives a failure share a decimal under ten percent and says when it rounds to nothing", () => {
    expect(failureShare(0)).toBe("0");
    expect(failureShare(0.0004)).toBe("<0.1%");
    expect(failureShare(0.027)).toBe("2.7%");
    expect(failureShare(0.12)).toBe("12%");
  });

  it("calls a cost of nothing local rather than free", () => {
    expect(display(t, METRICS.cost, 0)).toBe("Local");
    expect(display(t, METRICS.cost, 0.041)).toBe("$0.041");
  });

  it("shades failure cells to full at twelve percent and task cells between 60 and 98", () => {
    expect(failureIntensity(0)).toBe(0);
    expect(failureIntensity(0.06)).toBeCloseTo(0.5);
    expect(failureIntensity(0.4)).toBe(1);
    expect(completionIntensity(0.6)).toBe(0);
    expect(completionIntensity(0.98)).toBeCloseTo(1);
    expect(completionIntensity(null)).toBe(0);
  });
});

describe("board", () => {
  const ranked = [
    series({ key: "a", completedRate: 0.8 }),
    series({ key: "b", completedRate: 0.95 }),
    series({ key: "few", completedRate: 1, few: true, runs: 12 }),
    series({ key: "none", completedRate: null }),
  ];

  it("ranks best first, leaves series without the figure out, and puts the few last unranked", () => {
    const entries = board(ranked, METRICS.completed);

    expect(entries.map((entry) => [entry.series.key, entry.best])).toEqual([
      ["b", true],
      ["a", false],
      ["few", false],
    ]);
  });

  it("ranks lower first for a figure where lower is better", () => {
    const entries = board(
      [series({ key: "slow", medianRunMs: 13_000 }), series({ key: "fast", medianRunMs: 4_100 })],
      METRICS.runTime,
    );

    expect(entries.map((entry) => entry.series.key)).toEqual(["fast", "slow"]);
  });

  it("says the best and the worst ranked series in one sentence", () => {
    expect(boardSentence(t, rt, METRICS.completed, board(ranked, METRICS.completed))).toBe(
      "[b] finishes 95 of every 100 runs; [a] finishes 80.",
    );
  });

  it("counts failed tool calls as one in so many", () => {
    const entries = board(
      [
        series({ key: "careful", toolFailureRate: 0.0167 }),
        series({ key: "sloppy", toolFailureRate: 0.09 }),
      ],
      METRICS.toolFailures,
    );

    expect(boardSentence(t, rt, METRICS.toolFailures, entries)).toBe(
      "[careful] gets 1 in 60 tool calls wrong; [sloppy] gets 1 in 11 wrong.",
    );
  });

  it("says a series with no failed tool calls gets none wrong in words", () => {
    const entries = board(
      [
        series({ key: "clean", toolFailureRate: 0 }),
        series({ key: "sloppy", toolFailureRate: 0.09 }),
      ],
      METRICS.toolFailures,
    );

    expect(boardSentence(t, rt, METRICS.toolFailures, entries)).toBe(
      "[clean] gets no tool calls wrong; [sloppy] gets 1 in 11 wrong.",
    );
  });

  it("says a local provider's run costs nothing", () => {
    const entries = board(
      [
        series({ key: "lmstudio", pricedRuns: 0, costPerCompletedRun: null }),
        series({ key: "azure", costPerCompletedRun: "0.041" }),
      ],
      METRICS.cost,
    );

    expect(boardSentence(t, rt, METRICS.cost, entries)).toBe(
      "[lmstudio] runs on your own hardware, so a run costs nothing; [azure] costs $0.041.",
    );
  });

  it("does not call a priced series whose runs cost nothing local", () => {
    const entries = board(
      [
        series({ key: "free-tier", pricedRuns: 12, costPerCompletedRun: "0" }),
        series({ key: "azure", costPerCompletedRun: "0.041" }),
      ],
      METRICS.cost,
    );

    expect(boardSentence(t, rt, METRICS.cost, entries)).toBe(
      "[free-tier] costs $0.000 per finished run; [azure] costs $0.041.",
    );
  });

  it("names only the best series when it is the one ranked", () => {
    const entries = board(
      [series({ key: "solo", approvalRate: 0.72 }), series({ key: "few", few: true })],
      METRICS.approval,
    );

    expect(boardSentence(t, rt, METRICS.approval, entries)).toBe(
      "People accept 72% of [solo]'s proposals as written.",
    );
  });

  it("has nothing to say when no series has enough runs to rank", () => {
    const entries = board([series({ key: "few", few: true })], METRICS.completed);

    expect(boardSentence(t, rt, METRICS.completed, entries)).toBeNull();
  });
});

describe("placeLabels", () => {
  it("puts a label left of its dot when the right would leave the plot", () => {
    const [label] = placeLabels([{ key: "edge", cx: 590, cy: 100, r: 6, width: 80 }], 600);

    expect(label?.left).toBe(true);
    expect(label?.y).toBe(100);
  });

  it("moves a label off a neighbour's label instead of drawing one over the other", () => {
    const labels = placeLabels(
      [
        { key: "upper", cx: 100, cy: 100, r: 6, width: 80 },
        { key: "lower", cx: 102, cy: 104, r: 6, width: 80 },
      ],
      600,
    );
    const upper = labels.find((label) => label.key === "upper");
    const lower = labels.find((label) => label.key === "lower");

    expect(upper?.left).toBe(false);
    expect(lower?.left !== upper?.left || Math.abs((lower?.y ?? 0) - (upper?.y ?? 0)) >= 16).toBe(
      true,
    );
  });
});

describe("stackLabels", () => {
  it("drops a label a row only when it would touch the one before it", () => {
    expect(
      stackLabels(
        [
          { start: 0, end: 100 },
          { start: 60, end: 160 },
          { start: 200, end: 260 },
          { start: 210, end: 300 },
        ],
        12,
      ),
    ).toEqual([0, 1, 0, 1]);
  });
});

describe("run time", () => {
  it("puts one second at the start of the axis and three minutes at its end", () => {
    expect(runTimePosition(1000)).toBe(0);
    expect(runTimePosition(500)).toBe(0);
    expect(runTimePosition(180_000)).toBeCloseTo(1);
    expect(runTimePosition(600_000)).toBe(1);
  });

  it("orders by the outcome's median and sends series without such runs last", () => {
    const rows = [
      series({
        key: "none",
        runTimes: [{ outcome: "Completed", runs: 9, p25Ms: 1, medianMs: 2, p75Ms: 3, p95Ms: 4 }],
      }),
      series({
        key: "slow",
        runTimes: [{ outcome: "Failed", runs: 9, p25Ms: 1, medianMs: 60_000, p75Ms: 3, p95Ms: 4 }],
      }),
      series({
        key: "quick",
        runTimes: [{ outcome: "Failed", runs: 9, p25Ms: 1, medianMs: 9_000, p75Ms: 3, p95Ms: 4 }],
      }),
    ];

    expect(runTimeOrder(rows, "Failed")).toEqual(["quick", "slow", "none"]);
  });
});

describe("trend", () => {
  it("puts each point on the server's bucket so a quiet one is a gap", () => {
    const buckets = [986_400, 990_000, 993_600, 997_200];

    const lines = trendLines(
      [
        {
          seriesKey: "a",
          bucketStart: 986_400,
          runs: 3,
          completedRate: 0.9,
          toolFailureRate: null,
          medianRunMs: null,
        },
        {
          seriesKey: "a",
          bucketStart: 997_200,
          runs: 2,
          completedRate: 0.5,
          toolFailureRate: null,
          medianRunMs: null,
        },
        {
          seriesKey: "z",
          bucketStart: 990_000,
          runs: 2,
          completedRate: 1,
          toolFailureRate: null,
          medianRunMs: null,
        },
      ],
      "completed",
      ["a"],
      buckets,
    );

    expect(lines.get("a")).toEqual([0.9, null, null, 0.5]);
    expect(lines.has("z")).toBe(false);
  });

  it("keeps a point on its local bucket when the viewer is not in UTC", () => {
    const newYork = [1_791_979_200, 1_792_000_800, 1_792_022_400];

    const lines = trendLines(
      [
        {
          seriesKey: "a",
          bucketStart: 1_792_000_800,
          runs: 4,
          completedRate: 0.75,
          toolFailureRate: null,
          medianRunMs: null,
        },
      ],
      "completed",
      ["a"],
      newYork,
    );

    expect(lines.get("a")).toEqual([null, 0.75, null]);
  });

  it("places an instant inside an uneven bucket by how far through it is", () => {
    const days = [0, 90_000, 176_400];

    expect(bucketPosition(days, 45_000, 86_400)).toBe(0.5);
    expect(bucketPosition(days, 176_400 + 43_200, 86_400)).toBe(2.5);
    expect(bucketPosition(days, -10, 86_400)).toBe(0);
    expect(bucketPosition([], 10, 86_400)).toBe(0);
  });

  it("lifts the pen over a gap", () => {
    expect(
      linePath(
        [1, 2, null, 4],
        (index) => index * 10,
        (value) => value,
      ),
    ).toBe("M0.0,1.0L10.0,2.0M30.0,4.0");
  });
});

describe("whoseRows", () => {
  it("splits failed calls into the model's and the tool's as shares of all calls", () => {
    const [row] = whoseRows([
      series({
        key: "qwen",
        toolCalls: 200,
        toolFailuresByCause: [
          { cause: "WrongArguments", blame: "Model", calls: 10 },
          { cause: "ToolTimedOut", blame: "Tool", calls: 4 },
        ],
      }),
    ]);

    expect(row?.modelTotal).toBeCloseTo(0.05);
    expect(row?.toolTotal).toBeCloseTo(0.02);
    expect(row?.model.find((part) => part.cause === "WrongArguments")?.share).toBeCloseTo(0.05);
  });
});

describe("sameModelPair", () => {
  const groq = series({
    key: "llama|groq",
    label: "Groq",
    modelKey: "llama",
    providerId: "groq",
    completedRate: 0.86,
    medianRunMs: 3400,
    toolFailureRate: 0.041,
    firstTokenMedianMs: 179,
    approvalRate: 0.58,
  });
  const vllm = series({
    key: "llama|vllm",
    label: "Workstation vLLM",
    modelKey: "llama",
    providerId: "vllm",
    completedRate: 0.73,
    medianRunMs: 12_000,
    toolFailureRate: 0.046,
    firstTokenMedianMs: 907,
    approvalRate: 0.59,
  });

  const picked = { modelKey: "llama", betterKey: "llama|groq", worseKey: "llama|vllm" };
  const shown = () => false;

  it("puts the server's better deployment on the left and counts its wins", () => {
    const pair = sameModelPair(
      [vllm, groq, series({ key: "gpt|azure", modelKey: "gpt", providerId: "azure" })],
      picked,
      shown,
    );

    expect(pair?.left.label).toBe("Groq");
    expect(pair?.right.label).toBe("Workstation vLLM");
    expect(pair?.rows.map((row) => [row.metric.key, row.leftWins])).toEqual([
      ["completed", true],
      ["runTime", true],
      ["toolFailures", true],
      ["firstToken", true],
      ["approval", false],
    ]);
    expect(pair?.wins).toBe(4);
  });

  it("shows no pair when the server named none, one side is missing, or one is hidden", () => {
    expect(sameModelPair([groq, vllm], null, shown)).toBeNull();
    expect(sameModelPair([groq], picked, shown)).toBeNull();
    expect(sameModelPair([groq, vllm], picked, (item) => item.key === "llama|vllm")).toBeNull();
  });
});

describe("moveAhead", () => {
  it("moves a provider to just ahead of another and leaves the rest in order", () => {
    expect(moveAhead(["lmstudio", "vllm", "openai", "groq"], "groq", "vllm")).toEqual([
      "lmstudio",
      "groq",
      "vllm",
      "openai",
    ]);
  });

  it("changes nothing when the provider already goes first or is not in the routing", () => {
    expect(moveAhead(["groq", "vllm"], "groq", "vllm")).toBeNull();
    expect(moveAhead(["vllm"], "groq", "vllm")).toBeNull();
  });
});

describe("analyticsCsv", () => {
  it("writes one row per group and one per deployment, escaping what a spreadsheet would run", () => {
    const analytics = {
      series: [
        series({ key: "gpt", label: "GPT-5.1, preview" }),
        series({ key: "evil", label: "=HYPERLINK(1)" }),
      ],
      deployments: [
        series({ key: "gpt|azure", label: "Azure", models: ["GPT-5.1"], completedRate: 0.9312 }),
      ],
    } as unknown as AgentAnalytics;

    const [header, group, evil, deployment] = analyticsCsv(analytics, t).split("\n");

    expect(header?.startsWith("Group,Providers,Runs,Completed %")).toBe(true);
    expect(group?.startsWith('"GPT-5.1, preview",,100,90.00,')).toBe(true);
    expect(evil?.startsWith("'=HYPERLINK(1),,100,")).toBe(true);
    expect(deployment?.startsWith("GPT-5.1 · Azure,,100,93.12,")).toBe(true);
  });
});

describe("scope", () => {
  it("drops hidden keys for series that are gone, and shows everything when all are hidden", () => {
    const series = [{ key: "a" }, { key: "b" }];

    expect(hiddenAmong(["a", "gone"], series)).toEqual(["a"]);
    expect(hiddenAmong(["a", "b"], series)).toEqual([]);
    expect(hiddenAmong(["a"], [])).toEqual([]);
  });

  it("filters on a task only once the catalog knows it", () => {
    const labels = new Map([["AssistantChat", "Assistant chat"]] as const);

    expect(knownTask(null, labels, false, false)).toBeNull();
    expect(knownTask("AssistantChat", labels, true, false)).toBe("AssistantChat");
    expect(knownTask("Bogus", new Map(), false, false)).toBeUndefined();
    expect(knownTask("Bogus", labels, true, false)).toBeNull();
    expect(knownTask("Bogus", new Map(), false, true)).toBeNull();
  });

  it("names the filters that left the scope empty", () => {
    const window = "the last 7 days";

    expect(emptyScopeMessage(t, { window, agent: true, task: true })).toBe(
      "Nothing ran for this agent and task in the last 7 days.",
    );
    expect(emptyScopeMessage(t, { window, agent: true, task: false })).toBe(
      "Nothing ran for this agent in the last 7 days.",
    );
    expect(emptyScopeMessage(t, { window, agent: false, task: true })).toBe(
      "Nothing ran for this task in the last 7 days.",
    );
    expect(emptyScopeMessage(t, { window, agent: false, task: false })).toBe(
      "Nothing ran in the last 7 days.",
    );
  });
});
