// A stand-in for the Trenova API, enough to sign in and drive the Shipments
// board without Postgres, Redis or the Go service. See README.md.
import http from "node:http";
import { handleGraphQL } from "./handlers/graphql.mjs";
import { handleRest } from "./handlers/rest.mjs";
import { openStream, publish } from "./realtime.mjs";
import { createState, SCENARIO_DEFAULTS } from "./state.mjs";

const PORT = Number(process.env.MOCK_API_PORT ?? 8080);
const LOG_UNKNOWN = process.env.MOCK_API_LOG !== "0";
const ALLOWED_ORIGINS = new Set(
  (process.env.MOCK_API_ORIGINS ?? "http://localhost:5173,http://127.0.0.1:5173")
    .split(",")
    .map((origin) => origin.trim())
    .filter(Boolean),
);

let state = createState(SCENARIO_DEFAULTS);

function send(res, status, payload, headers = {}) {
  res.writeHead(status, { "Content-Type": "application/json", ...headers });
  res.end(payload === undefined ? "" : JSON.stringify(payload));
}

/** A multipart form as the document upload route reads it: its text fields and the file's name, type and size. */
function parseMultipart(buffer, contentType) {
  const boundary = /boundary=(?:"([^"]+)"|([^;]+))/i.exec(contentType);
  if (!boundary) return undefined;
  const delimiter = Buffer.from(`--${boundary[1] ?? boundary[2]}`);
  const form = { fields: {}, file: null };
  let start = buffer.indexOf(delimiter);
  while (start !== -1) {
    const next = buffer.indexOf(delimiter, start + delimiter.length);
    if (next === -1) break;
    const part = buffer.subarray(start + delimiter.length + 2, next - 2);
    const split = part.indexOf("\r\n\r\n");
    if (split !== -1) {
      const head = part.subarray(0, split).toString("utf8");
      const content = part.subarray(split + 4);
      const name = /name="([^"]*)"/i.exec(head)?.[1];
      const fileName = /filename="([^"]*)"/i.exec(head)?.[1];
      if (fileName !== undefined) {
        form.file = { name: fileName, type: /content-type:\s*([^\r\n]+)/i.exec(head)?.[1]?.trim() ?? "", size: content.length };
      } else if (name) {
        form.fields[name] = content.toString("utf8");
      }
    }
    start = next;
  }
  return form;
}

function readBody(req) {
  return new Promise((resolve) => {
    const chunks = [];
    req.on("data", (chunk) => chunks.push(chunk));
    req.on("end", () => {
      const buffer = Buffer.concat(chunks);
      if (buffer.length === 0) return resolve(undefined);
      const contentType = req.headers["content-type"] ?? "";
      if (/^multipart\/form-data/i.test(contentType)) return resolve(parseMultipart(buffer, contentType));
      try {
        resolve(JSON.parse(buffer.toString("utf8")));
      } catch {
        resolve(undefined);
      }
    });
  });
}

const server = http.createServer((req, res) => {
  handle(req, res).catch((error) => {
    console.error(`[mock-api] ${req.method} ${req.url} failed:`, error);
    if (!res.headersSent) send(res, 500, { message: String(error?.message ?? error) });
  });
});

async function handle(req, res) {
  const origin = req.headers.origin;
  if (origin && ALLOWED_ORIGINS.has(origin)) {
    res.setHeader("Access-Control-Allow-Origin", origin);
    res.setHeader("Access-Control-Allow-Credentials", "true");
    res.setHeader("Vary", "Origin");
  }
  res.setHeader(
    "Access-Control-Allow-Headers",
    req.headers["access-control-request-headers"] ?? "Content-Type, X-CSRF-Token",
  );
  res.setHeader("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS");
  if (req.method === "OPTIONS") {
    res.writeHead(204);
    return res.end();
  }

  const url = new URL(req.url, `http://localhost:${PORT}`);
  const body = await readBody(req);

  if (url.pathname === "/__mock/scenario") {
    if (req.method === "POST") {
      state.dispose();
      state = createState({ ...state.scenario, ...(body ?? {}) });
      publish("reset", {});
    }
    return send(res, 200, { ...state.scenario, anchor: state.anchor, triageAnchor: state.triageAnchor });
  }

  if (url.pathname.startsWith("/graphql")) {
    const result = await handleGraphQL(state, body ?? {});
    if (!result.known && LOG_UNKNOWN) {
      console.log(`[mock-api] unhandled GraphQL operation ${body?.operationName}`);
    }
    return send(res, 200, result.payload);
  }

  const result = handleRest(state, { method: req.method, path: url.pathname, query: url.searchParams, body });
  if (!result.known && LOG_UNKNOWN) {
    console.log(`[mock-api] unhandled ${req.method} ${url.pathname}`);
  }
  if (result.sse) {
    res.writeHead(200, { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" });
    let closed = false;
    req.on("close", () => (closed = true));
    let at = 0;
    for (const [index, frame] of result.sse.entries()) {
      at += frame.delay ?? 0;
      setTimeout(() => {
        if (closed) return;
        res.write(`id: ${index + 1}\nevent: ${frame.event}\ndata: ${JSON.stringify(frame.data ?? {})}\n\n`);
        if (index === result.sse.length - 1) res.end();
      }, at);
    }
    return undefined;
  }
  if (result.stream) {
    openStream(req, res, { presenceUsers: url.searchParams.get("presence") === "users" });
    return undefined;
  }
  if (result.raw !== undefined) {
    res.writeHead(200, { "Content-Type": result.contentType ?? "text/plain", "Cache-Control": "max-age=3600" });
    return res.end(result.raw);
  }
  return send(res, result.status ?? 200, result.payload, result.headers);
}

server.listen(PORT, () => {
  console.log(`[mock-api] listening on http://localhost:${PORT}`);
  console.log(`[mock-api] scenario ${JSON.stringify(state.scenario)}`);
});
