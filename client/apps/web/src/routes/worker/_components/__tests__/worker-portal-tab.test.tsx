import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { Operation } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerPortalTab from "../worker-portal-tab";

const { fetchWorkerPortalStatus } = vi.hoisted(() => ({ fetchWorkerPortalStatus: vi.fn() }));
const granted = vi.hoisted(() => ({ ops: new Set<number>() }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchWorkerPortalStatus,
  inviteWorkerToPortal: vi.fn(),
  revokeWorkerPortalAccess: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: granted.ops.has(operation),
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/hooks/use-can-read-kind", () => ({ useCanReadKind: () => true }));

vi.mock("../portal/profile-change-requests", () => ({
  ProfileChangeRequests: () => <div>profile-change-requests</div>,
}));

vi.mock("../portal/policy-acknowledgements", () => ({
  PolicyAcknowledgements: () => <div>policy-acknowledgements</div>,
}));

const notSetUp = { linked: false, portalUser: null, pendingInvitation: null, invitations: [] };
const linked = {
  linked: true,
  portalUser: {
    id: "usr_1",
    name: "Dana Driver",
    emailAddress: "dana@example.com",
    lastLoginAt: 1,
  },
  pendingInvitation: null,
  invitations: [],
};

function renderTab(...ops: number[]) {
  granted.ops = new Set(ops);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <WorkerPortalTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  fetchWorkerPortalStatus.mockReset();
});

describe("WorkerPortalTab", () => {
  it("reads no portal status without driver portal read", () => {
    renderTab();

    expect(fetchWorkerPortalStatus).not.toHaveBeenCalled();
    expect(screen.queryByText("Dash access")).toBeNull();
    expect(screen.getByText("profile-change-requests")).toBeTruthy();
    expect(screen.getByText("policy-acknowledgements")).toBeTruthy();
  });

  it("offers the invitation only to a person who may invite", async () => {
    fetchWorkerPortalStatus.mockResolvedValue(notSetUp);
    renderTab(Operation.Read, Operation.Create);

    expect(await screen.findByText("Send invitation")).toBeTruthy();
  });

  it("says the driver is not invited to a reader who may not invite", async () => {
    fetchWorkerPortalStatus.mockResolvedValue(notSetUp);
    renderTab(Operation.Read);

    expect(await screen.findByText("This driver has not been invited to Dash.")).toBeTruthy();
    expect(screen.queryByText("Send invitation")).toBeNull();
  });

  it("offers revoking access only to a person who may revoke", async () => {
    fetchWorkerPortalStatus.mockResolvedValue(linked);
    renderTab(Operation.Read, Operation.Create);

    expect(await screen.findByText("Dana Driver")).toBeTruthy();
    expect(screen.queryByText("Revoke access")).toBeNull();

    cleanup();
    renderTab(Operation.Read, Operation.Cancel);
    expect(await screen.findByText("Revoke access")).toBeTruthy();
  });
});
