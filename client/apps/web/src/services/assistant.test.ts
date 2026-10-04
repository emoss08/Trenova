import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }));

vi.mock("@trenova/shared/lib/api", () => ({
  api: { get: mocks.get, put: mocks.put },
  withCsrfHeader: vi.fn(),
}));

import { AssistantService, assistantTranscriptUrl } from "./assistant";

describe("AssistantService.listProviders", () => {
  beforeEach(() => {
    mocks.get.mockReset();
  });

  it("returns the providers the organization has configured", async () => {
    mocks.get.mockResolvedValue({
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
    });

    const providers = await new AssistantService().listProviders();

    expect(providers).toHaveLength(2);
    expect(providers.map((provider) => provider.model)).toEqual([
      "nvidia/nemotron-3-super-120b-a12b",
      "minimax/minimax-m3",
    ]);
  });

  it("returns an array, not a promise property, when the list is empty", async () => {
    mocks.get.mockResolvedValue({ results: [] });
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
  beforeEach(() => {
    mocks.put.mockReset();
  });

  it("saves the changed values on the thread's proposal", async () => {
    mocks.put.mockResolvedValue({
      proposalId: "ap_1",
      pendingModifications: { subject: "PO needed" },
    });

    const edits = await new AssistantService().saveProposalEdits("athr_1", "ap_1", {
      subject: "PO needed",
    });

    expect(mocks.put).toHaveBeenCalledWith("/assistant/threads/athr_1/proposals/ap_1/edits/", {
      modifications: { subject: "PO needed" },
    });
    expect(edits.pendingModifications).toEqual({ subject: "PO needed" });
  });

  it("reads a cleared edit as nothing saved", async () => {
    mocks.put.mockResolvedValue({ proposalId: "ap_1", pendingModifications: null });

    const edits = await new AssistantService().saveProposalEdits("athr_1", "ap_1", {});

    expect(edits.pendingModifications).toBeNull();
  });
});
