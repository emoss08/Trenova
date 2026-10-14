import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentCustomerTracking from "../shipment-customer-tracking";

const NOW = 1_800_000_000;

const mocks = vi.hoisted(() => ({
  getProfile: vi.fn(),
  listLinks: vi.fn(),
  listNotifications: vi.fn(),
  issueLink: vi.fn(),
  revokeLink: vi.fn(),
  canShare: true,
}));

vi.mock("@/lib/graphql/customer-tracking", () => ({
  getCustomerTrackingProfile: mocks.getProfile,
  listCustomerTrackingLinks: mocks.listLinks,
  listCustomerStatusNotifications: mocks.listNotifications,
  issueCustomerTrackingLink: mocks.issueLink,
  revokeCustomerTrackingLink: mocks.revokeLink,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canShare, isLoading: false }),
}));

const profile = {
  id: "ctp_1",
  customerId: "cus_1",
  linkEnabled: true,
  visibleFields: ["Status", "Windows", "Actuals", "Eta", "References"],
  linkTtlDays: 30,
  notifyPickedUp: true,
  notifyDelayed: true,
  notifyDelivered: false,
  notifyPodReady: false,
  includeLink: true,
  recipients: "shipping@customer.com, ops@customer.com",
  notificationsEnabledAt: NOW - 86_400,
  version: 2,
  updatedAt: NOW - 86_400,
};

function link(overrides: Record<string, unknown> = {}) {
  return {
    id: "ctl_1",
    shipmentId: "shp_1",
    customerId: "cus_1",
    purpose: "Shared",
    recipientEmail: "receiving@customer.com",
    expiresAt: NOW + 86_400,
    revokedAt: null,
    lastViewedAt: NOW - 600,
    viewCount: 3,
    createdById: "usr_1",
    createdAt: NOW - 3600,
    ...overrides,
  };
}

const shipment = {
  id: "shp_1",
  customerId: "cus_1",
  status: "InTransit",
  customer: { name: "Acme Foods" },
} as unknown as Shipment;

function renderTab(subject: Shipment = shipment) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={client}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  }
  return render(<ShipmentCustomerTracking shipment={subject} />, { wrapper: Wrapper });
}

describe("ShipmentCustomerTracking", () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ["Date"] });
    vi.setSystemTime(NOW * 1000);
    mocks.canShare = true;
    mocks.getProfile.mockReset().mockResolvedValue(profile);
    mocks.listLinks.mockReset().mockResolvedValue([]);
    mocks.listNotifications.mockReset().mockResolvedValue([]);
    mocks.issueLink.mockReset();
    mocks.revokeLink.mockReset().mockResolvedValue(1);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("says who the load is shared with and how much they see", async () => {
    renderTab();

    expect(await screen.findByText("Acme Foods")).toBeDefined();
    expect(await screen.findByText("Sees 5 of 13 details")).toBeDefined();
    expect(screen.getByText("Links on · 30 days")).toBeDefined();
    expect(screen.getByRole("button", { name: "Preview" })).toBeDefined();
    expect(screen.getByRole("link", { name: /Settings/ }).getAttribute("href")).toContain(
      "panelEntityId=cus_1",
    );
  });

  it("lists live links and folds away the ones that no longer open", async () => {
    mocks.listLinks.mockResolvedValue([
      link(),
      link({ id: "ctl_2", recipientEmail: null, viewCount: 0, lastViewedAt: null }),
      link({ id: "ctl_3", recipientEmail: "old@customer.com", expiresAt: NOW - 60 }),
    ]);
    renderTab();

    expect(await screen.findByText("receiving@customer.com")).toBeDefined();
    expect(screen.getByText("Copied link")).toBeDefined();
    expect(screen.getByText("2 live")).toBeDefined();
    expect(screen.getByText("1 expired or revoked link")).toBeDefined();
    expect(screen.queryByText("old@customer.com")).toBeNull();
    expect(screen.getByRole("button", { name: "Revoke all" })).toBeDefined();
  });

  it("revokes one link from its row", async () => {
    mocks.listLinks.mockResolvedValue([link()]);
    renderTab();

    await userEvent.click(await screen.findByRole("button", { name: "Revoke" }));
    await waitFor(() => expect(mocks.revokeLink).toHaveBeenCalledWith("shp_1", "ctl_1"));
    expect(screen.queryByRole("button", { name: "Revoke all" })).toBeNull();
  });

  it("shows each status email against the customer's rules, with a failure's reason", async () => {
    mocks.getProfile.mockResolvedValue({ ...profile, notifyDelivered: true });
    mocks.listNotifications.mockResolvedValue([
      {
        id: "csn_1",
        shipmentId: "shp_1",
        event: "PickedUp",
        stopId: "stp_1",
        occurredAt: NOW - 7200,
        estimatedAt: null,
        status: "Sent",
        skipReason: null,
        recipients: ["shipping@customer.com"],
        attempts: 1,
        lastError: null,
        sentAt: NOW - 7200,
        createdAt: NOW - 7200,
      },
      {
        id: "csn_2",
        shipmentId: "shp_1",
        event: "Delayed",
        stopId: "stp_2",
        occurredAt: NOW - 600,
        estimatedAt: NOW + 3600,
        status: "Pending",
        skipReason: null,
        recipients: ["shipping@customer.com"],
        attempts: 2,
        lastError: "550 mailbox unavailable",
        sentAt: null,
        createdAt: NOW - 600,
      },
    ]);
    renderTab();

    expect(await screen.findByText("Retrying")).toBeDefined();
    expect(screen.getByText(/^Sent /)).toBeDefined();
    expect(screen.getByText("Not yet")).toBeDefined();
    expect(screen.getByText("Off")).toBeDefined();
    expect(screen.getByText("Delayed: retrying after 2 failures")).toBeDefined();
    expect(screen.getByText("550 mailbox unavailable")).toBeDefined();
    expect(screen.getByText("To shipping@customer.com +1")).toBeDefined();
  });

  it("explains why it cannot share when the customer's links are off", async () => {
    mocks.getProfile.mockResolvedValue({ ...profile, linkEnabled: false });
    renderTab();

    expect(await screen.findByText("Tracking links are turned off for this customer.")).toBeDefined();
    expect(screen.getByRole("link", { name: "Turn them on" })).toBeDefined();
    expect(screen.queryByRole("button", { name: "Send" })).toBeNull();
  });

  it("emails a new link and shows it once to copy", async () => {
    mocks.issueLink.mockResolvedValue({
      link: link({ id: "ctl_new" }),
      url: "https://tms.example.com/shipment-tracking/tok",
      emailed: true,
      emailError: null,
    });
    renderTab();

    const form = (await screen.findByRole("button", { name: "Send" })).closest("form");
    expect(form).not.toBeNull();
    await userEvent.type(
      within(form as HTMLElement).getByRole("textbox", { name: "Email the link to" }),
      "receiving@customer.com",
    );
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() =>
      expect(mocks.issueLink).toHaveBeenCalledWith({
        shipmentId: "shp_1",
        recipientEmail: "receiving@customer.com",
      }),
    );
    expect(await screen.findByText("Emailed to receiving@customer.com")).toBeDefined();
    expect(screen.getByText("https://tms.example.com/shipment-tracking/tok")).toBeDefined();
  });

  it("keeps the share form from someone who cannot change the shipment", async () => {
    mocks.canShare = false;
    renderTab();

    expect(await screen.findByText("Acme Foods")).toBeDefined();
    expect(screen.queryByRole("button", { name: "Send" })).toBeNull();
  });
});
