import type { AgentWait, AssistantArtifact, AssistantChapter, AssistantProposal } from "@/types/assistant";

/** What one chapter holds: what it made, what waits on the person, and what the agent waits on. */
export type ChapterWork = {
  artifacts: number;
  decisions: number;
  waits: number;
};

/** A chapter as the rail lists it: its place in the conversation and what it holds. */
export type ChapterEntry = {
  chapter: AssistantChapter;
  /** Its place, from 1. */
  number: number;
  work: ChapterWork;
  /** It opens with the reply still being written. */
  live: boolean;
};

/** How many chapters a conversation has before any of them is drawn. */
export const MIN_SHOWN_CHAPTERS = 2;

type Dated = { createdAt: number };

/**
 * The chapters a reader sees: those with a message saved, the one the reply
 * being written opens, and the newest one, whose reply may have just been
 * saved before the chapters are read again. An older chapter whose turn saved
 * nothing is the server's to remove, and is not shown meanwhile.
 */
export function shownChapters(
  chapters: readonly AssistantChapter[],
  liveTurnId: string | null,
): AssistantChapter[] {
  const newest = chapters.length - 1;
  return chapters.filter(
    (chapter, idx) =>
      chapter.startSequence !== null || chapter.turnId === liveTurnId || idx === newest,
  );
}

function bucketOf(starts: readonly number[], at: number): number {
  let bucket = -1;
  for (let idx = 0; idx < starts.length; idx++) {
    if (starts[idx] > at) {
      break;
    }
    bucket = idx;
  }
  return bucket;
}

function count<T extends Dated>(
  starts: readonly number[],
  items: readonly T[],
  keep: (item: T) => boolean,
): number[] {
  const totals = starts.map(() => 0);
  for (const item of items) {
    if (!keep(item) || item.createdAt <= 0) {
      continue;
    }
    const bucket = bucketOf(starts, item.createdAt);
    if (bucket >= 0) {
      totals[bucket] += 1;
    }
  }
  return totals;
}

/**
 * Each chapter with what was made, decided and waited on while it was the
 * conversation's chapter: everything dated from its turn starting until the
 * next chapter's did. Work from before the first chapter belongs to none.
 */
export function chapterEntries(
  chapters: readonly AssistantChapter[],
  work: {
    artifacts: readonly AssistantArtifact[];
    proposals: readonly AssistantProposal[];
    waits: readonly AgentWait[];
  },
  liveTurnId: string | null,
): ChapterEntry[] {
  const starts = chapters.map((chapter) => chapter.startedAt);
  const artifacts = count(starts, work.artifacts, (item) => item.kind !== "decision_request");
  const decisions = count(starts, work.proposals, (item) => item.status === "Pending");
  const waits = count(starts, work.waits, (item) => item.status === "Waiting");

  return chapters.map((chapter, idx) => ({
    chapter,
    number: idx + 1,
    work: { artifacts: artifacts[idx], decisions: decisions[idx], waits: waits[idx] },
    live: chapter.turnId === liveTurnId && chapter.startSequence === null,
  }));
}

