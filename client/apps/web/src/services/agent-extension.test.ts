import { api } from "@trenova/shared/lib/api";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AgentExtensionService } from "./agent-extension";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

afterEach(() => {
  vi.clearAllMocks();
});

const continued = {
  id: "aprn_1",
  siteId: "apst_1",
  siteName: "Acme DC",
  extensionType: "BrowserUse",
  mode: "Read",
  status: "Running",
  instruction: "On Acme DC, find out: Which slots are open?",
  details: null,
  usedSignIn: true,
  answer: "",
  visitedUrls: ["https://portal.acme.com/login"],
  offSiteUrls: [],
  steps: 2,
  costUsd: "0.03",
  failure: "",
  agentRunId: null,
  threadId: "athr_1",
  proposalId: null,
  requestedById: "usr_1",
  hasRecording: true,
  codePrompt: "",
  codeExpiresAt: null,
  codeAttempts: 1,
  startedAt: 1_790_150_400,
  finishedAt: null,
};

describe("AgentExtensionService.sendPortalSignInCode", () => {
  it("posts the code to the run waiting on it and reads the run it continued", async () => {
    vi.mocked(api.post).mockResolvedValue(continued);

    const run = await new AgentExtensionService().sendPortalSignInCode("aprn_1", "123 456");

    expect(api.post).toHaveBeenCalledWith("/agent-portals/runs/aprn_1/sign-in-code/", {
      code: "123 456",
    });
    expect(run.status).toBe("Running");
    expect(run.codeAttempts).toBe(1);
    expect(run.codeExpiresAt).toBeNull();
  });

  it("reads the IDs the server leaves unset as empty", async () => {
    vi.mocked(api.post).mockResolvedValue(continued);

    const run = await new AgentExtensionService().sendPortalSignInCode("aprn_1", "123 456");

    expect(run.agentRunId).toBe("");
    expect(run.proposalId).toBe("");
    expect(run.threadId).toBe("athr_1");
  });
});

describe("AgentExtensionService.confirmPortalSignIn", () => {
  it("says the person signed in themselves, with no code, and reads the run it continued", async () => {
    vi.mocked(api.post).mockResolvedValue(continued);

    const run = await new AgentExtensionService().confirmPortalSignIn("aprn_1");

    expect(api.post).toHaveBeenCalledWith("/agent-portals/runs/aprn_1/sign-in-code/", {
      signedIn: true,
    });
    expect(run.status).toBe("Running");
  });
});

describe("AgentExtensionService.portalRunLive", () => {
  it("reads the live browser's address for the run", async () => {
    vi.mocked(api.get).mockResolvedValue({ liveUrl: "https://live.browser-use.com/s/1" });

    const live = await new AgentExtensionService().portalRunLive("aprn_1");

    expect(api.get).toHaveBeenCalledWith("/agent-portals/runs/aprn_1/live/");
    expect(live.liveUrl).toBe("https://live.browser-use.com/s/1");
  });
});

describe("AgentExtensionService.portalRun", () => {
  it("reads one run under the caller's signal", async () => {
    vi.mocked(api.get).mockResolvedValue(continued);
    const controller = new AbortController();

    const run = await new AgentExtensionService().portalRun("aprn_1", {
      signal: controller.signal,
    });

    expect(api.get).toHaveBeenCalledWith("/agent-portals/runs/aprn_1/", {
      signal: controller.signal,
    });
    expect(run.siteName).toBe("Acme DC");
  });
});
