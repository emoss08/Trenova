import type { ThreadAsk, ThreadAskRequest } from "@/components/assistant/ask-requests";
import type { AssistantMessage } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskAskDock, DeskAskRecord, deskAskPlaceholder } from "../desk-ask";

const mocks = vi.hoisted(() => ({ searchMentions: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { assistantService: { searchMentions: mocks.searchMentions } },
}));

afterEach(() => {
  cleanup();
  mocks.searchMentions.mockReset();
});

const reportWindow: ThreadAskRequest = {
  sequence: 3,
  callId: "call_3",
  question: "Which window should the report cover?",
  options: [
    { value: "7", label: "7 days", detail: "the last week" },
    { value: "30", label: "30 days", detail: "" },
  ],
  allowOther: true,
  otherHint: "Number of days",
  suggestion: "",
  input: { type: "text" },
  fields: [],
};

/** The dock sits in a conversation's room, beside the composer. */
function inRoom(children: ReactNode) {
  return (
    <div className="dk-room">
      <textarea aria-label="Message" />
      {children}
    </div>
  );
}

function dock(request: ThreadAskRequest, { disabled = false } = {}) {
  const onAnswer = vi.fn();
  render(inRoom(<DeskAskDock request={request} disabled={disabled} onAnswer={onAnswer} />));
  return onAnswer;
}

describe("DeskAskDock", () => {
  it("is a region named for what it is", () => {
    dock(reportWindow);

    expect(screen.getByRole("region", { name: "Question from the agent" })).toHaveTextContent(
      "Which window should the report cover?",
    );
  });

  it("sends the choice's value, not its label", async () => {
    const onAnswer = dock(reportWindow);

    await userEvent.click(screen.getByRole("button", { name: "30 days" }));

    expect(onAnswer).toHaveBeenCalledWith("30");
  });

  // The line of consequence used to hide in a tooltip nobody hovered.
  it("shows each choice's detail under it, and describes the choice with it", () => {
    dock(reportWindow);

    expect(screen.getByText("the last week")).toBeVisible();
    expect(screen.getByRole("button", { name: "7 days" })).toHaveAccessibleDescription(
      "the last week",
    );
    expect(screen.getByRole("button", { name: "30 days" })).not.toHaveAttribute("aria-describedby");
  });

  it("answers with a choice's number key while focus is not in a field", () => {
    const onAnswer = dock(reportWindow);

    fireEvent.keyDown(document.body, { key: "2" });

    expect(onAnswer).toHaveBeenCalledWith("30");
  });

  it("leaves a number typed in the composer to the composer", () => {
    const onAnswer = dock(reportWindow);

    fireEvent.keyDown(screen.getByRole("textbox", { name: "Message" }), { key: "1" });
    fireEvent.keyDown(document.body, { key: "3" });
    fireEvent.keyDown(document.body, { key: "1", metaKey: true });

    expect(onAnswer).not.toHaveBeenCalled();
  });

  it("ignores a number key pressed outside the conversation", () => {
    const onAnswer = dock(reportWindow);
    const elsewhere = document.createElement("button");
    document.body.append(elsewhere);

    fireEvent.keyDown(elsewhere, { key: "1" });

    expect(onAnswer).not.toHaveBeenCalled();
    elsewhere.remove();
  });

  // The composer is the box for a value in words; a second one beside it
  // was the defect.
  it("draws no box of its own for a value in words", () => {
    dock(reportWindow);

    expect(screen.getAllByRole("textbox")).toEqual([
      screen.getByRole("textbox", { name: "Message" }),
    ]);
    expect(screen.getByText("Pick one, or type your own answer below.")).toBeInTheDocument();
  });

  it("leaves focus where it is when the answer is typed in the composer", () => {
    dock(reportWindow);

    expect(document.body).toHaveFocus();
  });

  it("takes focus on its first choice when the set is closed", () => {
    dock({ ...reportWindow, allowOther: false });

    expect(screen.getByRole("button", { name: "7 days" })).toHaveFocus();
  });

  it("takes a number in a number box and sends it as typed", async () => {
    const onAnswer = dock({ ...reportWindow, options: [], input: { type: "number" } });
    const send = screen.getByRole("button", { name: "Send" });
    const box = screen.getByRole("spinbutton", { name: "Number of days" });

    expect(box).toHaveFocus();
    expect(send).toBeDisabled();
    await userEvent.type(box, "45");
    await userEvent.click(send);

    expect(onAnswer).toHaveBeenCalledWith("45", []);
  });

  it("sends a form as one line per value once every value it needs is there", async () => {
    const onAnswer = dock({
      ...reportWindow,
      options: [],
      allowOther: false,
      fields: [
        { key: "pickup", label: "Pickup date", type: "date", hint: "", optional: false },
        {
          key: "pallets",
          label: "Pallets",
          type: "number",
          hint: "Whole pallets",
          optional: false,
        },
        { key: "note", label: "Note", type: "text", hint: "", optional: true },
      ],
    });
    const send = screen.getByRole("button", { name: "Send" });

    expect(screen.getByLabelText("Pallets")).toHaveAccessibleDescription("Whole pallets");
    fireEvent.change(screen.getByLabelText("Pickup date"), { target: { value: "2026-10-14" } });
    expect(send).toBeDisabled();
    await userEvent.type(screen.getByLabelText("Pallets"), "12");
    expect(send).toBeEnabled();
    await userEvent.click(send);

    expect(onAnswer).toHaveBeenCalledWith("Pickup date: 2026-10-14\nPallets: 12", []);
  });

  it("finds a record with the @ search and sends it as a mention", async () => {
    mocks.searchMentions.mockResolvedValue([
      { type: "customer", id: "cus_1", label: "Acme Freight", subtitle: "Chicago, IL" },
      { type: "shipment", id: "shp_1", label: "Acme load", subtitle: "" },
    ]);
    const onAnswer = dock({
      ...reportWindow,
      options: [],
      otherHint: "Customer",
      input: { type: "record", recordKind: "customer" },
    });

    const search = screen.getByRole("combobox", { name: "Customer" });
    await userEvent.type(search, "acme");
    const found = await screen.findByRole("option", { name: /Acme Freight/ });
    // The search runs over the customer tab; only a customer answers.
    expect(screen.queryByRole("option", { name: /Acme load/ })).toBeNull();
    await userEvent.click(found);
    await userEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(mocks.searchMentions).toHaveBeenCalledWith("acme", "customer", expect.anything());
    expect(onAnswer).toHaveBeenCalledWith("@Acme Freight", [
      { type: "customer", id: "cus_1", label: "Acme Freight" },
    ]);
  });

  it("takes no answer while a reply is being written", () => {
    const onAnswer = dock(reportWindow, { disabled: true });

    expect(screen.getByRole("button", { name: "7 days" })).toBeDisabled();
    expect(screen.getByText("You can answer once the reply finishes.")).toBeInTheDocument();
    fireEvent.keyDown(document.body, { key: "1" });
    expect(onAnswer).not.toHaveBeenCalled();
  });
});

function said(content: string, overrides: Partial<AssistantMessage> = {}): AssistantMessage {
  return {
    id: "msg_a",
    threadId: "athr_1",
    role: "User",
    kind: "Message",
    content,
    sequence: 4,
    createdAt: 0,
    ...overrides,
  } as AssistantMessage;
}

describe("DeskAskRecord", () => {
  it("records the question and the choice the person picked, by its label", () => {
    const ask: ThreadAsk = { request: reportWindow, answer: said("30") };
    render(<DeskAskRecord ask={ask} docked={false} />);

    expect(screen.getByText("Which window should the report cover?")).toBeInTheDocument();
    expect(screen.getByText("30 days")).toBeInTheDocument();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("records a value the person typed as they typed it", () => {
    render(<DeskAskRecord ask={{ request: reportWindow, answer: said("45") }} docked={false} />);

    expect(screen.getByText("45")).toBeInTheDocument();
  });

  it("draws a record the person picked as the chip a mention draws", () => {
    const answer = said("@Acme Freight", {
      mentions: [{ type: "customer", id: "cus_1", label: "Acme Freight" }],
    });
    const { container } = render(
      <DeskAskRecord ask={{ request: reportWindow, answer }} docked={false} />,
    );

    expect(container.querySelector(".dk-mn-chip")).toHaveTextContent("Acme Freight");
  });

  it("points at the composer while the question waits there", () => {
    render(<DeskAskRecord ask={{ request: reportWindow, answer: null }} docked />);

    expect(screen.getByText("Waiting for your answer below")).toBeInTheDocument();
    expect(screen.queryByRole("button")).toBeNull();
  });

  it("says a question nothing answered was not answered", () => {
    render(<DeskAskRecord ask={{ request: reportWindow, answer: null }} docked={false} />);

    expect(screen.getByText("Not answered yet")).toBeInTheDocument();
  });

  // The answer is read under its question from the moment it is sent, so
  // nothing moves when it is saved.
  it("records the answer the person has just sent, before it is saved", () => {
    const sent = { callId: reportWindow.callId, content: "30", mentions: [] };
    render(
      <DeskAskRecord ask={{ request: reportWindow, answer: null }} docked={false} sent={sent} />,
    );

    expect(screen.getByText("30 days")).toBeInTheDocument();
    expect(screen.queryByText("Not answered yet")).toBeNull();
  });

  it("records the saved answer over the one being sent", () => {
    const sent = { callId: reportWindow.callId, content: "7", mentions: [] };
    render(
      <DeskAskRecord
        ask={{ request: reportWindow, answer: said("30") }}
        docked={false}
        sent={sent}
      />,
    );

    expect(screen.getByText("30 days")).toBeInTheDocument();
    expect(screen.queryByText("7 days")).toBeNull();
  });
});

describe("DeskAskDock with a suggested answer", () => {
  const partnerCode: ThreadAskRequest = {
    ...reportWindow,
    question: "What partner code should Meritor have in Trenova?",
    options: [],
    otherHint: "A short code, such as MERITOR",
    suggestion: "MERITOR",
  };

  it("says the suggestion is a key away in the composer", () => {
    dock(partnerCode);

    expect(screen.getByRole("region", { name: "Question from the agent" })).toHaveTextContent(
      "Press Tab to use the suggested answer, or type your own below.",
    );
  });

  it("says only to type when there is no suggestion", () => {
    dock({ ...partnerCode, suggestion: "" });

    expect(screen.getByRole("region", { name: "Question from the agent" })).toHaveTextContent(
      "Type your answer below.",
    );
  });
});

describe("deskAskPlaceholder", () => {
  const t = ((key: string) => key) as TranslateFn;

  it("says what to type when the composer is the box", () => {
    expect(deskAskPlaceholder(reportWindow, t)).toBe("Number of days");
    expect(deskAskPlaceholder({ ...reportWindow, otherHint: "" }, t)).toBe("Type your answer…");
  });

  it("points above when the answer is a choice, a shape or a form", () => {
    expect(deskAskPlaceholder({ ...reportWindow, allowOther: false }, t)).toBe(
      "Answer above, or reply…",
    );
    expect(deskAskPlaceholder({ ...reportWindow, input: { type: "date" } }, t)).toBe(
      "Answer above, or reply…",
    );
  });
});
