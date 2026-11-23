import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { WorkerSectionStrip, WorkerStripProvider } from "../worker-strip-slot";

afterEach(cleanup);

describe("a tab placing its own section strip", () => {
  it("draws the strip the shell handed it where the tab puts it, after what comes first", () => {
    render(
      <WorkerStripProvider strip={<nav aria-label="Employment views" />}>
        <div data-testid="tab">
          <section aria-label="Lifecycle route" />
          <WorkerSectionStrip />
        </div>
      </WorkerStripProvider>,
    );

    const children = Array.from(screen.getByTestId("tab").children).map((node) =>
      node.getAttribute("aria-label"),
    );
    expect(children).toEqual(["Lifecycle route", "Employment views"]);
  });

  it("draws nothing outside a strip-placing tab", () => {
    const { container } = render(<WorkerSectionStrip />);

    expect(container.innerHTML).toBe("");
  });
});
