import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation } from "@trenova/shared/types/permission";
import { stubLayout } from "@/test/layout";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import WorkerPortalTab from "../worker-portal-tab";

const mocks = vi.hoisted(() => ({
  fetchWorkerPortalStatus: vi.fn(),
  revokeWorkerPortalAccess: vi.fn(),
  inviteWorkerToPortal: vi.fn(),
}));
const granted = vi.hoisted(() => ({ ops: new Set<number>() }));

vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  fetchWorkerPortalStatus: mocks.fetchWorkerPortalStatus,
  inviteWorkerToPortal: mocks.inviteWorkerToPortal,
  revokeWorkerPortalAccess: mocks.revokeWorkerPortalAccess,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: granted.ops.has(operation),
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/hooks/use-can-read-kind", () => ({ useCanReadKind: () => true }));

vi.mock("../profile-change-list", () => ({
  ProfileChangeList: ({ onReview }: { onReview: (id: string) => void }) => (
    <button type="button" onClick={() => onReview("pcr_1")}>
      profile-change-list
    </button>
  ),
}));

vi.mock("../policy-acknowledgements", () => ({
  PolicyAcknowledgements: () => <div>policy-acknowledgements</div>,
}));

const worker = { firstName: "Maria" } as never;
const notSetUp = { linked: false, portalUser: null, pendingInvitation: null, invitations: [] };
const linked = {
  linked: true,
  portalUser: {
    id: "usr_1",
    name: "Maria Alvarez",
    emailAddress: "maria.alvarez@example.com",
    lastLoginAt: 1_795_000_000,
  },
  pendingInvitation: null,
  invitations: [
    {
      id: "inv_1",
      email: "maria.alvarez@example.com",
      status: "Accepted",
      expiresAt: 1_700_000_000,
      acceptedAt: 1_663_000_000,
      createdAt: 1_662_950_000,
      invitedBy: { id: "usr_2", name: "Marcus Bell" },
    },
  ],
};

function renderTab(ops: number[], onOpen = vi.fn()) {
  granted.ops = new Set(ops);
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkerPortalTab workerId="wrk_1" worker={worker} onOpen={onOpen} />
    </QueryClientProvider>,
  );
  return onOpen;
}

let restoreLayout = () => {};
beforeEach(() => {
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerPortalTab", () => {
  it("reads no portal status without driver portal read, and still lists the driver's changes", () => {
    renderTab([]);

    expect(mocks.fetchWorkerPortalStatus).not.toHaveBeenCalled();
    expect(screen.queryByText("Dash access")).toBeNull();
    expect(screen.getByText("profile-change-list")).toBeTruthy();
    expect(screen.getByText("policy-acknowledgements")).toBeTruthy();
  });

  it("offers the invitation only to a person who may invite", async () => {
    mocks.fetchWorkerPortalStatus.mockResolvedValue(notSetUp);
    renderTab([Operation.Read, Operation.Create]);

    expect(await screen.findByRole("button", { name: "Send invitation" })).toBeTruthy();
  });

  it("says the driver is not invited to a reader who may not invite", async () => {
    mocks.fetchWorkerPortalStatus.mockResolvedValue(notSetUp);
    renderTab([Operation.Read]);

    expect(await screen.findByText("Maria has not been invited to Dash.")).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Send invitation" })).toBeNull();
  });

  it("names invitation states in words, never by their code", async () => {
    mocks.fetchWorkerPortalStatus.mockResolvedValue(linked);
    renderTab([Operation.Read]);

    expect(await screen.findByText("Accepted")).toBeTruthy();
    expect(screen.getByText(/Sent .* by Marcus Bell · accepted/)).toBeTruthy();
  });

  it("revokes access only after the confirmation", async () => {
    mocks.fetchWorkerPortalStatus.mockResolvedValue(linked);
    mocks.revokeWorkerPortalAccess.mockResolvedValue(true);
    renderTab([Operation.Read, Operation.Cancel]);
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Revoke access" }));
    expect(mocks.revokeWorkerPortalAccess).not.toHaveBeenCalled();
    expect(await screen.findByText("Revoke Maria's Dash access?")).toBeTruthy();

    const confirm = screen
      .getAllByRole("button", { name: "Revoke access" })
      .find((button) => button.closest('[role="alertdialog"]'));
    await user.click(confirm as HTMLElement);
    await waitFor(() => expect(mocks.revokeWorkerPortalAccess).toHaveBeenCalledWith("wrk_1"));
  });

  it("hides revoking from a reader who may not revoke", async () => {
    mocks.fetchWorkerPortalStatus.mockResolvedValue(linked);
    renderTab([Operation.Read, Operation.Create]);

    expect(await screen.findByText("maria.alvarez@example.com", { selector: "span" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Revoke access" })).toBeNull();
  });

  it("opens the page's review pane for a profile change", async () => {
    const onOpen = renderTab([]);
    await userEvent.setup().click(screen.getByText("profile-change-list"));

    expect(onOpen).toHaveBeenCalledWith({ tab: "files", view: "driver-portal", request: "pcr_1" });
  });
});
