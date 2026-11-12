import type { DeskAttentionGroup, DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { watchtowerItem } from "../../watchtower/__tests__/item-fixture";
import { AttentionRow } from "../attention-row";

const NOW = 1_800_000_000;

function item(overrides: Partial<DeskAttentionItem> = {}): DeskAttentionItem {
  return { ...watchtowerItem(), deskLens: "Watchtower", aboutId: "shp_1", ...overrides };
}

function group(
  items: DeskAttentionItem[],
  overrides: Partial<DeskAttentionGroup> = {},
): DeskAttentionGroup {
  return {
    key: "shp_1",
    record: {
      kind: "shipment",
      noun: "Shipment",
      id: "shp_1",
      label: "S-240412",
      subtitle: "Dallas, TX → Memphis, TN",
      status: "In transit",
      readable: true,
    },
    severity: "Critical",
    dueAt: NOW + 47 * 60,
    dueLabel: "Relay must leave",
    unseen: false,
    items,
    ownership: null,
    ...overrides,
  };
}

afterEach(cleanup);

describe("a row of Needs you, on its page and on Today", () => {
  it("counts the signals by queue instead of listing each one", () => {
    const decisions = Array.from({ length: 12 }, (_, index) =>
      item({
        id: `ap_${index}`,
        deskLens: "Decisions",
        title: "Load monitor wants to evaluate service failures",
      }),
    );
    render(
      <AttentionRow
        group={group([
          item({
            id: "wt_1",
            title: "Delivery will miss its window",
            summary: "Projected 7:05 PM",
          }),
          item({ id: "wt_2", title: "Missed delivery on S-240412" }),
          item({ id: "tr_1", deskLens: "Triage", title: "Acme asked for the ETA on PO 88213" }),
          ...decisions,
        ])}
        now={NOW}
        onOpen={vi.fn()}
      />,
    );

    const row = screen.getByRole("button");
    expect(within(row).getByText("Delivery will miss its window")).toBeInTheDocument();
    expect(within(row).queryByText("Missed delivery on S-240412")).toBeNull();
    expect(within(row).queryByText("Load monitor wants to evaluate service failures")).toBeNull();
    expect(
      within(row).getByLabelText("Watchtower: 2, Triage: 1, and Decisions: 12"),
    ).toHaveTextContent("2112");
  });

  it("tells assistive tech where it sits in a list whose other rows are not mounted", () => {
    render(
      <AttentionRow
        group={group([item()])}
        now={NOW}
        selected={false}
        position={{ index: 4, size: 230 }}
        onOpen={vi.fn()}
      />,
    );

    const row = screen.getByRole("option");
    expect(row).toHaveAttribute("aria-posinset", "5");
    expect(row).toHaveAttribute("aria-setsize", "230");
  });

  it("names the record and what the server says of it on one line", () => {
    render(<AttentionRow group={group([item()])} now={NOW} onOpen={vi.fn()} />);

    expect(
      screen.getByText("S-240412 · Dallas, TX → Memphis, TN · In transit"),
    ).toBeInTheDocument();
  });

  it("says a change waits on the person and draws the time left as critical inside four hours", () => {
    render(
      <AttentionRow group={group([item({ deskLens: "Decisions" })])} now={NOW} onOpen={vi.fn()} />,
    );

    expect(screen.getByText("Needs your approval")).toBeInTheDocument();
    expect(screen.getByText("47m")).toHaveClass("text-danger-subtle-foreground");
  });

  it("names a record the reader may not open without its label", () => {
    render(
      <AttentionRow
        group={group([item()], {
          record: {
            kind: "shipment",
            noun: "Shipment",
            id: "shp_1",
            label: "",
            subtitle: "",
            status: "",
            readable: false,
          },
        })}
        now={NOW}
        onOpen={vi.fn()}
      />,
    );

    expect(screen.getAllByText(/A record you can't open/).length).toBeGreaterThan(0);
    expect(screen.queryByText("S-240412")).toBeNull();
  });

  it("opens the record when clicked", () => {
    const onOpen = vi.fn();
    render(<AttentionRow group={group([item()])} now={NOW} onOpen={onOpen} />);

    fireEvent.click(screen.getByRole("button"));

    expect(onOpen).toHaveBeenCalledTimes(1);
  });

  it("is an option of the list only where the person moves through one", () => {
    render(<AttentionRow group={group([item()])} now={NOW} selected onOpen={vi.fn()} />);

    expect(screen.getByRole("option")).toHaveAttribute("aria-selected", "true");
  });

  it("says whose case it is once someone has picked it up", () => {
    render(
      <AttentionRow
        group={group([item()], {
          ownership: { owner: { id: "usr_1", name: "Dana Ruiz" }, auto: false, canWork: false },
        })}
        now={NOW}
        onOpen={vi.fn()}
      />,
    );

    expect(screen.getByLabelText("Dana Ruiz's case").textContent).toBe("Dana Ruiz");
  });

  it("says nothing about a case nobody has picked up", () => {
    render(
      <AttentionRow
        group={group([item()], {
          ownership: { owner: { id: "usr_1", name: "Dana Ruiz" }, auto: true, canWork: false },
        })}
        now={NOW}
        onOpen={vi.fn()}
      />,
    );

    expect(screen.queryByLabelText(/case$/)).toBeNull();
  });

  describe("read through the Decisions lens", () => {
    const decision = () =>
      item({
        id: "ap_1",
        deskLens: "Decisions",
        title: "Dispatch desk wants to request worker credential renewal",
        facts: [
          { label: "Agent", value: "Dispatch desk", at: null },
          { label: "Change", value: "request worker credential renewal", at: null },
        ],
      });

    it("leads with the change and names who proposes it, instead of saying it waits", () => {
      render(
        <AttentionRow group={group([decision()])} now={NOW} lens="Decisions" onOpen={vi.fn()} />,
      );

      const row = screen.getByRole("button");
      expect(within(row).getByText("Request worker credential renewal")).toBeInTheDocument();
      expect(within(row).queryByText(/wants to/)).toBeNull();
      expect(
        within(row).getByText("Dispatch desk · S-240412 · Dallas, TX → Memphis, TN · In transit"),
      ).toBeInTheDocument();
      expect(within(row).queryByText("Needs your approval")).toBeNull();
    });

    it("still says what is not the same on every row: a write that failed", () => {
      const failed = item({
        id: "wt_2",
        agentWork: { failed: true, status: "NeedsYou" } as DeskAttentionItem["agentWork"],
      });
      render(
        <AttentionRow
          group={group([decision(), failed])}
          now={NOW}
          lens="Decisions"
          onOpen={vi.fn()}
        />,
      );

      expect(screen.getByText("Didn't go through")).toBeInTheDocument();
    });

    it("reads as it does everywhere else when the item names no change", () => {
      const bare = item({
        id: "ap_2",
        deskLens: "Decisions",
        title: "An agent proposes",
        facts: [],
      });
      render(<AttentionRow group={group([bare])} now={NOW} lens="Decisions" onOpen={vi.fn()} />);

      expect(screen.getByText("An agent proposes")).toBeInTheDocument();
    });
  });

  it("keeps the approval badge outside the Decisions lens, where rows differ", () => {
    render(
      <AttentionRow
        group={group([item({ deskLens: "Decisions" })])}
        now={NOW}
        lens="Everything"
        onOpen={vi.fn()}
      />,
    );

    expect(screen.getByText("Needs your approval")).toBeInTheDocument();
  });
});
