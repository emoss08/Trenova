import { translate } from "@trenova/shared/i18n/runtime";
import { assistantMessageSchema, type AssistantMessage } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { stepsFromExchanges } from "../activity";
import { classifyMessage } from "../classify-message";
import { delegateView, handOffHeadline, handOffStatus, madeLine } from "../delegation";
import { turnEndByMessage } from "../proposal-state";
import { delegatedOwners, groupThread, turnPlacements } from "../thread-view";

const t = translate;

let sequence = 0;

/**
 * A saved message as the messages API serves it (domain/conversation
 * Message): the agent fields are omitted on the conversation's own messages
 * and set on another agent's, so each fixture goes through the schema.
 */
function saved(fields: Record<string, unknown>): AssistantMessage {
  sequence += 1;
  return assistantMessageSchema.parse({
    id: `amsg_${sequence}`,
    threadId: "thr_1",
    sequence,
    role: "User",
    kind: "Message",
    content: "",
    createdAt: 1_700_000_000 + sequence,
    ...fields,
  });
}

const AGENT = "agdef_report_builder";
const CALL = "call_delegate_1";

function fenced(payload: unknown): string {
  return `Result from delegate_task:\n<untrusted_data>\n${JSON.stringify(payload)}\n</untrusted_data>\n\n[Report Builder did this as the same person.]`;
}

const REPORT = {
  delegateCallId: CALL,
  agentId: AGENT,
  agentName: "Report Builder",
  status: "completed",
  reply: 'I saved the report "On-time this month".',
  made: [
    {
      toolName: "create_report",
      callId: "call_inner_1",
      tier: "AutoExecute",
      summary: "On-time this month",
      result: {
        action: "created",
        kind: "report",
        name: "On-time this month",
        ids: { definitionId: "rd_1" },
      },
    },
  ],
  awaiting: [
    { toolName: "share_report", callId: "call_inner_2", tier: "Propose", summary: "On-time" },
  ],
  published: [],
  toolCallsUsed: 2,
};

/** One turn that handed a task over, saved the way FinishTurn keeps it. */
function handOffThread(result: string = fenced(REPORT), failed = false) {
  sequence = 0;
  const delegated = { kind: "Delegated", agentId: AGENT, agentName: "Report Builder" };
  return [
    saved({ role: "User", content: "Build a dashboard of on-time delivery" }),
    saved({
      role: "Assistant",
      content: "",
      toolCalls: [
        {
          id: CALL,
          name: "delegate_task",
          arguments: { agentId: AGENT, task: "Create a report of on-time deliveries" },
          effect: "delegate",
        },
      ],
    }),
    saved({
      ...delegated,
      delegateCallId: CALL,
      role: "User",
      content: "Create a report of on-time deliveries",
    }),
    saved({
      ...delegated,
      delegateCallId: CALL,
      role: "Assistant",
      toolCalls: [{ id: "call_inner_1", name: "create_report", arguments: {} }],
    }),
    saved({
      ...delegated,
      delegateCallId: CALL,
      role: "Tool",
      toolCallId: "call_inner_1",
      toolName: "create_report",
      content: "{}",
      summary: "On-time this month",
    }),
    saved({
      ...delegated,
      delegateCallId: CALL,
      role: "Assistant",
      content: "I saved the report.",
    }),
    saved({
      role: "Tool",
      toolCallId: CALL,
      toolName: "delegate_task",
      toolFailed: failed,
      content: result,
      summary: "Report Builder",
    }),
    saved({ role: "Assistant", content: "Your dashboard is ready." }),
  ];
}

/**
 * Another agent's steps are saved between the delegate_task call and its
 * result, the task first in the User role. Drawn by role they would read as
 * the person asking and the conversation's agent answering; they belong
 * under the call and nowhere else.
 */
describe("groupThread with hand-offs", () => {
  it("never draws another agent's steps as turns of their own", () => {
    const entries = groupThread(handOffThread());

    expect(entries.map((entry) => entry.kind)).toEqual(["user", "assistant", "assistant"]);
    expect(entries.map((entry) => entry.message.content)).toEqual([
      "Build a dashboard of on-time delivery",
      "",
      "Your dashboard is ready.",
    ]);
  });

  it("keeps them under the delegate_task call, in order, with its result", () => {
    const [, turn] = groupThread(handOffThread());
    if (turn.kind !== "assistant") throw new Error("expected an assistant entry");

    expect(turn.tools).toHaveLength(1);
    const [exchange] = turn.tools;
    expect(exchange.call.id).toBe(CALL);
    expect(exchange.result?.toolName).toBe("delegate_task");
    expect(exchange.delegated?.map((message) => message.role)).toEqual([
      "User",
      "Assistant",
      "Tool",
      "Assistant",
    ]);
  });

  it("keeps the reply one turn: the task does not start a new one", () => {
    const entries = groupThread(handOffThread());
    const placements = turnPlacements(entries);

    expect(placements.get(entries[1].message.id)?.continued).toBe(false);
    expect(placements.get(entries[2].message.id)?.continued).toBe(true);
  });

  // The page holding the call is older than the page in view.
  it("puts steps whose call is out of view under the call's result", () => {
    const thread = handOffThread();
    const entries = groupThread(thread.slice(2));

    expect(entries.map((entry) => entry.kind)).toEqual(["assistant"]);
    const [turn] = entries;
    if (turn.kind !== "assistant") throw new Error("expected an assistant entry");
    expect(turn.tools[0]).toMatchObject({ orphan: true, call: { id: CALL } });
    expect(turn.tools[0].delegated).toHaveLength(4);
  });

  it("names the entry each of another agent's messages and calls is drawn under", () => {
    const thread = handOffThread();
    const owners = delegatedOwners(groupThread(thread));

    expect(owners.get(thread[3].id)).toBe(thread[1].id);
    expect(owners.get("call_inner_1")).toBe(thread[1].id);
    expect(owners.has(thread[7].id)).toBe(false);
  });
});

describe("classifyMessage", () => {
  it("reads another agent's step as delegated whatever its role or refusal", () => {
    for (const role of ["User", "Assistant", "Tool"]) {
      expect(
        classifyMessage(saved({ role, kind: "Delegated", refused: true, delegateCallId: CALL })),
      ).toBe("delegated");
    }
  });

  it("reads a kind this client does not know as an ordinary message", () => {
    expect(classifyMessage(saved({ role: "User", kind: "Whisper" }))).toBe("user");
  });
});

/**
 * What the other agent proposed is recorded against its own message, which
 * is never drawn as an entry. The card goes where the turn's own do: under
 * the turn's last words.
 */
describe("turnEndByMessage with hand-offs", () => {
  it("anchors another agent's messages to the end of the turn that handed it the task", () => {
    const thread = handOffThread();
    const ends = turnEndByMessage(thread);

    expect(ends.get(thread[3].id)).toBe(thread[7].id);
    expect(ends.get(thread[1].id)).toBe(thread[7].id);
  });

  it("leaves them unanchored when none of the turn's own words are in view", () => {
    const thread = handOffThread();
    const ends = turnEndByMessage(thread.slice(2, 6));

    expect(ends.has(thread[3].id)).toBe(false);
  });
});

/**
 * A past conversation reads the same as the one being watched: the hand-off
 * is rebuilt from the saved steps and the account in the call's result.
 */
describe("delegateView from a saved thread", () => {
  function savedStep(result?: string, failed?: boolean) {
    const [, turn] = groupThread(handOffThread(result, failed));
    if (turn.kind !== "assistant") throw new Error("expected an assistant entry");
    const [step] = stepsFromExchanges(turn.tools, turn.message.createdAt);
    return step;
  }

  it("reads the task, the other agent's steps, its answer and the account", () => {
    const step = savedStep();
    const view = delegateView(step);

    expect(step.effect).toBe("delegate");
    expect(view).toMatchObject({
      agentId: AGENT,
      agentName: "Report Builder",
      task: "Create a report of on-time deliveries",
      reply: 'I saved the report "On-time this month".',
      outcome: "completed",
    });
    expect(view?.steps.map((inner) => [inner.name, inner.summary])).toEqual([
      ["create_report", "On-time this month"],
    ]);
    expect(view?.report?.awaiting).toHaveLength(1);
    expect(handOffHeadline(view!, t)).toBe("Asked Report Builder");
  });

  it("reads a hand-off refused before it began as declined, with the reason", () => {
    const view = delegateView(
      savedStep('Tool "delegate_task" was not run: Report Builder is disabled.', true),
    );

    expect(view).toMatchObject({ outcome: "declined", reason: "Report Builder is disabled." });
    expect(handOffHeadline(view!, t)).toBe("Couldn't ask Report Builder");
    expect(handOffStatus(view!, t)).toEqual({
      text: "Could not be asked: Report Builder is disabled.",
      tone: "danger",
    });
  });

  it("reads a task that broke off as failed, keeping what it made", () => {
    const view = delegateView(
      savedStep(
        fenced({ ...REPORT, status: "failed", reply: "", reason: "The model stopped." }),
        true,
      ),
    );

    expect(view?.outcome).toBe("failed");
    expect(view?.report?.made).toHaveLength(1);
    expect(view?.reply).toBe("I saved the report.");
    expect(handOffStatus(view!, t)).toEqual({
      text: "Stopped partway: The model stopped.",
      tone: "danger",
    });
  });
});

/**
 * A conversation saved since the account was kept on the call's result reads
 * it from there: structured, whole where the text was cut, and carrying the
 * agent's mark. The fenced text is read only for one saved before.
 */
describe("delegateView from the account the result keeps", () => {
  function withAccount(
    thread: AssistantMessage[],
    account: Record<string, unknown>,
    stepFields: Record<string, unknown> = {},
  ): AssistantMessage[] {
    return thread.map((message) => {
      if (message.role === "Tool" && message.toolName === "delegate_task") {
        return assistantMessageSchema.parse({ ...message, delegateReport: account });
      }
      if (message.kind === "Delegated") {
        return assistantMessageSchema.parse({ ...message, ...stepFields });
      }
      return message;
    });
  }

  function firstStep(thread: AssistantMessage[]) {
    const [, turn] = groupThread(thread);
    if (turn.kind !== "assistant") throw new Error("expected an assistant entry");
    const [step] = stepsFromExchanges(turn.tools, turn.message.createdAt);
    return step;
  }

  const ACCOUNT = { ...REPORT, icon: "receipt", accent: "teal" };

  it("reads the account first, even when the result's text was cut short", () => {
    const view = delegateView(
      firstStep(
        withAccount(handOffThread("Result from delegate_task:\n<untrusted_data>\n{"), ACCOUNT),
      ),
    );

    expect(view).toMatchObject({
      outcome: "completed",
      reply: 'I saved the report "On-time this month".',
      icon: "receipt",
      accent: "teal",
    });
    expect(view?.report?.made).toHaveLength(1);
    expect(view?.report?.awaiting).toHaveLength(1);
  });

  it("draws the mark the thread served on the other agent's steps before the account's", () => {
    const view = delegateView(
      firstStep(withAccount(handOffThread(), ACCOUNT, { agentIcon: "truck", agentAccent: "rose" })),
    );

    expect(view).toMatchObject({ icon: "truck", accent: "rose" });
  });

  it("names the agent and its id from the account when its steps are out of view", () => {
    const thread = withAccount(handOffThread(), ACCOUNT).filter(
      (message) => message.kind !== "Delegated",
    );
    const step = firstStep(
      thread.map((message) =>
        message.role === "Tool"
          ? assistantMessageSchema.parse({ ...message, summary: "" })
          : message,
      ),
    );
    const view = delegateView(step);

    expect(view).toMatchObject({ agentId: AGENT, agentName: "Report Builder", icon: "receipt" });
    expect(handOffHeadline(view!, t)).toBe("Asked Report Builder");
  });

  it("prefers the other agent's own saved answer to an account that cut it short", () => {
    const view = delegateView(
      firstStep(withAccount(handOffThread(), { ...ACCOUNT, reply: "I saved the rep…" })),
    );

    expect(view?.reply).toBe("I saved the report.");
  });

  it("keeps what a bounded account says it left out", () => {
    const view = delegateView(
      firstStep(withAccount(handOffThread(), { ...ACCOUNT, moreMade: 3, morePublished: 1 })),
    );

    expect(view?.report).toMatchObject({ moreMade: 3, moreAwaiting: 0, morePublished: 1 });
  });

  it("reads a declined hand-off from its account", () => {
    const view = delegateView(
      firstStep(
        withAccount(
          handOffThread('Tool "delegate_task" was not run: Report Builder is disabled.', true),
          {
            ...ACCOUNT,
            status: "declined",
            reply: "",
            reason: "Report Builder is disabled.",
            made: [],
            awaiting: [],
          },
        ),
      ),
    );

    expect(view).toMatchObject({ outcome: "declined", reason: "Report Builder is disabled." });
  });

  it("links a write by the record its result names", () => {
    const view = delegateView(
      firstStep(
        withAccount(handOffThread(), {
          ...ACCOUNT,
          made: [
            {
              ...REPORT.made[0],
              result: {
                ...REPORT.made[0].result,
                ids: { definitionId: "rd_1", folderId: "f_1" },
                record: { entityType: "report", id: "rd_1" },
              },
            },
          ],
        }),
      ),
    );

    expect(view?.report?.made[0].result?.record).toEqual({ entityType: "report", id: "rd_1" });
    expect(madeLine(view!.report!.made[0], t).path).toBe("/reports/explore/rd_1");
  });
});
