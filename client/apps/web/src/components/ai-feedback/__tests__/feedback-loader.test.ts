import { describe, expect, it, vi } from "vitest";
import type { AiFeedback, AiFeedbackTarget } from "@/lib/graphql/ai-feedback";
import { answerMessageIds, delegatedAnswerId } from "../feedback-targets";
import { reasonsMatching, toggleReason } from "../feedback-reasons";
import { createAiFeedbackLoader, optimisticFeedback } from "../use-ai-feedback";
import type { AssistantMessage } from "@/types/assistant";

function row(target: AiFeedbackTarget, rating: 1 | -1): AiFeedback {
  return {
    id: `aifb_${target.targetId}`,
    targetType: target.targetType,
    targetId: target.targetId,
    targetPart: target.targetPart ?? "",
    rating,
    reasons: [],
    comment: "",
    version: 1,
    updatedAt: 1,
  };
}

describe("createAiFeedbackLoader", () => {
  it("answers every target asked for in one tick from one request", async () => {
    const a: AiFeedbackTarget = { targetType: "Insight", targetId: "ins_a" };
    const b: AiFeedbackTarget = { targetType: "Insight", targetId: "ins_b" };
    const fetcher = vi.fn().mockResolvedValue([row(b, 1)]);
    const loader = createAiFeedbackLoader(fetcher);

    const [first, second, again] = await Promise.all([
      loader.load(a),
      loader.load(b),
      loader.load(a),
    ]);

    expect(fetcher).toHaveBeenCalledTimes(1);
    expect(fetcher.mock.calls[0][0]).toEqual([a, b]);
    expect(first).toBeNull();
    expect(again).toBeNull();
    expect(second?.rating).toBe(1);
  });

  it("splits a screen larger than the server's limit", async () => {
    const fetcher = vi.fn().mockResolvedValue([]);
    const loader = createAiFeedbackLoader(fetcher);
    const targets = Array.from({ length: 201 }, (_, index) => ({
      targetType: "AssistantMessage" as const,
      targetId: `amsg_${index}`,
    }));

    await Promise.all(targets.map((target) => loader.load(target)));

    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(fetcher.mock.calls[0][0]).toHaveLength(200);
    expect(fetcher.mock.calls[1][0]).toHaveLength(1);
  });

  it("rejects every waiter when the read fails", async () => {
    const loader = createAiFeedbackLoader(vi.fn().mockRejectedValue(new Error("down")));

    await expect(loader.load({ targetType: "Briefing", targetId: "brf_1" })).rejects.toThrow(
      "down",
    );
  });

  it("does not ask for a target whose only query was cancelled", async () => {
    const fetcher = vi.fn().mockResolvedValue([]);
    const loader = createAiFeedbackLoader(fetcher);
    const controller = new AbortController();

    const pending = loader.load({ targetType: "Insight", targetId: "ins_a" }, controller.signal);
    controller.abort();

    await expect(pending).rejects.toBeDefined();
    await new Promise((resolve) => setTimeout(resolve, 5));
    expect(fetcher).not.toHaveBeenCalled();
  });
});

describe("optimisticFeedback", () => {
  it("keeps the saved row's id and version while showing the new rating", () => {
    const target: AiFeedbackTarget = { targetType: "Insight", targetId: "ins_a" };
    const next = optimisticFeedback({ ...row(target, 1), version: 4 }, target, {
      rating: -1,
      reasons: ["Inaccurate"],
      comment: "wrong lane",
    });

    expect(next.id).toBe("aifb_ins_a");
    expect(next.version).toBe(4);
    expect(next.rating).toBe(-1);
    expect(next.reasons).toEqual(["Inaccurate"]);
  });
});

describe("feedback reasons", () => {
  it("keeps only the reasons on the rating's side", () => {
    expect(reasonsMatching(1, ["Inaccurate", "Helpful", "SavedTime"])).toEqual([
      "Helpful",
      "SavedTime",
    ]);
    expect(reasonsMatching(-1, ["Helpful", "Unsafe"])).toEqual(["Unsafe"]);
  });

  it("toggles a reason on and off", () => {
    expect(toggleReason(["Inaccurate"], "Incomplete")).toEqual(["Inaccurate", "Incomplete"]);
    expect(toggleReason(["Inaccurate", "Incomplete"], "Inaccurate")).toEqual(["Incomplete"]);
  });
});

function message(id: string, content: string, role: AssistantMessage["role"] = "Assistant") {
  return { id, content, role } as AssistantMessage;
}

describe("answerMessageIds", () => {
  it("offers one rating per reply, on its last step that says anything", () => {
    const entries = [
      { kind: "user", message: message("q1", "Where is load 12?", "User") },
      { kind: "assistant", message: message("a1", "Let me look.") },
      { kind: "assistant", message: message("a2", "It is in Dallas.") },
      { kind: "assistant", message: message("a3", "") },
      { kind: "user", message: message("q2", "Thanks", "User") },
      { kind: "assistant", message: message("a4", "You're welcome.") },
    ];

    expect([...answerMessageIds(entries)]).toEqual(["a2", "a4"]);
  });

  it("offers nothing for a reply that said nothing", () => {
    const entries = [
      { kind: "user", message: message("q1", "Run it", "User") },
      { kind: "assistant", message: message("a1", "") },
    ];

    expect(answerMessageIds(entries).size).toBe(0);
  });
});

describe("delegatedAnswerId", () => {
  it("is the other agent's last message that says anything", () => {
    expect(
      delegatedAnswerId([
        message("d1", "Build the report", "User"),
        message("d2", "Working on it"),
        message("d3", "{}", "Tool"),
        message("d4", "Here it is."),
        message("d5", ""),
      ]),
    ).toBe("d4");
    expect(delegatedAnswerId([message("d1", "task", "User")])).toBeNull();
  });
});
