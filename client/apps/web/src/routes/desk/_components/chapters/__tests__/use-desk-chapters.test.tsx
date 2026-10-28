import { assistant } from "@/lib/queries/assistant";
import { assistantChapterSchema, type AssistantChapter, type AssistantThread } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDeskChapters } from "../use-desk-chapters";

vi.mock("@/components/assistant/use-active-turns", () => ({ useLiveTurns: () => [] }));

afterEach(cleanup);

const thread = {
  id: "athr_1",
  title: "Stuck loads",
  agentDefinitionId: "agdef_1",
  canContinue: true,
  pinnedFacts: [],
} as unknown as AssistantThread;

/* A chapter as GET /assistant/threads/:id/chapters/ serves it. */
function chapter(n: number): AssistantChapter {
  return assistantChapterSchema.parse({
    id: `achp_${n}`,
    threadId: thread.id,
    turnId: `atrn_${n}`,
    title: `Chapter ${n}`,
    titleSource: n === 1 ? "Question" : "Model",
    version: 1,
    createdAt: n * 100,
    updatedAt: n * 100,
    startSequence: n * 4,
    startedAt: n * 100,
  });
}

function renderChapters(chapters: AssistantChapter[]) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity, enabled: false } },
  });
  client.setQueryData(assistant.chapters(thread.id).queryKey, chapters);
  client.setQueryData(assistant.artifacts(thread.id)._ctx.summary.queryKey, { results: [] });
  client.setQueryData(assistant.proposals(thread.id).queryKey, { results: [] });
  client.setQueryData(assistant.waits(thread.id).queryKey, { items: [] });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );

  return renderHook(
    () =>
      useDeskChapters({
        thread,
        agent: null,
        agents: [],
        timezone: "UTC",
        onHandedOff: () => undefined,
      }),
    { wrapper },
  ).result.current;
}

describe("useDeskChapters", () => {
  it("draws no heading while a conversation has only its first chapter", () => {
    const chapters = renderChapters([chapter(1)]);

    expect(chapters.heading("atrn_1")).toBeNull();
    expect(chapters.liveTurnId).toBeNull();
  });

  it("draws every chapter's heading once there is a second", () => {
    const chapters = renderChapters([chapter(1), chapter(2)]);

    expect(chapters.heading("atrn_1")).not.toBeNull();
    expect(chapters.heading("atrn_2")).not.toBeNull();
    expect(chapters.heading("atrn_9")).toBeNull();
  });

  it("offers to start a chapter only where a turn opens none", () => {
    const chapters = renderChapters([chapter(1)]);

    expect(chapters.startAt({ id: "amsg_1", turnId: "atrn_1" } as never)).toBeUndefined();
    expect(chapters.startAt({ id: "amsg_5", turnId: "atrn_5" } as never)).toBeTypeOf("function");
    expect(chapters.startAt({ id: "amsg_6", turnId: null } as never)).toBeUndefined();
  });
});
