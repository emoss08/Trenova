import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Resource } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";

const allowed = vi.fn<(resource: string) => boolean>();

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
  vi.unstubAllEnvs();
});

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
});
