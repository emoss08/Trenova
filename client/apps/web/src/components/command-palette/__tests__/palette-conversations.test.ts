import type { DeskSearchResult } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import { conversationPage } from "../palette-conversations";

const t = ((text: string) => text) as TranslateFn;

function hit(overrides: Partial<DeskSearchResult>): DeskSearchResult {
  return {
    kind: "chat",
    id: "athr_1",
    threadId: "athr_1",
    agentId: "",
    title: "Which Dallas loads are late?",
    threadTitle: "Which Dallas loads are late?",
    artifactKind: undefined,
    status: "",
    at: 0,
    ...overrides,
  };
}

describe("a conversation in the command palette", () => {
  it("opens the conversation it found", () => {
    const page = conversationPage(hit({}), t);

    expect(page.href).toBe("/desk/t/athr_1");
    expect(page.title).toBe("Which Dallas loads are late?");
    expect(page.trail).toEqual(["Conversation"]);
  });

  it("opens an artifact beside its conversation, under the conversation's name", () => {
    const page = conversationPage(
      hit({ kind: "art", id: "art_9", title: "Six late shipments", threadTitle: "Dallas lanes" }),
      t,
    );

    expect(page.href).toBe("/desk/t/athr_1?a=art_9");
    expect(page.trail).toEqual(["Artifact", "Dallas lanes"]);
  });

  it("opens a chapter at its place in the conversation", () => {
    expect(conversationPage(hit({ kind: "chap", id: "chap_2" }), t).href).toBe(
      "/desk/t/athr_1?c=chap_2",
    );
  });

  it("opens a decision on Decisions", () => {
    expect(conversationPage(hit({ kind: "dec", id: "ap_1" }), t).href).toBe("/desk/decisions");
  });

  it("names an untitled conversation rather than leaving the row blank", () => {
    expect(conversationPage(hit({ title: "", threadTitle: "" }), t).title).toBe(
      "Untitled conversation",
    );
  });
});
