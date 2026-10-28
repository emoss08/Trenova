import type { DeskBoardCard } from "@/lib/graphql/desk-board-card";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DeskBoardBody } from "../desk-board-body";
import { block, boardArtifact, closeBoard, count, lateBoard } from "./fixtures";

const fetchers = vi.hoisted(() => ({ card: vi.fn() }));

vi.mock("@/lib/graphql/desk-board-card", () => ({ fetchDeskBoardCard: fetchers.card }));

// The shipment card is a chunk of its own; here it stands in as the record it was given,
// with its way back to the board.
vi.mock("../../../cards/shipment/desk-shipment", () => ({
  DeskShipment: ({
    recordId,
    onBack,
    backLabel,
  }: {
    recordId: string;
    onBack: () => void;
    backLabel?: string;
  }) => (
    <div>
      <p>Shipment card {recordId}</p>
      <button type="button" onClick={onBack}>
        {backLabel}
      </button>
    </div>
  ),
  DeskShipmentArtifact: () => null,
}));

function draw(card: DeskBoardCard | Promise<DeskBoardCard>) {
  fetchers.card.mockReturnValue(card instanceof Promise ? card : Promise.resolve(card));
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskBoardBody artifact={boardArtifact()} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function exhibit(name: string): HTMLElement {
  return screen.getByRole("region", { name });
}

beforeEach(() => {
  seedPermissions({ [Resource.Shipment]: Operation.Read });
  fetchers.card.mockReset();
});

afterEach(() => {
  clearPermissions();
});

describe("DeskBoardBody", () => {
  it("lays out the board it will read while its blocks are read", () => {
    const { container } = draw(new Promise<DeskBoardCard>(() => {}));

    const waiting = screen.getByLabelText("Reading the board…");
    expect(waiting).toHaveAttribute("aria-busy", "true");
    expect(
      within(waiting).getByRole("heading", {
        name: "Six loads are running late on the Dallas lanes",
      }),
    ).toBeInTheDocument();
    expect(within(waiting).getByText("Which Dallas loads are late and why?")).toBeInTheDocument();
    expect(within(waiting).getByText("Where they stand")).toBeInTheDocument();
    expect(within(waiting).getByText("The late loads")).toBeInTheDocument();
    const halves = [...container.querySelectorAll(".dk-bd-desk > *")].map((part) =>
      part.classList.contains("dk-bd-half"),
    );
    expect(halves).toEqual([false, true, true]);
    expect(fetchers.card).toHaveBeenCalledWith("athr_1", "art_board", expect.anything());
  });

  it("draws the answer with the phrases it rests on citing their exhibits", async () => {
    const { container } = draw(lateBoard());

    expect(await screen.findByText(/^Dispatch Desk answered at /u)).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Six loads are running late on the Dallas lanes" }),
    ).toBeInTheDocument();
    const verdict = container.querySelector(".dk-bd-verdict") as HTMLElement;
    expect(verdict).toHaveTextContent("Four are held at the dock1 and two are on the road.");
    expect(verdict.querySelector(".dk-bd-hl")).toHaveClass("dk-warn");
    expect(container.querySelector(".dk-bd-foot")).toHaveTextContent(
      "Reads16 shipments23 shipments3its history",
    );
  });

  it("says what a figure was when it moved since the answer, and which it could not read", async () => {
    draw(lateBoard());
    await screen.findByRole("region", { name: "Where they stand" });

    const figures = exhibit("Where they stand");
    expect(within(figures).getByText("Read from 6 shipments")).toBeInTheDocument();
    const late = within(figures).getByText("Late now").closest("div") as HTMLElement;
    expect(late).toHaveTextContent(/^Late now5→6since the answer at /u);
    expect(
      within(figures).getByText("Revenue at risk").closest("div") as HTMLElement,
    ).toHaveTextContent("Revenue at riskCould not be read");
  });

  it("lists the records with how far behind each runs, and one it may not read without naming it", async () => {
    const { container } = draw(lateBoard());
    await screen.findByRole("region", { name: "The late loads" });

    const records = exhibit("The late loads");
    expect(within(records).getByText("Read from 3 shipments · 2 you may read")).toBeInTheDocument();
    const rows = container.querySelectorAll<HTMLElement>(".dk-bd-rows > .dk-bd-row");
    expect(rows).toHaveLength(3);
    expect(rows[0]).toHaveTextContent("1h 40m behind");
    expect(rows[1]).toHaveTextContent("50m at the stop");
    expect(rows[0].querySelector(".dk-bd-track > i")).toHaveStyle({ width: "100%" });
    expect(rows[1].querySelector(".dk-bd-track > i")).toHaveStyle({ width: "50%" });
    expect(rows[1].querySelector(".dk-bd-track > i")).toHaveClass("dk-bd-held");
    expect(rows[2]).toHaveClass("dk-gone");
    expect(rows[2]).toHaveTextContent("This shipment could not be read");
  });

  it("opens a listed record in the board's place and comes back to the board", async () => {
    draw(lateBoard());
    await screen.findByRole("region", { name: "The late loads" });

    fireEvent.click(screen.getByRole("button", { name: /S-24130/u }));
    expect(await screen.findByText("Shipment card shp_2")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { level: 2 })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to the board" }));
    expect(
      await screen.findByRole("heading", { name: /Six loads are running late/u, level: 2 }),
    ).toBeInTheDocument();
  });

  it("links a record to its page for a reader without its card", async () => {
    seedPermissions({});
    draw(lateBoard());
    await screen.findByRole("region", { name: "The late loads" });

    expect(screen.queryByRole("button", { name: /S-24130/u })).not.toBeInTheDocument();
    const link = screen.getByRole("link", { name: /S-24130/u });
    expect(link.getAttribute("href")).toContain("/shipment-management/shipments");
    expect(link.getAttribute("href")).toContain("shp_2");
  });

  it("writes a note's words with the records it names, and the record's log with who acted", async () => {
    const { container } = draw(lateBoard());
    await screen.findByRole("region", { name: "What happened to S-24117" });

    const note = container.querySelector(".dk-bd-note") as HTMLElement;
    expect(note).toHaveClass("dk-warn", "dk-bd-half");
    expect(note).toHaveTextContent("Worth acting onCall the Memphis consigneeThey close at five.");
    expect(within(note).getByRole("button", { name: "S-24117" })).toBeInTheDocument();

    const log = within(exhibit("What happened to S-24117")).getAllByRole("listitem");
    expect(log).toHaveLength(2);
    expect(log[0]).toHaveTextContent("Driver reported a flat tireAna Ruiz");
    expect(log[1]).toHaveTextContent("Appointment movedAn API key");
    expect(screen.getByText("The rest are on schedule.")).toBeInTheDocument();
  });

  it("groups the checks worst first, fills the meter, and says which block could not be read", async () => {
    const { container } = draw(closeBoard());
    await screen.findByRole("region", { name: "Before the close" });

    const checks = container.querySelectorAll<HTMLElement>(".dk-bd-ck");
    expect(checks[0]).toHaveClass("dk-bad");
    expect(checks[0]).toHaveTextContent("Entries waiting to post23");
    expect(checks[0]).toHaveTextContent("Both are over a week old.");
    expect(checks[0]).toHaveTextContent("1 more you may not read");
    expect(within(checks[0]).getByRole("link", { name: "JE-1042" })).toBeInTheDocument();
    expect(checks[1]).toHaveClass("dk-ok");
    expect(checks[1]).toHaveTextContent("Bank reconciled0");

    const meter = within(exhibit("How close it is")).getByRole("meter", { name: "Checks clear" });
    expect(meter).toHaveAttribute("aria-valuenow", "1");
    expect(meter).toHaveAttribute("aria-valuemax", "2");
    expect(meter.querySelectorAll("i")).toHaveLength(2);
    expect(exhibit("How close it is")).toHaveTextContent("1of 2 checks clear50% clear");

    const gone = exhibit("Revenue by week");
    expect(gone).toHaveClass("dk-unread");
    expect(gone).toHaveTextContent("The analysis this came from could not be read");
    expect(gone.querySelector(".dk-bd-n")).toHaveClass("dk-off");
  });

  it("copies the board as text", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    draw(lateBoard());

    fireEvent.click(await screen.findByRole("button", { name: "Copy as text" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledWith(lateBoard().markdown));
    expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
  });

  it("says when the board cannot be read", async () => {
    fetchers.card.mockRejectedValue(new Error("nope"));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskBoardBody artifact={boardArtifact()} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    expect(await screen.findByText(/could not be read|nope/u)).toBeInTheDocument();
  });
});

/*
An operations board asks for five figures. The card sets them as three and
two, each row its own run, in the order they were given.
*/
describe("DeskBoardBody's figures", () => {
  it("sets five figures in a row of three and a row of two", async () => {
    const board = lateBoard();
    board.blocks = [
      block({
        type: "figures",
        title: "Today at a glance",
        numeral: 1,
        source: { kind: "metrics", resource: "", count: 0, readable: 0, title: "" },
        figures: [
          { label: "Active shipments", value: count(556), was: null, tone: "" },
          { label: "On-time %", value: count(0), was: null, tone: "" },
          { label: "At risk", value: count(54), was: null, tone: "" },
          { label: "Unassigned", value: count(337), was: null, tone: "" },
          { label: "Revenue today", value: count(0), was: null, tone: "" },
        ],
      }),
    ];
    draw(board);

    const figures = await screen.findByRole("region", { name: "Today at a glance" });
    const rows = [...figures.querySelectorAll(".dk-bd-read")].map((row) =>
      [...row.querySelectorAll("dt")].map((term) => term.textContent),
    );
    expect(rows).toEqual([
      ["Active shipments", "On-time %", "At risk"],
      ["Unassigned", "Revenue today"],
    ]);
  });
});
