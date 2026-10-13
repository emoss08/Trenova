import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentCustomerTracking from "../shipment-customer-tracking";

const NOW = Math.floor(Date.now() / 1000);

const mocks = vi.hoisted(() => ({
  listLinks: vi.fn(),
  listNotifications: vi.fn(),
  issue: vi.fn(),
  revoke: vi.fn(),
  canShare: true,
}));

vi.mock("@/lib/graphql/customer-tracking", () => ({
  listCustomerTrackingLinks: mocks.listLinks,
  listCustomerStatusNotifications: mocks.listNotifications,
  issueCustomerTrackingLink: mocks.issue,
  revokeCustomerTrackingLink: mocks.revoke,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canShare, isLoading: false }),
}));

function link(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    shipmentId: "shp_1",
    customerId: "cus_1",
    purpose: "Shared",
    recipientEmail: null,
    expiresAt: NOW + 86_400,
    revokedAt: null,
    lastViewedAt: null,
    viewCount: 0,
    createdById: null,
    createdAt: NOW - 600,
    ...overrides,
  };
}

function renderTab(status = "InTransit") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  }
  return render(
    <ShipmentCustomerTracking shipment={{ id: "shp_1", status } as unknown as Shipment} />,
    { wrapper: Wrapper },
  );
}

describe("ShipmentCustomerTracking", () => {
  beforeEach(() => {
    for (const fn of [mocks.listLinks, mocks.listNotifications, mocks.issue, mocks.revoke]) {
      fn.mockReset();
    }
    mocks.canShare = true;
    mocks.listLinks.mockResolvedValue([]);
    mocks.listNotifications.mockResolvedValue([]);
  });

  it("shares a link without emailing it and shows the link once", async () => {
    mocks.issue.mockResolvedValue({
      link: link("cstl_new"),
      url: "https://app.example.com/shipment-tracking/tok",
      emailed: false,
      emailError: null,
    });
    renderTab();

    await userEvent.click(await screen.findByRole("button", { name: /Share link/ }));

    await waitFor(() => expect(mocks.issue).toHaveBeenCalledTimes(1));
    expect(mocks.issue).toHaveBeenCalledWith({ shipmentId: "shp_1", recipientEmail: undefined });
    expect(
      await screen.findByDisplayValue("https://app.example.com/shipment-tracking/tok"),
    ).toBeDefined();
    expect(
      screen.getByText("This link is shown once. Share a new one if it is lost."),
    ).toBeDefined();
  });

  it("revokes one live link, and offers to revoke all when several are live", async () => {
    mocks.listLinks.mockResolvedValue([
      link("cstl_a", { recipientEmail: "ops@customer.example" }),
      link("cstl_b"),
      link("cstl_old", { revokedAt: NOW - 60 }),
    ]);
    mocks.revoke.mockResolvedValue(1);
    renderTab();

    expect(await screen.findByText("Emailed to ops@customer.example")).toBeDefined();
    expect(screen.getByText("Revoked")).toBeDefined();
    const revokeButtons = screen.getAllByRole("button", { name: /^Revoke$/ });
    expect(revokeButtons).toHaveLength(2);

    await userEvent.click(revokeButtons[0]);
    await waitFor(() => expect(mocks.revoke).toHaveBeenCalledWith("shp_1", "cstl_a"));

    await userEvent.click(screen.getByRole("button", { name: "Revoke all" }));
    await waitFor(() => expect(mocks.revoke).toHaveBeenCalledWith("shp_1", undefined));
  });

  it("lists each status email with what became of it", async () => {
    mocks.listNotifications.mockResolvedValue([
      {
        id: "csn_1",
        shipmentId: "shp_1",
        event: "PickedUp",
        stopId: null,
        occurredAt: NOW - 3600,
        estimatedAt: null,
        status: "Skipped",
        skipReason: "NoRecipients",
        recipients: [],
        attempts: 0,
        lastError: null,
        sentAt: null,
        createdAt: NOW - 3600,
      },
    ]);
    renderTab();

    expect(await screen.findByText("Picked up")).toBeDefined();
    expect(screen.getByText("Skipped")).toBeDefined();
    expect(screen.getByText("Nobody to send it to")).toBeDefined();
  });

  it("offers no sharing on a canceled shipment or without permission", async () => {
    renderTab("Canceled");
    expect(await screen.findByText("No link has been shared for this shipment.")).toBeDefined();
    expect(screen.queryByRole("button", { name: /Share link/ })).toBeNull();
  });
});
