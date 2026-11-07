import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ConsolePageActions } from "../console-page-actions";
import type { DispatchActions } from "../use-dispatch-actions";
import { resolveCenterMode } from "../url-state";

vi.mock("../dispatch-timeline", () => ({
  DispatchTimeline: () => <div data-testid="dispatch-timeline" />,
}));

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

function signIn(
  capabilities: OrganizationCapabilities,
  operations = Operation.Read | Operation.Assign,
) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Brokerage Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
  usePermissionStore.setState({
    manifest: {
      version: "1.0",
      permissions: { [Resource.ShipmentMove]: operations },
      routeAccess: {},
      availableOrgs: [],
      checksum: "abc123",
      expiresAt: 1_782_403_304,
    } as unknown as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
}

const actions = {
  canUndo: false,
  undoLabel: null,
  isAssigning: false,
  isDeciding: false,
  isPlanning: false,
  undo: vi.fn(),
} as unknown as DispatchActions;

function renderActions(onPropose = vi.fn()) {
  render(
    <NuqsTestingAdapter searchParams="">
      <ConsolePageActions actions={actions} onPropose={onPropose} />
    </NuqsTestingAdapter>,
  );
  return onPropose;
}

afterEach(() => {
  cleanup();
  useAuthStore.setState({ user: null, isAuthenticated: false });
  usePermissionStore.getState().clearPermissions();
});

describe("resolveCenterMode", () => {
  it("keeps the driver timeline for an organization that employs drivers", () => {
    expect(resolveCenterMode("timeline", hybrid)).toBe("timeline");
    expect(resolveCenterMode("queue", hybrid)).toBe("queue");
  });

  it("falls back to the queue for a brokerage", () => {
    expect(resolveCenterMode("timeline", brokerageOnly)).toBe("queue");
    expect(resolveCenterMode("queue", brokerageOnly)).toBe("queue");
  });
});

describe("dispatch console view switch", () => {
  it("offers the queue and the timeline to an organization that employs drivers", () => {
    signIn(hybrid);

    renderActions();

    const group = screen.getByRole("group", { name: "View" });
    expect(group).toHaveTextContent("Queue");
    expect(group).toHaveTextContent("Timeline");
  });

  it("withdraws the switch from a brokerage rather than offering one option", () => {
    signIn(brokerageOnly);

    renderActions();

    expect(screen.queryByRole("group", { name: "View" })).toBeNull();
  });
});

describe("dispatch console Propose coverage", () => {
  it("offers it to a dispatcher who may assign moves in an organization with drivers", () => {
    signIn(hybrid);

    renderActions();

    expect(screen.getByRole("button", { name: "Propose coverage" })).toBeInTheDocument();
  });

  it("withholds it from someone who may only read moves", () => {
    signIn(hybrid, Operation.Read);

    renderActions();

    expect(screen.queryByRole("button", { name: "Propose coverage" })).toBeNull();
  });

  it("leaves a brokerage no control at all that reaches the plan", async () => {
    signIn(brokerageOnly);

    const onPropose = renderActions();
    for (const control of screen.getAllByRole("button")) {
      await userEvent.click(control);
    }

    expect(onPropose).not.toHaveBeenCalled();
  });
});
