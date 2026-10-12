import type { AssistantArtifact } from "@/types/assistant";
import { act, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskBillingArtifact, DeskBillingRecord } from "../desk-billing-record";
import { useDeskBillingStore } from "../desk-billing-store";

const seen = vi.hoisted(() => ({
  props: [] as Array<{
    itemId: string;
    fallback: ReactNode;
    onSelect?: (itemId: string) => void;
    onBack?: () => void;
  }>,
}));

vi.mock("../desk-billing-item", () => ({
  DeskBillingItem: (props: (typeof seen.props)[number]) => {
    seen.props.push(props);
    return (
      <div>
        <p>Item {props.itemId}</p>
        <div data-testid="fallback">{props.fallback}</div>
        {props.onSelect && (
          <button type="button" onClick={() => props.onSelect?.("bqi_9")}>
            Next
          </button>
        )}
        {props.onBack && (
          <button type="button" onClick={props.onBack}>
            Back
          </button>
        )}
      </div>
    );
  },
}));

function card(overrides: Partial<AssistantArtifact> = {}): AssistantArtifact {
  return { id: "art_1", lineageId: "", payload: {}, ...overrides } as AssistantArtifact;
}

beforeEach(() => {
  seen.props = [];
  useDeskBillingStore.setState({ itemByLineage: {} });
});

describe("DeskBillingRecord", () => {
  it("opens the row's item in place, stepping and going back as the table asks", () => {
    const onSelect = vi.fn();
    const onBack = vi.fn();
    render(
      <DeskBillingRecord recordId="bqi_1" order={["bqi_1"]} onSelect={onSelect} onBack={onBack} />,
    );

    expect(screen.getByText("Item bqi_1")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    expect(onSelect).toHaveBeenCalledWith("bqi_9");
    expect(onBack).toHaveBeenCalledOnce();
  });

  it("holds the pane with the billing item's outline while the item is read", () => {
    render(<DeskBillingRecord recordId="bqi_1" order={[]} onSelect={vi.fn()} onBack={vi.fn()} />);

    const fallback = screen.getByTestId("fallback");
    expect(fallback.querySelector(".dk-rec-skel .dk-rec-checks")).not.toBeNull();
    expect(fallback).toHaveTextContent("Loading the billing item…");
  });
});

describe("DeskBillingArtifact", () => {
  it("shows the item it was opened on, with the server's view while it is read", () => {
    render(
      <DeskBillingArtifact artifact={card()} recordId="bqi_1" fallback={<p>Server view</p>} />,
    );

    expect(screen.getByText("Item bqi_1")).toBeInTheDocument();
    expect(screen.getByText("Server view")).toBeInTheDocument();
    expect(seen.props.at(-1)?.onBack).toBeUndefined();
  });

  it("shows whichever item a row or a step picked, kept by the artifact's lineage", () => {
    render(
      <DeskBillingArtifact
        artifact={card({ id: "art_2", lineageId: "art_1" })}
        recordId="bqi_1"
        fallback={null}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(screen.getByText("Item bqi_9")).toBeInTheDocument();
    expect(useDeskBillingStore.getState().itemByLineage).toEqual({ art_1: "bqi_9" });

    act(() => useDeskBillingStore.getState().select("art_1", "bqi_4"));
    expect(screen.getByText("Item bqi_4")).toBeInTheDocument();
  });
});
