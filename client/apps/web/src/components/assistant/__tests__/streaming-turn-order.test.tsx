import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { StreamingTurn } from "../streaming-turn";
import { initialTurnState, type TurnState } from "../turn-stream";

vi.mock("../report-run-card", () => ({
  ReportRunCard: ({ run }: { run: { runId: string } }) => (
    <div data-testid="report-run-card">{run.runId}</div>
  ),
}));

vi.mock("../choice-prompt", () => ({
  ChoicePrompt: ({ request }: { request: { callId: string } }) => (
    <div data-testid="choice-prompt">{request.callId}</div>
  ),
}));

const noop = () => {};

function fenced(payload: unknown): string {
  return `Result from tool:\n<untrusted_data>\n${JSON.stringify(payload)}\n</untrusted_data>`;
}

function precedes(first: HTMLElement, second: HTMLElement): boolean {
  return Boolean(first.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING);
}

function doneTurn(segments: TurnState["segments"], artifacts: TurnState["artifacts"] = []) {
  return {
    ...initialTurnState("Run the AR aging report", null, { startedAt: 0 }),
    status: "done",
    segments,
    artifacts,
  } satisfies TurnState;
}

/*
The saved thread shows what a step produced under that step, before the answer
written after it. The live turn must read the same way, or every card jumps
from under the answer to above it the moment the reply is saved.
*/
describe("StreamingTurn places what a step produced under that step", () => {
  it("shows a report run card before the answer written after the run", () => {
    render(
      <StreamingTurn
        turn={doneTurn([
          {
            kind: "tool",
            callId: "call_run",
            name: "run_report",
            arguments: { key: "ar_aging" },
            status: "done",
            content: fenced({ runId: "rrun_1", reportKey: "ar_aging" }),
          },
          { kind: "text", text: "Done — the run finished with 2 customers.", closed: true },
        ])}
        onDismiss={noop}
      />,
    );

    const card = screen.getByTestId("report-run-card");
    const answer = screen.getByText("Done — the run finished with 2 customers.");
    expect(precedes(card, answer)).toBe(true);
  });

  it("keeps a run checked again later under the step that first named it", () => {
    render(
      <StreamingTurn
        turn={doneTurn([
          {
            kind: "tool",
            callId: "call_run",
            name: "run_report",
            arguments: {},
            status: "done",
            content: fenced({ runId: "rrun_1" }),
          },
          { kind: "text", text: "Started it, checking on it.", closed: true },
          {
            kind: "tool",
            callId: "call_check",
            name: "get_report_run",
            arguments: {},
            status: "done",
            content: fenced({ runId: "rrun_1" }),
          },
          { kind: "text", text: "It finished.", closed: true },
        ])}
        onDismiss={noop}
      />,
    );

    const cards = screen.getAllByTestId("report-run-card");
    expect(cards).toHaveLength(1);
    expect(precedes(cards[0], screen.getByText("Started it, checking on it."))).toBe(true);
  });

  it("shows a question the agent asked before the words that follow it", () => {
    render(
      <StreamingTurn
        turn={doneTurn([
          {
            kind: "tool",
            callId: "call_ask",
            name: "ask_user",
            arguments: {},
            status: "done",
            content: fenced({
              question: "Which customer?",
              options: [{ label: "Acme", value: "Acme" }],
            }),
          },
          { kind: "text", text: "Tell me which one and I will run it.", closed: true },
        ])}
        onDismiss={noop}
        onAnswer={noop}
      />,
    );

    const prompt = screen.getByTestId("choice-prompt");
    expect(precedes(prompt, screen.getByText("Tell me which one and I will run it."))).toBe(true);
  });

  it("shows an artifact chip under the step that produced it", () => {
    render(
      <StreamingTurn
        turn={doneTurn(
          [
            {
              kind: "tool",
              callId: "call_run",
              name: "run_report",
              arguments: {},
              status: "done",
              content: fenced({ runId: "rrun_1" }),
            },
            { kind: "text", text: "Here is the table.", closed: true },
          ],
          [
            {
              id: "aart_1",
              kind: "report_run",
              status: "Ready",
              title: "AR Aging by Customer",
              sourceToolCallId: "call_run",
              path: "",
              draft: null,
            },
          ],
        )}
        onDismiss={noop}
        onOpenArtifact={noop}
      />,
    );

    const chip = screen.getByRole("button", { name: "Open AR Aging by Customer" });
    expect(precedes(chip, screen.getByText("Here is the table."))).toBe(true);
  });
});
