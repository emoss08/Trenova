import { describe, expect, it } from "vitest";
import { askRequestsFrom } from "./ask-requests";
import type { ToolExchange } from "./thread-view";

/** The runtime fences an ask exactly as it fences any other tool result. */
function fenced(payload: unknown): string {
  return `Result from ask_user:\n<untrusted_data>\n${JSON.stringify(payload)}\n</untrusted_data>`;
}

function exchange(name: string, content: string, sequence = 3): ToolExchange {
  return {
    call: { id: `call_${sequence}`, name, arguments: {} },
    result: {
      id: `amsg_${sequence}`,
      threadId: "athr_1",
      kind: "Message",
      sequence,
      role: "Tool",
      content,
      toolCallId: `call_${sequence}`,
      toolName: name,
      toolFailed: false,
      refused: false,
      scopeStage: "",
      scopeCategory: "",
      scopeReason: "",
      model: "",
      providerId: "",
      inputTokens: 0,
      outputTokens: 0,
      createdAt: 0,
    } as ToolExchange["result"],
  };
}

const window = {
  question: "Which window should the report cover?",
  options: [
    { value: "7", label: "7 days", detail: "the last week" },
    { value: "30", label: "30 days" },
  ],
  allowOther: true,
  otherHint: "Number of days",
  note: "The person has been shown this question…",
};

describe("askRequestsFrom", () => {
  it("reads the question and its choices back out of the thread", () => {
    expect(askRequestsFrom([exchange("ask_user", fenced(window))])).toEqual([
      {
        sequence: 3,
        callId: "call_3",
        question: "Which window should the report cover?",
        options: [
          { value: "7", label: "7 days", detail: "the last week" },
          { value: "30", label: "30 days", detail: "" },
        ],
        allowOther: true,
        otherHint: "Number of days",
      },
    ]);
  });

  it("keeps the free-text box off for a closed set", () => {
    const closed = { ...window, allowOther: false, otherHint: "" };
    expect(askRequestsFrom([exchange("ask_user", fenced(closed))])[0].allowOther).toBe(false);
  });

  it("labels a bare value with itself rather than rendering an empty button", () => {
    const bare = { ...window, options: [{ value: "Active" }] };
    expect(askRequestsFrom([exchange("ask_user", fenced(bare))])[0].options).toEqual([
      { value: "Active", label: "Active", detail: "" },
    ]);
  });

  it("answers a labelled option without a value with its label", () => {
    const labelled = {
      ...window,
      options: [
        { label: "Chicago to Dallas", detail: "the busiest lane" },
        { value: "", label: "Atlanta to Miami" },
        { detail: "neither a value nor a label" },
      ],
    };
    expect(askRequestsFrom([exchange("ask_user", fenced(labelled))])[0].options).toEqual([
      { value: "Chicago to Dallas", label: "Chicago to Dallas", detail: "the busiest lane" },
      { value: "Atlanta to Miami", label: "Atlanta to Miami", detail: "" },
    ]);
  });

  it("drops a question nothing can answer rather than rendering an empty card", () => {
    const unanswerable = { ...window, options: [], allowOther: false };
    expect(askRequestsFrom([exchange("ask_user", fenced(unanswerable))])).toEqual([]);
  });

  it("ignores the runtime's refusal to ask, which is prose and not a question", () => {
    expect(
      askRequestsFrom([exchange("ask_user", "ask_user needs a question. Say what you need…")]),
    ).toEqual([]);
  });

  it("ignores every other tool", () => {
    expect(askRequestsFrom([exchange("list_reports", fenced({ results: [] }))])).toEqual([]);
  });
});
