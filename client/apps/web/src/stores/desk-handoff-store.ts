import type { AssistantEntityRef } from "@/types/assistant";
import { create } from "zustand";

/** What a question asked at the Desk's front page carries besides its words. */
export type DeskHandoff = {
  threadId: string;
  files: File[];
  mentions: AssistantEntityRef[];
  /** The model chosen before the conversation existed; empty for Auto. */
  providerId: string;
};

interface DeskHandoffState {
  handoff: DeskHandoff | null;
  setHandoff: (handoff: DeskHandoff | null) => void;
}

/**
 * Files and named records from the front page, on their way to the
 * conversation the question started. Kept in memory only: a file picked
 * from disk cannot outlive the page it was picked on.
 */
export const useDeskHandoffStore = create<DeskHandoffState>()((set) => ({
  handoff: null,
  setHandoff: (handoff) => set({ handoff }),
}));
