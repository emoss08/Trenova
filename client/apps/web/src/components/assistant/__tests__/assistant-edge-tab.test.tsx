import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantEdgeTab } from "../assistant-edge-tab";
import { AssistantLauncher } from "../assistant-launcher";

afterEach(cleanup);

/**
 * Hiding the launcher is about space. The tab left behind still opens the
 * assistant and still says when a decision is waiting.
 */
describe("AssistantEdgeTab", () => {
  it("opens the assistant", () => {
    const onClick = vi.fn();
    render(<AssistantEdgeTab dock="bottom-right" pendingCount={0} onClick={onClick} />);

    const tab = screen.getByRole("button", { name: "Open the assistant" });
    fireEvent.click(tab);

    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("still says how many decisions are waiting", () => {
    render(<AssistantEdgeTab dock="bottom-left" pendingCount={4} onClick={vi.fn()} />);

    const tab = screen.getByRole("button", {
      name: "Open the assistant, 4 changes await your decision",
    });
    expect(tab).toHaveAttribute("data-pending");
  });

  it("stays quiet when nothing is waiting", () => {
    render(<AssistantEdgeTab dock="top-right" pendingCount={0} onClick={vi.fn()} />);

    expect(screen.getByRole("button")).not.toHaveAttribute("data-pending");
  });
});

describe("AssistantLauncher placement", () => {
  it("offers to hide itself when it can be hidden", () => {
    const onHide = vi.fn();
    const onClick = vi.fn();
    render(<AssistantLauncher pendingCount={0} onClick={onClick} onHide={onHide} />);

    fireEvent.click(screen.getByRole("button", { name: "Hide the assistant button" }));

    expect(onHide).toHaveBeenCalledTimes(1);
    // Hiding it is not opening it.
    expect(onClick).not.toHaveBeenCalled();
  });

  it("has no hide button where hiding is not offered", () => {
    render(<AssistantLauncher pendingCount={0} onClick={vi.fn()} />);

    expect(screen.queryByRole("button", { name: "Hide the assistant button" })).toBeNull();
  });

  it("still opens on a plain click", () => {
    const onClick = vi.fn();
    render(<AssistantLauncher pendingCount={0} onClick={onClick} onMove={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: "Open the assistant" }));

    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
