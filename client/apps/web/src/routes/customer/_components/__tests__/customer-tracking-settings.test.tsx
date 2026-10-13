import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CustomerTrackingSettingsTab } from "../customer-tracking-settings";

const mocks = vi.hoisted(() => ({
  getProfile: vi.fn(),
  updateProfile: vi.fn(),
  canEdit: true,
}));

vi.mock("@/lib/graphql/customer-tracking", () => ({
  getCustomerTrackingProfile: mocks.getProfile,
  updateCustomerTrackingProfile: mocks.updateProfile,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.canEdit, isLoading: false }),
}));

const defaults = {
  id: null,
  customerId: "cus_1",
  linkEnabled: true,
  visibleFields: ["Status", "Windows", "Actuals", "Eta", "References"],
  linkTtlDays: 30,
  notifyPickedUp: false,
  notifyDelayed: false,
  notifyDelivered: false,
  notifyPodReady: false,
  includeLink: true,
  recipients: null,
  notificationsEnabledAt: null,
  version: 0,
  updatedAt: 0,
};

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  }
  return render(<CustomerTrackingSettingsTab customerId="cus_1" />, { wrapper: Wrapper });
}

describe("CustomerTrackingSettingsTab", () => {
  beforeEach(() => {
    mocks.getProfile.mockReset();
    mocks.updateProfile.mockReset();
    mocks.canEdit = true;
  });

  it("saves the defaults as new settings, with no version to check", async () => {
    mocks.getProfile.mockResolvedValue(defaults);
    mocks.updateProfile.mockResolvedValue({ ...defaults, id: "ctp_1", notifyDelivered: true });
    renderTab();

    const save = await screen.findByRole("button", { name: "Save tracking settings" });
    expect(save).toHaveProperty("disabled", true);

    await userEvent.click(screen.getByRole("switch", { name: /^Delivered/ }));
    expect(save).toHaveProperty("disabled", false);
    await userEvent.click(save);

    await waitFor(() => expect(mocks.updateProfile).toHaveBeenCalledTimes(1));
    expect(mocks.updateProfile).toHaveBeenCalledWith({
      customerId: "cus_1",
      linkEnabled: true,
      visibleFields: ["Status", "Windows", "Actuals", "Eta", "References"],
      linkTtlDays: 30,
      notifyPickedUp: false,
      notifyDelayed: false,
      notifyDelivered: true,
      notifyPodReady: false,
      includeLink: true,
      recipients: "",
      expectedVersion: null,
    });
  });

  it("sends the version it read when the settings were saved before", async () => {
    mocks.getProfile.mockResolvedValue({ ...defaults, id: "ctp_1", version: 4 });
    mocks.updateProfile.mockResolvedValue({ ...defaults, id: "ctp_1", version: 5 });
    renderTab();

    await userEvent.click(await screen.findByRole("switch", { name: /^Picked up/ }));
    await userEvent.click(screen.getByRole("button", { name: "Save tracking settings" }));

    await waitFor(() => expect(mocks.updateProfile).toHaveBeenCalledTimes(1));
    expect(mocks.updateProfile.mock.calls[0][0]).toMatchObject({
      notifyPickedUp: true,
      expectedVersion: 4,
    });
  });

  it("offers no save to someone who cannot change the customer", async () => {
    mocks.canEdit = false;
    mocks.getProfile.mockResolvedValue(defaults);
    renderTab();

    expect(await screen.findByText("Status emails")).toBeDefined();
    expect(screen.queryByRole("button", { name: "Save tracking settings" })).toBeNull();
  });
});
