import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { describe, expect, it } from "vitest";
import { askedMessageCount, composerBlock, shouldSendOpeningQuestion } from "../thread-guard";

const agent = { id: "agdef_1", name: "Report Builder" } as AgentChoice;

describe("composerBlock", () => {
  it("opens the composer when the agent is known and the thread has room", () => {
    expect(composerBlock({ agent, agentsUnavailable: false, threadFull: false })).toBeNull();
  });

  it("closes a full thread whatever else is true", () => {
    expect(composerBlock({ agent, agentsUnavailable: true, threadFull: true })).toBe("full");
  });

  // An agent missing from a list that loaded was disabled or is no longer a
  // chat agent. An agent missing because the list never loaded is unknown,
  // and saying it was disabled would be a false report.
  it("tells a failed agent list apart from a disabled agent", () => {
    expect(composerBlock({ agent: null, agentsUnavailable: false, threadFull: false })).toBe(
      "agent-missing",
    );
    expect(composerBlock({ agent: null, agentsUnavailable: true, threadFull: false })).toBe(
      "agents-unavailable",
    );
  });

  it("treats a thread served without canContinue as open, the schema's default", () => {
    expect(composerBlock({ agent, agentsUnavailable: false, threadFull: false })).toBeNull();
    expect(
      composerBlock({ agent, agentsUnavailable: false, threadFull: false, canContinue: true }),
    ).toBeNull();
  });

  // The server decides whether the reader may still ask the agent, and it is
  // the one that would refuse the message. Its answer outranks everything the
  // client knows: an agent still in a list read a minute ago, a list that
  // failed to load, a thread with room left, or one that is full.
  it("makes a thread the server says cannot continue read-only, whatever else is true", () => {
    for (const agentsUnavailable of [false, true]) {
      for (const threadFull of [false, true]) {
        for (const known of [agent, null]) {
          expect(
            composerBlock({ agent: known, agentsUnavailable, threadFull, canContinue: false }),
          ).toBe("read-only");
        }
      }
    }
  });
});

describe("shouldSendOpeningQuestion", () => {
  const ready = {
    question: "Where is PRO S12345?",
    alreadySent: false,
    historyLoading: false,
    messageCount: 0,
  };

  it("sends a question handed over from the front page", () => {
    expect(shouldSendOpeningQuestion(ready)).toBe(true);
  });

  it("stays quiet when there is no question", () => {
    expect(shouldSendOpeningQuestion({ ...ready, question: undefined })).toBe(false);
    expect(shouldSendOpeningQuestion({ ...ready, question: "   " })).toBe(false);
  });

  it("sends only once", () => {
    expect(shouldSendOpeningQuestion({ ...ready, alreadySent: true })).toBe(false);
  });

  // The guard that matters. History arrives asynchronously, so before it
  // lands every thread looks empty; asking then would ask into a thread that
  // may already have said it.
  it("waits for history rather than trusting an empty thread", () => {
    expect(shouldSendOpeningQuestion({ ...ready, historyLoading: true })).toBe(false);
  });

  it("refuses a thread that already has messages", () => {
    expect(shouldSendOpeningQuestion({ ...ready, messageCount: 1 })).toBe(false);
  });
});

describe("askedMessageCount", () => {
  it("leaves out the brief a hand-off opens a conversation with", () => {
    expect(askedMessageCount([{ kind: "HandoffBrief" }])).toBe(0);
    expect(
      askedMessageCount([{ kind: "HandoffBrief" }, { kind: null }, { kind: "Steer" }, {}]),
    ).toBe(3);
    expect(askedMessageCount([])).toBe(0);
  });

  it("lets the question handed over be asked in a conversation holding only its brief", () => {
    expect(
      shouldSendOpeningQuestion({
        question: "When will the Acme load arrive?",
        alreadySent: false,
        historyLoading: false,
        messageCount: askedMessageCount([{ kind: "HandoffBrief" }]),
      }),
    ).toBe(true);
  });
});
