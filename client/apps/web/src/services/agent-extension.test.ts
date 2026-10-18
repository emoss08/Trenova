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

describe("AgentExtensionService.analysisRun", () => {
  const stored = {
    id: "aanr_1",
    extensionType: "E2B",
    status: "Succeeded",
    title: "Lane margin",
    purpose: "Margin by lane for Q3",
    code: "print(1)",
    datasets: null,
    stdout: "1\n",
    stderr: "",
    results: null,
    tables: null,
    figures: null,
    layout: null,
    errorName: "",
    errorValue: "",
    traceback: "",
    chartCount: 0,
    failure: "",
    durationMs: 900,
    costUsd: "0.0002",
    agentRunId: null,
    threadId: null,
    requestedById: null,
    startedAt: 1_790_000_000,
    finishedAt: 1_790_000_001,
  };

  it("reads a run with no agent run, thread or requester as empty IDs", async () => {
    vi.mocked(api.get).mockResolvedValue(stored);

    const run = await new AgentExtensionService().analysisRun("aanr_1");

    expect(api.get).toHaveBeenCalledWith("/agent-analysis/runs/aanr_1/", { signal: undefined });
    expect(run.agentRunId).toBe("");
    expect(run.threadId).toBe("");
    expect(run.requestedById).toBe("");
  });
});
