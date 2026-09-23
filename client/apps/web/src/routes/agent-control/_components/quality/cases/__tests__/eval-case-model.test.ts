import { describe, expect, it } from "vitest";
import type { AgentEvalCaseDetail } from "@/lib/graphql/agent-eval-cases";
import {
  evalCaseFormDefaults,
  evalCaseFormSchema,
  nextStatuses,
  readExpected,
  toExpected,
  toFormValues,
  toUpdateInput,
  type EvalCaseFormValues,
} from "../eval-case-model";
import { corrections } from "../../../activity/evaluation-comparison";

const stored = {
  toolMode: "Ordered",
  tools: [
    {
      name: "update_rate",
      args: { shipmentId: "shp_1", rate: 1350, pickupDate: "2026-09-01", ids: ["trc_1", "trc_2"] },
      rules: {
        rate: { kind: "numeric", abs: 10, rel: 0.02 },
        pickupDate: { kind: "dateWindow", windowSeconds: 86400 },
        ids: { kind: "setEq" },
        status: { kind: "oneOf", values: ["Available", "OutOfService"] },
        reason: { kind: "present" },
        note: { kind: "ignore" },
      },
    },
  ],
  forbiddenTools: ["cancel_shipment"],
  proposals: [
    {
      toolName: "update_rate",
      params: { shipmentId: "shp_1", rate: 1350 },
      rejected: false,
      rules: { rate: { kind: "numeric", abs: 5 } },
      sourceProposalId: "aprop_1",
    },
    { toolName: "cancel_shipment", params: { shipmentId: "shp_1" }, rejected: true },
  ],
  expectRefusal: false,
  mustMention: ["hold"],
  mustNotMention: ["cancelled"],
};

function detail(expected: unknown): AgentEvalCaseDetail {
  return {
    id: "aec_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    agentDefinitionId: "agd_1",
    title: "Rate S-100",
    source: "DecidedProposal",
    status: "Candidate",
    trigger: "Chat",
    input: "Rate S-100",
    sourceThreadId: null,
    sourceProposalId: "aprop_1",
    heldTools: ["update_rate"],
    expected,
    weight: 1,
    expiresAt: null,
    version: 3,
    createdAt: 1,
    updatedAt: 1,
    sourceRunId: null,
    sourceTurnId: null,
    sourceMessageId: null,
    sourceFeedbackId: null,
    history: [],
    pageContext: null,
    mentions: [],
    subjectType: "",
    subjectId: null,
    toolFixtures: [],
    rubric: "",
    redaction: null,
    contentHash: "hash",
    capturedFingerprint: null,
    createdByUserId: null,
  } as AgentEvalCaseDetail;
}

describe("readExpected", () => {
  it("keeps every tolerance rule the server stores", () => {
    const expected = readExpected(stored);

    expect(expected.toolMode).toBe("Ordered");
    expect(expected.tools[0].rules?.rate).toEqual({
      kind: "numeric",
      abs: 10,
      rel: 0.02,
      windowSeconds: undefined,
      values: undefined,
    });
    expect(expected.tools[0].rules?.status?.values).toEqual(["Available", "OutOfService"]);
    expect(expected.proposals[1].rejected).toBe(true);
  });

  it("drops what it does not recognise instead of failing", () => {
    const expected = readExpected({
      toolMode: "Sometimes",
      tools: [{ name: 3 }, { name: "get_shipment", rules: { id: { kind: "fuzzy" } } }],
      mustMention: ["hold", 4],
    });

    expect(expected.toolMode).toBe("AnyOrder");
    expect(expected.tools).toHaveLength(1);
    expect(expected.tools[0].rules).toEqual({});
    expect(expected.mustMention).toEqual(["hold"]);
    expect(readExpected(null).tools).toEqual([]);
  });
});

describe("form round trip", () => {
  it("writes back what it read, rules and values alike", () => {
    const values = toFormValues(detail(stored));
    const expected = toExpected(values);

    expect(expected.tools[0].args).toEqual({
      shipmentId: "shp_1",
      rate: 1350,
      pickupDate: "2026-09-01",
      ids: ["trc_1", "trc_2"],
    });
    expect(expected.tools[0].rules?.pickupDate).toEqual({
      kind: "dateWindow",
      windowSeconds: 86400,
    });
    expect(expected.tools[0].rules?.status).toEqual({
      kind: "oneOf",
      values: ["Available", "OutOfService"],
    });
    expect(expected.tools[0].rules?.shipmentId).toBeUndefined();
    expect(expected.proposals[0].params).toEqual({ shipmentId: "shp_1", rate: 1350 });
    expect(expected.proposals[0].rules).toEqual({ rate: { kind: "numeric", abs: 5 } });
    expect(expected.proposals[0].sourceProposalId).toBe("aprop_1");
  });

  it("sends the expiry only when it changed, and null to clear it", () => {
    const values = toFormValues(detail(stored));

    expect("expiresAt" in toUpdateInput(values, null)).toBe(false);
    expect(toUpdateInput({ ...values, expiresAt: 1_800_000_000 }, null).expiresAt).toBe(
      1_800_000_000,
    );
    expect(toUpdateInput({ ...values, expiresAt: null }, 1_800_000_000).expiresAt).toBeNull();
    expect(toUpdateInput(values, null).version).toBe(3);
  });
});

function valid(overrides: Partial<EvalCaseFormValues>): EvalCaseFormValues {
  return { ...evalCaseFormDefaults, agentDefinitionId: "agd_1", input: "Rate S-100", ...overrides };
}

function issuePaths(values: EvalCaseFormValues): string[] {
  const result = evalCaseFormSchema.safeParse(values);
  return result.success ? [] : result.error.issues.map((issue) => issue.path.join("."));
}

describe("evalCaseFormSchema", () => {
  it("accepts a plain case", () => {
    expect(issuePaths(valid({}))).toEqual([]);
  });

  it("refuses the combinations the server refuses", () => {
    const rule = {
      key: "rate",
      value: "",
      kind: "numeric" as const,
      abs: "-1",
      rel: "2",
      windowHours: "",
      values: [],
    };

    expect(issuePaths(valid({ tools: [{ name: "update_rate", args: [rule] }] }))).toEqual(
      expect.arrayContaining(["tools.0.args.0.value", "tools.0.args.0.abs", "tools.0.args.0.rel"]),
    );
    expect(
      issuePaths(
        valid({
          tools: [{ name: "cancel_shipment", args: [] }],
          forbiddenTools: ["cancel_shipment"],
        }),
      ),
    ).toContain("tools.0.name");
    expect(
      issuePaths(valid({ expectRefusal: true, tools: [{ name: "get_shipment", args: [] }] })),
    ).toContain("expectRefusal");
    expect(issuePaths(valid({ mustMention: ["Hold"], mustNotMention: ["hold"] }))).toContain(
      "mustNotMention",
    );
    expect(
      issuePaths(
        valid({
          proposals: [
            {
              toolName: "update_rate",
              rejected: false,
              params: "[1]",
              sourceProposalId: "",
              rules: [],
            },
          ],
        }),
      ),
    ).toContain("proposals.0.params");
  });

  it("does not ask a proposal rule for a value", () => {
    const rule = {
      key: "rate",
      value: "",
      kind: "numeric" as const,
      abs: "5",
      rel: "",
      windowHours: "",
      values: [],
    };

    expect(
      issuePaths(
        valid({
          proposals: [
            {
              toolName: "update_rate",
              rejected: false,
              params: '{"rate":1}',
              sourceProposalId: "",
              rules: [rule],
            },
          ],
        }),
      ),
    ).toEqual([]);
  });
});

describe("nextStatuses", () => {
  it("follows the server's workflow", () => {
    expect(nextStatuses("Candidate")).toEqual(["Active", "Retired"]);
    expect(nextStatuses("Active")).toEqual(["Quarantined", "Retired"]);
    expect(nextStatuses("Retired")).toEqual(["Active"]);
  });
});

describe("corrections", () => {
  it("lists what a person changed before approving", () => {
    expect(
      corrections({
        toolName: "update_rate",
        verdict: "Agreed",
        originalParams: { shipmentId: "shp_1", rate: 1200 },
        correctedParams: { shipmentId: "shp_1", rate: 1350 },
      }),
    ).toEqual([{ field: "rate", from: "1200", to: "1350" }]);
  });

  it("is empty when nobody changed anything", () => {
    expect(
      corrections({ toolName: "update_rate", verdict: "Agreed", originalParams: { rate: 1 } }),
    ).toEqual([]);
  });
});
