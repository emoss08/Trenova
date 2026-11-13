import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DeskListHead } from "../desk-list-head";
import { DESK_VIEWS_HEAD_CLASS, DeskViewDetail, DeskViewList, DeskViews } from "../desk-views";

describe("DeskViews, a list pane whose views change in place", () => {
  it("draws the head once, with each view's list and detail as cells beside it", () => {
    render(
      <DeskViews data-testid="frame">
        <DeskListHead className={DESK_VIEWS_HEAD_CLASS}>head</DeskListHead>
        <DeskViewList>first list</DeskViewList>
        <DeskViewDetail reading={false}>first detail</DeskViewDetail>
        <DeskViewList>second list</DeskViewList>
        <DeskViewDetail reading={false}>second detail</DeskViewDetail>
      </DeskViews>,
    );

    const frame = screen.getByTestId("frame");
    expect(screen.getAllByRole("banner")).toHaveLength(1);
    expect([...frame.children].map((cell) => cell.textContent)).toEqual([
      "head",
      "first list",
      "first detail",
      "second list",
      "second detail",
    ]);
  });

  it("marks the detail a row is open in, which a narrow window gives itself to", () => {
    render(
      <DeskViews>
        <DeskViewDetail reading>open</DeskViewDetail>
        <DeskViewDetail reading={false}>closed</DeskViewDetail>
      </DeskViews>,
    );

    expect(screen.getByText("open")).toHaveAttribute("data-reading");
    expect(screen.getByText("closed")).not.toHaveAttribute("data-reading");
  });
});
