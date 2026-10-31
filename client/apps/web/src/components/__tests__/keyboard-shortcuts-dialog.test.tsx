import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Resource } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";

const allowed = vi.fn<(resource: string) => boolean>();
const platform = vi.hoisted(() => ({ mac: false }));

vi.mock("@trenova/shared/lib/shortcuts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/shortcuts")>()),
  isMacPlatform: () => platform.mac,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({ allowed: allowed(resource), isLoading: false }),
  usePageAccessOffered: () => allowed(Resource.Role),
}));

beforeEach(() => {
  vi.stubEnv("DEV", false);
  useAppDialogsStore.setState({ active: "shortcuts", prepared: [] });
});

afterEach(() => {
  cleanup();
  allowed.mockReset();
  platform.mac = false;
  vi.unstubAllEnvs();
});

/** The keys a row of the dialog shows, run together: its last cell. */
async function keysFor(label: string) {
  const row = (await screen.findByText(label)).closest("div")?.parentElement;
  return row?.lastElementChild?.textContent;
}

describe("KeyboardShortcutsDialog", () => {
  it("lists the page access shortcut, from its keybind entry, where the panel is offered", async () => {
    allowed.mockImplementation((resource) => resource === Resource.Role);
    render(<KeyboardShortcutsDialog />);

    expect(await screen.findByText("Page access")).toBeInTheDocument();
  });

  it("leaves the page access shortcut out where the panel is not offered", async () => {
    allowed.mockReturnValue(false);
    render(<KeyboardShortcutsDialog />);

    expect(await screen.findByRole("dialog", { name: "Keyboard shortcuts" })).toBeInTheDocument();
    expect(screen.queryByText("Page access")).not.toBeInTheDocument();
  });

  it("spells shortcuts with the Mac's modifier symbols on a Mac, and never says Ctrl there", async () => {
    platform.mac = true;
    allowed.mockReturnValue(true);
    render(<KeyboardShortcutsDialog />);

    expect(await keysFor("User settings")).toBe("⇧⌘S");
    expect(screen.getByRole("dialog")).not.toHaveTextContent("Ctrl");
  });

  it("spells shortcuts with Ctrl off a Mac", async () => {
    allowed.mockReturnValue(true);
    render(<KeyboardShortcutsDialog />);

    expect(await keysFor("User settings")).toBe("CtrlShiftS");
  });
});
