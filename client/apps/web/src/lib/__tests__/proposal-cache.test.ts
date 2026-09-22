import {
  applyDecision,
  decidedStatus,
  invalidateProposalViews,
  markProposalDecided,
} from "@/lib/proposal-cache";
import { queries } from "@/lib/queries";
import type { AssistantProposal } from "@/types/assistant";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

function pending(id: string): AssistantProposal {
  return {
    id,
    status: "Pending",
    toolName: "create_report",
    executionError: "",
  } as AssistantProposal;
}

/*
Deciding and executing are two steps. The resolve call records the decision and
returns; the tool runs after it. While the card waited for a refetch to tell it
that, it kept its buttons — and a second click in that window reached a
proposal the server had already decided, refusing it with a message that reads
like a bug: "this proposal has already been decided: it is executed".
*/
describe("decidedStatus", () => {
  it("records what the server will record", () => {
    expect(decidedStatus("Accepted")).toBe("Accepted");
    expect(decidedStatus("Modified")).toBe("Modified");
    expect(decidedStatus("Rejected")).toBe("Rejected");
  });

  // Never Executed: whether the write happened is the execution columns' to
  // say, and claiming it here would tell someone their change landed before
  // anything had run.
  it("never claims the tool ran", () => {
    expect(decidedStatus("Accepted")).not.toBe("Executed");
  });
});

describe("applyDecision", () => {
  it("takes the decided proposal out of the awaiting state", () => {
    const [first] = applyDecision([pending("prop_1")], "prop_1", "Accepted");

    expect(first.status).toBe("Accepted");
  });

  it("leaves every other proposal alone", () => {
    const after = applyDecision([pending("prop_1"), pending("prop_2")], "prop_1", "Rejected");

    expect(after[0].status).toBe("Rejected");
    expect(after[1].status).toBe("Pending");
  });

  it("does not mutate the cached list", () => {
    const before = [pending("prop_1")];
    applyDecision(before, "prop_1", "Accepted");

    expect(before[0].status).toBe("Pending");
  });

  it("is a no-op for a proposal it does not hold", () => {
    const after = applyDecision([pending("prop_1")], "prop_other", "Accepted");

    expect(after[0].status).toBe("Pending");
  });
});

/*
These two go through a real QueryClient seeded at the key a component actually
reads, because the bug they close was not in the update — it was in the address.

createQueryKeys prepends its scope and the method name: what a component reads
under `queries.assistant.proposals(id)` lives at ["assistant", "proposals",
"assistant-proposals", id]. This file used to name ["assistant-proposals", id],
which is a prefix of nothing, and TanStack matches from the start of the key.
Nothing threw. The request succeeded, the mutation resolved, and the cache was
never touched — so an approved proposal kept its buttons until the thread was
remounted, which is indistinguishable from the click not working.

So neither test asserts on the key it targets. They seed the live key and ask
whether the cache moved; a key written out by hand fails them.
*/
const THREAD = "athr_1";

function seededClient() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(queries.assistant.proposals(THREAD).queryKey, {
    results: [pending("prop_1"), pending("prop_2")],
  });

  return client;
}

function proposalsIn(client: QueryClient, threadId: string): AssistantProposal[] {
  return (
    client.getQueryData<{ results: AssistantProposal[] }>(
      queries.assistant.proposals(threadId).queryKey,
    )?.results ?? []
  );
}

describe("markProposalDecided", () => {
  it("writes the decision into the list the thread is reading", () => {
    const client = seededClient();

    markProposalDecided(client, "prop_1", "Accepted");

    expect(proposalsIn(client, THREAD).map((p) => p.status)).toEqual(["Accepted", "Pending"]);
  });

  // The same proposal can be cached under more than one thread — AI Control and
  // the chat are different threads over the same records — so the write is not
  // scoped to the one that was clicked.
  it("reaches every thread's cached list", () => {
    const client = seededClient();
    client.setQueryData(queries.assistant.proposals("athr_2").queryKey, {
      results: [pending("prop_1")],
    });

    markProposalDecided(client, "prop_1", "Rejected");

    expect(proposalsIn(client, "athr_2")[0]?.status).toBe("Rejected");
  });

  it("leaves a thread that does not hold the proposal untouched", () => {
    const client = seededClient();

    markProposalDecided(client, "prop_elsewhere", "Accepted");

    expect(proposalsIn(client, THREAD).map((p) => p.status)).toEqual(["Pending", "Pending"]);
  });
});

/** Whether a refetch was actually asked for, rather than which key was named. */
function invalidated(client: QueryClient, queryKey: readonly unknown[]): boolean {
  return client.getQueryState(queryKey)?.isInvalidated === true;
}

describe("invalidateProposalViews", () => {
  it("makes the deciding thread's own views stale", async () => {
    const client = seededClient();
    client.setQueryData(queries.assistant.plans(THREAD).queryKey, { results: [] });
    client.setQueryData(queries.assistant.messages(THREAD).queryKey, { results: [] });
    client.setQueryData(queries.assistant.artifacts(THREAD).queryKey, { results: [] });

    await invalidateProposalViews(client, THREAD);

    expect(invalidated(client, queries.assistant.proposals(THREAD).queryKey)).toBe(true);
    expect(invalidated(client, queries.assistant.plans(THREAD).queryKey)).toBe(true);
    expect(invalidated(client, queries.assistant.messages(THREAD).queryKey)).toBe(true);
    expect(invalidated(client, queries.assistant.artifacts(THREAD).queryKey)).toBe(true);
  });

  // A decision made in AI Control has no thread. It still has to reach the
  // chat, which is why the thread-scoped keys fall back to their scope.
  it("makes every thread's views stale when the decision had no thread", async () => {
    const client = seededClient();
    client.setQueryData(queries.assistant.proposals("athr_2").queryKey, { results: [] });

    await invalidateProposalViews(client);

    expect(invalidated(client, queries.assistant.proposals(THREAD).queryKey)).toBe(true);
    expect(invalidated(client, queries.assistant.proposals("athr_2").queryKey)).toBe(true);
  });

  // The other side of the same complaint: a card approved in the chat left AI
  // Control's tables and the attention badge showing a question already
  // answered.
  it("makes the AI Control tables and the waiting counts stale", async () => {
    const client = seededClient();
    const plain = [
      ["agent-proposal-list"],
      ["agent-plan-list"],
      ["agent-run-list"],
      ["pending-decisions"],
      ["pending-decision-summary"],
      ["attention"],
    ];
    for (const key of plain) {
      client.setQueryData([...key, "page-1"], {});
    }

    await invalidateProposalViews(client, THREAD);

    for (const key of plain) {
      expect(invalidated(client, [...key, "page-1"])).toBe(true);
    }
  });
});
