import json

Refused = OSError

_state = {"result": False}


def _describe(entry):
    lines = [entry["name"], "", entry.get("description", "")]
    if entry.get("writes"):
        lines += ["", "A change: calling it prepares it for the script's batch; it does not run."]
    parameters = entry.get("parameters") or {}
    properties = parameters.get("properties") or {}
    required = parameters.get("required") or []
    if properties:
        lines += ["", "Parameters:"]
        for key in properties:
            spec = properties[key]
            mark = " (required)" if key in required else ""
            lines.append("  " + key + mark + ": " + str(spec.get("description", spec.get("type", ""))))
    return "\n".join(lines)


def _answer(tool, reply):
    if reply.get("status") == "ok":
        return reply.get("data")
    raise Refused(tool + ": " + str(reply.get("status", "failed")) + ": " + str(reply.get("message", "")))


class _Pending:
    def __init__(self, tool, args):
        self.tool = tool
        self.args = args

    def __repr__(self):
        return "<trenova deferred " + self.tool + ">"


class _Tool:
    def __init__(self, entry):
        self.name = entry["name"]
        self.writes = bool(entry.get("writes"))
        self.__doc__ = _describe(entry)

    def __call__(self, **args):
        reply = json.loads(_trenova_read(json.dumps({"tool": self.name, "args": args})))
        return _answer(self.name, reply)

    def defer(self, **args):
        return _Pending(self.name, args)

    def __repr__(self):
        return "<trenova " + ("change " if self.writes else "read ") + self.name + ">"


class _Namespace:
    def __init__(self, name, tools):
        self._name = name
        for verb in tools:
            setattr(self, verb, tools[verb])

    def __repr__(self):
        return "<trenova namespace " + self._name + ">"


class _Trenova:
    def __init__(self, index, inputs):
        self.Refused = Refused
        self.inputs = inputs
        self._names = sorted(entry["name"] for entry in index)
        spaces = {}
        for entry in index:
            space, _, verb = entry["name"].partition("_")
            spaces.setdefault(space, {})[verb] = _Tool(entry)
        for space in spaces:
            setattr(self, space, _Namespace(space, spaces[space]))

    def tools(self):
        return list(self._names)

    def gather(self, *pending, return_exceptions=False):
        for call in pending:
            if not isinstance(call, _Pending):
                raise TypeError("gather() takes calls made with .defer(...)")
        calls = [{"tool": call.tool, "args": call.args} for call in pending]
        replies = json.loads(_trenova_gather(json.dumps(calls)))
        answers = []
        for call, reply in zip(pending, replies):
            try:
                answers.append(_answer(call.tool, reply))
            except Refused as refused:
                if not return_exceptions:
                    raise
                answers.append(refused)
        return answers

    def result(self, value):
        if _state["result"]:
            raise RuntimeError("result() may be called once")
        _state["result"] = True
        reply = json.loads(_trenova_result(json.dumps(value)))
        if reply.get("status") != "ok":
            raise Refused("result: " + str(reply.get("status")) + ": " + str(reply.get("message", "")))

    def __repr__(self):
        return "<trenova: " + str(len(self._names)) + " tools; tools() lists them>"


def help(obj=None):
    print(getattr(obj, "__doc__", None) or repr(obj))


trenova = _Trenova(json.loads(_TRENOVA_INDEX), json.loads(_TRENOVA_INPUTS))
inputs = trenova.inputs
