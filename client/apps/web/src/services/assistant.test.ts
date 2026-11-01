import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

import { AssistantService, assistantTranscriptUrl } from "./assistant";

const requestGraphQLMock = vi.mocked(requestGraphQL);

function answer(operations: Record<string, unknown>) {
  requestGraphQLMock.mockImplementation((async (params: { operationName?: string }) => {
    const name = params.operationName ?? "";
    if (!(name in operations)) {
      throw new Error(`unexpected operation ${name}`);
    }
    return operations[name];
  }) as typeof requestGraphQL);
}

function variablesOf(operationName: string) {
  const call = requestGraphQLMock.mock.calls.find(
    ([params]) => (params as { operationName?: string }).operationName === operationName,
  );
  return (call?.[0] as { variables?: Record<string, unknown> } | undefined)?.variables;
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("AssistantService.listProviders", () => {
  it("returns the providers the organization has configured", async () => {
    answer({
      AssistantProviders: {
        assistantProviders: {
          results: [
            {
              id: "aiprv_01M2ZX9KNWNT4TN19C45ZT63WG",
              name: "OpenRouter",
              kind: "OpenAIChat",
              model: "nvidia/nemotron-3-super-120b-a12b",
              trusted: true,
            },
            {
              id: "aiprv_01M301NZM073J2744DF4VXNZ6Z",
              name: "Minimax",
              kind: "OpenAIChat",
              model: "minimax/minimax-m3",
              trusted: true,
            },
          ],
        },
      },
    });

    const providers = await new AssistantService().listProviders();

    expect(providers).toHaveLength(2);
    expect(providers.map((provider) => provider.model)).toEqual([
      "nvidia/nemotron-3-super-120b-a12b",
      "minimax/minimax-m3",
    ]);
  });

  it("returns an array, not a promise property, when the list is empty", async () => {
    answer({ AssistantProviders: { assistantProviders: { results: [] } } });
    await expect(new AssistantService().listProviders()).resolves.toEqual([]);
  });
});

/**
 * The transcript is a file the server names and serves, so the client needs
 * only the address: a same-origin link carries the session with it.
 */
describe("assistantTranscriptUrl", () => {
  it("points at the thread's transcript on the API", () => {
    expect(assistantTranscriptUrl("athr_01M3034Q2N7JD99RA1D8DGH1ZF")).toMatch(
      /\/assistant\/threads\/athr_01M3034Q2N7JD99RA1D8DGH1ZF\/transcript\/$/u,
    );
  });

  it("escapes an id that is not a plain identifier", () => {
    expect(assistantTranscriptUrl("a/b")).toContain("/threads/a%2Fb/transcript/");
  });
});

/**
 * A person's rewording of a draft is saved on the proposal behind it, so it
 * survives a reload; saving is not deciding.
 */
describe("AssistantService.saveProposalEdits", () => {
  it("saves the changed values on the thread's proposal", async () => {
    answer({
      SaveAssistantProposalEdits: {
        saveAssistantProposalEdits: {
          proposalId: "ap_1",
          pendingModifications: { subject: "PO needed" },
        },
      },
    });

    const edits = await new AssistantService().saveProposalEdits("athr_1", "ap_1", {
      subject: "PO needed",
    });

    expect(variablesOf("SaveAssistantProposalEdits")).toEqual({
      threadId: "athr_1",
      proposalId: "ap_1",
      modifications: { subject: "PO needed" },
    });
    expect(edits.pendingModifications).toEqual({ subject: "PO needed" });
  });

  it("reads a cleared edit as nothing saved", async () => {
    answer({
      SaveAssistantProposalEdits: {
        saveAssistantProposalEdits: { proposalId: "ap_1", pendingModifications: null },
      },
    });

    const edits = await new AssistantService().saveProposalEdits("athr_1", "ap_1", {});

    expect(edits.pendingModifications).toBeNull();
  });
});

describe("AssistantService.startTurn", () => {
  it("sends the page under the message as the server reads it", async () => {
    answer({
      StartAssistantTurn: {
        startAssistantTurn: {
          turnId: "atrn_1",
          threadId: "athr_1",
          streamUrl: "/api/v1/assistant/turns/atrn_1/stream/",
          status: "Pending",
        },
      },
    });

    const turn = await new AssistantService().startTurn("athr_1", "Where is PRO 1001?", {
      context: {
        path: "/shipments",
        entityType: "",
        entityId: "",
        title: "Shipments",
        view: null,
        draft: null,
      },
      surface: "Desk",
      mentions: [{ type: "shipment", id: "shp_1", label: "PRO 1001" }],
    });

    expect(turn.turnId).toBe("atrn_1");
    expect(variablesOf("StartAssistantTurn")).toEqual({
      threadId: "athr_1",
      input: {
        content: "Where is PRO 1001?",
        context: {
          path: "/shipments",
          entityType: "",
          entityId: "",
          title: "Shipments",
          view: null,
          draft: null,
        },
        surface: "Desk",
        providerId: "",
        attachmentDocumentIds: [],
        mentions: [{ type: "shipment", id: "shp_1", label: "PRO 1001" }],
        directedAgentId: null,
      },
    });
  });
});

describe("AssistantService.listActiveTurns", () => {
  it("lists the replies in progress under the items the panel reads", async () => {
    answer({
      AssistantLiveTurns: {
        assistantLiveTurns: [
          {
            turnId: "atrn_1",
            threadId: "athr_1",
            threadTitle: "Late loads this week",
            origin: "DecisionFollowUp",
            startedAt: 1_700_000_000,
          },
        ],
      },
    });

    const live = await new AssistantService().listActiveTurns();

    expect(live.items).toHaveLength(1);
    expect(live.items[0]?.threadTitle).toBe("Late loads this week");
  });
});

describe("AssistantService.activeTurn", () => {
  it("is null when the conversation is producing nothing", async () => {
    answer({ AssistantActiveTurn: { assistantActiveTurn: null } });

    await expect(new AssistantService().activeTurn("athr_1")).resolves.toBeNull();
  });
});

describe("AssistantService.requestMore", () => {
  it("answers how many administrators were asked", async () => {
    answer({ RequestAssistantAllowance: { requestAssistantAllowance: 2 } });

    await expect(new AssistantService().requestMore("athr_1", "budget")).resolves.toEqual({
      sent: 2,
    });
    expect(variablesOf("RequestAssistantAllowance")).toEqual({
      threadId: "athr_1",
      kind: "budget",
    });
  });
});
