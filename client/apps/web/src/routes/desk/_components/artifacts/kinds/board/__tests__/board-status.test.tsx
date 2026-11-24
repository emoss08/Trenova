import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { BoardStatus } from "../board-parts";

afterEach(cleanup);

/*
A row's status is the product's own word for it, drawn with the shared badge in
the tone the status's owner gave it on the server: the client keeps no list of
status names and never draws a state as a neutral outline.
*/
describe("BoardStatus", () => {
  it("draws the status in the tone its owner gave it", () => {
    render(
      <>
        <BoardStatus status="Expired" tone="bad" />
        <BoardStatus status="Expiring soon" tone="warn" />
        <BoardStatus status="In transit" tone="info" />
        <BoardStatus status="Posted" tone="ok" />
        <BoardStatus status="Draft" tone="" />
      </>,
    );

    expect(screen.getByText("Expired").className).toContain("var(--danger)");
    expect(screen.getByText("Expiring soon").className).toContain("var(--warning)");
    expect(screen.getByText("In transit").className).toContain("var(--info)");
    expect(screen.getByText("Posted").className).toContain("var(--success)");
    expect(screen.getByText("Draft").className).toContain("var(--neutral)");
    expect(screen.getByText("Expired").className).not.toContain("bg-transparent");
  });

  it("draws nothing for a record with no status", () => {
    const { container } = render(<BoardStatus status="" tone="" />);
    expect(container).toBeEmptyDOMElement();
  });
});
