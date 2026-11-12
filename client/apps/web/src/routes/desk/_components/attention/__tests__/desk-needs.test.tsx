import { render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskNeeds } from "../desk-needs";
import type { NeedsLens } from "../lens";

const can = vi.hoisted(() => ({ watch: true, decide: true }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "agent_proposal" ? can.decide : can.watch,
    isLoading: false,
  }),
}));

vi.mock("../use-lens-counts", () => ({
  useLensCounts: () => ({ everything: 9, watchtower: 5, decisions: 4 }),
}));

vi.mock("../needs-attention", () => ({
  NeedsAttention: ({ lens, lenses }: { lens: NeedsLens; lenses: readonly NeedsLens[] }) => (
    <div data-testid="by-record">{`${lens}|${lenses.join(",")}`}</div>
  ),
}));

vi.mock("../../decisions/decision-flow", () => ({
  DecisionFlow: ({
    head,
  }: {
    head: (progress: {
      progressLabel: string;
      done: number;
      total: number;
      waiting: number;
      keysEnabled: boolean;
    }) => ReactNode;
  }) => (
    <div data-testid="decisions">
      {head({
        progressLabel: "1 of 8 decided",
        done: 1,
        total: 8,
        waiting: 7,
        keysEnabled: true,
      })}
    </div>
  ),
}));

afterEach(() => {
  can.watch = true;
  can.decide = true;
});

function renderAt(search: string) {
  render(
    <NuqsTestingAdapter searchParams={search}>
      <DeskNeeds />
    </NuqsTestingAdapter>,
  );
}

describe("DeskNeeds, one page for every lens", () => {
  it("lists by record under Everything and Watchtower", () => {
    renderAt("?lens=watchtower");

    expect(screen.getByTestId("by-record")).toHaveTextContent(
      "Watchtower|Everything,Watchtower,Decisions",
    );
    expect(screen.queryByTestId("decisions")).toBeNull();
  });

  it("draws the decision queue under the same head for the Decisions lens", () => {
    renderAt("?lens=decisions");

    expect(screen.queryByTestId("by-record")).toBeNull();
    expect(screen.getByText("Needs you")).toBeInTheDocument();
    expect(screen.getByRole("progressbar", { name: "1 of 8 decided" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Everything 9" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Decisions 4" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("opens on Decisions, counted from the queue, for someone who may not read the Watchtower", () => {
    can.watch = false;
    renderAt("");

    expect(screen.queryByTestId("by-record")).toBeNull();
    expect(screen.getAllByRole("tab")).toHaveLength(1);
    expect(screen.getByRole("tab", { name: "Decisions 7" })).toBeInTheDocument();
  });

  it("falls back to Everything when the address asks for a lens the person may not read", () => {
    can.decide = false;
    renderAt("?lens=decisions");

    expect(screen.getByTestId("by-record")).toHaveTextContent("Everything|Everything,Watchtower");
  });
});
