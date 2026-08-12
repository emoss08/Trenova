import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import RightStack from "../index";
import {
  ALL_MODULES,
  availableRightStackModules,
  useRightStackStore,
  type RightStackModuleId,
} from "../right-stack-store";

vi.mock("../unassigned-queue", () => ({
  UnassignedQueue: () => <div data-testid="panel-unassigned" />,
}));
vi.mock("../exceptions-inbox", () => ({
  ExceptionsInbox: () => <div data-testid="panel-exceptions" />,
}));
vi.mock("../hos-watch", () => ({
  HosWatch: () => <div data-testid="panel-hos" />,
}));
vi.mock("../certification-watch", () => ({
  CertificationWatch: () => <div data-testid="panel-certification" />,
}));

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

function signIn(capabilities: OrganizationCapabilities) {
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
}

function setPreference(order: RightStackModuleId[], hidden: RightStackModuleId[] = []) {
  useRightStackStore.setState({ order, hidden });
}

describe("availableRightStackModules", () => {
  it("keeps every module for an organization that employs drivers", () => {
    expect(availableRightStackModules(ALL_MODULES, hybrid)).toEqual([...ALL_MODULES]);
  });

  it("drops the driver-watching modules for a brokerage", () => {
    expect(availableRightStackModules(ALL_MODULES, brokerageOnly)).toEqual([
      "unassigned",
      "exceptions",
    ]);
  });

  it("preserves the stored order of the modules it keeps", () => {
    const stored: RightStackModuleId[] = ["hos", "exceptions", "certification", "unassigned"];

    expect(availableRightStackModules(stored, brokerageOnly)).toEqual(["exceptions", "unassigned"]);
  });

  it("returns an empty list rather than throwing when nothing survives", () => {
    expect(availableRightStackModules(["hos", "certification"], brokerageOnly)).toEqual([]);
  });
});

describe("RightStack under a stored preference that names a hidden module", () => {
  beforeEach(() => {
    setPreference([...ALL_MODULES], []);
  });

  afterEach(() => {
    cleanup();
    setPreference([...ALL_MODULES], []);
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("mounts all four panels for an organization that employs drivers", () => {
    signIn(hybrid);

    render(<RightStack />);

    expect(screen.getByTestId("panel-unassigned")).toBeInTheDocument();
    expect(screen.getByTestId("panel-exceptions")).toBeInTheDocument();
    expect(screen.getByTestId("panel-hos")).toBeInTheDocument();
    expect(screen.getByTestId("panel-certification")).toBeInTheDocument();
  });

  it("leaves the driver panels unmounted for a brokerage", () => {
    signIn(brokerageOnly);

    render(<RightStack />);

    expect(screen.getByTestId("panel-unassigned")).toBeInTheDocument();
    expect(screen.getByTestId("panel-exceptions")).toBeInTheDocument();
    expect(screen.queryByTestId("panel-hos")).toBeNull();
    expect(screen.queryByTestId("panel-certification")).toBeNull();
  });

  it("honours a stored order that leads with a module the brokerage cannot use", () => {
    signIn(brokerageOnly);
    setPreference(["hos", "unassigned", "certification", "exceptions"]);

    render(<RightStack />);

    const panels = screen.getAllByTestId(/^panel-/).map((node) => node.dataset.testid);
    expect(panels).toEqual(["panel-unassigned", "panel-exceptions"]);
  });

  it("does not offer a hidden driver module back through the add-panel menu", () => {
    signIn(brokerageOnly);
    setPreference([...ALL_MODULES], ["hos", "certification"]);

    render(<RightStack />);

    expect(screen.queryByRole("button", { name: /Add panel/ })).toBeNull();
  });

  it("counts only the modules the organization can actually restore", () => {
    signIn(brokerageOnly);
    setPreference([...ALL_MODULES], ["exceptions", "hos", "certification"]);

    render(<RightStack />);

    expect(screen.getByRole("button", { name: "Add panel (1)" })).toBeInTheDocument();
  });

  it("falls back to the empty state when the only visible modules are gated away", () => {
    signIn(brokerageOnly);
    setPreference([...ALL_MODULES], ["unassigned", "exceptions"]);

    render(<RightStack />);

    expect(screen.getByText("All panels hidden")).toBeInTheDocument();
    expect(screen.queryByTestId("panel-hos")).toBeNull();
    expect(screen.queryByTestId("panel-certification")).toBeNull();
  });
});
