import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantService } from "../assistant";

const server = vi.hoisted(() => ({ body: undefined as unknown, urls: [] as string[] }));

vi.mock("@trenova/shared/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/api")>();
  return {
    ...actual,
    api: {
      ...actual.api,
      get: vi.fn(async (url: string) => {
        server.urls.push(url);
        return server.body;
      }),
    },
  };
});

afterEach(() => {
  server.body = undefined;
  server.urls = [];
});

/* A chapter as GET /assistant/threads/:id/chapters/ serves it (conversation.Chapter). */
const served = {
  id: "achp_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  threadId: "athr_1",
  turnId: "atrn_1",
  title: "Acme invoice dispute",
  titleSource: "Person",
  version: 2,
  createdAt: 1_761_600_000,
  updatedAt: 1_761_600_100,
  startSequence: 4,
  startedAt: 1_761_600_000,
};

describe("AssistantService.listChapters", () => {
  it("resolves to the conversation's chapters, never undefined", async () => {
    server.body = { items: [served, { ...served, id: "achp_2", turnId: "atrn_2", startSequence: null }] };

    const chapters = await new AssistantService().listChapters("athr_1");

    expect(server.urls).toEqual(["/assistant/threads/athr_1/chapters/"]);
    expect(chapters.map((chapter) => [chapter.id, chapter.startSequence])).toEqual([
      ["achp_1", 4],
      ["achp_2", null],
    ]);
    expect(chapters[0]).toMatchObject({ title: "Acme invoice dispute", titleSource: "Person" });
  });

  it("resolves to an empty list when the conversation has none or the list is absent", async () => {
    server.body = { items: null };
    await expect(new AssistantService().listChapters("athr_1")).resolves.toEqual([]);

    server.body = {};
    await expect(new AssistantService().listChapters("athr_1")).resolves.toEqual([]);
  });
});
