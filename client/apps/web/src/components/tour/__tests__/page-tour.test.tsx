import type { PageTourOutcome, PageTourView } from "@/lib/graphql/page-tour";
import { pageTour } from "@/lib/queries/page-tour";
import { usePageTourStore } from "@/stores/page-tour-store";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Popover, PopoverContent } from "@trenova/shared/components/ui/popover";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { PageTour } from "../page-tour";
import { tourTarget, type TourDefinition } from "../tour-model";
import { TourReplayButton } from "../tour-replay-button";

type RecordInput = { tour: string; tourVersion: number; outcome: PageTourOutcome };

const mocks = vi.hoisted(() => ({
  views: vi.fn<() => Promise<PageTourView[]>>(),
  record: vi.fn<(input: RecordInput) => Promise<PageTourView>>(),
  reportFailure: vi.fn(),
}));

vi.mock("@/lib/graphql/page-tour", () => ({
  fetchMyPageTourViews: mocks.views,
  recordMyPageTour: mocks.record,
}));

vi.mock("@trenova/shared/lib/failure-reporter", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/failure-reporter")>();
  return { ...actual, reportFailure: mocks.reportFailure };
});

const SEEN_AT = 1_794_000_000;

const tour: TourDefinition = {
  key: "test-page",
  version: 2,
  steps: () => [
    { title: "Welcome", body: "What this page is for." },
    { target: "figures", title: "The figures", body: "The numbers that matter." },
    { target: "absent", title: "A hidden section", body: "Not on this page." },
    { target: "moves", title: "The moves", body: "What to do about it.", side: "left" },
  ],
};

function view(tourVersion: number, outcome: PageTourOutcome = "Finished"): PageTourView {
  return { tour: tour.key, tourVersion, outcome, seenAt: SEEN_AT } as PageTourView;
}

const bothParts = (
  <>
    <div {...tourTarget("figures")}>Figures</div>
    <div {...tourTarget("moves")}>Moves</div>
  </>
);

function renderTour(options: { ready?: boolean; parts?: ReactNode } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const result = render(
    <QueryClientProvider client={queryClient}>
      {options.parts ?? bothParts}
      <PageTour tour={tour} ready={options.ready} />
    </QueryClientProvider>,
  );
  return { queryClient, ...result };
}

function part(name: string): HTMLElement {
  return document.querySelector<HTMLElement>(`[data-tour="${name}"]`)!;
}

function card() {
  return screen.findByRole("dialog");
}

function noCard() {
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
}

describe("PageTour", () => {
  beforeEach(() => {
    mocks.views.mockReset().mockResolvedValue([]);
    mocks.record
      .mockReset()
      .mockImplementation(async (input) => view(input.tourVersion, input.outcome));
    mocks.reportFailure.mockReset();
    usePageTourStore.setState({
      available: [],
      requested: null,
      showing: null,
      always: false,
    });
  });

  it("opens at its first step the first time a person reaches the page", async () => {
    renderTour();

    const dialog = await card();

    expect(dialog).toHaveAccessibleName("Welcome");
    expect(dialog).toHaveAccessibleDescription("What this page is for.");
    expect(dialog).toHaveTextContent("Step 1 of 4");
    expect(screen.getByRole("button", { name: "Next" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Back" })).not.toBeInTheDocument();
    expect(mocks.record).not.toHaveBeenCalled();
  });

  it("walks forward and back one step at a time", async () => {
    const user = userEvent.setup();
    renderTour();
    const dialog = await card();

    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(dialog).toHaveTextContent("The figures");
    expect(dialog).toHaveTextContent("Step 2 of 4");

    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(dialog).toHaveTextContent("Welcome");
    expect(dialog).toHaveTextContent("Step 1 of 4");
  });

  it("moves with the arrow keys", async () => {
    renderTour();
    const dialog = await card();

    fireEvent.keyDown(dialog, { key: "ArrowRight" });
    expect(dialog).toHaveTextContent("The figures");

    fireEvent.keyDown(dialog, { key: "ArrowLeft" });
    expect(dialog).toHaveTextContent("Welcome");

    fireEvent.keyDown(dialog, { key: "ArrowLeft" });
    expect(dialog).toHaveTextContent("Welcome");
    expect(mocks.record).not.toHaveBeenCalled();
  });

  it("passes over a step whose target is not on the page, in both directions", async () => {
    const user = userEvent.setup();
    renderTour();
    const dialog = await card();

    await user.click(screen.getByRole("button", { name: "Next" }));
    await user.click(screen.getByRole("button", { name: "Next" }));
    expect(dialog).toHaveTextContent("The moves");
    expect(dialog).toHaveTextContent("Step 4 of 4");
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Back" }));
    expect(dialog).toHaveTextContent("The figures");
  });

  it("passes over a part inside a folded panel, which is on the page but switched off", async () => {
    const user = userEvent.setup();
    renderTour({
      parts: (
        <>
          <div inert>
            <div {...tourTarget("figures")}>Figures</div>
          </div>
          <div {...tourTarget("moves")}>Moves</div>
        </>
      ),
    });
    const dialog = await card();

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(dialog).toHaveTextContent("The moves");
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
  });

  it("tells its page while it is open, so the page can draw every part", async () => {
    const user = userEvent.setup();
    renderTour();
    expect(usePageTourStore.getState().showing).toBeNull();

    await card();
    expect(usePageTourStore.getState().showing).toBe("test-page");

    await user.keyboard("{Escape}");
    expect(usePageTourStore.getState().showing).toBeNull();
  });

  it("brings a part that fits to the middle of the window", async () => {
    const user = userEvent.setup();
    renderTour();
    await card();
    const scrollIntoView = vi.fn();
    part("figures").scrollIntoView = scrollIntoView;

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(scrollIntoView).toHaveBeenCalledExactlyOnceWith({
      block: "center",
      inline: "nearest",
      behavior: "smooth",
    });
  });

  it("moves a list longer than the box it scrolls in only as far as it must", async () => {
    const user = userEvent.setup();
    renderTour({
      parts: (
        <div style={{ overflow: "auto" }} data-testid="scroller">
          <div {...tourTarget("figures")}>Figures</div>
        </div>
      ),
    });
    await card();
    screen.getByTestId("scroller").getBoundingClientRect = () =>
      DOMRect.fromRect({ x: 0, y: 300, width: 900, height: 300 });
    part("figures").getBoundingClientRect = () =>
      DOMRect.fromRect({ x: 0, y: 300, width: 900, height: 2000 });
    const scrollIntoView = vi.fn();
    part("figures").scrollIntoView = scrollIntoView;

    await user.click(screen.getByRole("button", { name: "Next" }));

    expect(scrollIntoView).toHaveBeenCalledExactlyOnceWith({
      block: "nearest",
      inline: "nearest",
      behavior: "smooth",
    });
  });

  it("picks up a part drawn after its step opened", async () => {
    const user = userEvent.setup();
    const { rerender, queryClient } = renderTour({ parts: <div>Nothing yet</div> });
    const dialog = await card();
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();

    rerender(
      <QueryClientProvider client={queryClient}>
        {bothParts}
        <PageTour tour={tour} />
      </QueryClientProvider>,
    );
    await user.click(await screen.findByRole("button", { name: "Next" }));

    expect(dialog).toHaveTextContent("The figures");
  });

  it("remembers a finished tour at its version and closes", async () => {
    const user = userEvent.setup();
    const { queryClient } = renderTour();
    await card();

    await user.click(screen.getByRole("button", { name: "Next" }));
    await user.click(screen.getByRole("button", { name: "Next" }));
    await user.click(screen.getByRole("button", { name: "Done" }));

    noCard();
    expect(mocks.record).toHaveBeenCalledExactlyOnceWith({
      tour: "test-page",
      tourVersion: 2,
      outcome: "Finished",
    });
    await waitFor(() =>
      expect(queryClient.getQueryData(pageTour.views().queryKey)).toEqual([view(2, "Finished")]),
    );
    noCard();
  });

  it("remembers a tour skipped from its first step", async () => {
    const user = userEvent.setup();
    renderTour();
    const dialog = await card();

    await user.click(
      Array.from(dialog.querySelectorAll("button")).find(
        (button) => button.textContent === "Skip tour",
      )!,
    );

    noCard();
    expect(mocks.record).toHaveBeenCalledExactlyOnceWith({
      tour: "test-page",
      tourVersion: 2,
      outcome: "Skipped",
    });
  });

  it("remembers a tour closed from its corner or with Escape as skipped", async () => {
    const user = userEvent.setup();
    const first = renderTour();
    const dialog = await card();
    await user.click(screen.getByRole("button", { name: "Next" }));

    await user.click(
      Array.from(dialog.querySelectorAll("button")).find(
        (button) => button.getAttribute("aria-label") === "Skip tour",
      )!,
    );

    noCard();
    expect(mocks.record).toHaveBeenLastCalledWith({
      tour: "test-page",
      tourVersion: 2,
      outcome: "Skipped",
    });
    first.unmount();

    renderTour();
    await card();
    await user.keyboard("{Escape}");

    noCard();
    expect(mocks.record).toHaveBeenCalledTimes(2);
    expect(mocks.record).toHaveBeenLastCalledWith({
      tour: "test-page",
      tourVersion: 2,
      outcome: "Skipped",
    });
  });

  it("stays open when the dimmed page is pressed", async () => {
    const user = userEvent.setup();
    const { container } = renderTour();
    const dialog = await card();

    await user.click(container.ownerDocument.querySelector('[data-slot="tour-spotlight"]')!);

    expect(dialog).toBeInTheDocument();
    expect(dialog).toHaveTextContent("Welcome");
    expect(mocks.record).not.toHaveBeenCalled();
  });

  it("stays closed for a person who has seen this version, finished or skipped", async () => {
    mocks.views.mockResolvedValue([{ ...view(9), tour: "another-page" }, view(2, "Skipped")]);
    const { queryClient } = renderTour();

    await waitFor(() =>
      expect(queryClient.getQueryState(pageTour.views().queryKey)?.status).toBe("success"),
    );

    noCard();
    expect(document.querySelector('[data-slot="tour-spotlight"]')).toBeNull();
  });

  it("opens on every visit for a developer who switched that on, seen or not", async () => {
    mocks.views.mockResolvedValue([view(2)]);
    usePageTourStore.setState({ always: true });
    renderTour();

    expect(await card()).toHaveTextContent("Welcome");
  });

  it("opens at once when that switch is turned on with the page already open", async () => {
    mocks.views.mockResolvedValue([view(2)]);
    const { queryClient } = renderTour();
    await waitFor(() =>
      expect(queryClient.getQueryState(pageTour.views().queryKey)?.status).toBe("success"),
    );
    noCard();

    act(() => {
      window.tour!.always();
    });

    expect(await card()).toHaveTextContent("Welcome");
  });

  it("opens again for a person who saw an older version", async () => {
    mocks.views.mockResolvedValue([view(1)]);
    renderTour();

    expect(await card()).toHaveTextContent("Welcome");
  });

  it("stays closed when what a person has seen cannot be read", async () => {
    mocks.views.mockRejectedValue(new Error("unreachable"));
    const { queryClient } = renderTour();

    await waitFor(() =>
      expect(queryClient.getQueryState(pageTour.views().queryKey)?.status).toBe("error"),
    );

    noCard();
  });

  it("neither reads nor opens until its page is ready", async () => {
    const { rerender, queryClient } = renderTour({ ready: false });

    expect(mocks.views).not.toHaveBeenCalled();
    expect(usePageTourStore.getState().available).toEqual([]);
    noCard();

    rerender(
      <QueryClientProvider client={queryClient}>
        <div {...tourTarget("figures")}>Figures</div>
        <div {...tourTarget("moves")}>Moves</div>
        <PageTour tour={tour} ready />
      </QueryClientProvider>,
    );

    expect(await card()).toHaveTextContent("Welcome");
    expect(usePageTourStore.getState().available).toEqual(["test-page"]);
  });

  it("replays for a person who has seen it, when asked, and records the replay", async () => {
    const user = userEvent.setup();
    mocks.views.mockResolvedValue([view(2)]);
    const { queryClient } = renderTour();
    await waitFor(() =>
      expect(queryClient.getQueryState(pageTour.views().queryKey)?.status).toBe("success"),
    );
    noCard();

    act(() => usePageTourStore.getState().request("another-page"));
    noCard();

    act(() => usePageTourStore.getState().request("test-page"));
    expect(await card()).toHaveTextContent("Welcome");

    await user.keyboard("{Escape}");

    noCard();
    expect(usePageTourStore.getState().requested).toBeNull();
    expect(mocks.record).toHaveBeenCalledExactlyOnceWith({
      tour: "test-page",
      tourVersion: 2,
      outcome: "Skipped",
    });

    act(() => usePageTourStore.getState().request("test-page"));
    expect(await card()).toHaveTextContent("Step 1 of 4");
  });

  it("stays closed and reports it when the tour cannot be remembered", async () => {
    const user = userEvent.setup();
    const failure = new Error("write failed");
    mocks.record.mockRejectedValue(failure);
    renderTour();
    await card();

    await user.keyboard("{Escape}");

    await waitFor(() =>
      expect(mocks.reportFailure).toHaveBeenCalledExactlyOnceWith(failure, {
        source: "background",
        operation: "recordPageTour",
      }),
    );
    noCard();
  });

  it("withdraws its tour when the page leaves", async () => {
    const { unmount } = renderTour();
    await card();
    act(() => usePageTourStore.getState().request("test-page"));

    unmount();

    expect(usePageTourStore.getState()).toMatchObject({ available: [], requested: null });
  });
});

describe("TourReplayButton", () => {
  beforeEach(() => {
    usePageTourStore.setState({
      available: [],
      requested: null,
      showing: null,
      always: false,
    });
  });

  function renderNote() {
    return render(
      <Popover open>
        <PopoverContent>
          <TourReplayButton tour="test-page" />
        </PopoverContent>
      </Popover>,
    );
  }

  it("is not offered while the page's tour cannot run", () => {
    usePageTourStore.setState({ available: ["another-page"] });
    renderNote();

    expect(screen.queryByRole("button", { name: "Take the tour" })).not.toBeInTheDocument();
  });

  it("asks the page to replay its tour", async () => {
    const user = userEvent.setup();
    usePageTourStore.setState({ available: ["test-page"] });
    renderNote();

    await user.click(await screen.findByRole("button", { name: "Take the tour" }));

    expect(usePageTourStore.getState().requested).toBe("test-page");
  });
});
