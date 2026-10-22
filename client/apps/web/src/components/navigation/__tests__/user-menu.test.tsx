import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { UserMenu } from "../user-menu";

const state = vi.hoisted(() => ({ offered: true }));

vi.mock("@/hooks/use-permission", () => ({
  usePageAccessOffered: () => state.offered,
}));
vi.mock("@/hooks/use-sign-out", () => ({ useSignOut: () => () => Promise.resolve() }));
vi.mock("../language-submenu", () => ({ LanguageSubmenu: () => null }));
vi.mock("@/components/resolved-user-avatar", () => ({ ResolvedUserAvatar: () => null }));

beforeEach(() => {
  state.offered = true;
  useAppDialogsStore.setState({ active: null, prepared: [] });
  useAuthStore.setState({ user: { name: "Dana Reyes" } } as never);
});

afterEach(() => {
  cleanup();
});

async function openMenu() {
  render(
    <MemoryRouter>
      <UserMenu />
    </MemoryRouter>,
  );
  await userEvent.click(screen.getByRole("button", { name: /Dana Reyes/ }));
}

describe("UserMenu", () => {
  it("offers page access, asks for its code when pointed at and opens it", async () => {
    await openMenu();
    const entry = await screen.findByRole("menuitem", { name: "Page access" });

    await userEvent.hover(entry);
    expect(useAppDialogsStore.getState()).toMatchObject({
      active: null,
      prepared: ["page-access"],
    });

    await userEvent.click(entry);
    expect(useAppDialogsStore.getState().active).toBe("page-access");
  });

  it("leaves page access out for someone it is not offered to", async () => {
    state.offered = false;
    await openMenu();

    expect(await screen.findByRole("menuitem", { name: "Settings" })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Page access" })).not.toBeInTheDocument();
  });
});
