import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { GrantState, StaffProfile } from "../../types/support-access";
import { SupportAccessPage } from "../admin/support-access/page";
import { SupportConsolePage } from "../support-console/page";

const mocks = vi.hoisted(() => ({
  grantState: vi.fn(),
  createGrant: vi.fn(),
  revokeGrant: vi.fn(),
  staffProfile: vi.fn(),
  grantedOrganizations: vi.fn(),
  startSession: vi.fn(),
  assign: vi.fn(),
}));

vi.mock("../../services/support-access", () => ({
  supportAccessService: {
    grantState: mocks.grantState,
    createGrant: mocks.createGrant,
    revokeGrant: mocks.revokeGrant,
    staffProfile: mocks.staffProfile,
    grantedOrganizations: mocks.grantedOrganizations,
    startSession: mocks.startSession,
  },
}));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({ isCloud: true }),
}));

vi.mock("@/components/navigation/sidebar-layout", () => ({
  PageLayout: ({
    children,
    pageHeaderProps,
  }: {
    children: ReactNode;
    pageHeaderProps: { title: string };
  }) => (
    <main>
      <h1>{pageHeaderProps.title}</h1>
      {children}
    </main>
  ),
}));

const NOW = Math.floor(Date.now() / 1000);

function grantAll(operations: number) {
  usePermissionStore.setState({
    manifest: {
      permissions: Object.fromEntries(
        Object.values(Resource).map((resource) => [resource, operations]),
      ),
      routeAccess: {},
    } as unknown as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
}

function renderPage(ui: ReactNode, path = "/") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>{ui}</MemoryRouter>
    </QueryClientProvider>,
  );
}

function state(overrides: Partial<GrantState> = {}): GrantState {
  return {
    grant: null,
    sessions: [],
    history: [],
    durationHours: [24, 72, 168, 336],
    maxDurationHours: 336,
    accessModes: ["read_only", "read_write"],
    serverTime: NOW,
    ...overrides,
  };
}

describe("SupportAccessPage", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((mock) => mock.mockReset());
    grantAll(0xffff);
  });

  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("allows read-only access for a day by default", async () => {
    mocks.grantState.mockResolvedValue(state());
    mocks.createGrant.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage(<SupportAccessPage />);

    await user.click(await screen.findByRole("button", { name: "Allow access" }));

    await waitFor(() =>
      expect(mocks.createGrant).toHaveBeenCalledWith({
        durationHours: 24,
        accessMode: "read_only",
        note: "",
      }),
    );
  });

  it("shows the active grant and the sessions support opened, and revokes on confirmation", async () => {
    mocks.grantState.mockResolvedValue(
      state({
        grant: {
          id: "sag_1",
          organizationId: "org_1",
          grantedById: "usr_1",
          accessMode: "read_write",
          note: "Look at invoice 4411",
          startsAt: NOW,
          expiresAt: NOW + 72 * 3600,
          revokedAt: null,
          revokedById: "",
          createdAt: NOW,
        },
        sessions: [
          {
            id: "sps_1",
            staffName: "Jordan Lee",
            status: "active",
            mode: "read_only",
            reason: "Customer reported a missing invoice",
            ticketReference: "SUP-9",
            startedAt: NOW,
            expiresAt: NOW + 3600,
            lastSeenAt: NOW,
            elevationCount: 1,
            elevatedUntil: null,
            elevationReason: "",
            endedAt: null,
            endReason: "",
          },
        ],
      }),
    );
    mocks.revokeGrant.mockResolvedValue(undefined);
    const user = userEvent.setup();
    renderPage(<SupportAccessPage />);

    expect(await screen.findByText("Trenova support can access this organization")).toBeVisible();
    expect(screen.getByText("Look at invoice 4411")).toBeInTheDocument();
    const sessions = screen.getByRole("region", { name: "Support sessions" });
    expect(within(sessions).getByText("Jordan Lee")).toBeInTheDocument();
    expect(within(sessions).getByText("In progress")).toBeInTheDocument();
    expect(within(sessions).getByText("SUP-9")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Revoke access" }));
    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("Any support session in progress ends immediately.");
    const confirm = Array.from(dialog.querySelectorAll("button")).find(
      (button) => button.textContent === "Revoke access",
    );
    await user.click(confirm!);

    await waitFor(() => expect(mocks.revokeGrant).toHaveBeenCalled());
  });

  it("does not let someone without organization update change access", async () => {
    grantAll(1);
    mocks.grantState.mockResolvedValue(state());
    renderPage(<SupportAccessPage />);

    expect(await screen.findByRole("button", { name: "Allow access" })).toBeDisabled();
  });
});

const PROFILE: StaffProfile = {
  isStaff: true,
  role: "support",
  mfaEnrolled: true,
  sessionVerified: true,
  openSessions: [],
  maxSessionHours: 4,
  elevationMinutes: 30,
};

describe("SupportConsolePage", () => {
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

  it("turns away someone who is not platform staff", async () => {
    mocks.staffProfile.mockResolvedValue({ ...PROFILE, isStaff: false });
    renderPage(<SupportConsolePage />);

    expect(await screen.findByText("Trenova staff only")).toBeInTheDocument();
    expect(mocks.grantedOrganizations).not.toHaveBeenCalled();
  });

  it("will not open a session without a two-factor sign-in", async () => {
    mocks.staffProfile.mockResolvedValue({ ...PROFILE, sessionVerified: false });
    mocks.grantedOrganizations.mockResolvedValue([
      {
        grantId: "sag_1",
        organizationId: "org_1",
        businessUnitId: "bu_1",
        organizationName: "Acme Freight",
        accessMode: "read_only",
        note: "",
        startsAt: NOW,
        expiresAt: NOW + 3600,
      },
    ]);
    renderPage(<SupportConsolePage />);

    expect(await screen.findByText("Sign in again with your authenticator code")).toBeVisible();
    expect(await screen.findByRole("button", { name: "Open support session" })).toBeDisabled();
  });

  it("opens a session with a reason and enters the organization", async () => {
    mocks.staffProfile.mockResolvedValue(PROFILE);
    mocks.grantedOrganizations.mockResolvedValue([
      {
        grantId: "sag_1",
        organizationId: "org_1",
        businessUnitId: "bu_1",
        organizationName: "Acme Freight",
        accessMode: "read_write",
        note: "Invoice question",
        startsAt: NOW,
        expiresAt: NOW + 3600,
      },
    ]);
    mocks.startSession.mockResolvedValue({});
    const user = userEvent.setup();
    renderPage(<SupportConsolePage />);

    await user.click(await screen.findByRole("button", { name: "Open support session" }));
    const dialog = await screen.findByRole("dialog");
    await user.type(
      within(dialog).getByPlaceholderText("Investigate the missing invoice the customer reported"),
      "Look into the invoice the customer reported missing",
    );
    await user.type(within(dialog).getByPlaceholderText("SUP-1234"), "SUP-77");
    const open = Array.from(dialog.querySelectorAll("button")).find(
      (button) => button.textContent === "Open support session",
    );
    await user.click(open!);

    await waitFor(() =>
      expect(mocks.startSession).toHaveBeenCalledWith({
        organizationId: "org_1",
        businessUnitId: "bu_1",
        reason: "Look into the invoice the customer reported missing",
        ticketReference: "SUP-77",
      }),
    );
    await waitFor(() => expect(mocks.assign).toHaveBeenCalledWith("/"));
  });

  it("says why the last session ended", async () => {
    mocks.staffProfile.mockResolvedValue(PROFILE);
    mocks.grantedOrganizations.mockResolvedValue([]);
    renderPage(<SupportConsolePage />, "/support-console?ended=grant_revoked");

    expect(await screen.findByText("Your Trenova support session has ended")).toBeVisible();
    expect(screen.getByText("Access revoked")).toBeInTheDocument();
  });
});
