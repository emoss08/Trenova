import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TooltipProvider } from "@trenova/shared/components/ui/tooltip";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FavoriteToggle } from "../header";

const mocks = vi.hoisted(() => ({
  check: vi.fn<(pageUrl: string) => Promise<{ favorited: boolean }>>(),
  toggle: vi.fn<(values: { pageUrl: string; pageTitle: string }) => Promise<unknown>>(),
}));

vi.mock("@/services/page-favorite", () => ({
  pageFavoriteService: {
    checkPageFavorite: mocks.check,
    togglePageFavorite: mocks.toggle,
    listPageFavorites: vi.fn().mockResolvedValue([]),
  },
}));

vi.mock("@/hooks/use-breadcrumb", () => ({ useBreadcrumbs: () => [] }));

function renderStar() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter initialEntries={["/dispatch/outlook"]}>
          <FavoriteToggle />
        </MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

describe("FavoriteToggle", () => {
  beforeEach(() => {
    mocks.check.mockReset().mockResolvedValue({ favorited: false });
    mocks.toggle.mockReset().mockImplementation(() => new Promise(() => {}));
  });

  it("is a plain outlined button, as tall as the page bar's other controls, on a page that is not a favorite", async () => {
    renderStar();

    const star = await screen.findByRole("button", { name: "Add to favorites" });
    await waitFor(() => expect(star).toBeEnabled());

    expect(star).toHaveClass("size-7", "border");
    expect(star).not.toHaveClass("bg-warning-subtle");
    expect(star.querySelector("svg")).not.toHaveClass("fill-warning");
  });

  it("takes the star's colour, muted and bordered, on a page that is a favorite", async () => {
    mocks.check.mockResolvedValue({ favorited: true });
    renderStar();

    const star = await screen.findByRole("button", { name: "Remove from favorites" });

    expect(star).toHaveClass(
      "size-7",
      "border-warning-border",
      "bg-warning-subtle",
      "text-warning-foreground",
    );
    expect(star.querySelector("svg")).toHaveClass("fill-warning");
  });

  it("does not replay the star landing when a favorite page is opened", async () => {
    mocks.check.mockResolvedValue({ favorited: true });
    renderStar();

    const star = await screen.findByRole("button", { name: "Remove from favorites" });

    expect(star.querySelector("svg")).not.toHaveClass("animate-confirm");
  });

  it("turns at once when pressed, before the server answers, and the star lands", async () => {
    const user = userEvent.setup();
    renderStar();
    const star = await screen.findByRole("button", { name: "Add to favorites" });
    await waitFor(() => expect(star).toBeEnabled());

    await user.click(star);

    expect(star).toHaveAccessibleName("Remove from favorites");
    expect(star).toHaveClass("bg-warning-subtle", "border-warning-border");
    expect(star.querySelector("svg")).toHaveClass("fill-warning", "animate-confirm");
    expect(star).toBeEnabled();
    expect(star).toHaveAttribute("aria-busy", "true");
    expect(mocks.toggle).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ pageUrl: "/dispatch/outlook" }),
    );
  });

  it("refuses a second press while the first is still on its way", async () => {
    const user = userEvent.setup();
    renderStar();
    const star = await screen.findByRole("button", { name: "Add to favorites" });
    await waitFor(() => expect(star).toBeEnabled());

    await user.click(star);
    await user.click(star);

    expect(mocks.toggle).toHaveBeenCalledTimes(1);
    expect(star).toHaveAccessibleName("Remove from favorites");
  });

  it("goes back to plain at once when a favorite is removed, with no landing", async () => {
    const user = userEvent.setup();
    mocks.check.mockResolvedValue({ favorited: true });
    renderStar();
    const star = await screen.findByRole("button", { name: "Remove from favorites" });

    await user.click(star);

    expect(star).toHaveAccessibleName("Add to favorites");
    expect(star).not.toHaveClass("bg-warning-subtle");
    expect(star.querySelector("svg")).not.toHaveClass("fill-warning");
    expect(star.querySelector("svg")).not.toHaveClass("animate-confirm");
  });
});
