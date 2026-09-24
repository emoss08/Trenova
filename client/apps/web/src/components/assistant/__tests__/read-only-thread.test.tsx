import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { ThreadAskRequest } from "../ask-requests";
import { ChoicePrompt } from "../choice-prompt";
import { ReadOnlyThreadNotice } from "../read-only-thread-notice";
import { StreamingTurn } from "../streaming-turn";
import { initialTurnState, type TurnState } from "../turn-stream";

afterEach(() => {
  cleanup();
});

const request: ThreadAskRequest = {
  sequence: 3,
  callId: "call_3",
  question: "Which window should the report cover?",
  options: [
    { value: "7", label: "7 days", detail: "" },
    { value: "30", label: "30 days", detail: "" },
  ],
  allowOther: true,
  otherHint: "Number of days",
};

/**
 * A conversation whose reader lost access to its agent stays readable, and
 * nothing on it offers to send: the server would refuse whatever it sent.
 */
describe("a conversation that can no longer continue", () => {
  it("says so where the composer was, and who can change it", () => {
    render(<ReadOnlyThreadNotice />);

    const notice = screen.getByRole("status");
    expect(notice).toHaveTextContent(
      "You no longer have access to this agent. An administrator can give one of your roles access to it.",
    );
    expect(screen.queryByRole("textbox")).toBeNull();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("forwards the ref, so the thread pads its last message clear of the notice", () => {
    let measured: HTMLDivElement | null = null;
    render(
      <ReadOnlyThreadNotice
        ref={(element) => {
          measured = element;
        }}
      />,
    );

    expect(measured).not.toBeNull();
    expect(measured).toContainElement(screen.getByRole("status"));
  });

  it("keeps a question the agent asked as a record, with nothing to answer it", () => {
    render(<ChoicePrompt request={request} answered={false} />);

    expect(screen.getByText("Which window should the report cover?")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "7 days" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "30 days" })).toBeDisabled();
    expect(screen.queryByLabelText("Answer in your own words")).toBeNull();
    expect(screen.queryByLabelText("Send this answer")).toBeNull();
  });

  it("offers no retry on a reply that failed", () => {
    const failed: TurnState = {
      ...initialTurnState("Where is PRO S12345?", null),
      status: "error",
      error: "The reply was cut off before it finished.",
    };

    render(<StreamingTurn turn={failed} onDismiss={() => undefined} />);

    expect(screen.getByText("The reply was cut off before it finished.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /try again|retry/i })).toBeNull();
  });
});
