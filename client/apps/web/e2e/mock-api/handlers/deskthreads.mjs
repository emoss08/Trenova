// The Desk rail: the person's conversations a page at a time, and how many
// each shelf holds. Both follow conversationrepository: pinned first, then
// newest by when the conversation was last touched (its last message, or its
// start), and conversations bound to a page (Ask, Import, Formula) left out.
import { deskThread } from "../fixtures/desk-cards.mjs";
import { deskCaseThreads } from "./deskcase.mjs";
import { formulaThreads } from "./formula-nova.mjs";

const UNLISTED_ORIGINS = new Set(["Ask", "Import", "Formula"]);
const DEFAULT_LIMIT = 50;
const MAX_LIMIT = 100;

function touchedAt(thread) {
  return Math.max(thread.lastMessageAt ?? 0, thread.createdAt ?? 0);
}

/** Every conversation the mock serves, newest first with pinned ones on top. */
function threads(state) {
  const cards = deskThread(state.anchor, state.scenario.deskSlug ?? "tractor-t-101").thread;
  const triage = [...(state.triage?.assistantThreads?.values() ?? [])];
  return [cards, ...deskCaseThreads(state), ...triage, ...formulaThreads(state)]
    .filter((thread) => thread.id !== undefined && !UNLISTED_ORIGINS.has(thread.origin))
    .sort(
      (a, b) =>
        Number(b.pinned) - Number(a.pinned) ||
        touchedAt(b) - touchedAt(a) ||
        (a.id < b.id ? 1 : a.id > b.id ? -1 : 0),
    );
}

function encodeCursor(thread) {
  return Buffer.from(JSON.stringify([thread.pinned, touchedAt(thread), thread.id])).toString(
    "base64url",
  );
}

function decodeCursor(cursor) {
  try {
    const [pinned, touched, id] = JSON.parse(Buffer.from(cursor, "base64url").toString("utf8"));
    return { pinned, touched, id };
  } catch {
    return null;
  }
}

/** Whether a row sorts after the cursor: (pinned, touchedAt, id) < the cursor's, in that order. */
function after(thread, cursor) {
  const key = [Number(thread.pinned), touchedAt(thread), thread.id];
  const at = [Number(cursor.pinned), cursor.touched, cursor.id];
  for (let i = 0; i < key.length; i += 1) {
    if (key[i] !== at[i]) return key[i] < at[i];
  }
  return false;
}

function listThreads(state, _path, _body, query) {
  const limit = Math.min(Number(query?.get("limit")) || DEFAULT_LIMIT, MAX_LIMIT);
  const pinned = query?.get("pinned");
  const from = Number(query?.get("touchedFrom") ?? 0);
  const before = Number(query?.get("touchedBefore") ?? 0);
  const cursor = query?.get("cursor") ? decodeCursor(query.get("cursor")) : null;
  if (query?.get("cursor") && !cursor) {
    return {
      __status: 400,
      body: {
        type: "validation-error",
        title: "Validation Failed",
        status: 400,
        detail: "The cursor is not one this list gave out",
      },
    };
  }
  const rows = threads(state).filter(
    (thread) =>
      (pinned === null || pinned === undefined || String(thread.pinned) === pinned) &&
      (from <= 0 || touchedAt(thread) >= from) &&
      (before <= 0 || touchedAt(thread) < before) &&
      (!cursor || after(thread, cursor)),
  );
  const items = rows.slice(0, limit);
  return { items, nextCursor: rows.length > limit ? encodeCursor(items.at(-1)) : "" };
}

/** One count per band between the edges, newest first, the band below the last edge at the end. */
function countShelves(state, _path, _body, query) {
  const edges = (query?.getAll("edge") ?? []).map(Number);
  const counts = { pinned: 0, bands: edges.map(() => 0).concat(0) };
  for (const thread of threads(state)) {
    if (thread.pinned) {
      counts.pinned += 1;
      continue;
    }
    const band = edges.findIndex((edge) => touchedAt(thread) >= edge);
    counts.bands[band === -1 ? edges.length : band] += 1;
  }
  return counts;
}

export const DESK_THREAD_ROUTES = [
  ["GET", /^\/api\/v1\/assistant\/threads\/shelves\/?$/, countShelves],
  ["GET", /^\/api\/v1\/assistant\/threads\/?$/, listThreads],
];
