import { assistant } from "@/lib/queries/assistant";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { ConnectionDraftSkeleton } from "../connection-draft-skeleton";
import { DeskConnectionDraftBody } from "../desk-connection-draft-body";

const THREAD = "ath_ecd";
const DIGEST = "sha256:5f1c9a0b7e3d2c1f";

function source(page: number, quote: string) {
  return { kind: "Document", page, quote, section: "2.1" };
}

function draftPayload(overrides: Record<string, unknown> = {}): Record<string, unknown> {
  return {
    documentId: "doc_1",
    fileName: "acme-guide.pdf",
    pageCount: 7,
    documentDigest: DIGEST,
    ready: false,
    changes: 6,
    blockers: ["Decide Time zone"],
    mappingCount: 1,
    truncated: false,
    partner: {
      match: "New",
      change: "Add",
      status: "Inactive",
      fields: [
        {
          field: "partner.name",
          label: "Name",
          value: "Acme Retail Group",
          change: "Add",
          status: "Accepted",
          confidence: 0.97,
          required: true,
          source: source(1, "Acme Retail Group"),
        },
        {
          field: "partner.contactEmail",
          label: "Contact email",
          value: "",
          change: "None",
          status: "Unverified",
          required: false,
          source: source(2, "edi@acme.example"),
          reason: "The quote is not on page 2",
        },
      ],
    },
    connection: {
      method: "AS2",
      inbound: true,
      outbound: true,
      sets: [
        { transactionSet: "204", direction: "Inbound" },
        { transactionSet: "214", direction: "Outbound" },
      ],
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
          keys: ["as2.privateKey"],
          label: "AS2 private key",
          kind: "Secret",
          required: true,
          missing: true,
        },
        {
          keys: ["as2.partnerCertificate"],
          label: "Partner signing certificate",
          kind: "Secret",
          required: false,
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
        requirements: 5,
        unmet: [
          {
            segmentId: "MS1",
            elementPosition: 1,
            usage: "Mandatory",
            codes: [],
            problem: "The drafted map does not write MS1",
            source: source(5, "MS1 is mandatory"),
          },
        ],
        unverified: [],
        controlNumbers: [],
      },
    ],
    project: {
      projectId: "",
      role: "carrier",
      change: "Add",
      files: [
        { path: "partner.lading", kind: "partner", change: "Add" },
        { path: "guides/214.lading", kind: "guide", change: "Add" },
      ],
      errors: 0,
      warnings: 1,
      diagnostics: [],
    },
    mappings: [
      {
        entityType: "Location",
        sourceId: "DC04",
        targetLabel: "Acme DC 4",
        targetId: "loc_1",
        change: "Add",
        status: "Accepted",
        confidence: 0.9,
        source: source(6, "DC04 Joliet"),
      },
    ],
    samples: [
      {
        page: 6,
        transactionSet: "204",
        direction: "Inbound",
        parsed: true,
        overview: {
          x12Version: "004010",
          summary: { segmentCount: 31, errorCount: 0, warningCount: 0 },
        },
        checks: [
          {
            field: "isaSenderId",
            label: "Sender id",
            expected: "ACMERETAIL",
            found: "ACMERETAIL",
            matches: true,
          },
        ],
        unmapped: [],
      },
    ],
    gaps: [
      {
        kind: "Unsupported",
        summary: "Appendix B is missing",
        source: source(7, "See Appendix B"),
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
          { value: "America/Chicago", confidence: 0.9, source: source(2, "Central time") },
          { value: "America/New_York", confidence: 0.88, source: source(4, "Eastern time") },
        ],
      },
    ],
    failures: [],
    ...overrides,
  };
}

function artifactOf(payload: Record<string, unknown>): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: THREAD,
    kind: "edi_connection_draft",
    status: "Ready",
    title: "EDI connection for Acme Retail Group",
    payload,
    sourceToolCallId: "call_1",
    messageId: "msg_1",
    pinned: false,
    createdAt: 0,
    updatedAt: 0,
  } as unknown as AssistantArtifact;
}

function renderBody(payload: Record<string, unknown>, proposals: unknown[] = []) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
  });
  client.setQueryData(assistant.proposals(THREAD).queryKey, {
    results: proposals,
    count: proposals.length,
  });
  return render(
    <QueryClientProvider client={client}>
      <DeskConnectionDraftBody artifact={artifactOf(payload)} />
    </QueryClientProvider>,
  );
}

function applyProposal(digest: string, status: string) {
  return {
    id: "apr_1",
    toolName: "edi_apply_connection_draft",
    arguments: { documentDigest: digest },
    autonomyTier: "Propose",
    status,
  };
}

beforeEach(() => {
  useDeskStore.setState({ asks: {} });
});

describe("DeskConnectionDraftBody", () => {
  it("draws each setting with the page it was read from and what still needs a look", () => {
    renderBody(draftPayload());

    expect(screen.getByRole("heading", { name: "Acme Retail Group" })).toBeInTheDocument();
    expect(screen.getByText("A new trading partner")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Page 1: what the guide says" })).toBeInTheDocument();
    expect(
      screen.getByText("Not found on its page · The quote is not on page 2"),
    ).toBeInTheDocument();
    expect(screen.getByText("MS1, element 1")).toBeInTheDocument();
    expect(screen.getByText("Appendix B is missing")).toBeInTheDocument();
    expect(screen.getByText("guides/214.lading")).toBeInTheDocument();
    expect(screen.getByText("214 to the partner")).toBeInTheDocument();
    expect(screen.getByText("The envelope matches the drafted profile.")).toBeInTheDocument();
  });

  it("asks the agent to draft again with the reading the person picks for a held value", () => {
    renderBody(draftPayload());

    fireEvent.click(screen.getByRole("button", { name: "Use America/Chicago" }));

    expect(useDeskStore.getState().asks[THREAD]).toBe(
      'For the EDI connection drafted from acme-guide.pdf, use "America/Chicago" for Time zone (page 2) and draft it again.',
    );
    expect(screen.getByRole("button", { name: "Use America/New_York" })).toBeDisabled();
  });

  it("names the credentials to enter and marks the ones the connection cannot run without", () => {
    renderBody(draftPayload());

    const required = screen.getByText("AS2 private key").closest("li") as HTMLElement;
    expect(within(required).getByText("Required before it can be turned on")).toBeInTheDocument();
    const optional = screen.getByText("Partner signing certificate").closest("li") as HTMLElement;
    expect(within(optional).getByText("If the partner uses it")).toBeInTheDocument();
  });

  it("cannot be proposed while something is left to settle", () => {
    renderBody(draftPayload());

    expect(screen.getByRole("button", { name: "Propose saving it" })).toBeDisabled();
    expect(screen.getByText("Decide Time zone")).toBeInTheDocument();
  });

  it("asks for the proposal of a ready draft with the digest the person reviewed", () => {
    renderBody(draftPayload({ ready: true, blockers: [], decisions: [] }), [
      applyProposal("sha256:0000", "Pending"),
    ]);

    fireEvent.click(screen.getByRole("button", { name: "Propose saving it" }));

    expect(useDeskStore.getState().asks[THREAD]).toBe(
      `Propose saving the EDI connection drafted from acme-guide.pdf, with document digest ${DIGEST}.`,
    );
  });

  it("follows the proposal made for this draft to its approval and its save", () => {
    const ready = draftPayload({ ready: true, blockers: [], decisions: [] });
    const { unmount } = renderBody(ready, [applyProposal(DIGEST, "Pending")]);
    expect(screen.getByText("Waiting on your approval")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Propose saving it" })).not.toBeInTheDocument();
    unmount();

    renderBody(ready, [applyProposal(DIGEST, "Executed")]);
    expect(screen.getByText("Saved")).toBeInTheDocument();
  });

  it("asks which partner the guide is for when it matches more than one", () => {
    renderBody(
      draftPayload({
        partner: {
          match: "Ambiguous",
          change: "None",
          fields: [],
          candidates: [
            { partnerId: "edip_1", code: "ACME", name: "Acme Retail", reasons: ["Same ISA id"] },
            { partnerId: "edip_2", code: "ACMEW", name: "Acme West", reasons: ["Same name"] },
          ],
        },
      }),
    );

    fireEvent.click(screen.getByRole("button", { name: "It is ACMEW" }));

    expect(useDeskStore.getState().asks[THREAD]).toBe(
      "The guide acme-guide.pdf is for the existing trading partner Acme West (ACMEW, id edip_2). Draft it again for that partner.",
    );
  });
});

describe("ConnectionDraftSkeleton", () => {
  it("announces itself once and draws the card's own outline", () => {
    const { container } = render(<ConnectionDraftSkeleton />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading the EDI connection draft…");
    expect(container.querySelectorAll(".dk-ecd-v").length).toBeGreaterThan(0);
    expect(container.querySelectorAll(".dk-ecd-prof")).toHaveLength(2);
  });
});
