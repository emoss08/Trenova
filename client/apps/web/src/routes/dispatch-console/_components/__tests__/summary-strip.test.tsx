import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SummaryStrip } from "../summary-strip";

afterEach(cleanup);

function renderStrip(state: { isPending: boolean; isError: boolean }) {
  render(
    <SummaryStrip
      summary={undefined}
      proposalsWaiting={0}
      exceptions={null}
      isPending={state.isPending}
      isError={state.isError}
      urgencyFocus={null}
      onUrgencyFocus={vi.fn()}
      onShowExceptions={vi.fn()}
      onShowOpenDrivers={vi.fn()}
      showsDrivers
    />,
  );
}

describe("SummaryStrip", () => {
  it("says the figures are unknown after a failed read instead of shimmering forever", () => {
    renderStrip({ isPending: false, isError: true });

    expect(screen.getAllByText("not updated")).toHaveLength(6);
    expect(screen.queryByLabelText(/loading/i)).toBeNull();
  });
});
