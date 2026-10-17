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
});
