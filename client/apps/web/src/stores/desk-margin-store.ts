import { create } from "zustand";

/** A business record a question at the Desk is about, as the record-link registry names it. */
export type DeskSubject = {
  kind: string;
  id: string;
  label: string;
  /** The agent working the record's case: the only one a question about it goes to. */
  agentId?: string;
};

/**
 * Nova's margin beside the work: a conversation that already exists, and the
 * record it is about. A new conversation never starts here: it opens full width.
 */
export type DeskMargin = {
  threadId: string;
  subject: DeskSubject | null;
};

/** A case or a composed view opened at the Desk this session, newest first, for the rail's Open work. */
export type DeskOpenWork = DeskSubject & {
  /** Where it opens when it is not a record's case: a composed view's own page. */
  path?: string;
};

const OPEN_WORK_LIMIT = 4;

interface DeskMarginState {
  margin: DeskMargin | null;
  /** What the page in front of the person is about, which ⌘J and Ask take with them. */
  pageSubject: DeskSubject | null;
  /** What the page in front of the person is called, when only the page knows: a ledger line's title. */
  pageTitle: string | null;
  /**
   * The case in front of the person. While one is, the margin holds that
   * case's conversation and no other: a conversation about something else is
   * opened as its own page, never swapped in beside the case.
   */
  casePage: DeskSubject | null;
  /**
   * The cases opened this session. Held in memory only: a record's name is
   * never written to browser storage.
   */
  openWork: DeskOpenWork[];

  showThread: (threadId: string, subject?: DeskSubject | null) => void;
  closeMargin: () => void;
  setPageSubject: (subject: DeskSubject | null) => void;
  setPageTitle: (title: string | null) => void;
  setCasePage: (subject: DeskSubject | null) => void;
  noteOpenWork: (work: DeskOpenWork) => void;
  /** Takes a case or a view opened this session off Open work, by where it opens. */
  forgetOpenWork: (kind: string, id: string) => void;
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
  pageTitle: null,
  casePage: null,
  openWork: [],

  showThread: (threadId, subject) =>
    set((state) => {
      const about = subject === undefined ? (state.margin?.subject ?? null) : subject;
      if (state.casePage !== null && !sameSubject(about, state.casePage)) {
        return state;
      }
      return { margin: { threadId, subject: about } };
    }),
  closeMargin: () => set({ margin: null }),
  setPageSubject: (subject) => {
    if (sameSubject(get().pageSubject, subject) && get().pageSubject?.label === subject?.label) {
      return;
    }
    set({ pageSubject: subject });
  },
  setPageTitle: (title) => {
    if (get().pageTitle !== title) {
      set({ pageTitle: title });
    }
  },
  setCasePage: (subject) => {
    const { casePage, margin } = get();
    if (sameSubject(casePage, subject) && casePage?.agentId === subject?.agentId) {
      return;
    }
    // A conversation about something else folds when a case opens: beside a
    // case sits only the case's own.
    const held = subject !== null && margin !== null && !sameSubject(margin.subject, subject);
    set({ casePage: subject, margin: held ? null : margin });
  },
  noteOpenWork: (work) => set((state) => ({ openWork: withOpenWork(state.openWork, work) })),
  forgetOpenWork: (kind, id) =>
    set((state) => ({
      openWork: state.openWork.filter((entry) => !(entry.kind === kind && entry.id === id)),
    })),
  clearSession: () =>
    set({ margin: null, pageSubject: null, pageTitle: null, casePage: null, openWork: [] }),
}));
