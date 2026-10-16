import { describe, expect, it } from "vitest";
import type { AssistantMessage } from "@/types/assistant";
import {
  answersInComposer,
  askAnswerIds,
  askRequestsFrom,
  liveAskRequest,
  threadAsks,
  type ThreadAskRequest,
} from "./ask-requests";
import type { ThreadEntry, ToolExchange } from "./thread-view";
import type { TurnSegment } from "./turn-stream";

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
        suggestion: "",
        input: { type: "text" },
        fields: [],
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

describe("askRequestsFrom, the agent's suggested answer", () => {
  const partnerCode = {
    question: "What partner code should Meritor have in Trenova?",
    options: [],
    allowOther: true,
    otherHint: "A short code, such as MERITOR",
    suggestion: "MERITOR",
    note: "The person has been shown this question…",
  };
  const suggestionOf = (payload: object) =>
    askRequestsFrom([exchange("ask_user", fenced(payload))])[0].suggestion;

  it("reads the value the agent would use for a value in words", () => {
    expect(suggestionOf(partnerCode)).toBe("MERITOR");
  });

  it("is empty when the agent offered none", () => {
    expect(suggestionOf({ ...partnerCode, suggestion: undefined })).toBe("");
    expect(suggestionOf({ ...partnerCode, suggestion: null })).toBe("");
    expect(suggestionOf({ ...partnerCode, suggestion: 42 })).toBe("");
    expect(suggestionOf({ ...partnerCode, suggestion: " \n " })).toBe("");
  });

  it("folds it onto one line", () => {
    expect(suggestionOf({ ...partnerCode, suggestion: "  Chicago\n  to\tDallas " })).toBe(
      "Chicago to Dallas",
    );
  });

  it("drops one longer than the contract allows rather than cutting it", () => {
    expect(suggestionOf({ ...partnerCode, suggestion: "é".repeat(120) })).toBe("é".repeat(120));
    expect(suggestionOf({ ...partnerCode, suggestion: "a".repeat(121) })).toBe("");
  });

  it("offers none beside choices, a value in a shape or a form", () => {
    expect(suggestionOf({ ...partnerCode, options: [{ value: "MER", label: "MER" }] })).toBe("");
    expect(suggestionOf({ ...partnerCode, input: { type: "number" } })).toBe("");
    expect(
      suggestionOf({
        ...partnerCode,
        fields: [{ key: "code", label: "Partner code", type: "text" }],
      }),
    ).toBe("");
  });
});

describe("askRequestsFrom, the value the person types", () => {
  const ask = (input: unknown) =>
    askRequestsFrom([exchange("ask_user", fenced({ ...window, input }))])[0].input;

  it("reads an absent or null input as words", () => {
    expect(askRequestsFrom([exchange("ask_user", fenced(window))])[0].input).toEqual({
      type: "text",
    });
    expect(ask(null)).toEqual({ type: "text" });
    expect(ask("number")).toEqual({ type: "text" });
  });

  it("keeps a number, a day and a moment", () => {
    expect(ask({ type: "number" })).toEqual({ type: "number" });
    expect(ask({ type: "date" })).toEqual({ type: "date" });
    expect(ask({ type: "datetime" })).toEqual({ type: "datetime" });
  });

  it("reads a type it cannot draw as words", () => {
    expect(ask({ type: "money" })).toEqual({ type: "text" });
    expect(ask({ type: 3 })).toEqual({ type: "text" });
  });

  it("keeps a record of a kind the @ search can find", () => {
    expect(ask({ type: "record", recordKind: "customer" })).toEqual({
      type: "record",
      recordKind: "customer",
    });
    expect(ask({ type: "record", recordKind: "invoice_dispute" })).toEqual({
      type: "record",
      recordKind: "invoice_dispute",
    });
    expect(ask({ type: "record", recordKind: "location" })).toEqual({
      type: "record",
      recordKind: "location",
    });
  });

  // A kind the registry knows but the @ search does not offer has no picker
  // to draw, and a record with no kind has nothing to search among.
  it("reads a record it cannot search for as words", () => {
    expect(ask({ type: "record", recordKind: "service_type" })).toEqual({ type: "text" });
    expect(ask({ type: "record", recordKind: "not_a_kind" })).toEqual({ type: "text" });
    expect(ask({ type: "record" })).toEqual({ type: "text" });
    expect(ask({ type: "record", recordKind: null })).toEqual({ type: "text" });
  });

  it("asks an old question with null options for a value of the person's own", () => {
    const old = { question: "How many days?", options: null, allowOther: true, note: "" };
    const [request] = askRequestsFrom([exchange("ask_user", fenced(old))]);

    expect(request.options).toEqual([]);
    expect(request.allowOther).toBe(true);
    expect(request.fields).toEqual([]);
  });

  it("drops an old question with null options and nothing of the person's own", () => {
    const old = { question: "Which status?", options: null, allowOther: false, note: "" };
    expect(askRequestsFrom([exchange("ask_user", fenced(old))])).toEqual([]);
  });
});

describe("askRequestsFrom, a small form", () => {
  const form = {
    question: "What should the new stop be?",
    options: [],
    allowOther: false,
    fields: [
      { key: "customer", label: "Customer", type: "record", recordKind: "customer" },
      { key: "date", label: "Pickup date", type: "date", hint: "The day the truck arrives" },
      { key: "weight", label: "Weight", type: "number", optional: true },
    ],
    note: "",
  };

  it("reads each field with the control its kind takes", () => {
    expect(askRequestsFrom([exchange("ask_user", fenced(form))])[0].fields).toEqual([
      {
        key: "customer",
        label: "Customer",
        type: "record",
        recordKind: "customer",
        hint: "",
        optional: false,
      },
      {
        key: "date",
        label: "Pickup date",
        type: "date",
        hint: "The day the truck arrives",
        optional: false,
      },
      { key: "weight", label: "Weight", type: "number", hint: "", optional: true },
    ]);
  });

  // The contract says a form comes with no options and no value of the
  // person's own; a model that sends them anyway is not obeyed.
  it("offers neither choices nor a value of the person's own beside a form", () => {
    const mixed = { ...form, options: [{ value: "x", label: "X" }], allowOther: true };
    const [request] = askRequestsFrom([exchange("ask_user", fenced(mixed))]);

    expect(request.options).toEqual([]);
    expect(request.allowOther).toBe(false);
    expect(request.fields).toHaveLength(3);
  });

  it("drops a malformed field and keeps the rest", () => {
    const messy = {
      ...form,
      fields: [
        "Customer",
        null,
        { label: "No key" },
        { key: "", label: "Blank key" },
        { key: "ref", type: "text" },
        { key: "ref", label: "Duplicate key" },
        { key: "kind", label: "Kind", type: "colour" },
        { key: "lane", label: "Lane", type: "record", recordKind: "service_type" },
        { key: "rush", label: "Rush", optional: "yes" },
      ],
    };

    expect(askRequestsFrom([exchange("ask_user", fenced(messy))])[0].fields).toEqual([
      { key: "ref", label: "ref", type: "text", hint: "", optional: false },
      { key: "kind", label: "Kind", type: "text", hint: "", optional: false },
      { key: "lane", label: "Lane", type: "text", hint: "", optional: false },
      { key: "rush", label: "Rush", type: "text", hint: "", optional: false },
    ]);
  });

  it("keeps a form to eight values", () => {
    const long = {
      ...form,
      fields: Array.from({ length: 10 }, (_, index) => ({ key: `f${index}`, label: "F" })),
    };
    expect(askRequestsFrom([exchange("ask_user", fenced(long))])[0].fields).toHaveLength(8);
  });

  it("falls back to the choices when no field survives", () => {
    const broken = {
      ...form,
      fields: [{ label: "No key" }],
      options: [{ value: "7", label: "7 days" }],
      allowOther: true,
    };
    const [request] = askRequestsFrom([exchange("ask_user", fenced(broken))]);

    expect(request.fields).toEqual([]);
    expect(request.options).toEqual([{ value: "7", label: "7 days", detail: "" }]);
    expect(request.allowOther).toBe(true);
  });

  it("drops a question whose form is all malformed and that offers nothing else", () => {
    const broken = { ...form, fields: [{ label: "No key" }], options: null, allowOther: false };
    expect(askRequestsFrom([exchange("ask_user", fenced(broken))])).toEqual([]);
  });

  it("reads an empty or null list of fields as no form", () => {
    for (const fields of [[], null]) {
      const [request] = askRequestsFrom([exchange("ask_user", fenced({ ...window, fields }))]);
      expect(request.fields).toEqual([]);
      expect(request.options).toHaveLength(2);
    }
  });
});

function messageAt(
  id: string,
  sequence: number,
  overrides: Partial<AssistantMessage> = {},
): AssistantMessage {
  return {
    ...(exchange("ask_user", "", sequence).result as AssistantMessage),
    id,
    sequence,
    role: "User",
    kind: "Message",
    content: "",
    toolCallId: "",
    toolName: "",
    ...overrides,
  };
}

/** A reply that asked the question in `payload`, its result saved at `sequence`. */
function asking(id: string, sequence: number, payload: unknown = window): ThreadEntry {
  return {
    kind: "assistant",
    message: messageAt(id, sequence - 1, { role: "Assistant" }),
    tools: [exchange("ask_user", fenced(payload), sequence)],
  };
}

function person(id: string, sequence: number, content: string): ThreadEntry {
  return { kind: "user", message: messageAt(id, sequence, { content }) };
}

describe("threadAsks", () => {
  it("pairs each question with the first thing the person said after it", () => {
    const entries = [
      person("q1", 1, "Run the late report"),
      asking("r1", 3),
      person("q2", 4, "30"),
      person("q3", 5, "And for last month?"),
    ];
    const { byReply, open } = threadAsks(entries);

    expect(open).toBeNull();
    const [ask] = byReply.get("r1") ?? [];
    expect(ask.request.callId).toBe("call_3");
    expect(ask.answer?.id).toBe("q2");
  });

  it("leaves the newest unanswered question open", () => {
    const entries = [
      person("q1", 1, "Run the late report"),
      asking("r1", 3),
      person("q2", 4, "30"),
      asking("r2", 6, { ...window, question: "Which customer?" }),
      asking("r3", 8, { ...window, question: "Which lane?" }),
    ];
    const { byReply, open } = threadAsks(entries);

    expect(open?.question).toBe("Which lane?");
    expect(byReply.get("r2")?.[0].answer).toBeNull();
    expect(byReply.get("r3")?.[0].answer).toBeNull();
    expect(byReply.get("r1")?.[0].answer?.id).toBe("q2");
  });

  // Only what the person said answers; a compaction summary is saved in the
  // User role because the model reads it in their place, but they never said it.
  it("is not answered by a message the person did not write", () => {
    const entries: ThreadEntry[] = [
      asking("r1", 3),
      { kind: "compaction", message: messageAt("c1", 4, { kind: "Compaction" }) },
    ];

    expect(threadAsks(entries).open?.callId).toBe("call_3");
  });

  it("names the messages that answered a question, and no other", () => {
    const entries = [
      person("q1", 1, "Run the late report"),
      asking("r1", 3),
      person("q2", 4, "30"),
      person("q3", 5, "And for last month?"),
      asking("r2", 7, { ...window, question: "Which customer?" }),
    ];

    expect([...askAnswerIds(threadAsks(entries))]).toEqual(["q2"]);
  });

  it("holds nothing for a thread that asked nothing", () => {
    const { byReply, open } = threadAsks([person("q1", 1, "Hello")]);

    expect(byReply.size).toBe(0);
    expect(open).toBeNull();
  });
});

describe("liveAskRequest", () => {
  const tool = (callId: string, status: "running" | "done" | "failed", payload: unknown) =>
    ({
      kind: "tool",
      callId,
      name: "ask_user",
      arguments: {},
      status,
      content: status === "running" ? "" : fenced(payload),
    }) satisfies TurnSegment;

  it("reads the newest question the reply has finished asking", () => {
    const request = liveAskRequest([
      tool("call_a", "done", window),
      { kind: "text", text: "One moment.", closed: true },
      tool("call_b", "done", { ...window, question: "Which customer?" }),
    ]);

    expect(request?.callId).toBe("call_b");
    expect(request?.question).toBe("Which customer?");
    expect(request?.sequence).toBe(Number.MAX_SAFE_INTEGER);
  });

  it("asks nothing while the call is running or after it failed", () => {
    expect(liveAskRequest([tool("call_a", "running", window)])).toBeNull();
    expect(liveAskRequest([tool("call_a", "failed", window)])).toBeNull();
  });
});

describe("answersInComposer", () => {
  const base: ThreadAskRequest = {
    sequence: 3,
    callId: "call_3",
    question: "Which window?",
    options: [],
    allowOther: true,
    otherHint: "",
    suggestion: "",
    input: { type: "text" },
    fields: [],
  };

  it("is the composer for a value in words", () => {
    expect(answersInComposer(base)).toBe(true);
  });

  it("is not the composer for a value in a shape, a closed set or a form", () => {
    expect(answersInComposer({ ...base, input: { type: "number" } })).toBe(false);
    expect(answersInComposer({ ...base, allowOther: false })).toBe(false);
    expect(
      answersInComposer({
        ...base,
        allowOther: false,
        fields: [{ key: "a", label: "A", type: "text", hint: "", optional: false }],
      }),
    ).toBe(false);
  });
});
