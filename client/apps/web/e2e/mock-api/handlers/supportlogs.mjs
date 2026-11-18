// "Send logs to Trenova support" on the support access page (supportLogRequests and
// requestSupportLogs in services/tms/internal/api/graphql/schema/support_access.graphqls).
// There is no operator here, so a request the page makes moves on by itself the way the instance
// reports it: Collecting after 4 seconds, Sent after 12, each announced on the realtime stream as
// support_log_request.updated, which is what makes the page read the list again.
import { publishInvalidation } from "../realtime.mjs";
import { isManaged } from "./managedplan.mjs";
import { TriageError } from "./triage-model.mjs";

const HOUR = 3600;
const DAY = 24 * HOUR;
const DAILY_LIMIT = 5;
const COLLECTING_AFTER_MS = 4000;
const SENT_AFTER_MS = 12000;
const RESOURCE = "support_log_request";

const SELF_HOSTED = { managed: false, inFlight: false, remainingToday: DAILY_LIMIT, requests: [] };

function seed(state) {
  const now = state.anchor;
  const scenario = state.scenario.supportLogs;
  if (scenario === "empty") {
    return [];
  }
  if (scenario === "inFlight") {
    return [
      {
        id: "slr_01JMOCKCOLLECTING00000000000",
        period: "Day",
        status: "Collecting",
        requestedByName: "Admin User",
        requestedAt: now - 6 * 60,
        finishedAt: null,
        failureMessage: null,
      },
    ];
  }
  if (scenario === "limit") {
    return Array.from({ length: DAILY_LIMIT }, (_, index) => ({
      id: `slr_01JMOCKLIMIT${String(index).padStart(15, "0")}`,
      period: "Hour",
      status: "Sent",
      requestedByName: "Admin User",
      requestedAt: now - (index + 1) * 2 * HOUR,
      finishedAt: now - (index + 1) * 2 * HOUR + 3 * 60,
      failureMessage: null,
    }));
  }
  return [
    {
      id: "slr_01JMOCKSENT0000000000000000",
      period: "Week",
      status: "Sent",
      requestedByName: "Admin User",
      requestedAt: now - 3 * DAY,
      finishedAt: now - 3 * DAY + 9 * 60,
      failureMessage: null,
    },
    {
      id: "slr_01JMOCKFAILED00000000000000",
      period: "Hour",
      status: "Failed",
      requestedByName: "Dana Whitfield",
      requestedAt: now - 5 * DAY,
      finishedAt: now - 5 * DAY + 30 * 60,
      failureMessage:
        "The instance did not start collecting the logs within 30 minutes. Try again, and contact Trenova support if it happens again.",
    },
  ];
}

function requestsOf(state) {
  state.supportLogs ??= seed(state);
  return state.supportLogs;
}

function overview(state) {
  const requests = requestsOf(state);
  const since = Math.floor(Date.now() / 1000) - DAY;
  const recent = requests.filter((request) => request.requestedAt > since).length;
  return {
    managed: true,
    inFlight: requests.some(
      (request) => request.status === "Requested" || request.status === "Collecting",
    ),
    remainingToday: Math.max(DAILY_LIMIT - recent, 0),
    requests: requests.slice(0, 10),
  };
}

function refuse(message) {
  throw new TriageError(message, { code: "INVALID", field: "period" });
}

function moveOn(request) {
  setTimeout(() => {
    request.status = "Collecting";
    publishInvalidation({ resource: RESOURCE, recordId: request.id });
  }, COLLECTING_AFTER_MS).unref?.();
  setTimeout(() => {
    request.status = "Sent";
    request.finishedAt = Math.floor(Date.now() / 1000);
    publishInvalidation({ resource: RESOURCE, recordId: request.id });
  }, SENT_AFTER_MS).unref?.();
}

export const SUPPORT_LOG_HANDLERS = {
  SupportLogRequests: (state) => ({
    supportLogRequests: isManaged(state) ? overview(state) : SELF_HOSTED,
  }),
  RequestSupportLogs: (state, variables) => {
    if (!isManaged(state)) {
      refuse("Sending logs to Trenova support is available only on a Managed Trenova instance");
    }
    const current = overview(state);
    if (current.inFlight) {
      refuse("Logs are already being sent to Trenova support. Wait until that request has finished");
    }
    if (current.remainingToday === 0) {
      refuse(
        "This instance has sent logs to Trenova support 5 times in the last 24 hours. Try again later",
      );
    }
    const now = Math.floor(Date.now() / 1000);
    const request = {
      id: `slr_01JMOCKNEW${String(now).padStart(17, "0")}`,
      period: variables.input.period,
      status: "Requested",
      requestedByName: "Admin User",
      requestedAt: now,
      finishedAt: null,
      failureMessage: null,
    };
    requestsOf(state).unshift(request);
    publishInvalidation({ resource: RESOURCE, action: "created", recordId: request.id });
    moveOn(request);
    return { requestSupportLogs: { id: request.id, status: request.status } };
  },
};
