import type {
  AIRetrievalReindexEstimate,
  AIRetrievalSource,
  AIRetrievalStatus,
  AIRetrievalUnavailableReason,
} from "@/lib/graphql/ai-retrieval";
import { describe, expect, it } from "vitest";
import {
  SOURCE_STATE,
  UNAVAILABLE_NOTICE,
  availabilityNotice,
  budgetReached,
  estimateExceedsBudget,
  failedEntryStatus,
  modelChangeShare,
  monthCost,
  retrievalRailState,
  retrievalRefetchInterval,
  parseBudgetCents,
  retrievalSentence,
  retrievalTotals,
  sourceState,
  sourceWaiting,
  sourceShares,
  RETRIEVAL_REFRESH_MS,
} from "../retrieval-model";

// Every value of AIRetrievalUnavailableReason in airetrieval.graphqls.
const EVERY_REASON: AIRetrievalUnavailableReason[] = [
  "ExtensionMissing",
  "SchemaMissing",
  "TooOld",
  "NoProvider",
  "Disabled",
  "BudgetPaused",
  "NotIndexed",
  "QueryTimeout",
  "ProviderFailed",
];

function source(overrides: Partial<AIRetrievalSource> = {}): AIRetrievalSource {
  return {
    sourceType: "Document",
    enabled: true,
    total: 0,
    indexed: 0,
    pending: 0,
    failed: 0,
    skipped: 0,
    lastIndexedAt: null,
    lastAttemptAt: null,
    ...overrides,
  };
}

function status(overrides: Partial<AIRetrievalStatus> = {}): AIRetrievalStatus {
  return {
    availability: {
      available: true,
      reason: null,
      extensionInstalled: true,
      extensionVersion: "0.8.1",
    },
    settings: {
      memoryEnabled: true,
      documentsEnabled: true,
      inboundMessagesEnabled: true,
      monthlyIndexingBudgetUsd: "10.00",
      paused: false,
      pausedReason: null,
      pausedAt: null,
      activeModelKey: "api.voyageai.com/voyage-3.5@1024",
      dimensions: 1024,
      pendingModelKey: null,
      pendingDimensions: null,
      version: 3,
      updatedAt: 1_790_000_000,
    },
    sources: [],
    monthStartedAt: 1_788_220_800,
    indexingCostMonthUsd: "0.00",
    indexingUnpricedCalls: 0,
    retrievalCostMonthUsd: "0.00",
    retrievalUnpricedCalls: 0,
    lastIndexedAt: null,
    modelChange: null,
    configuredModelKey: "api.voyageai.com/voyage-3.5@1024",
    configuredModelDiffers: false,
    ...overrides,
  } as AIRetrievalStatus;
}

function unavailable(reason: AIRetrievalUnavailableReason): AIRetrievalStatus["availability"] {
  return { available: false, reason, extensionInstalled: true, extensionVersion: null };
}

describe("availabilityNotice", () => {
  it("shows nothing while search by meaning works", () => {
    expect(availabilityNotice(status().availability)).toBeNull();
  });

  // The schema says the reason is absent when retrieval is available; an
  // unavailable answer without one names nothing to fix, so nothing is invented.
  it("shows nothing when the server gives no reason", () => {
    expect(
      availabilityNotice({
        available: false,
        reason: null,
        extensionInstalled: false,
        extensionVersion: null,
      }),
    ).toBeNull();
  });

  it("has a notice for every reason the server can give", () => {
    for (const reason of EVERY_REASON) {
      const notice = availabilityNotice(unavailable(reason));
      expect(notice, reason).not.toBeNull();
      expect(notice?.title, reason).not.toBe("");
      expect(notice?.message, reason).not.toBe("");
    }
    expect(Object.keys(UNAVAILABLE_NOTICE).sort()).toEqual([...EVERY_REASON].sort());
  });

  it("points each reason at the fix the operator can make", () => {
    const fixOf = (reason: AIRetrievalUnavailableReason) =>
      availabilityNotice(unavailable(reason))?.fix;

    expect(fixOf("ExtensionMissing")).toBe("command");
    expect(fixOf("TooOld")).toBe("command");
    expect(fixOf("SchemaMissing")).toBe("command");
    expect(fixOf("NoProvider")).toBe("providers");
    expect(fixOf("QueryTimeout")).toBe("providers");
    expect(fixOf("ProviderFailed")).toBe("providers");
    expect(fixOf("Disabled")).toBe("settings");
    expect(fixOf("BudgetPaused")).toBe("settings");
    expect(fixOf("NotIndexed")).toBe("none");
  });

  it("names pgvector 0.8 for a missing extension and the Providers tab for no provider", () => {
    expect(UNAVAILABLE_NOTICE.ExtensionMissing.message).toContain("pgvector 0.8");
    expect(UNAVAILABLE_NOTICE.NoProvider.message).toContain("Providers tab");
    expect(UNAVAILABLE_NOTICE.NotIndexed.message).toContain("starts on its own");
  });
});

describe("source status", () => {
  // The phases the section is specified with: Pending is queued, Indexing is
  // active, Indexed is complete, Failed is failed, Paused is attention.
  it("maps each state to its lifecycle phase", () => {
    expect(SOURCE_STATE.pending.phase).toBe("queued");
    expect(SOURCE_STATE.indexing.phase).toBe("active");
    expect(SOURCE_STATE.indexed.phase).toBe("complete");
    expect(SOURCE_STATE.failed.phase).toBe("failed");
    expect(SOURCE_STATE.paused.phase).toBe("attention");
    expect(SOURCE_STATE.off.phase).toBe("closed");
  });

  it("is off when the source is turned off, whatever its counts", () => {
    expect(sourceState(source({ enabled: false, total: 9, failed: 3 }), status())).toBe("off");
  });

  it("is paused when a person paused indexing or the budget did", () => {
    const manual = status({
      settings: { ...status().settings, paused: true, pausedReason: "Manual" },
    });
    const budget = status({
      settings: { ...status().settings, paused: true, pausedReason: "Budget" },
    });

    expect(sourceState(source({ total: 4, pending: 4 }), manual)).toBe("paused");
    expect(sourceState(source({ total: 4, indexed: 4 }), budget)).toBe("paused");
  });

  it("is paused while the database or the routing stops the indexer", () => {
    for (const reason of ["ExtensionMissing", "TooOld", "SchemaMissing", "NoProvider"] as const) {
      expect(
        sourceState(source({ total: 4 }), status({ availability: unavailable(reason) })),
        reason,
      ).toBe("paused");
    }
  });

  // A slow or failing query provider does not stop indexing; the source says
  // where its own index stands.
  it("is not paused by a failing search", () => {
    expect(
      sourceState(
        source({ total: 2, indexed: 2 }),
        status({ availability: unavailable("ProviderFailed") }),
      ),
    ).toBe("indexed");
  });

  it("is pending until the indexer has reached it", () => {
    expect(sourceState(source({ total: 5 }), status())).toBe("pending");
    expect(sourceState(source({ total: 5, pending: 5 }), status())).toBe("pending");
    expect(
      sourceState(source({ total: 5 }), status({ availability: unavailable("NotIndexed") })),
    ).toBe("pending");
  });

  it("is indexing once work on it has begun and some is still queued", () => {
    expect(sourceState(source({ total: 5, indexed: 2, pending: 3 }), status())).toBe("indexing");
    expect(sourceState(source({ total: 5, pending: 5, lastAttemptAt: 10 }), status())).toBe(
      "indexing",
    );
  });

  it("is failed when nothing is queued and some items failed", () => {
    expect(sourceState(source({ total: 5, indexed: 4, failed: 1 }), status())).toBe("failed");
  });

  // Queued work outranks old failures: the retries are part of what is running.
  it("is indexing, not failed, while failures sit beside queued work", () => {
    expect(sourceState(source({ total: 5, indexed: 2, failed: 1, pending: 2 }), status())).toBe(
      "indexing",
    );
  });

  it("is indexed when every item is indexed or skipped on purpose, or there are none", () => {
    expect(sourceState(source({ total: 5, indexed: 3, skipped: 2 }), status())).toBe("indexed");
    expect(sourceState(source({ total: 0 }), status())).toBe("indexed");
  });
});

describe("sourceWaiting", () => {
  it("counts queued entries and rows the sweep has not reached", () => {
    expect(sourceWaiting(source({ total: 10, indexed: 4, skipped: 1, failed: 1 }))).toBe(4);
    expect(sourceWaiting(source({ total: 10, indexed: 4, pending: 6 }))).toBe(6);
  });

  it("is never negative when entries outlive their rows", () => {
    expect(sourceWaiting(source({ total: 2, indexed: 5 }))).toBe(0);
  });

  it("is nothing for a source that is off", () => {
    expect(sourceWaiting(source({ enabled: false, total: 10, pending: 3 }))).toBe(0);
  });
});

describe("retrievalTotals", () => {
  it("adds up every source, waiting only where the source is on", () => {
    expect(
      retrievalTotals([
        source({ sourceType: "Memory", total: 10, indexed: 8, pending: 2 }),
        source({ sourceType: "Document", total: 7, indexed: 3, failed: 2 }),
        source({ sourceType: "InboundMessage", enabled: false, total: 50, indexed: 5 }),
      ]),
    ).toEqual({ indexed: 16, waiting: 4, failed: 2 });
  });
});

describe("cost", () => {
  it("adds indexing and search for the month", () => {
    expect(
      monthCost(status({ indexingCostMonthUsd: "1.25", retrievalCostMonthUsd: "0.10" })),
    ).toBeCloseTo(1.35);
    expect(
      monthCost(status({ indexingCostMonthUsd: "oops", retrievalCostMonthUsd: "0.10" })),
    ).toBeCloseTo(0.1);
  });

  it("counts the budget as reached at the budget, not only past it", () => {
    expect(budgetReached(status({ indexingCostMonthUsd: "9.99" }))).toBe(false);
    expect(budgetReached(status({ indexingCostMonthUsd: "10.00" }))).toBe(true);
    expect(budgetReached(status({ indexingCostMonthUsd: "12.00" }))).toBe(true);
  });
});

describe("retrievalRefetchInterval", () => {
  it("reads again only while the index is moving", () => {
    expect(retrievalRefetchInterval(undefined)).toBe(false);
    expect(retrievalRefetchInterval(status({ sources: [source({ total: 3, indexed: 3 })] }))).toBe(
      false,
    );
    expect(retrievalRefetchInterval(status({ sources: [source({ total: 3, pending: 1 })] }))).toBe(
      RETRIEVAL_REFRESH_MS,
    );
    expect(
      retrievalRefetchInterval(
        status({
          modelChange: {
            fromModelKey: "a@768",
            toModelKey: "b@1024",
            dimensions: 1024,
            total: 3,
            indexed: 3,
            pending: 0,
            failed: 0,
          },
        }),
      ),
    ).toBe(RETRIEVAL_REFRESH_MS);
  });

  it("does not poll for a source that is off", () => {
    expect(
      retrievalRefetchInterval(status({ sources: [source({ enabled: false, pending: 4 })] })),
    ).toBe(false);
  });
});

describe("modelChangeShare", () => {
  const change = { fromModelKey: "a@768", toModelKey: "b@1024", dimensions: 1024, pending: 0 };

  it("is the share indexed under the new model", () => {
    expect(modelChangeShare({ ...change, total: 40, indexed: 10, failed: 0 })).toBe(0.25);
  });

  it("stays within 0 and 1", () => {
    expect(modelChangeShare({ ...change, total: 0, indexed: 0, failed: 0 })).toBe(0);
    expect(modelChangeShare({ ...change, total: 2, indexed: 5, failed: 0 })).toBe(1);
  });
});

describe("retrievalRailState", () => {
  it("carries the reason only when retrieval is unavailable", () => {
    expect(
      retrievalRailState(
        status({ sources: [source({ total: 4, indexed: 1, failed: 1, pending: 2 })] }),
      ),
    ).toEqual({ available: true, reason: null, failed: 1, waiting: 2 });
    expect(retrievalRailState(status({ availability: unavailable("BudgetPaused") }))).toMatchObject(
      { available: false, reason: "BudgetPaused" },
    );
  });
});

describe("retrievalSentence", () => {
  it("says nothing is routed while no model is configured, counting what waits", () => {
    const base = status();
    const unrouted = status({
      availability: { ...base.availability, available: false, reason: "NoProvider" },
      settings: { ...base.settings, activeModelKey: null },
      configuredModelKey: null,
      sources: [
        source({ sourceType: "Memory", enabled: true, total: 10, skipped: 2 }),
        source({ sourceType: "Document", enabled: false, total: 500 }),
      ],
    });

    expect(retrievalSentence(unrouted)).toEqual({ kind: "unrouted", waiting: 8 });
  });

  it("says indexing is paused, and whether the budget paused it", () => {
    const base = status();

    expect(
      retrievalSentence(
        status({ settings: { ...base.settings, paused: true, pausedReason: "Manual" } }),
      ),
    ).toEqual({ kind: "paused", budget: false });
    expect(
      retrievalSentence(
        status({
          availability: { ...base.availability, available: false, reason: "BudgetPaused" },
          settings: { ...base.settings, paused: true, pausedReason: "Budget" },
        }),
      ),
    ).toEqual({ kind: "paused", budget: true });
  });

  it("says how much is left while the indexer works, and what failed once it is done", () => {
    const base = status();
    const indexing = status({
      sources: [source({ sourceType: "Memory", enabled: true, total: 10, indexed: 4, pending: 6 })],
    });
    const done = status({
      sources: [source({ sourceType: "Memory", enabled: true, total: 10, indexed: 8, failed: 2 })],
    });

    expect(retrievalSentence(indexing)).toEqual({
      kind: "indexing",
      model: base.settings.activeModelKey,
      waiting: 6,
    });
    expect(retrievalSentence(done)).toEqual({
      kind: "done",
      model: base.settings.activeModelKey,
      failed: 2,
    });
  });
});

describe("parseBudgetCents", () => {
  it("reads dollars typed with or without a sign, commas or cents", () => {
    expect(parseBudgetCents("12.5")).toBe(1250);
    expect(parseBudgetCents("$1,000")).toBe(100_000);
    expect(parseBudgetCents(" 0 ")).toBe(0);
  });

  it("refuses text, a negative, a third decimal or more than the server takes", () => {
    expect(parseBudgetCents("ten")).toBeNull();
    expect(parseBudgetCents("-5")).toBeNull();
    expect(parseBudgetCents("1.234")).toBeNull();
    expect(parseBudgetCents("100000.01")).toBeNull();
    expect(parseBudgetCents("100000")).toBe(10_000_000);
  });
});

describe("sourceShares", () => {
  it("draws each part of the bar as a share of everything the source holds", () => {
    expect(sourceShares(source({ total: 200, indexed: 100, failed: 20, skipped: 10 }))).toEqual({
      indexed: 50,
      failed: 10,
      skipped: 5,
    });
  });

  it("draws nothing for an empty source rather than dividing by zero", () => {
    expect(sourceShares(source({ total: 0, indexed: 0, failed: 0, skipped: 0 }))).toEqual({
      indexed: 0,
      failed: 0,
      skipped: 0,
    });
  });
});

describe("estimateExceedsBudget", () => {
  const estimate: AIRetrievalReindexEstimate = {
    sourceType: "Document",
    modelKey: "api.voyageai.com/voyage-3.5@1024",
    sources: 100,
    averageChunks: 4,
    chunksMeasured: true,
    averageTokensPerChunk: 311.5,
    estimatedTokens: 124_600,
    inputCostPerMillionUsd: "0.06",
    estimatedCostUsd: "0.007476",
    remainingBudgetUsd: "6.75",
  };

  it("warns only when the estimate is more than what is left", () => {
    expect(estimateExceedsBudget(estimate)).toBe(false);
    expect(estimateExceedsBudget({ ...estimate, remainingBudgetUsd: "0.00" })).toBe(true);
    expect(
      estimateExceedsBudget({ ...estimate, estimatedCostUsd: "6.75", remainingBudgetUsd: "6.75" }),
    ).toBe(false);
  });

  // An unpriced provider's cost is unknown, not zero, and not over budget.
  it("does not warn about a cost nobody knows", () => {
    expect(
      estimateExceedsBudget({
        ...estimate,
        estimatedCostUsd: null,
        inputCostPerMillionUsd: null,
        remainingBudgetUsd: "0.00",
      }),
    ).toBe(false);
  });
});

describe("failedEntryStatus", () => {
  it("reads a pending entry as a scheduled retry and a failed one as final", () => {
    expect(failedEntryStatus({ status: "Pending" })).toMatchObject({
      phase: "queued",
      text: "Retrying",
    });
    expect(failedEntryStatus({ status: "Failed" })).toMatchObject({
      phase: "failed",
      text: "Failed",
    });
  });
});
