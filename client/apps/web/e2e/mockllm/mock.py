"""A scripted OpenAI-compatible chat model for the Desk's end-to-end tests.

It answers /v1/chat/completions (streaming or not), fills structured-output
requests from their JSON schema, and follows a script of tool calls so a turn
exercises the real runtime: tool steps, artifacts, proposals and text.

The script is a list of rules. The first rule with a `match` phrase inside the
latest user message wins: its `tools` are called one per model request, in
order (a tool the runtime did not offer on that request is skipped), and then
its `text` is streamed word by word.

  {ref}            in text: the artifact reference the last tool result offered.
  "{{id:prefix_}}" as a whole argument string: the newest id with that prefix
                   found in this turn's tool results, so a step can use an id
                   an earlier step looked up (an email profile, say).

Environment:
  MOCK_SCRIPT      the rules file (default: script.json beside this file)
  MOCK_LOG_DIR     where requests.jsonl and offered.log go (default: a temp dir)
  MOCK_WORD_DELAY  seconds between streamed words (default 0.03)

fail.json beside the script switches a failure on, as {"mode": "..."}:
offtopic (the guard refuses), busy (429), fallback (529 on mock-sonnet),
down (500), cut (the reply drops after a few words).
"""
import json, os, re, sys, tempfile, time, uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPT = os.path.abspath(os.environ.get("MOCK_SCRIPT", os.path.join(HERE, "script.json")))
LOG_DIR = os.environ.get("MOCK_LOG_DIR", os.path.join(tempfile.gettempdir(), "trenova-mockllm"))
os.makedirs(LOG_DIR, exist_ok=True)
LOG = os.path.join(LOG_DIR, "requests.jsonl")
OFFERED = os.path.join(LOG_DIR, "offered.log")
FAIL = os.path.join(os.path.dirname(SCRIPT), "fail.json")
WORD_DELAY = float(os.environ.get("MOCK_WORD_DELAY", "0.03"))

ARTIFACT_REF = re.compile(r"\[[^\[\]\n]+\]\(artifact:[A-Za-z0-9_]+\)")
ID_SLOT = re.compile(r"^\{\{id:([a-z]+_)\}\}$")


def load_script():
    # Read on every request, so the rules can change while the mock runs.
    try:
        with open(SCRIPT) as f:
            return json.load(f)
    except FileNotFoundError:
        return {"rules": [], "default": "I looked into it."}


def fill(schema):
    t = schema.get("type")
    if "enum" in schema:
        return schema["enum"][0]
    if t == "object" or "properties" in schema:
        return {k: fill(v) for k, v in schema.get("properties", {}).items()}
    if t == "array":
        return []
    if t in ("integer", "number"):
        return 0
    if t == "boolean":
        return False
    return "ok"


def text_of(content):
    if isinstance(content, list):
        return " ".join(p.get("text", "") for p in content if isinstance(p, dict))
    return content or ""


def last_user(messages):
    for m in reversed(messages):
        if m.get("role") == "user":
            return text_of(m.get("content"))
    return ""


def tool_results_since_user(messages):
    out = []
    for m in reversed(messages):
        if m.get("role") == "user":
            break
        if m.get("role") == "tool":
            out.append(m)
    return list(reversed(out))


def with_ref(text, messages):
    """Fills {ref} with the artifact reference the last tool result offered, as a model would."""
    if "{ref}" not in text:
        return text
    for m in reversed(tool_results_since_user(messages)):
        found = ARTIFACT_REF.search(text_of(m.get("content")))
        if found:
            return text.replace("{ref}", found.group(0))
    return text.replace("{ref}", "the record beside this")


def with_ids(value, messages):
    """Fills "{{id:prefix_}}" arguments from this turn's tool results, newest first."""
    if isinstance(value, dict):
        return {k: with_ids(v, messages) for k, v in value.items()}
    if isinstance(value, list):
        return [with_ids(v, messages) for v in value]
    if isinstance(value, str):
        slot = ID_SLOT.match(value)
        if slot:
            pattern = re.compile(re.escape(slot.group(1)) + r"[0-9A-Z]{26}")
            for m in reversed(tool_results_since_user(messages)):
                found = pattern.search(text_of(m.get("content")))
                if found:
                    return found.group(0)
    return value


def read_mode():
    try:
        with open(FAIL) as f:
            return json.load(f).get("mode", "")
    except (FileNotFoundError, ValueError):
        return ""


def plan(body):
    """What to answer: a tool call to make next, or the final text."""
    messages = body.get("messages", [])
    tools = {t["function"]["name"] for t in body.get("tools", []) if t.get("type") == "function"}
    question = last_user(messages).lower()
    with open(OFFERED, "a") as log:
        log.write(question[:80].replace("\n", " ") + " | " + ",".join(sorted(tools)) + "\n")
    done = len(tool_results_since_user(messages))
    script = load_script()
    for rule in script["rules"]:
        if any(k in question for k in rule["match"]):
            steps = [s for s in rule.get("tools", []) if s["name"] in tools]
            if done < len(steps):
                step = dict(steps[done])
                step["arguments"] = with_ids(step.get("arguments", {}), messages)
                return {"tool": step}
            return {"text": with_ref(rule["text"], messages)}
    return {"text": script.get("default", "Done.")}


def chunk(model, delta, finish=None):
    return {
        "id": "chatcmpl-mock",
        "object": "chat.completion.chunk",
        "created": int(time.time()),
        "model": model,
        "choices": [{"index": 0, "delta": delta, "finish_reason": finish}],
    }


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def do_GET(self):
        if self.path.rstrip("/").endswith("/models"):
            return self.json({"object": "list", "data": [{"id": "mock-sonnet", "object": "model"}]})
        self.json({"ok": True, "script": SCRIPT})

    def json(self, payload, status=200):
        data = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
        with open(LOG, "a") as f:
            f.write(json.dumps({"path": self.path, "body": body}) + "\n")
        model = body.get("model", "mock-sonnet")
        fmt = body.get("response_format") or {}
        is_guard = fmt.get("type") == "json_schema"
        mode = read_mode()
        if not is_guard and mode == "busy":
            return self.json({"error": {"message": "Overloaded", "type": "overloaded_error"}}, 429)
        if not is_guard and mode == "fallback" and model == "mock-sonnet":
            return self.json({"error": {"message": "Overloaded", "type": "overloaded_error"}}, 529)
        if not is_guard and mode == "down":
            return self.json({"error": {"message": "Internal error"}}, 500)
        if is_guard:
            schema = fmt["json_schema"].get("schema", {})
            name = fmt["json_schema"].get("name", "")
            value = fill(schema)
            if name == "scope_classification":
                value = {"category": "TransportationOperations", "reasoning": "Freight work."}
                if mode == "offtopic":
                    value = {"category": "GeneralKnowledge", "reasoning": "A general trivia question."}
            return self.reply(body, model, {"text": json.dumps(value)})
        step = plan(body)
        if mode == "cut" and "text" in step:
            step = {**step, "cut": True}
        return self.reply(body, model, step)

    def reply(self, body, model, step):
        usage = {"prompt_tokens": 120, "completion_tokens": 40, "total_tokens": 160}
        if not body.get("stream"):
            message = {"role": "assistant", "content": step.get("text")}
            finish = "stop"
            if "tool" in step:
                message = {"role": "assistant", "content": None, "tool_calls": [self.call(step["tool"])]}
                finish = "tool_calls"
            return self.json({
                "id": "chatcmpl-mock", "object": "chat.completion", "created": int(time.time()), "model": model,
                "choices": [{"index": 0, "message": message, "finish_reason": finish}], "usage": usage,
            })
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "close")
        self.end_headers()

        def send(obj):
            self.wfile.write(b"data: " + json.dumps(obj).encode() + b"\n\n")
            self.wfile.flush()

        try:
            send(chunk(model, {"role": "assistant", "content": ""}))
            if "tool" in step:
                time.sleep(float(step["tool"].get("delay", 0.8)))
                send(chunk(model, {"tool_calls": [{"index": 0, **self.call(step["tool"])}]}))
                send(chunk(model, {}, "tool_calls"))
            else:
                time.sleep(0.4)
                for i, word in enumerate(step["text"].split(" ")):
                    if step.get("cut") and i == 12:
                        self.connection.close()
                        return
                    send(chunk(model, {"content": word + " "}))
                    time.sleep(WORD_DELAY)
                send(chunk(model, {}, "stop"))
            final = chunk(model, {})
            final["choices"] = []
            final["usage"] = usage
            send(final)
            self.wfile.write(b"data: [DONE]\n\n")
            self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError):
            # The runtime hung up: a stopped turn closes the stream mid-reply.
            pass

    def call(self, tool):
        return {
            "id": "call_" + uuid.uuid4().hex[:12],
            "type": "function",
            "function": {"name": tool["name"], "arguments": json.dumps(tool.get("arguments", {}))},
        }


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 5055
    print(f"mock LLM on :{port}, script {SCRIPT}, logs in {LOG_DIR}", flush=True)
    ThreadingHTTPServer(("0.0.0.0", port), Handler).serve_forever()
