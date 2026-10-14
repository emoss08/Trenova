import type { AgentProfile } from "@/lib/graphql/agent-profile";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskProfileCard } from "../desk-profile-card";

const api = vi.hoisted(() => ({
  fetchAgentProfile: vi.fn(),
  addAgentProfileGuidance: vi.fn(),
  retireAgentProfileGuidance: vi.fn(),
  regenerateAgentProfile: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-profile", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-profile")>()),
  ...api,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

const line = {
  id: "aprg_9",
  subject: "User" as const,
  content: "I only handle the Midwest accounts.",
  status: "Active" as const,
  source: "Person" as const,
  createdById: "usr_1",
  createdByName: "Avery Lane",
  threadId: null,
  retiredAt: null,
  createdAt: 1_789_000_000,
};

function mine(overrides: Partial<AgentProfile> = {}): AgentProfile {
  return {
    subject: "User",
    current: null,
    latestFailure: null,
    guidance: [],
    ...overrides,
  };
}

function renderCard() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <DeskProfileCard />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.fetchAgentProfile.mockResolvedValue(mine());
  api.addAgentProfileGuidance.mockResolvedValue(line);
  api.retireAgentProfileGuidance.mockResolvedValue({ ...line, status: "Retired" });
  api.regenerateAgentProfile.mockResolvedValue(undefined);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("DeskProfileCard", () => {
  it("reads the person's own profile, never the organization's", async () => {
    renderCard();

    expect(await screen.findByText(/Nothing yet/)).toBeInTheDocument();
    expect(api.fetchAgentProfile).toHaveBeenCalledWith("mine", expect.anything());
  });

  it("shows the write-up and the lines the person gave", async () => {
    api.fetchAgentProfile.mockResolvedValue(
      mine({
        current: {
          id: "aprf_3",
          subject: "User",
          status: "Current",
          trigger: "Guidance",
          content: "This person handles the Midwest accounts' billing.",
          facts: [],
          guidanceIds: ["aprg_9"],
          model: null,
          errorMessage: null,
          checkedAt: 1_790_000_000,
          createdAt: 1_790_000_000,
        },
        guidance: [line],
      }),
    );
    renderCard();

    expect(
      await screen.findByText("This person handles the Midwest accounts' billing."),
    ).toBeInTheDocument();
    expect(screen.getByText("I only handle the Midwest accounts.")).toBeInTheDocument();
  });

  it("adds a line about the person's work and retires one", async () => {
    const user = userEvent.setup();
    api.fetchAgentProfile.mockResolvedValue(mine({ guidance: [line] }));
    renderCard();

    const box = await screen.findByRole("textbox", { name: "Tell agents about your work" });
    await user.type(box, "I'm in billing now.");
    await user.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(() =>
      expect(api.addAgentProfileGuidance).toHaveBeenCalledWith("mine", "I'm in billing now."),
    );

    await user.click(
      screen.getByRole("button", { name: "Retire “I only handle the Midwest accounts.”" }),
    );
    await waitFor(() =>
      expect(api.retireAgentProfileGuidance).toHaveBeenCalledWith("mine", "aprg_9"),
    );
  });

  it("draws the card's shape while the profile loads, then the card", async () => {
    let resolve: (value: AgentProfile) => void = () => undefined;
    api.fetchAgentProfile.mockReturnValue(
      new Promise<AgentProfile>((done) => {
        resolve = done;
      }),
    );
    renderCard();

    const card = screen.getByRole("region", { name: "What agents know about your work" });
    expect(card).toHaveAttribute("aria-busy", "true");
    expect(card.querySelectorAll(".dk-sk").length).toBeGreaterThan(0);
    expect(
      screen.queryByRole("textbox", { name: "Tell agents about your work" }),
    ).not.toBeInTheDocument();

    resolve(mine());
    expect(await screen.findByText(/Nothing yet/)).toBeInTheDocument();
    expect(
      screen.getByRole("region", { name: "What agents know about your work" }),
    ).not.toHaveAttribute("aria-busy");
  });

  it("says when the profile could not be loaded instead of showing nothing", async () => {
    api.fetchAgentProfile.mockRejectedValue(new Error("assistant access is required"));
    renderCard();

    expect(await screen.findByRole("alert")).toHaveTextContent("assistant access is required");
    expect(
      screen.queryByRole("textbox", { name: "Tell agents about your work" }),
    ).not.toBeInTheDocument();
  });
});
