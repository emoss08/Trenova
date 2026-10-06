import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { formatRemaining, secondsUntil } from "../../../lib/support-access";
import type { SessionView } from "../../../types/support-access";
import { SupportSessionBanner, sessionEndedPath } from "../support-session-banner";

const mocks = vi.hoisted(() => ({
  endSession: vi.fn(),
  dropElevation: vi.fn(),
  elevate: vi.fn(),
  assign: vi.fn(),
}));

vi.mock("../../../services/support-access", () => ({
  supportAccessService: {
    endSession: mocks.endSession,
    dropElevation: mocks.dropElevation,
    elevate: mocks.elevate,
  },
}));

const NOW = Math.floor(Date.now() / 1000);

function session(overrides: Partial<SessionView> = {}): SessionView {
  return {
    id: "sps_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    organizationName: "Acme Freight",
    staffName: "Jordan Lee",
    mode: "read_only",
    grantMode: "read_write",
    reason: "Customer reported a missing invoice",
    ticketReference: "SUP-1",
    startedAt: NOW,
    expiresAt: NOW + 41 * 60,
    elevatedUntil: null,
    canElevate: true,
    serverTime: NOW,
    ...overrides,
  };
}

function renderBanner(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("SupportSessionBanner", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
    vi.stubGlobal("location", {
      origin: "http://localhost",
      href: "http://localhost/",
      pathname: "/",
      search: "",
      assign: mocks.assign,
    });
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("says unmistakably that this is a read-only support session and when it ends", () => {
    renderBanner(<SupportSessionBanner session={session()} fetchedAt={NOW} elevationMinutes={30} />);

    const banner = screen.getByTestId("support-session-banner");
    expect(banner).toHaveAttribute("data-mode", "read_only");
    expect(banner).toHaveTextContent("Trenova support session");
    expect(banner).toHaveTextContent("Acme Freight");
    expect(banner).toHaveTextContent("Read-only");
    expect(banner).toHaveTextContent("Expires in 41m");
    expect(screen.getByRole("button", { name: "Elevate to write" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Exit" })).toBeInTheDocument();
  });

  it("offers no elevation when the organization allowed read-only access", () => {
    renderBanner(
      <SupportSessionBanner
        session={session({ grantMode: "read_only", canElevate: false })}
        fetchedAt={NOW}
        elevationMinutes={30}
      />,
    );

    expect(screen.queryByRole("button", { name: "Elevate to write" })).not.toBeInTheDocument();
  });

  it("shows write access, its countdown and the way back to read-only", async () => {
    mocks.dropElevation.mockResolvedValue(session());
    const user = userEvent.setup();
    renderBanner(
      <SupportSessionBanner
        session={session({ mode: "read_write", elevatedUntil: NOW + 30 * 60 })}
        fetchedAt={NOW}
        elevationMinutes={30}
      />,
    );

    const banner = screen.getByTestId("support-session-banner");
    expect(banner).toHaveAttribute("data-mode", "read_write");
    expect(banner).toHaveTextContent("Read-write");
    expect(banner).toHaveTextContent("Write access ends in 30m");

    await user.click(screen.getByRole("button", { name: "Return to read-only" }));
    await waitFor(() => expect(mocks.dropElevation).toHaveBeenCalled());
  });

  it("ends the session on the server and returns to the console on exit", async () => {
    mocks.endSession.mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderBanner(<SupportSessionBanner session={session()} fetchedAt={NOW} elevationMinutes={30} />);

    await user.click(screen.getByRole("button", { name: "Exit" }));

    await waitFor(() => expect(mocks.assign).toHaveBeenCalledWith("/support-console"));
    expect(mocks.endSession).toHaveBeenCalled();
  });

  it("asks for a password, a code, a ticket and a reason before elevating", async () => {
    const user = userEvent.setup();
    renderBanner(<SupportSessionBanner session={session()} fetchedAt={NOW} elevationMinutes={30} />);

    await user.click(screen.getByRole("button", { name: "Elevate to write" }));
    const dialog = await screen.findByRole("dialog");
    const confirm = Array.from(dialog.querySelectorAll("button")).find(
      (button) => button.textContent === "Elevate to write",
    );
    expect(confirm).toBeDefined();
    await user.click(confirm!);

    expect(mocks.elevate).not.toHaveBeenCalled();
    expect(await screen.findByText("Password is required")).toBeInTheDocument();
  });
});

describe("support session time", () => {
  const t = (message: string | null | undefined, ...args: unknown[]) =>
    (message ?? "").replace(/\{(\d)\}/g, (_, index: string) => String(args[Number(index)]));

  it("measures the deadline on the server's clock", () => {
    expect(secondsUntil(1000, 900, 5000, 5030)).toBe(70);
    expect(secondsUntil(1000, 900, 5000, 4990)).toBe(100);
  });

  it("formats what is left coarsely", () => {
    expect(formatRemaining(0, t)).toBe("now");
    expect(formatRemaining(30, t)).toBe("under a minute");
    expect(formatRemaining(41 * 60, t)).toBe("41m");
    expect(formatRemaining(3 * 3600 + 5 * 60, t)).toBe("3h 5m");
    expect(formatRemaining(2 * 86400, t)).toBe("2d");
  });

  it("sends an ended session to the console with its reason", () => {
    expect(sessionEndedPath("grant_revoked")).toBe("/support-console?ended=grant_revoked");
    expect(sessionEndedPath("")).toBe("/support-console?ended=ended");
  });
});
