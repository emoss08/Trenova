import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { stubLayout, stubScrollTo } from "@/test/layout";
import { DndContext } from "@dnd-kit/core";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { CoverageKanban } from "../coverage-kanban";
import type { UrgencyBucket } from "../dispatch-vocabulary";

// Fields as the board query selects them (DispatchBoardMove in dispatch_console.graphqls);
// temperatureMin and temperatureMax are null for a dry move.
function move(index: number, overrides: Partial<DispatchBoardMove> = {}): DispatchBoardMove {
  const pro = `PRO-${String(index).padStart(3, "0")}`;
  return {
    moveId: `smv_${index}`,
    shipmentId: `shp_${index}`,
    proNumber: pro,
    sequence: 0,
    moveCount: 1,
    distance: 240,
    revenue: 1_800,
    customerName: "Acme Foods",
    temperatureMin: null,
    temperatureMax: null,
    hasHazmat: false,
    hasActiveHold: false,
    urgency: "Late",
    minutesToPickup: -30,
    isCovered: false,
    originName: "Acme DC",
    originCity: "Dallas",
    originState: "TX",
    originWindowStart: 1_720_000_000,
    destinationName: "Acme Store",
    destinationCity: "Tulsa",
    destinationState: "OK",
    assignedWorkerName: "",
    assignedTractorCode: "",
    coverageType: "unassigned",
    assignedCarrierName: "",
    carrierTotalCost: null,
    liveTender: null,
    ...overrides,
  } as unknown as DispatchBoardMove;
}

function renderBoard({
  moves,
  urgencyFocus = null,
  selectedMoveId = null,
}: {
  moves: DispatchBoardMove[];
  urgencyFocus?: UrgencyBucket | null;
  selectedMoveId?: string | null;
}) {
  const ui = (selected: string | null) => (
    <DndContext>
      <CoverageKanban
        moves={moves}
        isLoading={false}
        urgencyFocus={urgencyFocus}
        selectedMoveId={selected}
        onSelectMove={() => undefined}
      />
    </DndContext>
  );
  const view = render(ui(selectedMoveId));
  return { select: (next: string | null) => view.rerender(ui(next)) };
}

// Columns are virtualized, and jsdom lays nothing out: without a size no card would mount.
let restoreLayout = () => {};
let restoreScroll = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(320);
  restoreScroll = stubScrollTo();
});

afterEach(() => {
  restoreScroll();
  restoreLayout();
  cleanup();
});

describe("coverage kanban on a long column", () => {
  const late = Array.from({ length: 200 }, (_, index) => move(index));

  it("mounts only the cards near the viewport", () => {
    renderBoard({ moves: late, urgencyFocus: "Late" });

    expect(screen.getByText("PRO-000")).toBeInTheDocument();
    expect(screen.queryByText("PRO-199")).toBeNull();
  });

  it("scrolls a move selected elsewhere into its column", async () => {
    const { select } = renderBoard({ moves: late, urgencyFocus: "Late" });
    expect(screen.queryByText("PRO-150")).toBeNull();

    select("smv_150");

    await waitFor(() => expect(screen.getByText("PRO-150")).toBeInTheDocument());
    expect(screen.queryByText("PRO-000")).toBeNull();
  });
});

describe("coverage kanban reefer mark", () => {
  it("names a reefer move's temperature range on its mark", () => {
    renderBoard({ moves: [move(1, { temperatureMin: 34, temperatureMax: 38 })] });

    expect(screen.getByRole("img", { name: "Reefer 34–38°F" })).toBeInTheDocument();
  });

  it("marks a reefer held from zero, which is set rather than absent", () => {
    renderBoard({ moves: [move(1, { temperatureMin: 0, temperatureMax: 4 })] });

    expect(screen.getByRole("img", { name: "Reefer 0–4°F" })).toBeInTheDocument();
  });

  it("leaves a dry move unmarked", () => {
    renderBoard({ moves: [move(1)] });

    expect(screen.queryByRole("img", { name: /^Reefer/ })).toBeNull();
  });

  it("keeps the range off the badge row", () => {
    renderBoard({ moves: [move(1, { temperatureMin: 34, temperatureMax: 38 })] });

    expect(screen.queryByText("34–38°F")).toBeNull();
    expect(screen.queryByText("Reefer 34–38°F")).toBeNull();
  });
});
