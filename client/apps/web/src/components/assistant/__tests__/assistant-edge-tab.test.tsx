import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantEdgeTab } from "../assistant-edge-tab";
import { AssistantLauncher } from "../assistant-launcher";
import { beaconState } from "../beacon-state";

const quiet = beaconState({
  pendingCount: 0,
  liveTurns: [],
  repliedAgentName: null,
  lastAgentName: "Billing exceptions",
});
const waiting = (pendingCount: number) =>
  beaconState({
    pendingCount,
    liveTurns: [],
    repliedAgentName: null,
    lastAgentName: "Billing exceptions",
  });

afterEach(cleanup);

/**
 * Hiding the launcher is about space. The tab left behind still opens the
 * assistant and still says when a decision is waiting.
 */
describe("AssistantEdgeTab", () => {
  it("opens the assistant", () => {
    const onClick = vi.fn();
    render(<AssistantEdgeTab dock="bottom-right" beacon={quiet} onClick={onClick} />);

    const tab = screen.getByRole("button", { name: "Open the assistant" });
    fireEvent.click(tab);

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("still says how many decisions are waiting", () => {
    render(<AssistantEdgeTab dock="bottom-left" beacon={waiting(4)} onClick={vi.fn()} />);

    const tab = screen.getByRole("button", {
      name: "Open the assistant, 4 changes await your decision",
    });
    expect(tab).toHaveAttribute("data-pending");
  });

  it("stays quiet when nothing is waiting", () => {
    render(<AssistantEdgeTab dock="top-right" beacon={quiet} onClick={vi.fn()} />);

    expect(screen.getByRole("button")).not.toHaveAttribute("data-pending");
  });
});

describe("AssistantLauncher placement", () => {
  it("offers to hide itself when it can be hidden", () => {
    const onHide = vi.fn();
    const onClick = vi.fn();
    render(<AssistantLauncher beacon={quiet} onClick={onClick} onHide={onHide} />);

    fireEvent.click(screen.getByRole("button", { name: "Hide the assistant button" }));

    expect(onHide).toHaveBeenCalledTimes(1);
    // Hiding it is not opening it.
    expect(onClick).not.toHaveBeenCalled();
  });

  it("has no hide button where hiding is not offered", () => {
    render(<AssistantLauncher beacon={quiet} onClick={vi.fn()} />);

    expect(screen.queryByRole("button", { name: "Hide the assistant button" })).toBeNull();
  });

  it("still opens on a plain click", () => {
    const onClick = vi.fn();
    render(<AssistantLauncher beacon={quiet} onClick={onClick} onMove={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Open the assistant" }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
