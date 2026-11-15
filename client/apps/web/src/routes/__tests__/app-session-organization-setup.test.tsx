import { renderAtRoutes } from "@trenova/shared/testing/test-router";
import { cleanup, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationSetup, User } from "@trenova/shared/types/user";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AppSession } from "../app-layout";

vi.mock("@/hooks/use-permission-polling", () => ({ usePermissionPolling: () => {} }));
vi.mock("@/hooks/use-realtime-connection", () => ({ useRealtimeConnection: () => {} }));
vi.mock("@/components/session-idle-timeout", () => ({ SessionIdleTimeout: () => null }));
vi.mock("@/components/session-context/support-session-banner", () => ({
  SupportSessionBanner: () => null,
}));

vi.mock("../session-gates", () => ({
  PasswordChangeGate: () => <p>password gate</p>,
  RoleActivationGate: () => <p>role gate</p>,
  OrganizationSetupGate: ({ setup }: { setup: OrganizationSetup }) => (
    <p>organization setup gate: {setup.missing.join(", ")}</p>
  ),
}));

function signIn(overrides: Partial<User>) {
  useAuthStore.setState({
    user: {
      id: "usr_01",
      currentOrganizationId: "org_01",
      businessUnitId: "bu_01",
      name: "Dana Whitfield",
      username: "dana",
      emailAddress: "dana@acme.example",
      timezone: "America/New_York",
      mustChangePassword: false,
      ...overrides,
    } as User,
    isAuthenticated: true,
  });
}

async function renderAt(path: string) {
  return renderAtRoutes([{ path: "$", component: () => <p>the page</p> }], {
    initialEntries: [path],
    layout: () => <AppSession>{(outlet) => outlet}</AppSession>,
  });
}

const incomplete: OrganizationSetup = {
  complete: false,
  missing: ["addressLine1", "scacCode"],
  canComplete: true,
};

describe("the organization setup gate", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
    usePermissionStore.setState({ manifest: null });
  });

  it("stands in front of any page while the organization is incomplete", async () => {
    signIn({ organizationSetup: incomplete });
    await renderAt("/billing/invoices?tab=open");

    expect(
      await screen.findByText("organization setup gate: addressLine1, scacCode"),
    ).toBeInTheDocument();
    expect(screen.queryByText("the page")).not.toBeInTheDocument();
  });

  it("lets the page through once the organization is complete", async () => {
    signIn({ organizationSetup: { complete: true, missing: [], canComplete: false } });
    await renderAt("/billing/invoices");

    expect(screen.getByText("the page")).toBeInTheDocument();
  });

  it("asks for a password change first", async () => {
    signIn({ organizationSetup: incomplete, mustChangePassword: true });
    await renderAt("/");

    expect(await screen.findByText("password gate")).toBeInTheDocument();
    expect(screen.queryByText(/organization setup gate/)).not.toBeInTheDocument();
  });
});
