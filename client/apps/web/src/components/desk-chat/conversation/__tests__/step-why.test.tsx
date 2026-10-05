import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StepWhy } from "../desk-citations";

describe("StepWhy", () => {
  it("shows the rows the agent filled in, labelled, and leaves out the rest", () => {
    const { container } = render(
      <StepWhy
        rows={[
          ["Saw", "8 items have no biller"],
          ["Because", ""],
          ["Instead of", "Assigning them one by one"],
        ]}
      />,
    );
    const rows = [...container.querySelectorAll(".dk-why > span")].map((row) => row.textContent);
    expect(rows).toEqual(["Saw8 items have no biller", "Instead ofAssigning them one by one"]);
  });
});
