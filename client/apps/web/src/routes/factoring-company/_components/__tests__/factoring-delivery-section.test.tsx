import type { FactoringTransport } from "@/lib/graphql/factoring";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringDeliverySection } from "../factoring-delivery-section";

const mocks = vi.hoisted(() => ({
  fetchFactoringTransport: vi.fn(),
  saveFactoringTransport: vi.fn(),
  testFactoringTransport: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/factoring", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/factoring")>()),
  fetchFactoringTransport: mocks.fetchFactoringTransport,
  saveFactoringTransport: mocks.saveFactoringTransport,
  testFactoringTransport: mocks.testFactoringTransport,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, error: mocks.toastError, info: vi.fn() },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function sftpTransport(overrides: Partial<FactoringTransport> = {}): FactoringTransport {
  return {
    id: "factrn_1",
    factoringCompanyId: "fac_rts",
    method: "Sftp",
    emailRecipients: [],
    attachPackets: true,
    sftpHost: "sftp.rtsfinancial.com",
    sftpPort: 22,
    sftpUsername: "trenova",
    sftpAuthMethod: "Password",
    sftpSecretSet: true,
    sftpKnownHostKey:
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl",
    sftpDirectory: "/incoming",
    lastTestedAt: null,
    lastTestError: null,
    version: 3,
    ...overrides,
  } as FactoringTransport;
}

function renderSection() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <FactoringDeliverySection factoringCompanyId="fac_rts" />
    </QueryClientProvider>,
  );
}

describe("factoring delivery section", () => {
  it("says a factor without delivery settings cannot take submissions", async () => {
    mocks.fetchFactoringTransport.mockResolvedValue(null);
    renderSection();

    expect(
      await screen.findByText(
        "Not set up. Invoices for this factor can't be submitted until it is.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Test" })).toBeNull();
    expect(screen.getByRole("button", { name: "Set up" })).toBeInTheDocument();
  });

  it("keeps the stored password when it is left blank, sending the version it was read at", async () => {
    mocks.fetchFactoringTransport.mockResolvedValue(sftpTransport());
    mocks.saveFactoringTransport.mockResolvedValue(sftpTransport({ version: 4 }));
    renderSection();

    expect(await screen.findByText("trenova@sftp.rtsfinancial.com:22")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Edit" }));
    const dialog = await screen.findByRole("dialog");
    await userEvent.click(within(dialog).getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(mocks.saveFactoringTransport).toHaveBeenCalledWith({
        factoringCompanyId: "fac_rts",
        version: 3,
        method: "Sftp",
        emailRecipients: [],
        attachPackets: true,
        sftpHost: "sftp.rtsfinancial.com",
        sftpPort: 22,
        sftpUsername: "trenova",
        sftpAuthMethod: "Password",
        sftpSecret: null,
        sftpKnownHostKey: sftpTransport().sftpKnownHostKey,
        sftpDirectory: "/incoming",
      }),
    );
  });

  it("asks for the key when sign-in moves from a password to a key, since the stored password is dropped", async () => {
    mocks.fetchFactoringTransport.mockResolvedValue(sftpTransport());
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Edit" }));
    const dialog = await screen.findByRole("dialog");
    await userEvent.click(within(dialog).getByRole("radio", { name: "Private key" }));
    await userEvent.click(within(dialog).getByRole("button", { name: "Save" }));

    expect(await within(dialog).findByText("Paste the private key")).toBeInTheDocument();
    expect(mocks.saveFactoringTransport).not.toHaveBeenCalled();
  });

  it("refuses a relative upload folder", async () => {
    mocks.fetchFactoringTransport.mockResolvedValue(sftpTransport());
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Edit" }));
    const dialog = await screen.findByRole("dialog");
    const folder = within(dialog).getByRole("textbox", { name: /Folder/ });
    await userEvent.clear(folder);
    await userEvent.type(folder, "incoming/../etc");
    await userEvent.click(within(dialog).getByRole("button", { name: "Save" }));

    expect(
      await within(dialog).findByText("Use an absolute folder such as /incoming, without .."),
    ).toBeInTheDocument();
    expect(mocks.saveFactoringTransport).not.toHaveBeenCalled();
  });

  it("reports a failed connection test with the server's reason", async () => {
    mocks.fetchFactoringTransport.mockResolvedValue(sftpTransport());
    mocks.testFactoringTransport.mockResolvedValue({
      passed: false,
      message: "ssh: handshake failed: knownhosts: key mismatch",
      transport: sftpTransport({
        lastTestedAt: 1_789_000_000,
        lastTestError: "ssh: handshake failed: knownhosts: key mismatch",
      }),
    });
    renderSection();

    await userEvent.click(await screen.findByRole("button", { name: "Test" }));

    await waitFor(() =>
      expect(mocks.toastError).toHaveBeenCalledWith("Could not connect to the factor", {
        description: "ssh: handshake failed: knownhosts: key mismatch",
      }),
    );
    expect(mocks.testFactoringTransport).toHaveBeenCalledWith("fac_rts");
    expect(
      await screen.findByText("ssh: handshake failed: knownhosts: key mismatch"),
    ).toBeInTheDocument();
  });
});
