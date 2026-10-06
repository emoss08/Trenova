import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { CurrentSession, StaffProfile } from "../../../types/support-access";
import { SupportAppBanner } from "../support-app-banner";

const mocks = vi.hoisted(() => ({
  staffProfile: vi.fn(),
  currentSession: vi.fn(),
  assign: vi.fn(),
  isCloud: true,
}));

vi.mock("../../../services/support-access", () => ({
  supportAccessService: {
    staffProfile: mocks.staffProfile,
    currentSession: mocks.currentSession,
  },
}));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({ isCloud: mocks.isCloud }),
}));

const NOW = Math.floor(Date.now() / 1000);

const STAFF: StaffProfile = {
  isStaff: true,
  role: "support",
  mfaEnrolled: true,
  sessionVerified: true,
  openSessions: [],
  maxSessionHours: 4,
  elevationMinutes: 30,
};

const ACTIVE: CurrentSession = {
  active: true,
  endedReason: "",
  session: {
    id: "sps_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    organizationName: "Acme Freight",
    staffName: "Jordan Lee",
    mode: "read_only",
    grantMode: "read_only",
    reason: "Customer reported a missing invoice",
    ticketReference: "",
    startedAt: NOW,
    expiresAt: NOW + 3600,
    elevatedUntil: null,
    canElevate: false,
    serverTime: NOW,
  },
};

function renderBanner(ui: ReactNode, client: QueryClient) {
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={["/shipments"]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

function newClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

describe("SupportAppBanner", () => {
  beforeEach(() => {
    mocks.staffProfile.mockReset();
    mocks.currentSession.mockReset();
    mocks.assign.mockReset();
    mocks.isCloud = true;
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

  it("shows nothing to someone who is not platform staff and never asks about sessions", async () => {
    mocks.staffProfile.mockResolvedValue({ ...STAFF, isStaff: false });
    const { container } = renderBanner(<SupportAppBanner />, newClient());

    await waitFor(() => expect(mocks.staffProfile).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
    expect(mocks.currentSession).not.toHaveBeenCalled();
  });

  it("asks nothing on a self-hosted install", () => {
    mocks.isCloud = false;
    const { container } = renderBanner(<SupportAppBanner />, newClient());

    expect(container).toBeEmptyDOMElement();
    expect(mocks.staffProfile).not.toHaveBeenCalled();
  });

  it("links the support console for staff in their own organization", async () => {
    mocks.staffProfile.mockResolvedValue(STAFF);
    mocks.currentSession.mockResolvedValue({ active: false, session: null, endedReason: "" });
    renderBanner(<SupportAppBanner />, newClient());

    expect(await screen.findByTestId("staff-strip")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Support console" })).toHaveAttribute(
      "href",
      "/support-console",
    );
  });

  it("shows the session banner inside a support session", async () => {
    mocks.staffProfile.mockResolvedValue(STAFF);
    mocks.currentSession.mockResolvedValue(ACTIVE);
    renderBanner(<SupportAppBanner />, newClient());

    expect(await screen.findByTestId("support-session-banner")).toHaveTextContent("Acme Freight");
  });

  it("takes the person out when the session ends underneath them", async () => {
    mocks.staffProfile.mockResolvedValue(STAFF);
    mocks.currentSession
      .mockResolvedValueOnce(ACTIVE)
      .mockResolvedValue({ active: false, session: null, endedReason: "grant_revoked" });
    const client = newClient();
    renderBanner(<SupportAppBanner />, client);

    await screen.findByTestId("support-session-banner");
    await client.refetchQueries();

    await waitFor(() =>
      expect(mocks.assign).toHaveBeenCalledWith("/support-console?ended=grant_revoked"),
    );
  });
});
