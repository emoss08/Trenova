import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentExtensionService } from "./agent-extension";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

afterEach(() => {
  vi.clearAllMocks();
});

// A portal run as the AgentPortalRunFields fragment answers it: an unset id is empty and
// details nobody gave are an empty object.
const continued = {
  id: "aprn_1",
  siteId: "apst_1",
  siteName: "Acme DC",
  extensionType: "BrowserUse",
  mode: "Read",
  status: "Running",
  instruction: "On Acme DC, find out: Which slots are open?",
  details: {},
  usedSignIn: true,
  answer: "",
  visitedUrls: ["https://portal.acme.com/login"],
  offSiteUrls: [],
  steps: 2,
  costUsd: "0.03",
  failure: "",
  agentRunId: "",
  threadId: "athr_1",
  proposalId: "",
  requestedById: "usr_1",
  hasRecording: true,
  codePrompt: "",
  codeExpiresAt: null,
  codeAttempts: 1,
  startedAt: 1_790_150_400,
  finishedAt: null,
};

describe("AgentExtensionService.sendPortalSignInCode", () => {
  it("sends the code to the run waiting on it and reads the run it continued", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ provideAgentPortalSignInCode: continued });

    const run = await new AgentExtensionService().sendPortalSignInCode("aprn_1", "123 456");

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "ProvideAgentPortalSignInCode",
        variables: { id: "aprn_1", input: { code: "123 456", signedIn: false } },
      }),
    );
    expect(run.status).toBe("Running");
    expect(run.codeAttempts).toBe(1);
    expect(run.codeExpiresAt).toBeNull();
    expect(run.costUsd).toBe(0.03);
  });

  it("reads the IDs the server leaves unset as empty", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ provideAgentPortalSignInCode: continued });

    const run = await new AgentExtensionService().sendPortalSignInCode("aprn_1", "123 456");

    expect(run.agentRunId).toBe("");
    expect(run.proposalId).toBe("");
    expect(run.threadId).toBe("athr_1");
  });
});

describe("AgentExtensionService.confirmPortalSignIn", () => {
  it("says the person signed in themselves, with no code, and reads the run it continued", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ provideAgentPortalSignInCode: continued });

    const run = await new AgentExtensionService().confirmPortalSignIn("aprn_1");

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "ProvideAgentPortalSignInCode",
        variables: { id: "aprn_1", input: { code: "", signedIn: true } },
      }),
    );
    expect(run.status).toBe("Running");
  });
});

describe("AgentExtensionService.portalRunLive", () => {
  it("reads the live browser's address for the run", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      agentPortalRunLive: { liveUrl: "https://live.browser-use.com/s/1" },
    });

    const live = await new AgentExtensionService().portalRunLive("aprn_1");

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({ operationName: "AgentPortalRunLive", variables: { id: "aprn_1" } }),
    );
    expect(live.liveUrl).toBe("https://live.browser-use.com/s/1");
  });
});

describe("AgentExtensionService.portalRun", () => {
  it("reads one run under the caller's signal", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({ agentPortalRun: continued });
    const controller = new AbortController();

    const run = await new AgentExtensionService().portalRun("aprn_1", {
      signal: controller.signal,
    });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "AgentPortalRun",
        variables: { id: "aprn_1" },
        signal: controller.signal,
      }),
    );
    expect(run.siteName).toBe("Acme DC");
  });
});

describe("AgentExtensionService.updateConfig", () => {
  it("sends the settings as key and value pairs", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      updateAgentExtensionConfig: {
        type: "Exa",
        enabled: true,
        availability: "AllAgents",
        fields: [{ key: "apiKey", value: "", hasValue: true }],
        spec: [],
        version: 2,
        updatedAt: 1_790_150_400,
      },
    });

    const config = await new AgentExtensionService().updateConfig("Exa", {
      enabled: true,
      availability: "AllAgents",
      configuration: { apiKey: "exa-secret" },
      version: 1,
    });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "UpdateAgentExtensionConfig",
        variables: {
          type: "Exa",
          input: {
            enabled: true,
            availability: "AllAgents",
            configuration: [{ key: "apiKey", value: "exa-secret" }],
            version: 1,
          },
        },
      }),
    );
    expect(config.fields).toEqual([{ key: "apiKey", value: "", hasValue: true }]);
  });
});

describe("AgentExtensionService.listWebWatches", () => {
  it("leaves out the filters the caller did not set", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      agentWebWatches: { items: [], nextBefore: 0 },
    });

    const page = await new AgentExtensionService().listWebWatches({ limit: 50 });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "AgentWebWatches",
        variables: { status: null, before: null, limit: 50 },
      }),
    );
    expect(page.items).toEqual([]);
  });
});
