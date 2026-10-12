import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AgentToolVerdictRow } from "@/lib/graphql/agent-scorecard";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";
import { ToolCallsPanel } from "../tool-calls-panel";

afterEach(cleanup);

function row(
  toolName: string,
  verdict: string,
  calls: number,
  topReasons: AgentToolVerdictRow["topReasons"] = [],
): AgentToolVerdictRow {
  return { toolName, verdict, calls, topReasons };
}

const title = (name: string) => describeToolCall(name, null).title;

/** Each tool's row: the buttons in the panel's list of tools, the outermost list. */
function toolButtons() {
  const [tools] = screen.queryAllByRole("list");
  return tools ? within(tools).getAllByRole("button") : [];
}

describe("ToolCallsPanel", () => {
  it("lists the tool with the most calls that did not go through first, with its share", () => {
    render(
      <ToolCallsPanel
        loading={false}
        failed={false}
        verdicts={[
          row("list_shipments", "ran", 40),
          row("update_shipment", "ran", 1),
          row("update_shipment", "invalid", 3, [{ reason: "stop 2 has no location", calls: 3 }]),
        ]}
      />,
    );

    const [first, second] = toolButtons();
    expect(first).toHaveTextContent(title("update_shipment"));
    expect(first).toHaveTextContent("75%");
    expect(first).toHaveTextContent("Not accepted");
    expect(second).toHaveTextContent(title("list_shipments"));
    expect(second).toHaveTextContent("0%");
    expect(screen.getByText("3 of 44 calls didn't go through")).toBeInTheDocument();
  });

  it("opens a tool onto the reasons its calls gave, and closes it again", async () => {
    const user = userEvent.setup();
    render(
      <ToolCallsPanel
        loading={false}
        failed={false}
        verdicts={[
          row("update_shipment", "ran", 1),
          row("update_shipment", "invalid", 3, [{ reason: "stop 2 has no location", calls: 3 }]),
          row("update_shipment", "failed", 1),
        ]}
      />,
    );

    const [button] = toolButtons();
    expect(button).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("stop 2 has no location")).not.toBeInTheDocument();

    await user.click(button);

    expect(button).toHaveAttribute("aria-expanded", "true");
    const region = document.getElementById(button.getAttribute("aria-controls") ?? "");
    expect(region).not.toBeNull();
    const reasons = within(region as HTMLElement);
    expect(reasons.getByText("stop 2 has no location")).toBeInTheDocument();
    expect(reasons.getByText("3 calls")).toBeInTheDocument();
    expect(reasons.getByText("No reason was recorded.")).toBeInTheDocument();
    expect(reasons.queryByRole("region", { name: "Ran" })).not.toBeInTheDocument();

    await user.click(button);
    expect(screen.queryByText("stop 2 has no location")).not.toBeInTheDocument();
  });

  it("says every call went through when a tool's calls all did", async () => {
    const user = userEvent.setup();
    render(
      <ToolCallsPanel loading={false} failed={false} verdicts={[row("get_customer", "ran", 5)]} />,
    );

    expect(screen.getByText("Every call went through")).toBeInTheDocument();
    await user.click(toolButtons()[0]);
    expect(screen.getByText("Every call went through.")).toBeInTheDocument();
  });

  it("shows the first eight tools until asked for all of them", async () => {
    const user = userEvent.setup();
    const verdicts = Array.from({ length: 10 }, (_, index) =>
      row(`tool_${String(index).padStart(2, "0")}`, "failed", 10 - index),
    );
    render(<ToolCallsPanel loading={false} failed={false} verdicts={verdicts} />);

    expect(toolButtons()).toHaveLength(8);
    await user.click(screen.getByRole("button", { name: "Show all 10 tools" }));
    expect(toolButtons()).toHaveLength(10);
    await user.click(screen.getByRole("button", { name: "Show fewer" }));
    expect(toolButtons()).toHaveLength(8);
  });

  it("says so when there were no tool calls", () => {
    render(<ToolCallsPanel loading={false} failed={false} verdicts={[]} />);

    expect(screen.getByText("No tool calls in the last 30 days.")).toBeInTheDocument();
    expect(toolButtons()).toHaveLength(0);
  });

  it("holds its place while loading and says when the scorecard could not be read", () => {
    const { rerender } = render(<ToolCallsPanel loading failed={false} verdicts={undefined} />);
    expect(document.querySelector("[aria-busy='true']")).not.toBeNull();
    expect(screen.queryByText("No tool calls in the last 30 days.")).not.toBeInTheDocument();

    rerender(<ToolCallsPanel loading={false} failed verdicts={undefined} />);
    expect(screen.getByRole("alert")).toHaveTextContent("The tool calls could not be loaded.");
  });
});
