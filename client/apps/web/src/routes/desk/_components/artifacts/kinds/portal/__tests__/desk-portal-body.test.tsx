import type { PortalRun } from "@/types/agent-extension";
import type { AssistantArtifact } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskPortalBody } from "../desk-portal-body";

const mocks = vi.hoisted(() => ({
  portalRun: vi.fn(),
  portalRunLive: vi.fn(),
  confirmPortalSignIn: vi.fn(),
  sendPortalSignInCode: vi.fn(),
  portalRunRecording: vi.fn(),
}));

vi.mock("@/services/api", () => ({ apiService: { agentExtensionService: mocks } }));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function run(overrides: Partial<PortalRun> = {}): PortalRun {
  return {
    id: "aprn_1",
    siteId: "apst_1",
    siteName: "Acme dock portal",
    extensionType: "BrowserUse",
    mode: "Read",
    status: "Running",
    instruction: "On Acme dock portal, find out: Which slots are open?",
    details: {},
    usedSignIn: true,
    answer: "",
    visitedUrls: ["https://portal.acme.com/login"],
    offSiteUrls: [],
    steps: 2,
    costUsd: 0.03,
    failure: "",
    agentRunId: "",
    threadId: "athr_1",
    proposalId: "",
    requestedById: "usr_1",
    hasRecording: true,
    codePrompt: "",
    codeExpiresAt: null,
    codeAttempts: 0,
    startedAt: 1_790_150_400,
    finishedAt: null,
    ...overrides,
  };
}

function artifact(payload: Record<string, unknown> = {}): AssistantArtifact {
  return {
    id: "aart_1",
    threadId: "athr_1",
    kind: "portal_task",
    status: "Ready",
    title: "Which slots are open?",
    payload: {
      portalRunId: "aprn_1",
      mode: "Read",
      site: "Acme dock portal",
      question: "Which slots are open?",
      awaitingSignInCode: true,
      portalSaid: "SIGN-IN CODE NEEDED: a code was texted to the phone ending 42.",
      ...payload,
    },
  } as unknown as AssistantArtifact;
}

function renderBody(shown: AssistantArtifact) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <DeskPortalBody artifact={shown} />
    </QueryClientProvider>,
  );
}

describe("DeskPortalBody", () => {
  it("offers the code box, the live browser and signing in by hand while the task waits", async () => {
    mocks.portalRun.mockResolvedValue(
      run({
        status: "AwaitingCode",
        codePrompt: "SIGN-IN CODE NEEDED: a code was texted to the phone ending 42.",
        codeExpiresAt: 1_790_151_000,
      }),
    );

    renderBody(artifact());

    expect(await screen.findByText("Needs a code")).toBeTruthy();
    expect(screen.getByText("Looked up on Acme dock portal")).toBeTruthy();
    expect(screen.getByText("Which slots are open?")).toBeTruthy();
    expect(screen.getByLabelText("Sign-in code")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Live browser" })).toBeTruthy();
  });

  it("carries the task on when the person says they signed in themselves", async () => {
    mocks.portalRun.mockResolvedValue(
      run({ status: "AwaitingCode", codeExpiresAt: 1_790_151_000 }),
    );
    mocks.confirmPortalSignIn.mockResolvedValue(run({ status: "Running", codeAttempts: 1 }));

    renderBody(artifact());
    fireEvent.click(await screen.findByRole("button", { name: "I've signed in" }));

    await waitFor(() => expect(mocks.confirmPortalSignIn).toHaveBeenCalledWith("aprn_1"));
    expect(mocks.sendPortalSignInCode).not.toHaveBeenCalled();
  });

  it("reads the live browser's address only when asked", async () => {
    mocks.portalRun.mockResolvedValue(run({ status: "Running" }));
    mocks.portalRunLive.mockResolvedValue({ liveUrl: "https://live.browser-use.com/s/1" });

    renderBody(artifact({ awaitingSignInCode: false, portalSaid: "" }));
    const watch = await screen.findByRole("button", { name: "Watch the browser" });
    expect(mocks.portalRunLive).not.toHaveBeenCalled();

    fireEvent.click(watch);

    expect(await screen.findByText("Open the live browser")).toBeTruthy();
    expect(mocks.portalRunLive).toHaveBeenCalledWith("aprn_1");
  });

  it("shows what the portal said as text from outside the organization once it is done", async () => {
    mocks.portalRun.mockResolvedValue(
      run({
        status: "Succeeded",
        answer: "Open slots: 8:00 AM, 1:30 PM.",
        visitedUrls: ["https://portal.acme.com/login", "https://portal.acme.com/appointments"],
        finishedAt: 1_790_150_500,
      }),
    );

    renderBody(artifact({ awaitingSignInCode: false, portalSaid: "" }));

    expect(await screen.findByText("Open slots: 8:00 AM, 1:30 PM.")).toBeTruthy();
    expect(
      screen.getByText(
        "Read from the portal, outside your organization. Check anything you act on.",
      ),
    ).toBeTruthy();
    expect(screen.getByText("2 pages visited")).toBeTruthy();
    expect(screen.queryByLabelText("Sign-in code")).toBeNull();
    expect(screen.queryByRole("button", { name: "Watch the browser" })).toBeNull();
    expect(screen.getByRole("button", { name: "Recording" })).toBeTruthy();
  });

  it("shows only what the agent was told when the run cannot be read", async () => {
    mocks.portalRun.mockRejectedValue(new Error("Portal run not found"));

    renderBody(
      artifact({
        awaitingSignInCode: false,
        portalSaid: "",
        answer: "Open slots: 8:00 AM.",
        pagesRead: ["https://portal.acme.com/appointments"],
      }),
    );

    expect(
      await screen.findByText(
        "This task could not be read as it stands now. What the agent was told is shown.",
      ),
    ).toBeTruthy();
    expect(screen.getByText("Open slots: 8:00 AM.")).toBeTruthy();
    expect(screen.queryByLabelText("Sign-in code")).toBeNull();
    expect(screen.queryByRole("button", { name: "I've signed in" })).toBeNull();
  });
});
