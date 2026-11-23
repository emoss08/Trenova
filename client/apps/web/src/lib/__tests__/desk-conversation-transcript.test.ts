import { describe, expect, it } from "vitest";
import type {
  AgentRunTranscript,
  AgentRunTranscriptMessage,
} from "@/lib/graphql/agent-activity-tables";
import { conversationItems, type ConversationItem } from "../desk-conversation-transcript";

/** One message as AgentRunTranscriptFields returns it, every field present. */
function message(fields: Partial<AgentRunTranscriptMessage>): AgentRunTranscriptMessage {
  return {
    role: "Assistant",
    kind: "Message",
    content: "",
    reasoning: "",
    toolCalls: [],
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    toolVerdict: "",
    toolSummary: "",
    agentDefinitionId: "",
    delegateCallId: "",
    omitted: false,
    createdAt: 0,
    ...fields,
  };
}

function transcript(
  messages: AgentRunTranscriptMessage[],
  omittedMessages = 0,
  omittedAt = 0,
): AgentRunTranscript {
  return { messages, omittedMessages, omittedAt };
}

const QUESTION = message({
  role: "User",
  content: "Which GlobalTrade shipments are open?",
  createdAt: 1_790_000_000,
});

const NOTE = message({
  content: "I'll pull the open GlobalTrade shipments and their charges.",
  toolCalls: [{ id: "call_1", name: "search_shipments", arguments: { customer: "GlobalTrade" } }],
  createdAt: 1_790_000_002,
});

const RESULT = message({
  role: "Tool",
  toolCallId: "call_1",
  toolName: "search_shipments",
  content: '{"count":3}',
  toolSummary: "3 shipments",
  createdAt: 1_790_000_004,
});

const ANSWER = message({
  content: "Three shipments are open, excluding the two already invoiced.",
  reasoning: "Open means not yet invoiced.",
  createdAt: 1_790_000_009,
});

function kinds(items: ConversationItem[]): string[] {
  return items.map((item) => item.kind);
}

function turnAt(items: ConversationItem[], index: number) {
  const item = items[index];
  if (item?.kind !== "turn") {
    throw new Error(`item ${index} is ${item?.kind ?? "missing"}, not a turn`);
  }
  return item;
}

describe("conversationItems", () => {
  it("sets the agent's last words as the answer and its notes and calls apart as its working", () => {
    const items = conversationItems("athr_01", transcript([QUESTION, NOTE, RESULT, ANSWER]));

    expect(kinds(items)).toEqual(["asked", "turn"]);
    expect(items[0]).toMatchObject({
      kind: "asked",
      tag: null,
      message: { content: "Which GlobalTrade shipments are open?" },
    });
    const turn = turnAt(items, 1);
    expect(turn.reply?.message.content).toBe(
      "Three shipments are open, excluding the two already invoiced.",
    );
    expect(turn.reply?.message.reasoning?.text).toBe("Open means not yet invoiced.");
    expect(turn.work.map((step) => step.message.content)).toEqual([
      "I'll pull the open GlobalTrade shipments and their charges.",
    ]);
    expect(turn.work[0].tools).toHaveLength(1);
    expect(turn.work[0].tools[0].call.name).toBe("search_shipments");
    expect(turn.work[0].tools[0].result?.summary).toBe("3 shipments");
    expect(turn.askedAt).toBe(1_790_000_000);
    expect(turn.at).toBe(1_790_000_009);
    expect(turn.clipped).toBe(false);
  });

  it("leaves a turn that only called tools without an answer, timed by its last step", () => {
    const items = conversationItems("athr_01", transcript([QUESTION, NOTE, RESULT]));
    const lookup = message({
      toolCalls: [{ id: "call_2", name: "get_invoice", arguments: {} }],
      createdAt: 1_790_000_006,
    });
    const onlyTools = conversationItems(
      "athr_01",
      transcript([QUESTION, { ...NOTE, content: "" }, RESULT, lookup]),
    );

    expect(turnAt(items, 1).reply?.message.content).toBe(
      "I'll pull the open GlobalTrade shipments and their charges.",
    );
    const turn = turnAt(onlyTools, 1);
    expect(turn.reply).toBeNull();
    expect(turn.work).toHaveLength(2);
    expect(turn.work.flatMap((step) => step.tools.map((tool) => tool.call.name))).toEqual([
      "search_shipments",
      "get_invoice",
    ]);
    expect(turn.at).toBe(1_790_000_006);
  });

  it("counts the messages left out of the middle where they fell and never pairs a call with a result across them", () => {
    const later = message({
      role: "User",
      content: "And the invoiced ones?",
      createdAt: 1_790_000_100,
    });
    const laterAnswer = message({ content: "Two, both paid.", createdAt: 1_790_000_105 });
    const items = conversationItems(
      "athr_01",
      transcript([QUESTION, NOTE, RESULT, later, laterAnswer], 12, 2),
    );

    expect(kinds(items)).toEqual(["asked", "turn", "gap", "asked", "turn"]);
    expect(items[2]).toEqual({ kind: "gap", key: "athr_01:gap", count: 12 });
    const before = turnAt(items, 1);
    const after = turnAt(items, 4);
    expect(before.reply?.message.content).toBe(
      "I'll pull the open GlobalTrade shipments and their charges.",
    );
    expect(before.reply?.tools[0].result).toBeNull();
    expect(items[3]).toMatchObject({
      kind: "asked",
      key: "athr_01:15",
      message: { content: "And the invoiced ones?" },
    });
    expect(after.reply?.message.id).toBe("athr_01:16");
    expect(after.reply?.tools[0]).toMatchObject({
      orphan: true,
      call: { id: "call_1" },
      result: { id: "athr_01:14" },
    });
    expect(after.askedAt).toBe(1_790_000_100);
  });

  it("marks the turn whose step or result was too large to keep whole", () => {
    const items = conversationItems(
      "athr_01",
      transcript([QUESTION, NOTE, { ...RESULT, omitted: true }, ANSWER]),
    );
    const untouched = conversationItems("athr_01", transcript([QUESTION, ANSWER]));

    expect(turnAt(items, 1).clipped).toBe(true);
    expect(turnAt(untouched, 1).clipped).toBe(false);
  });

  it("keeps what the person sent while the agent worked as their words, and drops a role it does not know", () => {
    const steer = message({
      role: "User",
      kind: "Steer",
      content: "Only this month",
      createdAt: 1_790_000_003,
    });
    const unknown = message({ role: "Narrator", content: "Meanwhile…" });
    const items = conversationItems(
      "athr_01",
      transcript([QUESTION, NOTE, steer, RESULT, unknown, ANSWER]),
    );

    expect(kinds(items)).toEqual(["asked", "turn", "asked", "turn"]);
    expect(items[2]).toMatchObject({ kind: "asked", tag: "steer" });
    expect(turnAt(items, 3).reply?.message.content).toBe(
      "Three shipments are open, excluding the two already invoiced.",
    );
  });

  it("sets an event between turns apart from both sides", () => {
    const decision = message({
      role: "User",
      kind: "DecisionNote",
      content: "Dana approved the rate change.",
      createdAt: 1_790_000_050,
    });
    const items = conversationItems("athr_01", transcript([QUESTION, ANSWER, decision]));

    expect(kinds(items)).toEqual(["asked", "turn", "event"]);
    expect(items[2]).toMatchObject({
      kind: "event",
      message: { kind: "DecisionNote", content: "Dana approved the rate change." },
    });
  });

  it("reads an empty conversation as nothing", () => {
    expect(conversationItems("athr_01", transcript([]))).toEqual([]);
  });
});
