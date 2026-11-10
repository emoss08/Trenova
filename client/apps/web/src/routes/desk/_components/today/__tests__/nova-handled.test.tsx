import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { NovaHandled } from "../nova-handled";

const NOW = 1_790_000_000;

function entry(overrides: Partial<DeskActivityEntry>): DeskActivityEntry {
  return {
    id: "ap_1",
    outcome: "OnItsOwn",
    at: NOW - 600,
    agent: null,
    runId: "arun_1",
    planId: null,
    title: "Filed 11 rate confirmations",
    detail: "",
    count: 11,
    records: [],
    recordsMore: 0,
    decidedBy: null,
    proposalIds: ["ap_1"],
    ...overrides,
  };
}

function draw(entries: DeskActivityEntry[]) {
  return render(
    <MemoryRouter>
      <NovaHandled entries={entries} timezone="America/Chicago" />
    </MemoryRouter>,
  );
}

afterEach(cleanup);

describe("Nova handled", () => {
  it("labels how each write ended and who stood behind it", () => {
    draw([
      entry({ id: "a", outcome: "OnItsOwn" }),
      entry({ id: "b", outcome: "Approved", decidedBy: { id: "usr_2", name: "Jordan Ellis" } }),
      entry({ id: "c", outcome: "Previewed" }),
      entry({ id: "d", outcome: "Failed", detail: "The period is closed for posting." }),
    ]);

    const rows = screen.getAllByRole("link");
    expect(within(rows[0]).getByText("Done on its own")).toBeInTheDocument();
    expect(within(rows[0]).getByText("Ran under AI control")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Done")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Approved by Jordan Ellis")).toBeInTheDocument();
    expect(within(rows[2]).getByText("Previewed only")).toBeInTheDocument();
    expect(within(rows[3]).getByText("Didn't go through")).toBeInTheDocument();
    expect(within(rows[3]).getByText("The period is closed for posting.")).toBeInTheDocument();
  });

  it("opens each line on its own page, by the id of the write it folds", () => {
    draw([entry({ id: "ap_9" })]);

    expect(screen.getByRole("link")).toHaveAttribute("href", "/desk/handled/ap_9");
  });

  it("names the records the reader may open and counts the rest", () => {
    draw([
      entry({
        records: [
          {
            kind: "shipment",
            noun: "Shipment",
            id: "shp_1",
            label: "S-240318",
            subtitle: "",
            status: "",
            readable: true,
          },
          {
            kind: "shipment",
            noun: "Shipment",
            id: "shp_2",
            label: "",
            subtitle: "",
            status: "",
            readable: false,
          },
        ],
        recordsMore: 2,
      }),
    ]);

    expect(screen.getByText("S-240318 and 3 more")).toBeInTheDocument();
  });

  it("puts the settled time in the gutter on the reader's clock", () => {
    draw([entry({ at: 1_790_000_000 })]);

    expect(screen.getByRole("link").querySelector("time")).toHaveTextContent(/\d{1,2}:\d{2}/u);
  });

  it("shows the first six lines and the rest it read behind one line", () => {
    draw(Array.from({ length: 9 }, (_, index) => entry({ id: `ap_${index}` })));

    expect(screen.getAllByRole("link")).toHaveLength(6);
    fireEvent.click(screen.getByRole("button", { name: "3 more" }));

    expect(screen.getAllByRole("link")).toHaveLength(9);
    expect(screen.queryByRole("button", { name: /more/u })).toBeNull();
  });

  it("offers no more line when every line it read is shown", () => {
    draw(Array.from({ length: 6 }, (_, index) => entry({ id: `ap_${index}` })));

    expect(screen.queryByRole("button")).toBeNull();
  });
});
