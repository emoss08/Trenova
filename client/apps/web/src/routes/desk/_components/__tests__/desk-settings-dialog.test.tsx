import { apiService } from "@/services/api";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskSettingsDialog } from "../desk-settings";

function grant(permissions: Record<string, number>) {
  usePermissionStore.setState({
    manifest: {
      version: "1.0",
      userId: "usr_01",
      organizationId: "org_01",
      activeRoleIds: [],
      authorizedRoleIds: [],
      activeRoles: [],
      authorizedRoles: [],
      requiresRoleActivation: false,
      maxSensitivity: "internal",
      permissions,
      routeAccess: {},
      availableOrgs: [],
      checksum: "abc123",
      expiresAt: 1_782_403_304,
    } as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
}

function renderDialog(initialSection?: "checklists") {
  vi.spyOn(apiService.caseChecklistService, "list").mockResolvedValue({
    kind: "ReadyToBill",
    organization: {
      id: "",
      kind: "ReadyToBill",
      customerId: "",
      customerName: "",
      items: [{ key: "delivered", mode: "Required" }],
      version: 0,
      updatedAt: 0,
    },
    customers: [],
    locked: ["delivered"],
  });
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <DeskSettingsDialog agents={[]} initialSection={initialSection} onClose={vi.fn()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/**
 * Case checklists are the organization's, so the Desk's settings only offer
 * them to someone who may read billing control, and a link can open the
 * dialog straight on them.
 */
describe("DeskSettingsDialog", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    usePermissionStore.setState({ manifest: null });
  });

  it("leaves the organization out for someone who may not read billing control", () => {
    grant({});
    renderDialog("checklists");

    expect(screen.queryByText("Organization")).toBeNull();
    expect(screen.queryByRole("button", { name: "Case checklists" })).toBeNull();
    expect(screen.getByRole("button", { name: "Appearance" })).toHaveAttribute("aria-current", "page");
  });

  it("opens on case checklists, wide, and without the personal reset", async () => {
    grant({ [Resource.BillingControl]: Operation.Read | Operation.Update });
    renderDialog("checklists");

    expect(screen.getByRole("button", { name: "Case checklists" })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByRole("dialog")).toHaveClass("dk-xl");
    expect(screen.queryByRole("button", { name: "Reset to defaults" })).toBeNull();
    expect(await screen.findByRole("heading", { name: "Your organization's checklist" })).toBeVisible();
  });
});
