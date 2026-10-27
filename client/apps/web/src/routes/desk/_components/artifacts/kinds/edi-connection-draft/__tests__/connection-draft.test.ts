import type { AssistantArtifact } from "@/types/assistant";
import { formatMessage } from "@trenova/shared/i18n/format-message";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import {
  connectionDraftFrom,
  draftOutline,
  partnerNameOf,
  resolutionQuestion,
  sourceLabel,
  valueState,
} from "../connection-draft";

const t: TranslateFn = (message, ...args) => formatMessage("en", message ?? "", args);

function artifactOf(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: "ath_1",
    kind: "edi_connection_draft",
    status: "Ready",
    title: "EDI connection for Acme Retail Group",
    payload,
  } as unknown as AssistantArtifact;
}

const payload = {
  documentId: "doc_1",
  fileName: "acme-guide.pdf",
  pageCount: 7,
  documentDigest: "sha256:5f1c9a",
  ready: false,
  changes: 9,
  blockers: ["Decide Time zone"],
  mappingCount: 2,
  truncated: false,
  partner: {
    match: "Existing",
    partnerId: "edip_1",
    version: 4,
    change: "Change",
    fields: [
      {
        field: "partner.name",
        label: "Name",
        value: "Acme Retail Group",
        current: "Acme Retail",
        change: "Change",
        status: "Accepted",
        confidence: 0.97,
        required: true,
        source: { kind: "Document", page: 1, quote: "Acme Retail Group", section: "1.1" },
      },
      { field: "partner.notes", label: "Notes", value: 42, change: "Nonsense", status: "Odd" },
    ],
  },
  connection: {
    method: "AS2",
    inbound: true,
    outbound: true,
    sets: [{ transactionSet: "204", direction: "Inbound" }],
  },
  interchange: [],
  transport: {
    name: "Acme AS2",
    method: "AS2",
    status: "Inactive",
    change: "Add",
    fields: [],
    credentials: [
      {
        keys: ["as2.partnerCertificate"],
        label: "Partner signing certificate",
        kind: "Secret",
        required: false,
        missing: true,
      },
      {
        keys: ["as2.privateKey"],
        label: "AS2 private key",
        kind: "Secret",
        required: true,
        missing: true,
      },
    ],
  },
  sets: [
    {
      transactionSet: "214",
      direction: "Outbound",
      change: "Add",
      fields: [],
      requirements: 4,
      unmet: [
        {
          segmentId: "MS1",
          elementPosition: 1,
          usage: "Mandatory",
          problem: "The drafted map does not write MS1",
          source: { kind: "Document", page: 5, quote: "MS1 is mandatory" },
        },
      ],
    },
  ],
  project: {
    role: "carrier",
    change: "Add",
    files: [{ path: "guides/214.lading", kind: "guide", change: "Add" }],
    errors: 2,
    warnings: 0,
    diagnostics: [],
  },
  mappings: [],
  samples: [{ page: 6, transactionSet: "204", parsed: true, overview: { x12Version: "004010" } }],
  gaps: [
    {
      kind: "Unsupported",
      summary: "Appendix B is missing",
      source: { kind: "Document", page: 7, quote: "See Appendix B" },
    },
  ],
  decisions: [
    {
      field: "partner.timezone",
      label: "Time zone",
      value: "",
      change: "None",
      status: "Held",
      reason: "The guide names two",
      candidates: [
        {
          value: "America/Chicago",
          confidence: 0.9,
          source: { kind: "Document", page: 2, quote: "Central time" },
        },
        {
          value: "America/New_York",
          confidence: 0.88,
          source: { kind: "Document", page: 4, quote: "Eastern" },
        },
      ],
    },
  ],
  failures: [{ record: "Communication profile", field: "as2.url", message: "URL must be https" }],
};

describe("connectionDraftFrom", () => {
  it("reads the draft the tool published", () => {
    const draft = connectionDraftFrom(artifactOf(payload));

    expect(draft.fileName).toBe("acme-guide.pdf");
    expect(draft.documentDigest).toBe("sha256:5f1c9a");
    expect(draft.partner.match).toBe("Existing");
    expect(draft.partner.fields[0]?.source).toEqual({
      kind: "Document",
      page: 1,
      quote: "Acme Retail Group",
      section: "1.1",
      note: "",
    });
    expect(draft.transport?.credentials).toHaveLength(2);
    expect(draft.sets[0]?.unmet[0]?.segmentId).toBe("MS1");
    expect(draft.sets[0]?.requirements).toBe(4);
    expect(draft.project?.files[0]).toEqual({
      path: "guides/214.lading",
      kind: "guide",
      change: "Add",
    });
    expect(draft.decisions[0]?.candidates).toHaveLength(2);
    expect(draft.samples[0]?.overview?.x12Version).toBe("004010");
  });

  it("reads a value it does not understand as absent rather than failing the card", () => {
    const draft = connectionDraftFrom(artifactOf(payload));
    const odd = draft.partner.fields[1];

    expect(odd?.value).toBe("");
    expect(odd?.change).toBe("None");
    expect(odd?.status).toBe("Unverified");
    expect(connectionDraftFrom(artifactOf({})).sets).toEqual([]);
    expect(connectionDraftFrom(artifactOf({})).project).toBeNull();
    expect(connectionDraftFrom(artifactOf({ transport: "nope" })).transport).toBeNull();
  });
});

describe("draftOutline", () => {
  it("counts what the person must still do and what the guide asks that nothing meets", () => {
    const outline = draftOutline(connectionDraftFrom(artifactOf(payload)));

    expect(outline.decisions).toBe(1);
    expect(outline.secretsToEnter).toEqual(["AS2 private key"]);
    expect(outline.unmet).toBe(1);
    expect(outline.projectErrors).toBe(2);
    expect(outline.gaps).toBe(1);
    expect(outline.failures).toBe(1);
  });
});

describe("partnerNameOf", () => {
  it("names the partner from its name field, else the file", () => {
    expect(partnerNameOf(connectionDraftFrom(artifactOf(payload)))).toBe("Acme Retail Group");
    expect(partnerNameOf(connectionDraftFrom(artifactOf({ fileName: "x.pdf" })))).toBe("x.pdf");
  });
});

describe("sourceLabel", () => {
  it("says where a value came from", () => {
    expect(sourceLabel({ kind: "Document", page: 3, quote: "", section: "", note: "" }, t)).toBe(
      "Page 3",
    );
    expect(sourceLabel({ kind: "Sample", page: 6, quote: "", section: "", note: "" }, t)).toBe(
      "Sample, page 6",
    );
    expect(sourceLabel({ kind: "Person", page: 0, quote: "", section: "", note: "" }, t)).toBe(
      "Your answer",
    );
    expect(
      sourceLabel({ kind: "Configuration", page: 0, quote: "", section: "", note: "" }, t),
    ).toBe("Current setup");
    expect(sourceLabel({ kind: "Default", page: 0, quote: "", section: "", note: "" }, t)).toBe(
      "Default",
    );
  });
});

describe("valueState", () => {
  it("marks what was held or failed for a look", () => {
    expect(valueState("Accepted")).toBe("ok");
    expect(valueState("Defaulted")).toBe("info");
    expect(valueState("Kept")).toBe("info");
    expect(valueState("Held")).toBe("warn");
    expect(valueState("Unverified")).toBe("warn");
    expect(valueState("Invalid")).toBe("bad");
    expect(valueState("Missing")).toBe("bad");
  });
});

describe("resolutionQuestion", () => {
  it("asks the agent to draft again with the person's choice and where it came from", () => {
    const draft = connectionDraftFrom(artifactOf(payload));
    const decision = draft.decisions[0]!;

    expect(resolutionQuestion(draft, decision, decision.candidates[0]!, t)).toBe(
      'For the EDI connection drafted from acme-guide.pdf, use "America/Chicago" for Time zone (page 2) and draft it again.',
    );
  });
});
