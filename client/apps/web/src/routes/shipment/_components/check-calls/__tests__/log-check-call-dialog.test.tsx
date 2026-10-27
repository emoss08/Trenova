import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { LogCheckCallDialog } from "../log-check-call-dialog";

vi.mock("@/lib/graphql/check-calls", () => ({
  logCheckCall: vi.fn(),
}));

const NOTE =
  "Arrivals and departures logged now won't change stop times — this load is in the billing queue.";

function renderDialog(stopTimesNote: string | null) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <LogCheckCallDialog
        open
        onOpenChange={vi.fn()}
        shipmentId="shp_1"
        moves={[{ id: "sm_1", label: "Move 1" }]}
        stops={[{ id: "stp_1", moveId: "sm_1", label: "1. Pickup · Origin" }]}
        defaultMoveId="sm_1"
        stopTimesNote={stopTimesNote}
      />
    </QueryClientProvider>,
  );
}

describe("LogCheckCallDialog", () => {
  it("raises the note only once an arrival or departure is chosen", async () => {
    const user = userEvent.setup();
    renderDialog(NOTE);

    expect(screen.getByText("Record what the carrier or driver told you.")).toBeInTheDocument();
    expect(screen.queryByText(NOTE)).not.toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Arrived" }));
    expect(screen.getByText(NOTE)).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Departed" }));
    expect(screen.getByText(NOTE)).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Position update" }));
    expect(screen.queryByText(NOTE)).not.toBeInTheDocument();
  });

  it("promises the stop's actual time when the server will record it", async () => {
    const user = userEvent.setup();
    renderDialog(null);

    expect(
      screen.getByText(
        "Record what the carrier or driver told you. An arrival or departure also records the stop's actual time.",
      ),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("radio", { name: "Arrived" }));
    expect(screen.queryByText(/won't change stop times/)).not.toBeInTheDocument();
  });
});
