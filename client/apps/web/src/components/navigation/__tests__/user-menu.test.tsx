import { renderAtRoutes, TestRouter } from "@trenova/shared/testing/test-router";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { UserMenu } from "../user-menu";

const state = vi.hoisted(() => ({
  offered: true,
  theme: "dark" as string,
  setTheme: vi.fn(),
  managed: false,
  trenovaSupport: { requestsUrl: "", newRequestUrl: "" },
}));

vi.mock("@trenova/shared/hooks/use-public-config", () => ({
  usePublicConfig: () => ({
    config: { managed: state.managed, trenovaSupport: state.trenovaSupport },
  }),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePageAccessOffered: () => state.offered,
}));
vi.mock("@/hooks/use-sign-out", () => ({ useSignOut: () => () => Promise.resolve() }));
vi.mock("../language-dialog", () => ({
  LanguageDialog: ({ open }: { open: boolean }) =>
    open ? <div role="dialog" aria-label="Choose your language" /> : null,
}));
vi.mock("@trenova/shared/components/theme-provider", () => ({
  useTheme: () => ({ theme: state.theme, setTheme: state.setTheme }),
}));
vi.mock("@trenova/shared/lib/shortcuts", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/shortcuts")>()),
  isMacPlatform: () => false,
}));
vi.mock("@/components/resolved-user-avatar", () => ({ ResolvedUserAvatar: () => null }));

beforeEach(() => {
  state.offered = true;
  state.theme = "dark";
  state.managed = false;
  state.trenovaSupport = { requestsUrl: "", newRequestUrl: "" };
  state.setTheme.mockClear();
  useAppDialogsStore.setState({ active: null, prepared: [] });
  useAuthStore.setState({ user: { name: "Dana Reyes" } } as never);
});

afterEach(() => {
  cleanup();
});

async function openMenu() {
  render(
    <TestRouter>
      <UserMenu />
    </TestRouter>,
  );
  await userEvent.click(screen.getByRole("button", { name: /Dana Reyes/ }));
}

describe("UserMenu", () => {
  it("leaves page access to its keybind, even for someone it is offered to", async () => {
    await openMenu();

    expect(await screen.findByRole("menuitem", { name: /^Settings/ })).toBeInTheDocument();
    expect(screen.queryByRole("menuitem", { name: "Page access" })).not.toBeInTheDocument();
    expect(useAppDialogsStore.getState()).toMatchObject({ active: null, prepared: [] });
  });
});

describe("UserMenu learn more", () => {
  it("links Trenova's academy, legal documents and license, each in a new tab", async () => {
    await openMenu();
    await userEvent.click(await screen.findByRole("menuitem", { name: "Learn more" }));

    const links = await screen.findAllByRole("menuitem", {
      name: /Academy|Terms|Privacy|Cloud|License/,
    });
    expect(
      links.map((link) => [
        link.textContent,
        link.getAttribute("href"),
        link.getAttribute("target"),
        link.getAttribute("rel"),
      ]),
    ).toEqual([
      ["Trenova Academy", "https://academy.trenova.app/", "_blank", "noopener noreferrer"],
      ["Terms of Service", "https://trenova.app/legal/terms/", "_blank", "noopener noreferrer"],
      ["Privacy policy", "https://trenova.app/legal/privacy/", "_blank", "noopener noreferrer"],
      [
        "Trenova Cloud terms",
        "https://trenova.app/legal/cloud-terms/",
        "_blank",
        "noopener noreferrer",
      ],
      [
        "Trenova Cloud privacy policy",
        "https://trenova.app/legal/cloud-privacy/",
        "_blank",
        "noopener noreferrer",
      ],
      [
        "License",
        "https://github.com/emoss08/Trenova/blob/master/LICENSE",
        "_blank",
        "noopener noreferrer",
      ],
    ]);
  });
});

describe("UserMenu Trenova support", () => {
  const FORM =
    "https://console.example.com/support/new?from=trenova&instance=0192f3a4-5b6c-7d8e-9f01-23456789abcd";

  it("opens the console's request form for this instance and its requests, each in a new tab", async () => {
    state.managed = true;
    state.trenovaSupport = {
      requestsUrl: "https://console.example.com/support",
      newRequestUrl: FORM,
    };
    await renderAtRoutes([{ path: "/shipments/$shipmentId", component: UserMenu }], {
      initialEntries: ["/shipments/shp_01JTESTRECORD"],
    });
    await userEvent.click(screen.getByRole("button", { name: /Dana Reyes/ }));

    // The page is the route's pattern: no record's id leaves the instance.
    const contact = await screen.findByRole("menuitem", { name: "Contact Trenova support" });
    expect(contact.getAttribute("href")).toBe(`${FORM}&page=%2Fshipments%2F%24shipmentId`);
    expect(contact.getAttribute("target")).toBe("_blank");
    expect(contact.getAttribute("rel")).toBe("noopener noreferrer");

    const requests = screen.getByRole("menuitem", { name: "Your support requests" });
    expect(requests.getAttribute("href")).toBe("https://console.example.com/support");
    expect(requests.getAttribute("target")).toBe("_blank");
  });

  it("offers only the link the control plane sent", async () => {
    state.managed = true;
    state.trenovaSupport = {
      requestsUrl: "https://console.example.com/support",
      newRequestUrl: "",
    };
    await openMenu();

    expect(await screen.findByRole("menuitem", { name: "Your support requests" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "Contact Trenova support" })).toBeNull();
  });

  it("offers nothing on a self-hosted instance, whatever its config holds", async () => {
    state.trenovaSupport = {
      requestsUrl: "https://console.example.com/support",
      newRequestUrl: FORM,
    };
    await openMenu();

    expect(await screen.findByRole("menuitem", { name: "Log out" })).toBeTruthy();
    expect(screen.queryByRole("menuitem", { name: "Contact Trenova support" })).toBeNull();
    expect(screen.queryByRole("menuitem", { name: "Your support requests" })).toBeNull();
  });
});

describe("UserMenu language", () => {
  it("opens the language dialog in place of a submenu", async () => {
    await openMenu();
    const item = await screen.findByRole("menuitem", { name: "Language" });
    expect(screen.queryByRole("dialog", { name: "Choose your language" })).not.toBeInTheDocument();

    await userEvent.click(item);

    expect(screen.getByRole("dialog", { name: "Choose your language" })).toBeInTheDocument();
  });
});

describe("UserMenu keybinds", () => {
  it("shows each item's keybind beside it, and none where there is no keybind", async () => {
    await openMenu();

    expect(await screen.findByRole("menuitem", { name: /Settings/ })).toHaveTextContent(
      "SettingsCtrl+Shift+S",
    );
    expect(screen.getByRole("menuitem", { name: /Keyboard shortcuts/ })).toHaveTextContent(
      "Keyboard shortcutsCtrl+/",
    );
    expect(screen.getByRole("menuitem", { name: "Language" })).toHaveTextContent(/^Language$/);
    expect(screen.getByRole("menuitem", { name: "Log out" })).toHaveTextContent(/^Log out$/);
  });
});

describe("UserMenu keyboard shortcuts", () => {
  it("opens the keyboard shortcuts dialog", async () => {
    await openMenu();

    await userEvent.click(await screen.findByRole("menuitem", { name: /Keyboard shortcuts/ }));

    expect(useAppDialogsStore.getState().active).toBe("shortcuts");
  });
});

describe("UserMenu theme", () => {
  it("marks the theme in use and switches to the one chosen", async () => {
    await openMenu();
    await userEvent.click(await screen.findByRole("menuitem", { name: "Theme" }));

    const choices = await screen.findAllByRole("menuitemradio");
    expect(
      choices.map((choice) => [choice.textContent, choice.getAttribute("aria-checked")]),
    ).toEqual([
      ["Light", "false"],
      ["Dark", "true"],
      ["System", "false"],
    ]);

    await userEvent.click(screen.getByRole("menuitemradio", { name: "Light" }));

    expect(state.setTheme).toHaveBeenCalledWith("light");
  });
});
