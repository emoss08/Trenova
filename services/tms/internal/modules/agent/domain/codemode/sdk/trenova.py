"""Trenova's SDK for agent scripts.

Every call is one of the agent's own tools, run by Trenova as the person. A
read answers with its result. A change, where the turn may prepare one, never
runs: Trenova prepares it, and when the script ends every change it prepared
is proposed as one batch for a person to approve. Nothing here can reach the
network: a call is written as a request frame on standard output and answered
by Trenova through a file.
"""

import itertools
import json
import os
import sys
import time

_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
with open(os.path.join(_ROOT, "config.json"), encoding="utf-8") as _f:
    _CONFIG = json.load(_f)
with open(os.path.join(_ROOT, "index.json"), encoding="utf-8") as _f:
    _INDEX = {entry["name"]: entry for entry in json.load(_f)}
with open(os.path.join(_ROOT, "inputs.json"), encoding="utf-8") as _f:
    inputs = json.load(_f)

_NONCE = _CONFIG["nonce"]
_POLL = _CONFIG["pollMs"] / 1000.0
_DEADLINE = time.monotonic() + _CONFIG["wallSeconds"]
_FRAME_BYTES = _CONFIG["frameBytes"]
_RESULT_BYTES = _CONFIG["resultBytes"]
_RPC = _CONFIG["rpcDir"]
_TERMINATOR = "\n\x1e" + _NONCE + "\n"
_SEQ = itertools.count(1)
_OUT = sys.stdout
_ROUND_TRIPS = []
_result_sent = False

__all__ = ["Refused", "gather", "inputs", "result", "tools"]


class Refused(Exception):
    """A read Trenova refused or could not answer."""

    def __init__(self, tool, status, message):
        super().__init__("%s: %s" % (tool, message))
        self.tool = tool
        self.status = status
        self.message = message


def _frame(kind, payload):
    body = json.dumps(payload, separators=(",", ":"), default=str)
    line = "\x1etrenova-%s:%s:%s\n" % (kind, _NONCE, body)
    _OUT.write(line)
    _OUT.flush()
    return len(body)


def _send(tool, args):
    seq = next(_SEQ)
    size = _frame("rpc", {"seq": seq, "tool": tool, "args": args})
    if size > _FRAME_BYTES:
        raise Refused(tool, "invalid", "the call's arguments are too large")
    return seq, time.monotonic()


def _wait(tool, seq, started):
    path = os.path.join(_RPC, "%d.json" % seq)
    while True:
        try:
            with open(path, encoding="utf-8") as handle:
                content = handle.read()
        except FileNotFoundError:
            content = ""
        if content.endswith(_TERMINATOR):
            _ROUND_TRIPS.append(int((time.monotonic() - started) * 1000))
            reply = json.loads(content[: -len(_TERMINATOR)])
            if reply.get("status") == "ok":
                return reply.get("data")
            raise Refused(tool, reply.get("status", "failed"), reply.get("message", ""))
        if time.monotonic() > _DEADLINE:
            raise Refused(tool, "limit", "the script ran out of time")
        time.sleep(_POLL)


class _Pending:
    def __init__(self, tool, args):
        self.tool = tool
        self.args = args


class _Tool:
    def __init__(self, entry):
        self.name = entry["name"]
        self.writes = bool(entry.get("writes"))
        self.__doc__ = _describe(entry)

    def __call__(self, **args):
        seq, started = _send(self.name, args)
        return _wait(self.name, seq, started)

    def defer(self, **args):
        """Prepare a call for gather()."""
        return _Pending(self.name, args)

    def __repr__(self):
        return "<trenova %s %s>" % ("change" if self.writes else "read", self.name)


def _describe(entry):
    lines = [entry["name"], "", entry.get("description", "")]
    if entry.get("writes"):
        lines += ["", "A change: calling it prepares it for the script's batch; it does not run."]
    props = (entry.get("parameters") or {}).get("properties") or {}
    required = set((entry.get("parameters") or {}).get("required") or [])
    if props:
        lines.append("")
        lines.append("Parameters:")
        for key, spec in props.items():
            mark = " (required)" if key in required else ""
            lines.append("  %s%s: %s" % (key, mark, spec.get("description", spec.get("type", ""))))
    return "\n".join(lines)


class _Namespace:
    def __init__(self, name):
        self._name = name

    def __getattr__(self, rest):
        full = self._name + "_" + rest
        entry = _INDEX.get(full)
        if entry is None:
            raise AttributeError(
                "trenova.%s.%s is not a read this turn holds; tools() lists them" % (self._name, rest)
            )
        return _Tool(entry)

    def __dir__(self):
        prefix = self._name + "_"
        return sorted(name[len(prefix):] for name in _INDEX if name.startswith(prefix))


def __getattr__(name):
    prefix = name + "_"
    if any(tool.startswith(prefix) for tool in _INDEX):
        return _Namespace(name)
    raise AttributeError("trenova has no namespace %r; tools() lists the reads" % name)


def tools():
    """The reads this script may call, and the changes it may prepare, by name."""
    return sorted(_INDEX)


def gather(*pending, return_exceptions=False):
    """Run several deferred reads together, answered in the order given."""
    sent = []
    for call in pending:
        if not isinstance(call, _Pending):
            raise TypeError("gather() takes calls made with .defer(...)")
        seq, started = _send(call.tool, call.args)
        sent.append((call.tool, seq, started))
    answers = []
    for tool, seq, started in sent:
        try:
            answers.append(_wait(tool, seq, started))
        except Refused as refused:
            if not return_exceptions:
                raise
            answers.append(refused)
    return answers


def result(value):
    """Hand back one structured value, once, at most %d bytes encoded."""
    global _result_sent
    if _result_sent:
        raise RuntimeError("result() may be called once")
    _result_sent = True
    size = _frame("result", {"value": value})
    if size > _RESULT_BYTES:
        raise Refused("result", "limit", "the result is too large")


def _end(status, limit="", error=None):
    _frame("end", {"status": status, "limit": limit, "error": error, "roundTripsMs": _ROUND_TRIPS})
