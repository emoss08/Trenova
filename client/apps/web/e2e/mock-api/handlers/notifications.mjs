// The notification panel's operations: one list per part of the panel, the counts on its
// tabs, and the writes that move a row between parts. A row's part is worked out from the
// row, as the server does, so a write shows up in the next read of every part.
import { buildNotifications } from "../fixtures/notifications.mjs";

const clock = () => Math.floor(Date.now() / 1000);

function notifications(state) {
  state.notifications ??= buildNotifications(clock());
  return state.notifications;
}

function lensOf(row, now) {
  if (row.dismissedAt !== null) return "done";
  if (row.surfacesAt > now) return "later";
  return row.reason === "team" ? "team" : "forYou";
}

function inLens(row, lens, now) {
  const own = lensOf(row, now);
  return lens === "open" ? own === "forYou" || own === "team" : own === lens;
}

const ORDER = {
  later: (a, b) => a.surfacesAt - b.surfacesAt,
  done: (a, b) => b.dismissedAt - a.dismissedAt,
};
const newestFirst = (a, b) => b.surfacesAt - a.surfacesAt;

function list(state, { input = {}, filter = {} }) {
  const now = clock();
  const lens = filter.lens ?? "open";
  const rows = notifications(state)
    .filter((row) => inLens(row, lens, now) && (!filter.unreadOnly || row.readAt === null))
    .sort(ORDER[lens] ?? newestFirst);
  const start = input.after ? Number(Buffer.from(input.after, "base64url").toString()) : 0;
  const first = input.first ?? 30;
  const page = rows.slice(start, start + first);
  const end = start + page.length;

  return {
    edges: page.map((node) => ({ node })),
    totalCount: rows.length,
    totalCountExact: true,
    pageInfo: {
      hasNextPage: end < rows.length,
      hasPreviousPage: start > 0,
      startCursor: null,
      endCursor: end < rows.length ? Buffer.from(String(end)).toString("base64url") : null,
    },
  };
}

function counts(state) {
  const now = clock();
  const out = { forYou: 0, forYouUnread: 0, team: 0, teamUnread: 0, later: 0 };
  for (const row of notifications(state)) {
    const lens = lensOf(row, now);
    if (lens === "done") continue;
    out[lens] += 1;
    if (lens !== "later" && row.readAt === null) out[`${lens}Unread`] += 1;
  }
  return out;
}

function change(state, ids, apply) {
  const now = clock();
  for (const row of notifications(state)) {
    if (ids.includes(row.id)) apply(row, now);
  }
  return true;
}

const finish = (row, now) => {
  row.dismissedAt = now;
  row.readAt ??= now;
};

export const NOTIFICATION_HANDLERS = {
  NotificationList: (state, v) => ({ notifications: list(state, v) }),
  NotificationCounts: (state) => ({ notificationCounts: counts(state) }),
  MarkNotificationsRead: (state, { ids }) => ({
    markNotificationsRead: change(state, ids, (row, now) => {
      row.readAt ??= now;
    }),
  }),
  MarkNotificationsUnread: (state, { ids }) => ({
    markNotificationsUnread: change(state, ids, (row) => {
      row.readAt = null;
    }),
  }),
  DismissNotifications: (state, { ids }) => ({
    dismissNotifications: change(state, ids, finish),
  }),
  DismissAllNotifications: (state, { lens }) => {
    const now = clock();
    for (const row of notifications(state)) {
      if (inLens(row, lens, now)) finish(row, now);
    }
    return { dismissAllNotifications: true };
  },
  RestoreNotifications: (state, { ids }) => ({
    restoreNotifications: change(state, ids, (row, now) => {
      row.dismissedAt = null;
      row.resolvedAt = null;
      row.resolution = "";
      row.surfacesAt = now;
    }),
  }),
  SnoozeNotifications: (state, { ids, until }) => ({
    snoozeNotifications: change(state, ids, (row) => {
      row.surfacesAt = until;
      row.readAt = null;
    }),
  }),
  WakeNotifications: (state, { ids }) => ({
    wakeNotifications: change(state, ids, (row, now) => {
      row.surfacesAt = now;
    }),
  }),
  AcknowledgeWatchtowerItem: (_state, { id }) => ({
    acknowledgeWatchtowerItem: { id, status: "Acknowledged", version: 2 },
  }),
  ApproveWorkerPto: (_state, { id }) => ({ approveWorkerPTO: { id, status: "Approved" } }),
};
