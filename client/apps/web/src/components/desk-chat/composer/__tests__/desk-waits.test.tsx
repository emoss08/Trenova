import type { ConversationWaits } from "@/components/assistant/use-conversation-waits";
import { waitIdOfNote } from "@/components/assistant/use-conversation-waits";
import type { AgentWait } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { api } from "@trenova/shared/lib/api";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskWaitNote } from "../../conversation/desk-wait-note";
import { DeskWaits } from "../desk-waits";

vi.mock("@trenova/shared/lib/api", () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function wait(overrides: Partial<AgentWait>): AgentWait {
  return {
    id: "awt_1",
    kind: "StopArrival",
    description: "Truck 2214 to reach Kroger DC",
    nextStep: "Tell Acme it is at the dock",
    threadId: "athr_1",
    status: "Waiting",
    dueAt: null,
    expiresAt: 1_800_086_400,
    resolvedAt: null,
    outcome: "",
    resumedTurnId: "",
    createdAt: 1_800_000_000,
    ...overrides,
  };
}

function waitsOf(open: AgentWait[]): ConversationWaits {
  return {
    open,
    byId: new Map(open.map((item) => [item.id, item])),
    cancel: vi.fn(async () => undefined),
  };
}

describe("DeskWaits", () => {
  it("shows nothing while nothing waits", () => {
    const { container } = render(<DeskWaits waits={waitsOf([])} timezone="UTC" />);

    expect(container).toBeEmptyDOMElement();
  });

  it("lists each wait in the agent's words, when it gives up, and cancels one", () => {
    const first = wait({});
    const waits = waitsOf([
      first,
      wait({ id: "awt_2", kind: "Time", description: "Call Acme back", dueAt: 1_800_003_600 }),
    ]);
    render(<DeskWaits waits={waits} timezone="UTC" />);

    expect(screen.getByText("Waiting on 2 things")).toBeInTheDocument();
    expect(screen.getByText("Truck 2214 to reach Kroger DC")).toBeInTheDocument();
    expect(screen.getByText("An arrival")).toBeInTheDocument();
    expect(screen.getByText(/^Gives up /u)).toBeInTheDocument();
    expect(screen.getByText(/^Due /u)).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole("button", { name: "Cancel the wait" })[0]);
    expect(waits.cancel).toHaveBeenCalledWith(first);
  });

  it("asks for the sign-in code a portal sent and sends it only to the portal", async () => {
    vi.mocked(api.post).mockResolvedValue({
      id: "aprn_1",
      siteId: "apst_1",
      siteName: "Acme DC",
      extensionType: "BrowserUse",
      mode: "Read",
      status: "Running",
      instruction: "On Acme DC, find out: Which slots are open?",
      usedSignIn: true,
      answer: "",
      visitedUrls: [],
      offSiteUrls: [],
      steps: 2,
      costUsd: "0.03",
      failure: "",
      hasRecording: true,
      codeAttempts: 1,
      startedAt: 1_800_000_000,
    });
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <DeskWaits
          waits={waitsOf([
            wait({
              kind: "PortalRun",
              description: "the task on Acme DC to finish",
              signInCode: {
                portalRunId: "aprn_1",
                prompt: "SIGN-IN CODE NEEDED: the portal texted a code to the phone ending 42.",
                expiresAt: 1_800_000_600,
                attempts: 0,
              },
            }),
          ])}
          timezone="UTC"
        />
      </QueryClientProvider>,
    );

    expect(screen.getByText("Waiting for your sign-in code")).toBeInTheDocument();
    expect(screen.getByText("The portal sent you a sign-in code.")).toBeInTheDocument();
    expect(
      screen.getByText("the portal texted a code to the phone ending 42."),
    ).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Sign-in code"), { target: { value: "123 456" } });
    fireEvent.click(screen.getByRole("button", { name: "Send code" }));

    await waitFor(() =>
      expect(api.post).toHaveBeenCalledWith("/agent-portals/runs/aprn_1/sign-in-code/", {
        code: "123 456",
      }),
    );
  });

  it("shows a file being read as being read, not as due at its next look", () => {
    render(
      <DeskWaits
        waits={waitsOf([
          wait({
            kind: "DocumentRead",
            description: "rate-con.pdf to finish being read",
            dueAt: 1_800_000_060,
          }),
        ])}
        timezone="UTC"
      />,
    );

    expect(screen.getByText("rate-con.pdf to finish being read")).toBeInTheDocument();
    expect(screen.getByText("A file")).toBeInTheDocument();
    expect(screen.getByText("Being read")).toBeInTheDocument();
    expect(screen.queryByText(/^Due /u)).not.toBeInTheDocument();
  });
});

describe("DeskWaitNote", () => {
  it("names the wait the note picked up, from the id the server ends it with", () => {
    expect(waitIdOfNote("…what you did. Wait id: awt_01J9ABC")).toBe("awt_01J9ABC");
    expect(waitIdOfNote("no id here")).toBeNull();
  });

  it("draws the wait it records and what came of it, not the agent's note", () => {
    const met = wait({
      status: "Met",
      outcome: "Arrived at stop stp_1.",
      resolvedAt: 1_800_001_000,
    });
    render(
      <DeskWaitNote
        content="[Notice from the system…] Pick the work up. Wait id: awt_1"
        waits={new Map([[met.id, met]])}
        timezone="UTC"
      />,
    );

    expect(screen.getByText("Picked up after a wait")).toBeInTheDocument();
    expect(screen.getByText("Truck 2214 to reach Kroger DC")).toBeInTheDocument();
    expect(screen.getByText("Arrived at stop stp_1.")).toBeInTheDocument();
    expect(screen.queryByText(/Notice from the system/u)).toBeNull();
  });

  it("says a wait that ran out ran out", () => {
    const ranOut = wait({ status: "TimedOut" });
    render(
      <DeskWaitNote
        content="Wait id: awt_1"
        waits={new Map([[ranOut.id, ranOut]])}
        timezone="UTC"
      />,
    );

    expect(screen.getByText("Picked up after the wait ran out")).toBeInTheDocument();
  });
});
