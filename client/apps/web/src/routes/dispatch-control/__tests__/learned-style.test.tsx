import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { DispatchStyle } from "@/lib/graphql/dispatch-style";
import { LearnedStyle } from "../_components/learned-style";

const mocks = vi.hoisted(() => ({
  fetch: vi.fn(),
  reset: vi.fn(),
  allowed: true,
}));

vi.mock("@/lib/graphql/dispatch-style", () => ({
  fetchDispatchStyle: mocks.fetch,
  resetDispatchStyle: mocks.reset,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: mocks.allowed, isLoading: false }),
}));

const empty: DispatchStyle = {
  enabled: true,
  decisions: 0,
  approvals: 0,
  overrides: 0,
  declines: 0,
  learningSince: null,
  updatedAt: null,
  weights: [],
};

const learned: DispatchStyle = {
  enabled: true,
  decisions: 312,
  approvals: 250,
  overrides: 50,
  declines: 12,
  learningSince: 1_793_000_000,
  updatedAt: 1_793_577_600,
  weights: [
    { component: "homeTime", multiplier: 1.6, samples: 312 },
    { component: "haulFit", multiplier: 0.75, samples: 1 },
    { component: "safety", multiplier: 1.02, samples: 40 },
  ],
};

function renderCard() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <LearnedStyle />
    </QueryClientProvider>,
  );
}

describe("LearnedStyle", () => {
  beforeEach(() => {
    mocks.allowed = true;
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("draws its own outline while the style loads", () => {
    mocks.fetch.mockImplementation(() => new Promise<never>(() => {}));

    renderCard();

    expect(screen.getByLabelText("Loading the learned dispatch style")).toHaveAttribute(
      "aria-busy",
      "true",
    );
    expect(mocks.fetch).toHaveBeenCalledWith({ signal: expect.any(AbortSignal) });
  });

  it("says nothing is learned yet and has nothing to reset", async () => {
    mocks.fetch.mockResolvedValue(empty);

    renderCard();

    expect(await screen.findByText(/Nothing learned yet/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reset learned style" })).toBeDisabled();
  });

  it("asks for learning to be turned on when it is off and nothing is learned", async () => {
    mocks.fetch.mockResolvedValue({ ...empty, enabled: false });

    renderCard();

    expect(
      await screen.findByText(
        "Learning is off. Turn on Learn our dispatch style to start learning.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText(/Nothing learned yet/)).toBeNull();
  });

  it("shows each learned weight against the organization's own cost", async () => {
    mocks.fetch.mockResolvedValue(learned);

    renderCard();

    const list = await screen.findByRole("list", { name: "Learned weights" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(3);
    expect(rows[0]).toHaveTextContent("Time away from home");
    expect(rows[0]).toHaveTextContent("1.6× your cost");
    expect(rows[0]).toHaveTextContent("learned from 312 decisions");
    expect(rows[1]).toHaveTextContent("Haul fit");
    expect(rows[1]).toHaveTextContent("0.75× your cost");
    expect(rows[1]).toHaveTextContent("learned from 1 decision");
    expect(rows[2]).toHaveTextContent("Safety history");
    expect(rows[2]).toHaveTextContent("As your cost");
    expect(
      screen.getByText(
        "Learned from 312 decisions: 250 took the plan's driver, 50 chose a driver it priced and 12 chose one it did not.",
      ),
    ).toBeInTheDocument();
  });

  it("keeps what was learned on view when learning is turned off", async () => {
    mocks.fetch.mockResolvedValue({ ...learned, enabled: false });

    renderCard();

    expect(
      await screen.findByText(/Learning is off, so the planner uses your own costs/),
    ).toBeInTheDocument();
    expect(screen.getByRole("list", { name: "Learned weights" })).toBeInTheDocument();
  });

  it("resets once after confirming and shows the style the server returns", async () => {
    const user = userEvent.setup();
    mocks.fetch.mockResolvedValue(learned);
    let settle: (style: DispatchStyle) => void = () => {};
    mocks.reset.mockImplementation(
      () =>
        new Promise<DispatchStyle>((resolve) => {
          settle = resolve;
        }),
    );

    renderCard();

    await user.click(await screen.findByRole("button", { name: "Reset learned style" }));
    const confirm = await screen.findByRole("button", { name: "Reset" });
    await user.click(confirm);
    await user.click(confirm);

    expect(mocks.reset).toHaveBeenCalledTimes(1);
    expect(confirm).toBeDisabled();

    settle({ ...empty, learningSince: 1_793_600_000 });

    expect(await screen.findByText(/Nothing learned yet/)).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("button", { name: "Reset" })).toBeNull());
  });

  it("offers no reset to someone who may not change dispatch control", async () => {
    mocks.allowed = false;
    mocks.fetch.mockResolvedValue(learned);

    renderCard();

    await screen.findByRole("list", { name: "Learned weights" });
    expect(screen.queryByRole("button", { name: "Reset learned style" })).toBeNull();
  });

  it("offers to try again when the style cannot be loaded", async () => {
    const user = userEvent.setup();
    mocks.fetch.mockRejectedValueOnce(new Error("boom")).mockResolvedValueOnce(empty);

    renderCard();

    await user.click(await screen.findByRole("button", { name: "Try again" }));

    expect(await screen.findByText(/Nothing learned yet/)).toBeInTheDocument();
    expect(mocks.fetch).toHaveBeenCalledTimes(2);
  });
});
