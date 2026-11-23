import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cardTime } from "../../../record-card/record-card-values";
import { writtenSegments } from "../board-card";
import { BoardWords, type CiteLight } from "../board-words";
import { ANSWERED_AT, CONVERSATION_SPEC, READ_AT, conversationBoard } from "./fixtures";

afterEach(cleanup);

function words(props: { cite?: CiteLight } = {}) {
  const card = conversationBoard();
  return render(
    <p data-testid="words">
      <BoardWords segments={card.verdict} board={card} cite={props.cite} />
    </p>,
  );
}

/*
A board's words are the agent's, but every figure in them is a reading the
server read: a button that says what it is and where it came from. The sentence
keeps what was answered; a reading that moved is marked and says both, and one
the reader may not read is said in words.
*/
describe("BoardWords", () => {
  it("writes the words with each cited phrase marked and numbered", () => {
    const { container } = words();

    expect(screen.getByTestId("words")).toHaveTextContent(
      "There are 6 late loads, the furthest 3h 10m behind, with an amount you may not read owed. Four wait on one driver's hours1.",
    );
    const phrase = container.querySelector(".dk-bd-hl");
    expect(phrase).toHaveTextContent("Four wait on one driver's hours");
    expect(phrase).toHaveClass("dk-warn");
    expect(container.querySelector(".dk-bd-n")).toHaveTextContent("1");
  });

  it("draws a reading as a button whose popover says what it reads, where from and when", async () => {
    words();

    const late = screen.getByRole("button", { name: "Late now: 6" });
    expect(late).toHaveTextContent("6");
    fireEvent.click(late);

    expect(
      await screen.findByText("Counted from 6 shipments · 4 you may read"),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Late now").length).toBeGreaterThan(0);
    expect(screen.getByText(`Read ${cardTime(READ_AT)}, with your access`)).toBeInTheDocument();
  });

  it("keeps the value answered for a reading that moved, and says what it was and is now", async () => {
    words();

    const furthest = screen.getByRole("button", {
      name: "Furthest behind: 3h 10m when answered, 1h 40m now",
    });
    expect(furthest).toHaveTextContent("3h 10m");
    fireEvent.click(furthest);

    expect(
      await screen.findByText(`3h 10m when answered at ${cardTime(ANSWERED_AT)}`),
    ).toBeInTheDocument();
    expect(
      screen.getByText(`1h 40m now, read ${cardTime(READ_AT)} with your access`),
    ).toBeInTheDocument();
  });

  it("says a reading the reader may not read in words, with nothing to press and no figure", () => {
    words();

    expect(screen.getByText("an amount you may not read")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Still owed on them/u })).not.toBeInTheDocument();
    expect(screen.getAllByRole("button").map((button) => button.textContent)).toEqual([
      "6",
      "3h 10m",
    ]);
  });

  it("draws a blank for each reading while the board is read", () => {
    const { container } = render(
      <p data-testid="words">
        <BoardWords segments={writtenSegments(CONVERSATION_SPEC.verdict)} board={null} />
      </p>,
    );

    expect(screen.queryAllByRole("button")).toHaveLength(0);
    expect(container.querySelectorAll(".dk-bd-blank")).toHaveLength(3);
    expect(screen.getByTestId("words")).toHaveTextContent(
      "There are late loads, the furthest behind, with owed. Four wait on one driver's hours1.",
    );
  });

  it("lights the exhibit a cited phrase cites while the phrase is pointed at or its numeral focused", () => {
    const onLight = vi.fn();
    const { container, rerender } = words({ cite: { lit: 0, onLight } });

    const phrase = container.querySelector(".dk-bd-hl") as HTMLElement;
    fireEvent.mouseEnter(phrase.parentElement as HTMLElement);
    expect(onLight).toHaveBeenLastCalledWith(1);
    fireEvent.mouseLeave(phrase.parentElement as HTMLElement);
    expect(onLight).toHaveBeenLastCalledWith(0);

    const numeral = screen.getByRole("button", { name: "Show exhibit 1" });
    fireEvent.focus(numeral);
    expect(onLight).toHaveBeenLastCalledWith(1);
    fireEvent.blur(numeral);
    expect(onLight).toHaveBeenLastCalledWith(0);

    const card = conversationBoard();
    rerender(
      <p data-testid="words">
        <BoardWords segments={card.verdict} board={card} cite={{ lit: 1, onLight }} />
      </p>,
    );
    expect(container.querySelector(".dk-bd-hl")).toHaveClass("dk-bd-lit");
    expect(screen.getByRole("button", { name: "Show exhibit 1" })).toHaveClass("dk-bd-lit");
  });
});
