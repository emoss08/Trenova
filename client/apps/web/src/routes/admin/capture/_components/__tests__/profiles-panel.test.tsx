import type { CaptureProfile } from "@/lib/graphql/capture";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ProfilesPanel } from "../profiles-panel";

const mocks = vi.hoisted(() => ({
  fetchCaptureProfiles: vi.fn(),
  deleteCaptureProfile: vi.fn(),
  createCaptureProfile: vi.fn(),
  updateCaptureProfile: vi.fn(),
  granted: new Set<string>(),
}));

vi.mock("@/lib/graphql/capture", () => ({
  fetchCaptureProfiles: mocks.fetchCaptureProfiles,
  fetchAvailableCaptureProfiles: vi.fn(),
  deleteCaptureProfile: mocks.deleteCaptureProfile,
  createCaptureProfile: mocks.createCaptureProfile,
  updateCaptureProfile: mocks.updateCaptureProfile,
}));

vi.mock("@/hooks/use-permission", () => {
  const has = (resource: string, operation: number) =>
    mocks.granted.has(`${resource}:${operation}`);
  return {
    usePermission: (resource: string, operation: number) => ({
      allowed: has(resource, operation),
      isLoading: false,
    }),
    usePermissions: (resource: string) => ({
      canRead: has(resource, Operation.Read),
      canCreate: has(resource, Operation.Create),
      canUpdate: has(resource, Operation.Update),
      canExport: false,
      canImport: false,
      isLoading: false,
    }),
  };
});

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }));

const paperwork: CaptureProfile = {
  id: "cprof_1",
  name: "Paperwork",
  description: "",
  status: "Active",
  isDefault: true,
  dpi: 300,
  pixelType: "BlackWhite",
  duplex: true,
  useFeeder: true,
  discardBlankPages: true,
  jpegQuality: 80,
  showDriverUi: false,
  separatorStrategies: ["PatchCode"],
  fixedPageCount: 0,
  version: 3,
  createdAt: 1_780_000_000,
  updatedAt: 1_780_000_000,
};

function grant(...operations: number[]) {
  for (const operation of operations) {
    mocks.granted.add(`${Resource.CaptureProfile}:${operation}`);
  }
}

function renderPanel() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<ProfilesPanel />, { wrapper });
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

describe("ProfilesPanel", () => {
  beforeEach(() => {
    mocks.granted.clear();
    mocks.fetchCaptureProfiles.mockResolvedValue([paperwork]);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the delete confirmation open until the delete settles, and shows why it failed", async () => {
    grant(Operation.Read, Operation.Delete);
    const pending = deferred<undefined>();
    mocks.deleteCaptureProfile.mockReturnValue(pending.promise);
    const user = userEvent.setup();

    renderPanel();
    await user.click(await screen.findByRole("button", { name: "Delete Paperwork" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Delete profile" }));

    expect(mocks.deleteCaptureProfile).toHaveBeenCalledWith("cprof_1");
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(within(screen.getByRole("alertdialog")).getByText("Delete Paperwork?")).toBeVisible();

    pending.reject(new Error("The profile is in use by a scan that is starting"));

    expect(
      await within(screen.getByRole("alertdialog")).findByText(
        "The profile is in use by a scan that is starting",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("closes the delete confirmation once the delete succeeds", async () => {
    grant(Operation.Read, Operation.Delete);
    mocks.deleteCaptureProfile.mockResolvedValue(undefined);
    const user = userEvent.setup();

    renderPanel();
    await user.click(await screen.findByRole("button", { name: "Delete Paperwork" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Delete profile" }));

    await waitFor(() => expect(screen.queryByRole("alertdialog")).not.toBeInTheDocument());
  });

  it("never titles an edit as a new profile while it closes", async () => {
    grant(Operation.Read, Operation.Update, Operation.Create);
    const user = userEvent.setup();

    renderPanel();
    await user.click(await screen.findByRole("button", { name: "Edit Paperwork" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).queryByRole("heading", { name: "New scan profile" })).toBeNull();

    const seen: string[] = [];
    const observer = new MutationObserver(() => {
      for (const heading of document.querySelectorAll("[role=dialog] h2")) {
        seen.push(heading.textContent ?? "");
      }
    });
    observer.observe(document.body, { subtree: true, childList: true, characterData: true });

    await user.click(within(dialog).getByRole("button", { name: "Close panel" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    observer.disconnect();

    expect(seen).not.toContain("New scan profile");
  });

  it("creates a profile from the panel and refreshes the list", async () => {
    grant(Operation.Read, Operation.Create);
    mocks.createCaptureProfile.mockResolvedValue({ ...paperwork, id: "cprof_2", name: "Legal" });
    const user = userEvent.setup();

    renderPanel();
    await screen.findByText("Paperwork");
    await user.click(screen.getByRole("button", { name: "New scan profile" }));
    const panel = await screen.findByRole("dialog");
    expect(within(panel).getByRole("heading", { name: "New scan profile" })).toBeInTheDocument();

    await user.type(within(panel).getByPlaceholderText("Paperwork"), "  Legal ");
    await user.click(within(panel).getByRole("button", { name: "Save & close" }));

    await waitFor(() =>
      expect(mocks.createCaptureProfile).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "Legal",
          dpi: 300,
          separatorStrategies: ["PatchCode", "CoverSheet"],
        }),
      ),
    );
    await waitFor(() => expect(mocks.fetchCaptureProfiles).toHaveBeenCalledTimes(2));
  });

  it("saves an edit against the version it was read at", async () => {
    grant(Operation.Read, Operation.Update);
    mocks.updateCaptureProfile.mockResolvedValue({ ...paperwork, name: "Paperwork 2", version: 4 });
    const user = userEvent.setup();

    renderPanel();
    await user.click(await screen.findByRole("button", { name: "Edit Paperwork" }));
    const panel = await screen.findByRole("dialog");
    const name = within(panel).getByPlaceholderText("Paperwork");
    await waitFor(() => expect(name).toHaveValue("Paperwork"));

    await user.type(name, " 2");
    await user.click(within(panel).getByRole("button", { name: "Save & close" }));

    await waitFor(() =>
      expect(mocks.updateCaptureProfile).toHaveBeenCalledWith(
        "cprof_1",
        3,
        expect.objectContaining({ name: "Paperwork 2", isDefault: true }),
      ),
    );
  });

  it("offers a retry when the profiles cannot be loaded", async () => {
    grant(Operation.Read);
    mocks.fetchCaptureProfiles.mockRejectedValueOnce(new Error("offline"));
    const user = userEvent.setup();

    renderPanel();
    await user.click(await screen.findByRole("button", { name: "Try again" }));

    expect(await screen.findByText("Paperwork")).toBeInTheDocument();
    expect(mocks.fetchCaptureProfiles).toHaveBeenCalledTimes(2);
  });

  it("marks the default profile without the brand colour", async () => {
    grant(Operation.Read);
    renderPanel();

    const badge = await screen.findByText("Default");
    expect(badge.closest("[data-variant]")?.getAttribute("data-variant")).not.toBe("brand");
  });
});
