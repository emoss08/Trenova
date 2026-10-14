import type { AgentProfile } from "@/lib/graphql/agent-profile";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationProfile } from "../organization-profile";

const api = vi.hoisted(() => ({
  fetchAgentProfile: vi.fn(),
  fetchAgentProfileHistory: vi.fn(),
  addAgentProfileGuidance: vi.fn(),
  retireAgentProfileGuidance: vi.fn(),
  regenerateAgentProfile: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-profile", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-profile")>()),
  ...api,
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

function profile(overrides: Partial<AgentProfile> = {}): AgentProfile {
  return {
    subject: "Organization",
    current: {
      id: "aprf_1",
      subject: "Organization",
      status: "Current",
      trigger: "Scheduled",
      content: "Acme Freight runs a regional fleet out of Dallas.",
      facts: [{ key: "fleet.tractors", text: "About 45 tractors." }],
      guidanceIds: ["aprg_1"],
      model: "model",
      errorMessage: null,
      checkedAt: 1_790_000_000,
      createdAt: 1_789_990_000,
    },
    latestFailure: null,
    guidance: [
      {
        id: "aprg_1",
        subject: "Organization",
        content: "The Laredo lane is seasonal.",
        status: "Active",
        source: "Agent",
        createdById: "usr_1",
        createdByName: "Avery Lane",
        threadId: "athr_1",
        retiredAt: null,
        createdAt: 1_789_000_000,
      },
    ],
    ...overrides,
  };
}

function renderProfile(permissions: { canCreate: boolean; canUpdate: boolean }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <OrganizationProfile {...permissions} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.fetchAgentProfile.mockResolvedValue(profile());
  api.addAgentProfileGuidance.mockResolvedValue(profile().guidance[0]);
  api.retireAgentProfileGuidance.mockResolvedValue(profile().guidance[0]);
  api.regenerateAgentProfile.mockResolvedValue(undefined);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("OrganizationProfile", () => {
  it("shows what agents are told, what it was written from, and who gave each line", async () => {
    renderProfile({ canCreate: true, canUpdate: true });

    expect(
      await screen.findByText("Acme Freight runs a regional fleet out of Dallas."),
    ).toBeInTheDocument();
    expect(screen.getByText("About 45 tractors.")).toBeInTheDocument();
    expect(screen.getByText("The Laredo lane is seasonal.")).toBeInTheDocument();
    expect(
      screen.getByText(/From Avery Lane in a conversation, written down by an agent/),
    ).toBeInTheDocument();
  });

  it("offers no change to someone who may only read", async () => {
    renderProfile({ canCreate: false, canUpdate: false });

    await screen.findByText("Acme Freight runs a regional fleet out of Dallas.");
    expect(screen.queryByRole("button", { name: "Retire" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Write it again/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("form", { name: "Add guidance" })).not.toBeInTheDocument();
  });

  it("adds guidance for the organization and clears the box", async () => {
    const user = userEvent.setup();
    renderProfile({ canCreate: true, canUpdate: true });

    const box = await screen.findByRole("textbox", { name: "Guidance" });
    await user.type(box, "We stopped running reefers in June.");
    await user.click(screen.getByRole("button", { name: /Add guidance/ }));

    await waitFor(() =>
      expect(api.addAgentProfileGuidance).toHaveBeenCalledWith(
        "organization",
        "We stopped running reefers in June.",
      ),
    );
    await waitFor(() => expect(box).toHaveValue(""));
  });

  it("retires a line and asks for the profile again", async () => {
    const user = userEvent.setup();
    renderProfile({ canCreate: true, canUpdate: true });

    await user.click(await screen.findByRole("button", { name: "Retire" }));
    await waitFor(() =>
      expect(api.retireAgentProfileGuidance).toHaveBeenCalledWith("organization", "aprg_1"),
    );

    await user.click(screen.getByRole("button", { name: /Write it again/ }));
    await waitFor(() => expect(api.regenerateAgentProfile).toHaveBeenCalledWith("organization"));
  });

  it("says a failed attempt left the current version in place", async () => {
    api.fetchAgentProfile.mockResolvedValue(
      profile({
        latestFailure: {
          ...profile().current!,
          id: "aprf_2",
          status: "Failed",
          content: null,
          errorMessage: "the profile stated figures the facts do not hold: 4,000",
        },
      }),
    );
    renderProfile({ canCreate: true, canUpdate: true });

    expect(
      await screen.findByText(
        /agents are still told the version above: the profile stated figures/,
      ),
    ).toBeInTheDocument();
  });

  it("says when there is no profile yet", async () => {
    api.fetchAgentProfile.mockResolvedValue(profile({ current: null, guidance: [] }));
    renderProfile({ canCreate: true, canUpdate: true });

    expect(await screen.findByText(/No profile yet/)).toBeInTheDocument();
  });

  it("draws the section's shape while the profile loads, then the profile", async () => {
    let resolve: (value: AgentProfile) => void = () => undefined;
    api.fetchAgentProfile.mockReturnValue(
      new Promise<AgentProfile>((done) => {
        resolve = done;
      }),
    );
    renderProfile({ canCreate: true, canUpdate: true });

    const section = screen.getByRole("region", { name: "Organization profile" });
    expect(section).toHaveAttribute("aria-busy", "true");
    expect(section.querySelectorAll('[data-slot="skeleton"]').length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: /Write it again/ })).not.toBeInTheDocument();

    resolve(profile());
    expect(
      await screen.findByText("Acme Freight runs a regional fleet out of Dallas."),
    ).toBeInTheDocument();
    expect(screen.getByRole("region", { name: "Organization profile" })).not.toHaveAttribute(
      "aria-busy",
    );
  });

  it("says when the profile could not be loaded instead of showing nothing", async () => {
    api.fetchAgentProfile.mockRejectedValue(new Error("agent memory read is not allowed"));
    renderProfile({ canCreate: true, canUpdate: true });

    expect(await screen.findByRole("alert")).toHaveTextContent("agent memory read is not allowed");
    expect(screen.queryByRole("form", { name: "Add guidance" })).not.toBeInTheDocument();
  });
});
