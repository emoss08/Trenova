import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ScrollBox } from "../layout";

afterEach(() => {
  cleanup();
});

/*
How tall the box grows, the fade at a clipped edge and keyboard focus once it
overflows all depend on layout (`.aic .scb` in ai-control.css and the shared
ScrollArea measuring its viewport), which jsdom does not compute. What a DOM can
show is that the list is one region named for what it holds.
*/
describe("ScrollBox", () => {
  it("holds its list in one region named for what it holds", () => {
    render(
      <ScrollBox label="Waiting for approval">
        <p>First suggestion</p>
        <p>Second suggestion</p>
      </ScrollBox>,
    );

    const region = screen.getByRole("region", { name: "Waiting for approval" });
    expect(within(region).getByText("First suggestion")).toBeInTheDocument();
    expect(within(region).getByText("Second suggestion")).toBeInTheDocument();
  });
});
