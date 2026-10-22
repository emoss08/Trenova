import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Resource } from "@trenova/shared/types/permission";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { KeyboardShortcutsDialog } from "../keyboard-shortcuts-dialog";

const allowed = vi.fn<(resource: string) => boolean>();

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({ allowed: allowed(resource), isLoading: false }),
}));

beforeEach(() => {
  vi.stubEnv("DEV", false);
  useAppDialogsStore.setState({ active: "shortcuts" });
});

afterEach(() => {
  cleanup();
  allowed.mockReset();
  vi.unstubAllEnvs();
});

describe("KeyboardShortcutsDialog", () => {
  it("lists the page access shortcut for whoever may read roles", async () => {
    allowed.mockImplementation((resource) => resource === Resource.Role);
    render(<KeyboardShortcutsDialog />);

    expect(await screen.findByText("Page access")).toBeInTheDocument();
  });

  it("leaves the page access shortcut out for everyone else", async () => {
    allowed.mockReturnValue(false);
    render(<KeyboardShortcutsDialog />);

    expect(await screen.findByRole("dialog", { name: "Keyboard shortcuts" })).toBeInTheDocument();
    expect(screen.queryByText("Page access")).not.toBeInTheDocument();
  });
});
