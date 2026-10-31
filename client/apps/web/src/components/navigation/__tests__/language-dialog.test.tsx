import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setLocale } from "@trenova/shared/i18n/runtime";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LanguageDialog } from "../language-dialog";

const switching = vi.hoisted(() => ({
  applyLocale: vi.fn(() => Promise.resolve()),
  persistLocale: vi.fn(() => Promise.resolve()),
}));

vi.mock("@/lib/locale-switch", () => switching);
vi.mock("@/hooks/use-api-mutation", () => ({
  useApiMutation: () => ({ mutateAsync: vi.fn() }),
}));
vi.mock("@/services/api", () => ({
  apiService: { userService: { updateMySettings: vi.fn() } },
}));
vi.mock("@trenova/shared/i18n/runtime", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/i18n/runtime")>()),
  loadCatalog: vi.fn(() => Promise.resolve()),
}));

beforeEach(async () => {
  await setLocale("en");
  switching.applyLocale.mockClear();
  switching.persistLocale.mockClear();
});

afterEach(() => {
  cleanup();
});

function renderDialog() {
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <LanguageDialog open onOpenChange={onOpenChange} />
    </QueryClientProvider>,
  );
  return { onOpenChange };
}

const languageButtons = () =>
  within(screen.getByRole("list", { name: "Languages" })).getAllByRole("button");

describe("LanguageDialog", () => {
  it("lists every language in itself and in English, with English marked as current", async () => {
    renderDialog();

    expect(await screen.findByRole("dialog", { name: "Choose your language" })).toBeVisible();
    expect(
      languageButtons().map((button) => [button.textContent, button.getAttribute("aria-pressed")]),
    ).toEqual([
      ["EnglishEnglish", "true"],
      ["EspañolSpanish", "false"],
      ["繁體中文Chinese (Traditional)", "false"],
      ["简体中文Chinese (Simplified)", "false"],
    ]);
  });

  it("puts the cursor in the search and narrows the list as you type", async () => {
    renderDialog();
    const search = await screen.findByRole("searchbox", { name: "Search languages" });
    await waitFor(() => expect(search).toHaveFocus());

    await userEvent.type(search, "espanol");

    expect(languageButtons().map((button) => button.textContent)).toEqual(["EspañolSpanish"]);
  });

  it("says so when no language matches the search", async () => {
    renderDialog();

    await userEvent.type(await screen.findByRole("searchbox"), "klingon");

    expect(screen.getByText("No languages match “klingon”.")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Languages" })).not.toBeInTheDocument();
  });

  it("applies the chosen language, closes, then saves it as the preference", async () => {
    const { onOpenChange } = renderDialog();

    await userEvent.click(await screen.findByRole("button", { name: /Español/ }));

    expect(switching.applyLocale).toHaveBeenCalledWith("es");
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(switching.persistLocale).toHaveBeenCalledWith(
      expect.objectContaining({ next: "es", previous: "en" }),
    );
  });

  it("chooses the first match when Enter is pressed in the search", async () => {
    renderDialog();

    await userEvent.type(await screen.findByRole("searchbox"), "简体{Enter}");

    expect(switching.applyLocale).toHaveBeenCalledWith("zh-CN");
  });

  it("closes without switching or saving when the current language is chosen", async () => {
    const { onOpenChange } = renderDialog();

    await userEvent.click(await screen.findByRole("button", { name: /^English/ }));

    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(switching.applyLocale).not.toHaveBeenCalled();
    expect(switching.persistLocale).not.toHaveBeenCalled();
  });
});
