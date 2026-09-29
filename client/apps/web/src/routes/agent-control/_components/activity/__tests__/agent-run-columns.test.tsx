import { translate } from "@trenova/shared/i18n/runtime";
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { afterEach, describe, expect, it } from "vitest";
import type { AgentRunRow } from "@/lib/graphql/agent-activity-tables";
import { getRunColumns } from "../agent-run-columns";

afterEach(cleanup);

function agentCell(row: Partial<AgentRunRow>): ReactElement {
  const column = getRunColumns(translate).find(
    (candidate) =>
      candidate.id === "agentType" ||
      ("accessorKey" in candidate && candidate.accessorKey === "agentType"),
  );
  if (!column || typeof column.cell !== "function") {
    throw new Error("expected the agent column to draw its cell");
  }
  const cell = column.cell as (context: {
    row: { original: Partial<AgentRunRow> };
  }) => ReactElement;

  return cell({ row: { original: { agentType: "AssistantChat", ...row } } });
}

/**
 * #628: the run of an agent a conversation's agent handed a task to is listed
 * with who handed it over, so the work it did is found where the rest is.
 */
describe("the agent run's agent column", () => {
  it("names the agent that handed a delegate its task", () => {
    render(agentCell({ handedBy: { id: "agdef_dispatch", name: "Dispatch" } }));

    expect(screen.getByText("Handed by Dispatch")).toBeTruthy();
  });

  it("says nothing of a run nothing handed a task", () => {
    render(agentCell({ handedBy: null }));

    expect(screen.queryByText(/Handed by/)).toBeNull();
  });
});
