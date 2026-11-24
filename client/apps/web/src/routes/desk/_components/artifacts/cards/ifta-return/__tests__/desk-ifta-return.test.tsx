import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { money } from "../../../record-card/record-card-values";
import { DeskIftaReturn } from "../desk-ifta-return";
import { draftCard } from "./fixtures";

const fetcher = vi.hoisted(() => ({ fetchDeskIftaReturnCard: vi.fn() }));

vi.mock("@/lib/graphql/ifta-return-desk-card", () => fetcher);

function draw(recordId = "ifr_q3") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <DeskIftaReturn recordId={recordId} onBack={vi.fn()} />
      </TestRouter>
    </QueryClientProvider>,
  );
}

function section(title: string): HTMLElement {
  const heading = screen.getByRole("heading", { name: title, level: 3 });
  const found = heading.closest("section");
  if (!found) throw new Error(`No section titled ${title}`);
  return found;
}

beforeEach(() => {
  fetcher.fetchDeskIftaReturnCard
    .mockReset()
    .mockImplementation(async (id: string) => draftCard({ id }));
});

describe("DeskIftaReturn", () => {
  it("holds the card's outline while the return is read, then draws what blocks it and the lines", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the IFTA return…");
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();

    expect(await screen.findByRole("heading", { name: "Q3 2026", level: 2 })).toBeInTheDocument();
    expect(fetcher.fetchDeskIftaReturnCard).toHaveBeenCalledWith(
      "ifr_q3",
      expect.objectContaining({}),
    );
    expect(screen.getByText(money("258.26"), { selector: "b" })).toBeInTheDocument();
    const checks = section("Before it can be filed");
    expect(within(checks).getByText("Missing tax rate")).toBeInTheDocument();
    expect(within(checks).getByText("Past the filing date")).toBeInTheDocument();
    const lines = section("Net tax by jurisdiction");
    expect(within(lines).getByText("Oklahoma")).toBeInTheDocument();
    expect(within(lines).getByText("Rate missing")).toBeInTheDocument();
    expect(within(lines).getByText(`${money("1352.40")} credit`)).toBeInTheDocument();
    expect(screen.queryByText(/ifrl_/u)).toBeNull();
    expect(screen.getByRole("link", { name: "Open IFTA return" })).toHaveAttribute(
      "href",
      "/fuel/ifta-returns?return=ifr_q3",
    );
  });

  it("says a return with no lines has none yet", async () => {
    fetcher.fetchDeskIftaReturnCard.mockResolvedValue(draftCard({ lines: [] }));
    draw();

    expect(
      await screen.findByText("The return has no jurisdiction lines yet."),
    ).toBeInTheDocument();
  });

  it("says the return could not be read when the server refuses it", async () => {
    fetcher.fetchDeskIftaReturnCard.mockRejectedValue(new Error("not found"));
    draw();

    expect(await screen.findByText("This IFTA return could not be read")).toBeInTheDocument();
  });
});
