// The realtime stream (`GET /api/v1/realtime/stream/`), framed the way
// docs/engineering/realtime.md "A stream" describes it: `ready` first, a
// `presence.snapshot` for `?presence=users`, `resource.invalidation` and
// `presence` events as the mock state changes, and a `heartbeat` on the interval
// `ready` announced so the browser's watchdog never trips.
import { ORG, USER } from "./fixtures/session.mjs";

const HEARTBEAT_MS = 15000;

const clients = new Map();
const disconnectListeners = new Set();
let sequence = 0;
let connections = 0;

function frame(client, { id, event, data }) {
  if (client.closed) return;
  const head = id ? `id: ${id}\n` : "";
  client.res.write(`${head}event: ${event}\ndata: ${JSON.stringify(data ?? {})}\n\n`);
}

/** Opens one browser's stream and keeps it until the browser goes away. */
export function openStream(req, res, { presenceUsers = false } = {}) {
  res.writeHead(200, {
    "Content-Type": "text/event-stream",
    "Cache-Control": "no-cache",
    Connection: "keep-alive",
  });
  connections += 1;
  const connectionId = `conn_mock_${connections}`;
  const client = { res, connectionId, closed: false };
  clients.set(connectionId, client);
  frame(client, {
    event: "ready",
    data: { connectionId, heartbeatIntervalMs: HEARTBEAT_MS, resumed: false, cursor: String(sequence) },
  });
  if (presenceUsers) {
    frame(client, {
      event: "presence.snapshot",
      data: { scope: "users", members: [{ userId: USER.id, connectionId, name: USER.name }] },
    });
  }
  const heartbeat = setInterval(() => frame(client, { event: "heartbeat", data: {} }), HEARTBEAT_MS);
  req.on("close", () => {
    client.closed = true;
    clearInterval(heartbeat);
    clients.delete(connectionId);
    for (const listener of disconnectListeners) listener(connectionId);
  });
  return connectionId;
}

/** Sends an event to every open stream. Presence and typing are never resumable, so they carry no id. */
export function publish(event, data, { resumable = true } = {}) {
  const id = resumable ? String(++sequence) : null;
  for (const client of clients.values()) frame(client, { id, event, data });
}

/** A record changed: the same envelope `realtimeservice` publishes (`type` is resource.action). */
export function publishInvalidation({ resource, action = "updated", recordId = null, fields = undefined }) {
  publish("resource.invalidation", {
    eventId: `evt_mock_${sequence + 1}`,
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    type: `${resource}.${action}`,
    resource,
    action,
    ...(recordId ? { recordId, entityId: recordId } : {}),
    ...(fields ? { fields } : {}),
    actorType: "system",
    occurredAt: new Date().toISOString(),
  });
}

export function onDisconnect(listener) {
  disconnectListeners.add(listener);
  return () => disconnectListeners.delete(listener);
}

export function isConnected(connectionId) {
  return clients.has(connectionId);
}
