import {
  realtimeClient,
  type RealtimePresenceActivity,
  type RealtimePresenceMember,
  type RealtimePresenceState,
} from "@trenova/shared/services/realtime";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useEffect } from "react";
import { create } from "zustand";

/** Everyone working Triage shares one scope; each member says which message they have open. */
export const TRIAGE_PRESENCE_SCOPE = "inbox:messages";
/** Joins the scope (POST, with `{connectionId, messageId, activity}`) and leaves it (DELETE). */
export const TRIAGE_PRESENCE_PATH = "/inbox/presence/";
/** How long a person stays on a message before the others are told; passing through says nothing. */
export const PRESENCE_SETTLE_MS = 300;

export type TriagePresenceActivity = RealtimePresenceActivity;

/** Someone else with a message open: who, and whether they are reading it or writing back. */
export type TriagePresenceMember = {
  userId: string;
  name: string;
  activity: TriagePresenceActivity;
};

type Announcement = { token: number; messageId: string; activity: TriagePresenceActivity };

type PresenceStore = {
  /** Other people by the message they have open, one entry per person. */
  byMessage: Readonly<Record<string, readonly TriagePresenceMember[]>>;
  /** What this tab has said it is doing, newest last. */
  announcements: readonly Announcement[];
};

const IDLE: RealtimePresenceState = { messageId: null, activity: "viewing" };
const NOBODY: readonly TriagePresenceMember[] = [];

const usePresenceStore = create<PresenceStore>(() => ({ byMessage: {}, announcements: [] }));

let nextToken = 0;
let publishTimer: ReturnType<typeof setTimeout> | null = null;

/**
 * Tells the scope what this tab is doing once it has stayed put for
 * PRESENCE_SETTLE_MS: stepping through a folder announces only where the
 * person stops, so moving between messages costs no request per step.
 */
function schedulePublish(): void {
  if (publishTimer !== null) {
    clearTimeout(publishTimer);
  }
  publishTimer = setTimeout(() => {
    publishTimer = null;
    realtimeClient.setScopeState(
      TRIAGE_PRESENCE_SCOPE,
      effectivePresence(usePresenceStore.getState().announcements),
    );
  }, PRESENCE_SETTLE_MS);
}

/** Writing outranks reading: the composer open beside a reader says "writing". */
export function effectivePresence(announcements: readonly Announcement[]): RealtimePresenceState {
  let latest: Announcement | undefined;
  for (const entry of announcements) {
    if (entry.activity === "writing" || latest?.activity !== "writing") {
      latest = entry;
    }
  }

  return latest === undefined ? IDLE : { messageId: latest.messageId, activity: latest.activity };
}

function sameMembers(
  a: readonly TriagePresenceMember[] | undefined,
  b: readonly TriagePresenceMember[],
): boolean {
  return (
    a !== undefined &&
    a.length === b.length &&
    a.every(
      (member, index) =>
        member.userId === b[index].userId &&
        member.name === b[index].name &&
        member.activity === b[index].activity,
    )
  );
}

/**
 * Members of the scope grouped by the message they have open, leaving out
 * the signed-in person (in every tab). A person with two tabs on one message
 * counts once, writing if either tab is. An unchanged message keeps its
 * array, so a row reading it does not re-render.
 */
export function groupPresence(
  members: readonly RealtimePresenceMember[],
  meId: string | null,
  previous: Readonly<Record<string, readonly TriagePresenceMember[]>> = {},
): Record<string, readonly TriagePresenceMember[]> {
  const grouped = new Map<string, Map<string, TriagePresenceMember>>();
  for (const member of members) {
    if (member.userId === meId || !member.messageId) {
      continue;
    }
    let people = grouped.get(member.messageId);
    if (people === undefined) {
      people = new Map();
      grouped.set(member.messageId, people);
    }
    const activity = member.activity ?? "viewing";
    const known = people.get(member.userId);
    if (known === undefined || (known.activity === "viewing" && activity === "writing")) {
      people.set(member.userId, { userId: member.userId, name: member.name, activity });
    }
  }

  const next: Record<string, readonly TriagePresenceMember[]> = {};
  for (const [messageId, people] of grouped) {
    const list = [...people.values()].sort((a, b) => a.userId.localeCompare(b.userId));
    next[messageId] = sameMembers(previous[messageId], list) ? previous[messageId] : list;
  }

  return next;
}

/**
 * Joins Triage's presence scope for as long as the page is mounted and keeps
 * who-has-what-open current. Mounted once, by the page.
 */
export function useTriagePresenceScope(): void {
  const meId = useAuthStore((state) => state.user?.id ?? null);

  useEffect(() => {
    if (meId === null) {
      return;
    }
    const unsubscribe = realtimeClient.subscribePresence(TRIAGE_PRESENCE_SCOPE, (members) => {
      usePresenceStore.setState((state) => ({
        byMessage: groupPresence(members, meId, state.byMessage),
      }));
    });
    const leave = realtimeClient.joinScope(
      TRIAGE_PRESENCE_SCOPE,
      TRIAGE_PRESENCE_PATH,
      effectivePresence(usePresenceStore.getState().announcements),
    );

    return () => {
      unsubscribe();
      leave();
      usePresenceStore.setState({ byMessage: {} });
    };
  }, [meId]);
}

/** Who else has each message open, by message id. */
export function useTriagePresence(): Readonly<Record<string, readonly TriagePresenceMember[]>> {
  return usePresenceStore((state) => state.byMessage);
}

/** Who else has one message open; the same empty array when nobody does. */
export function useMessagePresence(messageId: string): readonly TriagePresenceMember[] {
  return usePresenceStore((state) => state.byMessage[messageId] ?? NOBODY);
}

/**
 * Tells the others what this tab is doing with a message while the calling
 * component is mounted: the reader says "viewing", the composer "writing".
 * The latest wins, writing over viewing, and leaving the page goes back to
 * nothing open.
 */
export function useAnnouncePresence(
  messageId: string | null,
  activity: TriagePresenceActivity,
): void {
  useEffect(() => {
    if (messageId === null) {
      return;
    }
    nextToken += 1;
    const token = nextToken;

    usePresenceStore.setState((state) => ({
      announcements: [...state.announcements, { token, messageId, activity }],
    }));
    schedulePublish();

    return () => {
      usePresenceStore.setState((state) => ({
        announcements: state.announcements.filter((entry) => entry.token !== token),
      }));
      schedulePublish();
    };
  }, [messageId, activity]);
}
