import { create } from "zustand";

/** A business record a question at the Desk is about, as the record-link registry names it. */
export type DeskSubject = {
  kind: string;
  id: string;
  label: string;
};

/**
 * Nova's margin beside the work: the conversation it holds, or none yet while
 * the person is about to ask, and the record the question goes with.
 */
export type DeskMargin = {
  threadId: string | null;
  subject: DeskSubject | null;
};

/** A case opened from the Desk this session, newest first, for the rail's Open work. */
export type DeskOpenWork = DeskSubject;

const OPEN_WORK_LIMIT = 4;

interface DeskMarginState {
  margin: DeskMargin | null;
  /** What the page in front of the person is about, which ⌘J and Ask take with them. */
  pageSubject: DeskSubject | null;
  /**
   * The cases opened this session. Held in memory only: a record's name is
   * never written to browser storage.
   */
  openWork: DeskOpenWork[];

  openMargin: (subject?: DeskSubject | null) => void;
  showThread: (threadId: string, subject?: DeskSubject | null) => void;
  closeMargin: () => void;
  toggleMargin: () => void;
  setPageSubject: (subject: DeskSubject | null) => void;
  noteOpenWork: (work: DeskOpenWork) => void;
  clearSession: () => void;
}

function sameSubject(a: DeskSubject | null, b: DeskSubject | null): boolean {
  return a?.kind === b?.kind && a?.id === b?.id;
}

/** The open work with this case first, each case once, the oldest dropped past the limit. */
export function withOpenWork(current: readonly DeskOpenWork[], work: DeskOpenWork): DeskOpenWork[] {
  const rest = current.filter((entry) => !(entry.kind === work.kind && entry.id === work.id));
  return [work, ...rest].slice(0, OPEN_WORK_LIMIT);
}

export const useDeskMarginStore = create<DeskMarginState>()((set, get) => ({
  margin: null,
  pageSubject: null,
  openWork: [],

  openMargin: (subject) => {
    const about = subject === undefined ? get().pageSubject : subject;
    const current = get().margin;
    if (current && current.threadId !== null && sameSubject(current.subject, about)) {
      return;
    }
    set({ margin: { threadId: null, subject: about } });
  },
  showThread: (threadId, subject) =>
    set((state) => ({
      margin: {
        threadId,
        subject: subject === undefined ? (state.margin?.subject ?? null) : subject,
      },
    })),
  closeMargin: () => set({ margin: null }),
  toggleMargin: () => {
    if (get().margin) {
      set({ margin: null });
      return;
    }
    get().openMargin();
  },
  setPageSubject: (subject) => {
    if (sameSubject(get().pageSubject, subject) && get().pageSubject?.label === subject?.label) {
      return;
    }
    set({ pageSubject: subject });
  },
  noteOpenWork: (work) => set((state) => ({ openWork: withOpenWork(state.openWork, work) })),
  clearSession: () => set({ margin: null, pageSubject: null, openWork: [] }),
}));
