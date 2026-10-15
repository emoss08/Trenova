import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SuggestionsCard } from "../suggestions-card";

const suggestions = {
  suggested: 12,
  switched: 4,
  sentThrough: 2,
  autoSwitched: 1,
  kept: 2,
  passedOver: 3,
  takenRate: 7 / 12,
  agents: [
    {
      agentDefinitionId: "agdef_service",
      name: "Customer service",
      suggested: 8,
      switched: 3,
      sentThrough: 2,
      autoSwitched: 1,
      kept: 0,
      passedOver: 2,
      takenRate: 0.75,
    },
    {
      agentDefinitionId: "agdef_gone",
      name: "",
      suggested: 4,
      switched: 1,
      sentThrough: 0,
      autoSwitched: 0,
      kept: 2,
      passedOver: 1,
      takenRate: 0.25,
    },
  ],
};

describe("SuggestionsCard", () => {
  it("draws each agent named, most suggested first, with what became of its suggestions", () => {
    render(
      <div className="aic">
        <SuggestionsCard suggestions={suggestions} />
      </div>,
    );

    const card = screen.getByRole("region", { name: "Agent suggestions" });
    expect(card).toHaveTextContent("12 suggested");
    expect(card).toHaveTextContent("58% taken");

    const rows = card.querySelectorAll('[data-slot="suggestion-row"]');
    expect(rows).toHaveLength(2);
    expect(rows[0]).toHaveTextContent("Customer service");
    expect(rows[0]).toHaveTextContent("75%");
    expect(rows[1]).toHaveTextContent("Removed agent");

    expect(
      screen
        .getByRole("img", {
          name: "3 switched, 2 sent to it, 1 moved without asking, 0 kept their own agent, 2 passed it over",
        })
        .querySelectorAll("i"),
    ).toHaveLength(4);
  });
});
