import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { knownRemedies, PreviewRemedies, type PreviewRemedy } from "../preview-remedies";

const allowed = vi.hoisted(() => ({ value: true }));
const invite = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: allowed.value }),
}));
vi.mock("@trenova/shared/lib/graphql/driver-portal", () => ({
  inviteWorkerToPortal: invite,
}));

const worker = "wrk_01M717A1ADDJS2ZTZ8FYHQXNS0";

function remedy(overrides: Partial<PreviewRemedy> = {}): PreviewRemedy {
  return {
    kind: "invite_driver_to_portal",
    recordKind: "worker",
    recordId: worker,
    recordLabel: "Jane Doe",
    takenAt: null,
    takenNote: "",
    ...overrides,
  } as PreviewRemedy;
}

function renderRemedies(remedies: PreviewRemedy[]) {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PreviewRemedies warning={{ remedies }} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  invite.mockReset();
  allowed.value = true;
});

/**
 * A remedy is the server saying what would put a warning right
 * (AgentPreviewRemedy: kind, recordKind, recordId, recordLabel). The client
 * offers it by name, to a reader who may take it, and never shows the id.
 */
describe("PreviewRemedies", () => {
  it("invites the driver from the warning, then shows only that it was done", async () => {
    invite.mockResolvedValue({ emailSent: true, invitation: { email: "jane@example.com" } });
    renderRemedies([remedy()]);
    expect(screen.getByRole("link", { name: "Open Jane Doe" })).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Invite Jane Doe to the driver portal" }));

    await waitFor(() =>
      expect(screen.getByRole("status").textContent).toBe(
        "Invitation emailed to jane@example.com. Messages reach them once they accept it.",
      ),
    );
    expect(invite).toHaveBeenCalledWith({ workerId: worker, replacePending: false });
    expect(screen.queryByRole("button", { name: /Invite/ })).toBeNull();
    expect(screen.queryByRole("link", { name: "Open Jane Doe" })).toBeNull();
    expect(document.body.textContent).not.toContain(worker);
  });

  it("says so when the invitation could not be emailed", async () => {
    invite.mockResolvedValue({ emailSent: false, invitation: { email: "" } });
    renderRemedies([remedy()]);

    fireEvent.click(screen.getByRole("button", { name: "Invite Jane Doe to the driver portal" }));

    await waitFor(() =>
      expect(screen.getByRole("status").textContent).toContain("no email could be sent"),
    );
  });

  it("shows an invitation the server says is already out, and sends it again when asked", async () => {
    invite.mockResolvedValue({ emailSent: true, invitation: { email: "jane@example.com" } });
    renderRemedies([remedy({ takenAt: 1_794_300_000, takenNote: "jane@example.com" })]);

    expect(screen.getByRole("status").textContent).toMatch(
      /^Invitation sent to jane@example\.com on .+\. Messages reach them once they accept it\.$/u,
    );
    expect(screen.queryByRole("button", { name: /Invite Jane Doe/ })).toBeNull();
    expect(screen.queryByRole("link", { name: "Open Jane Doe" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Send it again" }));

    await waitFor(() =>
      expect(invite).toHaveBeenCalledWith({ workerId: worker, replacePending: true }),
    );
  });

  it("links to where the fix is made by hand", () => {
    renderRemedies([remedy(), remedy({ kind: "set_up_email_profile", recordId: "" })]);

    expect(screen.getByRole("link", { name: "Open Jane Doe" }).getAttribute("href")).toBe(
      `/hr/workers?panelType=edit&panelEntityId=${worker}`,
    );
    expect(screen.getByRole("link", { name: "Set up an email profile" }).getAttribute("href")).toBe(
      "/organization/email-profiles",
    );
  });

  it("offers nothing to a reader who may not take the remedy", () => {
    allowed.value = false;
    const { container } = renderRemedies([remedy()]);

    expect(container.textContent).toBe("");
  });

  it("leaves out a remedy this client has no entry for", () => {
    expect(knownRemedies({ remedies: [remedy({ kind: "something_newer" })] })).toEqual([]);
  });
});
