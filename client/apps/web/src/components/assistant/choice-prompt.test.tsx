import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ChoicePrompt } from "./choice-prompt";
import type { ThreadAskRequest } from "./ask-requests";

const request: ThreadAskRequest = {
  sequence: 3,
  callId: "call_3",
  question: "Which window should the report cover?",
  options: [
    { value: "7", label: "7 days", detail: "the last week" },
    { value: "30", label: "30 days", detail: "" },
  ],
  allowOther: true,
  otherHint: "Number of days",
};

describe("ChoicePrompt", () => {
  it("sends the option's value, not its label", async () => {
    const onAnswer = vi.fn();
    render(<ChoicePrompt request={request} answered={false} onAnswer={onAnswer} />);

    await userEvent.click(screen.getByRole("button", { name: "30 days" }));

    expect(onAnswer).toHaveBeenCalledWith("30");
  });

  it("takes a value the assistant never offered", async () => {
    const onAnswer = vi.fn();
    render(<ChoicePrompt request={request} answered={false} onAnswer={onAnswer} />);

    await userEvent.type(screen.getByLabelText("Answer in your own words"), "45");
    await userEvent.click(screen.getByLabelText("Send this answer"));

    expect(onAnswer).toHaveBeenCalledWith("45");
  });

  it("sends a typed answer on Enter", async () => {
    const onAnswer = vi.fn();
    render(<ChoicePrompt request={request} answered={false} onAnswer={onAnswer} />);

    await userEvent.type(screen.getByLabelText("Answer in your own words"), "45{Enter}");

    expect(onAnswer).toHaveBeenCalledWith("45");
  });

  it("will not send an empty or blank answer", async () => {
    const onAnswer = vi.fn();
    render(<ChoicePrompt request={request} answered={false} onAnswer={onAnswer} />);

    await userEvent.type(screen.getByLabelText("Answer in your own words"), "   {Enter}");

    expect(onAnswer).not.toHaveBeenCalled();
  });

  it("offers no free-text box when the set is closed", () => {
    render(
      <ChoicePrompt
        request={{ ...request, allowOther: false }}
        answered={false}
        onAnswer={vi.fn()}
      />,
    );

    expect(screen.queryByLabelText("Answer in your own words")).not.toBeInTheDocument();
  });

  // The question stays on screen as the record of what was asked, but a
  // settled question must not invite a second answer.
  it("stops accepting answers once the person has replied", () => {
    render(<ChoicePrompt request={request} answered onAnswer={vi.fn()} />);

    expect(screen.getByRole("button", { name: "30 days" })).toBeDisabled();
    expect(screen.queryByLabelText("Answer in your own words")).not.toBeInTheDocument();
    expect(screen.getByText("Which window should the report cover?")).toBeInTheDocument();
  });
});
