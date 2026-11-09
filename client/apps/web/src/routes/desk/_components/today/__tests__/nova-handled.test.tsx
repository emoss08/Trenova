import type { DeskActivityEntry } from "@/lib/graphql/desk-attention";
import { cleanup, render, screen, within } from "@testing-library/react";
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
      <NovaHandled entries={entries} now={NOW} timezone="America/Chicago" />
    </MemoryRouter>,
  );
}

afterEach(cleanup);

describe("Nova handled", () => {
  it("says how each write ended in words of its own", () => {
    draw([
      entry({ id: "a", outcome: "OnItsOwn" }),
      entry({ id: "b", outcome: "Approved", decidedBy: { id: "usr_2", name: "Jordan Ellis" } }),
      entry({ id: "c", outcome: "Approved", decidedBy: null }),
      entry({ id: "d", outcome: "Previewed" }),
      entry({ id: "e", outcome: "Failed", detail: "The period is closed for posting." }),
    ]);

    expect(screen.getByText("Ran on its own")).toBeInTheDocument();
    expect(screen.getByText("Approved by Jordan Ellis")).toBeInTheDocument();
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("Previewed only")).toBeInTheDocument();
    expect(screen.getByText("Didn't go through")).toBeInTheDocument();
    expect(screen.getByText("The period is closed for posting.")).toBeInTheDocument();
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

    const row = screen.getByRole("link");
    expect(row).toHaveAttribute("href", "/desk/r/shipment/shp_1");
    expect(within(row).getByText("S-240318 and 3 more")).toBeInTheDocument();
  });

  it("opens nothing when no record it touched may be read", () => {
    draw([entry({ records: [], recordsMore: 0 })]);

    expect(screen.queryByRole("link")).toBeNull();
    expect(screen.getByText("Filed 11 rate confirmations")).toBeInTheDocument();
  });
});
