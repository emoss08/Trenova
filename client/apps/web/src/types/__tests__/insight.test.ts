import { insightListSchema, insightSchema } from "@/types/insight";
import { describe, expect, it } from "vitest";

/**
 * The fixture is the Go struct on the wire, not what the UI would like to be
 * handed. `insight.Insight` (services/tms/internal/core/domain/insight/insight.go)
 * has two kinds of unset field and they serialize differently:
 *
 * - `DismissedByID` and `ProviderID` are `pulid.ID`. `pulid.ID.MarshalJSON`
 *   returns the literal `null` when the id is nil, so an insight nobody has
 *   dismissed, or that the narrator never touched, sends `null` for these.
 * - `Subject`, `Narrative`, `Recommendation`, `DismissReason` and
 *   `ModelIdentifier` are Go `string`. Their `nullzero` tag governs the database
 *   write, not the JSON, so an unset one is always `""` and never null.
 *
 * Getting that backwards is what broke the insights page: the schema declared
 * the ids as `z.string().optional()`, which admits `undefined` and nothing else,
 * so a single undismissed insight failed the parse and the whole list was
 * discarded.
 */
const undismissed = {
  id: "inst_01JINSIGHT00000000000000",
  businessUnitId: "bu_01JBUSINESSUNIT000000000",
  organizationId: "org_01JORGANIZATION00000000",
  detectorKey: "ontime-decline",
  category: "ServiceQuality",
  severity: "Warning",
  status: "Active",
  dedupeKey: "ontime-decline:cus_1",
  subject: "Acme Foods",
  headline: "On-time delivery for Acme Foods is 82.4%",
  narrative: "",
  recommendation: "",
  narrated: false,
  metrics: [],
  links: [],
  windowStart: 1_700_000_000,
  windowEnd: 1_702_592_000,
  detectedAt: 1_702_592_000,
  staleAt: 1_702_678_400,
  dismissedAt: null,
  dismissedById: null,
  dismissReason: "",
  modelIdentifier: "",
  providerId: null,
  version: 0,
  createdAt: 1_702_592_000,
  updatedAt: 1_702_592_000,
};

describe("insightSchema", () => {
  it("reads an insight whose unset ids the server wrote as null", () => {
    const parsed = insightSchema.parse(undismissed);

    expect(parsed.dismissedById).toBe("");
    expect(parsed.providerId).toBe("");
    expect(parsed.dismissedAt).toBeNull();
  });

  it("keeps the ids of an insight that was dismissed and narrated", () => {
    const parsed = insightSchema.parse({
      ...undismissed,
      narrated: true,
      status: "Dismissed",
      dismissedAt: 1_702_600_000,
      dismissedById: "usr_01JUSER000000000000000AB",
      dismissReason: "Known seasonal dip.",
      modelIdentifier: "qwen3:32b",
      providerId: "aiprv_01JPROVIDER00000000000AB",
    });

    expect(parsed.dismissedById).toBe("usr_01JUSER000000000000000AB");
    expect(parsed.providerId).toBe("aiprv_01JPROVIDER00000000000AB");
    expect(parsed.dismissReason).toBe("Known seasonal dip.");
  });

  it("still fills the ids in when the keys are absent", () => {
    const { dismissedById: _a, providerId: _b, ...withoutIds } = undismissed;
    const parsed = insightSchema.parse(withoutIds);

    expect(parsed.dismissedById).toBe("");
    expect(parsed.providerId).toBe("");
  });
});

describe("insightListSchema", () => {
  /**
   * The shape the reported failure actually came back as: a list where more
   * than one row carries a null id. One bad row discards the whole response,
   * so the list is what has to parse, not just a row in isolation.
   */
  it("reads a list of insights that have never been dismissed", () => {
    const parsed = insightListSchema.parse({
      results: [undismissed, { ...undismissed, id: "inst_01JINSIGHT00000000000001" }],
    });

    expect(parsed.results).toHaveLength(2);
    expect(parsed.results.map((insight) => insight.dismissedById)).toEqual(["", ""]);
  });
});
