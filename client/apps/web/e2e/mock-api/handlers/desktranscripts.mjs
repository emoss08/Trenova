// The administrator's Desk conversations page: everyone's conversations filtered by the
// timeframe, person, agent and origin beside the table, one opened read-only, the bounded count
// an export would hold, the exports and their downloads, and one conversation's Markdown file.
// The `deskTranscripts` scenario is full | empty | hold (every list left open, for the skeleton).
import { createDeskTranscriptState } from "../fixtures/desk-transcripts.mjs";
import { connection } from "./formula.mjs";

const DAY = 86400;
const MAX_EXPORT = 5000;
const MAX_SELECTED = 200;
const PRESET_DAYS = { Last7Days: 7, Last30Days: 30, Last90Days: 90, Last12Months: 365 };

function transcripts(state) {
  state.deskTranscripts ??= createDeskTranscriptState(state.anchor);
  return state.deskTranscripts;
}

function validation(field, message) {
  const error = new Error(message);
  error.extensions = { code: "VALIDATION_ERROR", field };
  throw error;
}

function startOfDay(seconds) {
  return seconds - (seconds % DAY);
}

function windowOf(state, timeframe) {
  if (!timeframe || timeframe.preset === "AllTime") return {};
  if (timeframe.preset === "Custom") {
    const from = Date.parse(`${timeframe.fromDate}T00:00:00Z`) / 1000;
    const to = Date.parse(`${timeframe.toDate}T00:00:00Z`) / 1000 + DAY;
    return { from, before: to };
  }
  const days = PRESET_DAYS[timeframe.preset] ?? 30;
  return { from: startOfDay(state.anchor) - (days - 1) * DAY };
}

function matching(state, filter = {}, query = "") {
  const { from, before } = windowOf(state, filter?.timeframe);
  const text = query.trim().toLowerCase();
  if (state.scenario.deskTranscripts === "empty") return [];
  return transcripts(state).conversations.filter(
    (row) =>
      (!filter?.userId || row.ownerId === filter.userId) &&
      (!filter?.agentId || row.agentId === filter.agentId) &&
      (!filter?.origins?.length || filter.origins.includes(row.origin)) &&
      (from === undefined || row.lastMessageAt >= from) &&
      (before === undefined || row.lastMessageAt < before) &&
      (!text || row.title.toLowerCase().includes(text)),
  );
}

function rowOf(row) {
  return {
    __typename: "DeskConversation",
    id: row.id,
    ownerName: row.ownerName,
    agentName: row.agentName,
    title: row.title,
    origin: row.origin,
    archived: row.archived,
    messageCount: row.messageCount,
    startedAt: row.startedAt,
    lastMessageAt: row.lastMessageAt,
  };
}

function requestExport(state, { input }) {
  const ids = input.conversationIds ?? [];
  if (input.filter && ids.length > 0) {
    validation(
      "conversationIds",
      "Export either a timeframe or the conversations you selected, not both",
    );
  }
  if (!input.filter && ids.length === 0) {
    validation("filter", "Choose a timeframe or select conversations to export");
  }
  if (ids.length > MAX_SELECTED) {
    validation(
      "conversationIds",
      `Select at most ${MAX_SELECTED} conversations to download at once, or export a timeframe instead`,
    );
  }
  const store = transcripts(state);
  const rows = input.filter
    ? matching(state, input.filter)
    : ids.map((id) => store.conversations.find((row) => row.id === id)).filter(Boolean);
  if (!input.filter && rows.length !== ids.length) {
    const error = new Error(
      `${ids.length - rows.length} of the selected conversations could not be found in your organization`,
    );
    error.extensions = { code: "NOT_FOUND" };
    throw error;
  }
  if (rows.length === 0) {
    validation(
      "filter",
      "No conversation matches. Choose a longer timeframe, or another person or agent.",
    );
  }
  const owners = [...new Set(rows.map((row) => row.ownerName))].sort();
  const person = store.people.find((candidate) => candidate.id === input.filter?.userId);
  const agent = store.agents.find((candidate) => candidate.id === input.filter?.agentId);
  const created = {
    __typename: "DeskTranscriptExport",
    id: `dtex_${store.exports.length + 1}`,
    requestedByUserId: "usr_admin",
    requestedByName: "System Administrator",
    kind: input.filter ? "Filter" : "Selection",
    coverage: {
      kind: input.filter ? "Filter" : "Selection",
      timeframe: input.filter
        ? {
            preset: input.filter.timeframe?.preset ?? "AllTime",
            fromDate: input.filter.timeframe?.fromDate ?? null,
            toDate: input.filter.timeframe?.toDate ?? null,
          }
        : null,
      personName: person?.name ?? null,
      agentName: agent?.name ?? null,
      origins: input.filter?.origins ?? [],
      selectedCount: input.filter ? 0 : rows.length,
      ownerNames: owners.slice(0, 20),
      ownerCount: owners.length,
    },
    status: "Running",
    conversationCount: rows.length,
    byteSize: 0,
    artifactExpiresAt: null,
    errorMessage: null,
    startedAt: state.now(),
    completedAt: null,
    downloadable: false,
    createdAt: state.now(),
  };
  store.exports.unshift(created);
  setTimeout(() => {
    Object.assign(created, {
      status: "Succeeded",
      byteSize: rows.length * 14_000,
      completedAt: state.now(),
      artifactExpiresAt: state.now() + 7 * DAY,
      downloadable: true,
    });
  }, 4000);
  return { requestDeskTranscriptExport: created };
}

export const DESK_TRANSCRIPT_HANDLERS = {
  DeskConversationTable: (state, v) => {
    if (state.scenario.deskTranscripts === "hold") return { __hold: true };
    const rows = matching(state, v.filter, v.input?.query ?? "");
    return {
      deskConversations: { ...connection(rows, v, rowOf), totalCountExact: true },
    };
  },
  DeskTranscriptExportTable: (state, v) => {
    if (state.scenario.deskTranscripts === "hold") return { __hold: true };
    const rows = state.scenario.deskTranscripts === "empty" ? [] : transcripts(state).exports;
    return {
      deskTranscriptExports: { ...connection(rows, v, (row) => row), totalCountExact: true },
    };
  },
  DeskTranscriptExportPreview: (state, { filter }) => {
    const count = Math.min(matching(state, filter).length, MAX_EXPORT + 1);
    return {
      deskTranscriptExportPreview: {
        count,
        exact: count <= MAX_EXPORT,
        limit: MAX_EXPORT,
        withinLimit: count <= MAX_EXPORT,
      },
    };
  },
  OpenDeskConversation: (state, { id }) => {
    const row = transcripts(state).conversations.find((candidate) => candidate.id === id);
    if (!row) {
      const error = new Error("Conversation not found");
      error.extensions = { code: "NOT_FOUND" };
      throw error;
    }
    return { openDeskConversation: { conversation: rowOf(row), transcript: row.transcript } };
  },
  RequestDeskTranscriptExport: requestExport,
  DeskTranscriptExportDownload: (state, { id }) => ({
    deskTranscriptExportDownload: {
      url: `http://localhost:8080/__mock/files/${id}`,
      fileName: "desk-conversations.zip",
      expiresAt: state.now() + 60,
    },
  }),
};

function markdownOf(state, path) {
  const id = decodeURIComponent(path.split("/")[4] ?? "");
  const row = transcripts(state).conversations.find((candidate) => candidate.id === id);
  if (!row) {
    return {
      __status: 404,
      body: { title: "Not Found", status: 404, detail: "Conversation not found" },
    };
  }
  const lines = [
    `# ${row.title || "Conversation"}`,
    "",
    `- **Person:** ${row.ownerName}`,
    `- **Agent:** ${row.agentName}`,
    "",
    ...row.transcript.messages.map((message) => `## ${message.role}\n\n${message.content}\n`),
  ];
  return { __raw: lines.join("\n"), contentType: "text/markdown; charset=utf-8" };
}

export const DESK_TRANSCRIPT_ROUTES = [
  ["GET", /^\/api\/v1\/desk-transcripts\/[^/]+\/download\/?$/, markdownOf],
];
