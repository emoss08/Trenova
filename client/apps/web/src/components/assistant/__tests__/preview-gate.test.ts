import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  approvalGate,
  batchPreviewDigests,
  canApprove,
  gateDigest,
  isPreviewConflict,
  isPreviewRefusal,
} from "../proposal-preview/preview-gate";
import {
  PREVIEW_WARNING_CODES,
  previewWarningText,
  previewWarningTone,
  visibleWarnings,
} from "../proposal-preview/preview-warnings";

const t = translate;

function graphQLError(type: string) {
  return new GraphQLRequestError({
    kind: "graphql",
    message: "refused",
    status: 200,
    graphQLErrors: [
      {
        message: "refused",
        extensions: { type: `https://trenova.app/problems/${type}` },
        type: `https://trenova.app/problems/${type}`,
      },
    ],
  });
}

describe("approvalGate", () => {
  it("holds Approve while the first read runs", () => {
    const gate = approvalGate({ data: undefined, isPending: true, isError: false });

    expect(gate).toEqual({ state: "loading" });
    expect(canApprove(gate)).toBe(false);
    expect(gateDigest(gate)).toBeUndefined();
  });

  it("approves against the digest on screen", () => {
    const gate = approvalGate({
      data: { digest: "sha256:a", stale: false },
      isPending: false,
      isError: false,
    });

    expect(canApprove(gate)).toBe(true);
    expect(gateDigest(gate)).toBe("sha256:a");
  });

  // The server refuses to approve a stale proposal; Reject still records
  // the digest the person saw.
  it("turns Approve off for a stale preview but keeps its digest", () => {
    const gate = approvalGate({
      data: { digest: "sha256:b", stale: true },
      isPending: false,
      isError: false,
    });

    expect(canApprove(gate)).toBe(false);
    expect(gateDigest(gate)).toBe("sha256:b");
  });

  // A refetch that fails behind a preview already on screen leaves it the
  // one approved against: the person saw it.
  it("keeps the preview on screen as the basis when a later read fails", () => {
    const gate = approvalGate({
      data: { digest: "sha256:c", stale: false },
      isPending: false,
      isError: true,
    });

    expect(gateDigest(gate)).toBe("sha256:c");
  });

  it("leaves approval possible without a digest when the preview cannot be read", () => {
    const gate = approvalGate({ data: undefined, isPending: false, isError: true });

    expect(canApprove(gate)).toBe(true);
    expect(gateDigest(gate)).toBeUndefined();
  });
});

describe("isPreviewConflict", () => {
  it("recognises the digest mismatch the server answers with a conflict", () => {
    expect(isPreviewConflict(graphQLError("resource-conflict"))).toBe(true);
  });

  it("is not fooled by another refusal", () => {
    expect(isPreviewConflict(graphQLError("business-rule-violation"))).toBe(false);
    expect(isPreviewConflict(new Error("network"))).toBe(false);
  });

  it("tells a refused draft from a preview that could not be read", () => {
    expect(isPreviewRefusal(graphQLError("validation-error"))).toBe(true);
    expect(isPreviewRefusal(graphQLError("business-rule-violation"))).toBe(false);
  });
});

describe("batchPreviewDigests", () => {
  it("sends a digest only for proposals whose preview was shown, in batch order", () => {
    const shown = new Map([
      ["aprop_3", "sha256:3"],
      ["aprop_1", "sha256:1"],
      ["aprop_other", "sha256:x"],
    ]);

    expect(batchPreviewDigests(["aprop_1", "aprop_2", "aprop_3"], shown)).toEqual([
      { proposalId: "aprop_1", digest: "sha256:1" },
      { proposalId: "aprop_3", digest: "sha256:3" },
    ]);
  });

  // The server refuses a batch that lists one proposal twice.
  it("never lists a proposal twice and skips an empty digest", () => {
    const shown = new Map([
      ["aprop_1", "sha256:1"],
      ["aprop_2", ""],
    ]);

    expect(batchPreviewDigests(["aprop_1", "aprop_1", "aprop_2"], shown)).toEqual([
      { proposalId: "aprop_1", digest: "sha256:1" },
    ]);
  });
});

describe("preview warnings", () => {
  it("translates every code the contract names, never falling back to the English", () => {
    for (const code of PREVIEW_WARNING_CODES) {
      const text = previewWarningText(
        { code, args: [], message: "SERVER ENGLISH", reasons: [] },
        t,
      );
      expect(text, code).not.toBe("SERVER ENGLISH");
      expect(text, code).not.toBe("");
    }
  });

  it("carries the step and the reason a warning names", () => {
    expect(
      previewWarningText({ code: "depends_on_step", args: ["2"], message: "", reasons: [] }, t),
    ).toBe("Step 2 changes this record first; it is shown as it would be after that step.");
    expect(
      previewWarningText(
        { code: "would_fail", args: ["Rate not found"], message: "", reasons: [] },
        t,
      ),
    ).toBe("This would not go through as it stands: Rate not found");
  });

  it("shows the server's words for a code this client does not know", () => {
    const warning = { code: "new_kind", args: [], message: "A new thing to know.", reasons: [] };

    expect(previewWarningText(warning, t)).toBe("A new thing to know.");
    expect(previewWarningTone(warning)).toBe("warning");
  });

  it("draws a write that will not do what was asked as danger", () => {
    expect(previewWarningTone({ code: "would_fail", args: [], message: "", reasons: [] })).toBe(
      "danger",
    );
    expect(previewWarningTone({ code: "unpinned", args: [], message: "", reasons: [] })).toBe(
      "info",
    );
  });

  it("drops the dependency warning inside a plan, where the step says it", () => {
    const warnings = [
      { code: "depends_on_step", args: ["1"], message: "", reasons: [] },
      { code: "withheld", args: [], message: "", reasons: [] },
    ];

    expect(visibleWarnings(warnings, { inPlan: true }).map((w) => w.code)).toEqual(["withheld"]);
    expect(visibleWarnings(warnings, { inPlan: false })).toHaveLength(2);
  });
});
