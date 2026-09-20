import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ get: vi.fn() }));

vi.mock("@trenova/shared/lib/api", () => ({
  api: { get: mocks.get },
  withCsrfHeader: vi.fn(),
}));

import { AssistantService } from "./assistant";

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
