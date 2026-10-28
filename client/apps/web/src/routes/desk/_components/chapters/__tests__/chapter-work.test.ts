import { assistantChapterSchema, type AgentWait, type AssistantArtifact, type AssistantProposal } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { chapterEntries, shownChapters } from "../chapter-work";

/* A chapter as GET /assistant/threads/:id/chapters/ serves it. */
function chapter(id: string, turnId: string, startedAt: number, startSequence: number | null) {
  return assistantChapterSchema.parse({
    id,
    businessUnitId: "bu_1",
    organizationId: "org_1",
    threadId: "athr_1",
    turnId,
    title: `Chapter ${id}`,
    titleSource: "Model",
    version: 1,
    createdAt: startedAt,
    updatedAt: startedAt,
    startSequence,
    startedAt,
  });
}

const artifact = (createdAt: number, kind = "table_view") =>
  ({ id: `aart_${createdAt}`, kind, createdAt }) as AssistantArtifact;
const proposal = (createdAt: number, status: string) =>
  ({ id: `aprp_${createdAt}`, status, createdAt }) as AssistantProposal;
const wait = (createdAt: number, status: string) =>
  ({ id: `awt_${createdAt}`, status, createdAt }) as AgentWait;

describe("shownChapters", () => {
  it("keeps chapters with a saved message, the live one and the newest, not an older turn that saved nothing", () => {
    const saved = chapter("achp_1", "atrn_1", 100, 0);
    const orphan = chapter("achp_2", "atrn_2", 200, null);
    const newest = chapter("achp_3", "atrn_3", 300, null);

    expect(shownChapters([saved, orphan, newest], "atrn_3").map((c) => c.id)).toEqual([
      "achp_1",
      "achp_3",
    ]);
    expect(shownChapters([saved, orphan, newest], "atrn_2").map((c) => c.id)).toEqual([
      "achp_1",
      "achp_2",
      "achp_3",
    ]);
    expect(shownChapters([saved, orphan], null).map((c) => c.id)).toEqual(["achp_1", "achp_2"]);
    expect(shownChapters([saved, orphan, newest], null).map((c) => c.id)).toEqual([
      "achp_1",
      "achp_3",
    ]);
  });

  it("reads a server list without a start as a chapter still being written", () => {
    const served = assistantChapterSchema.parse({
      id: "achp_1",
      threadId: "athr_1",
      turnId: "atrn_1",
      title: "Acme invoice dispute",
      titleSource: "Person",
      version: 2,
      createdAt: 1,
      updatedAt: 2,
    });

    expect(served.startSequence).toBeNull();
    expect(served.startedAt).toBe(0);
  });
});

describe("chapterEntries", () => {
  it("counts what each chapter made, what waits on the person and what the agent waits on, by when it happened", () => {
    const chapters = [chapter("achp_1", "atrn_1", 100, 0), chapter("achp_2", "atrn_2", 200, 6)];

    const entries = chapterEntries(
      chapters,
      {
        artifacts: [
          artifact(50),
          artifact(120),
          artifact(150),
          artifact(160, "decision_request"),
          artifact(250),
        ],
        proposals: [proposal(130, "Pending"), proposal(210, "Approved"), proposal(220, "Pending")],
        waits: [wait(140, "Met"), wait(230, "Waiting"), wait(240, "Waiting")],
      },
      null,
    );

    expect(entries.map((entry) => [entry.number, entry.work])).toEqual([
      [1, { artifacts: 2, decisions: 1, waits: 0 }],
      [2, { artifacts: 1, decisions: 1, waits: 2 }],
    ]);
  });

  it("marks the chapter the live reply opens, and only while nothing of it is saved", () => {
    const chapters = [chapter("achp_1", "atrn_1", 100, 0), chapter("achp_2", "atrn_2", 200, null)];

    const entries = chapterEntries(chapters, { artifacts: [], proposals: [], waits: [] }, "atrn_2");

    expect(entries.map((entry) => entry.live)).toEqual([false, true]);
  });
});
