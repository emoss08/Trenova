import type { RosterStepper as RosterStepperState } from "@/routes/worker/roster-context-store";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RosterStepper } from "../roster-stepper";

const mocks = vi.hoisted(() => ({
  navigate: vi.fn(),
  stepper: null as unknown as RosterStepperState,
}));

vi.mock("@tanstack/react-router", () => ({ useNavigate: () => mocks.navigate }));
vi.mock("@/routes/worker/roster-context-store", () => ({
  useRosterStepper: () => mocks.stepper,
  useRosterContextStore: { getState: () => ({ context: null, setContext: vi.fn() }) },
}));

function stepper(overrides: Partial<RosterStepperState> = {}): RosterStepperState {
  return {
    available: true,
    position: 3,
    total: 14,
    label: "Workers missing a medical card",
    rosterSearch: "",
    canPrevious: true,
    canNext: true,
    stepping: null,
    step: vi.fn(async (direction) => (direction === "next" ? "wrk_4" : "wrk_2")),
    ...overrides,
  };
}

beforeEach(() => {
  mocks.navigate.mockReset().mockResolvedValue(undefined);
});
afterEach(cleanup);

describe("the roster stepper", () => {
  it("says where the worker stands in the roster", () => {
    mocks.stepper = stepper();
    render(<RosterStepper workerId="wrk_3" tab="overview" view={null} />);

    expect(screen.getByText("3 of 14")).toBeTruthy();
  });

  it("opens the next worker on the same tab and view", async () => {
    mocks.stepper = stepper();
    render(<RosterStepper workerId="wrk_3" tab="qualification" view="dq-file" />);

    fireEvent.click(screen.getByRole("button", { name: "Next worker" }));

    await waitFor(() =>
      expect(mocks.navigate).toHaveBeenCalledWith({
        to: "/hr/workers/wrk_4?tab=qualification&view=dq-file",
      }),
    );
    expect(mocks.stepper.step).toHaveBeenCalledWith("next");
  });

  it("holds the first worker's previous button", () => {
    mocks.stepper = stepper({ canPrevious: false, position: 1 });
    render(<RosterStepper workerId="wrk_1" tab="overview" view={null} />);

    expect(screen.getByRole("button", { name: "Previous worker" })).toHaveProperty(
      "disabled",
      true,
    );
  });

  it("holds both buttons while a step is reading the next page", () => {
    mocks.stepper = stepper({ stepping: "next" });
    render(<RosterStepper workerId="wrk_3" tab="overview" view={null} />);

    expect(screen.getByRole("button", { name: "Next worker" })).toHaveProperty("disabled", true);
    expect(screen.getByRole("button", { name: "Previous worker" })).toHaveProperty(
      "disabled",
      true,
    );
  });

  it("draws nothing on a cold link, where no roster is in memory", () => {
    mocks.stepper = stepper({ available: false });
    const { container } = render(<RosterStepper workerId="wrk_3" tab="overview" view={null} />);

    expect(container.innerHTML).toBe("");
  });
});
