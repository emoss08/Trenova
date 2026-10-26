import json
import datetime

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


_TIME_KINDS = ("date", "datetime")


def _parse_time(value, kind):
    if not isinstance(value, str):
        return value
    try:
        if kind == "date":
            return datetime.date.fromisoformat(value[:10])
        return datetime.datetime.fromisoformat(value)
    except ValueError:
        return value


def _load_rows(entry):
    columns = [column["name"] for column in entry["columns"]]
    times = [(index, column["kind"]) for index, column in enumerate(entry["columns"]) if column["kind"] in _TIME_KINDS]
    rows = []
    offset = 0
    while True:
        page = json.loads(_trenova_rows(json.dumps({"dataset": entry["name"], "offset": offset})))
        chunk = page.get("rows") or []
        for raw in chunk:
            for index, kind in times:
                raw[index] = _parse_time(raw[index], kind)
            rows.append(dict(zip(columns, raw)))
        offset += len(chunk)
        if page.get("done") or not chunk:
            return rows


def _plain(value):
    if isinstance(value, datetime.datetime) or isinstance(value, datetime.date):
        return value.isoformat()
    if value is None or isinstance(value, bool) or isinstance(value, int) or isinstance(value, str):
        return value
    if isinstance(value, float):
        if value != value or value in (float("inf"), float("-inf")):
            return None
        return value
    return str(value)


def _show(payload):
    reply = json.loads(_trenova_show(json.dumps(payload)))
    if reply.get("status") != "ok":
        print("[not shown] " + str(reply.get("message", "")))


def show_figure(label, value, kind=None, note="", currency=""):
    _show({"type": "figure", "label": str(label), "value": _plain(value), "kind": kind or "", "note": str(note), "currency": str(currency)})


def show_table(rows, title="", columns=None, kinds=None):
    if isinstance(rows, dict):
        names = list(columns or ["Name", "Value"])
        values = [[_plain(key), _plain(rows[key])] for key in rows]
    else:
        rows = list(rows)
        if columns is not None:
            names = list(columns)
        elif rows and isinstance(rows[0], dict):
            names = list(rows[0].keys())
        else:
            width = len(rows[0]) if rows else 0
            names = ["Column " + str(index + 1) for index in range(width)]
        values = []
        for row in rows[:200]:
            if isinstance(row, dict):
                values.append([_plain(row.get(name)) for name in names])
            else:
                values.append([_plain(cell) for cell in row])
    kinds = kinds or {}
    _show({
        "type": "table",
        "title": str(title),
        "columns": [{"name": str(name), "kind": kinds.get(name, "")} for name in names],
        "rows": values[:200],
        "totalRows": len(rows) if not isinstance(rows, dict) else len(values),
    })


def show_chart(kind, x, y, title="", x_label="", y_label="", value_kind=None, stacked=False, horizontal=False):
    if isinstance(y, dict):
        series = [{"name": str(name), "values": [_plain(value) for value in y[name]]} for name in y]
    else:
        series = [{"name": str(y_label or "Value"), "values": [_plain(value) for value in y]}]
    _show({
        "type": "chart",
        "kind": str(kind),
        "title": str(title),
        "x": {"label": str(x_label), "values": [_plain(value) for value in x]},
        "series": series,
        "valueKind": value_kind or "",
        "stacked": bool(stacked),
        "horizontal": bool(horizontal),
    })


def _numbers(values):
    return [value for value in values if (isinstance(value, int) or isinstance(value, float)) and not isinstance(value, bool)]


def mean(values):
    numbers = _numbers(values)
    if not numbers:
        raise ValueError("mean() needs at least one number")
    return sum(numbers) / len(numbers)


def median(values):
    return percentile(values, 50)


def percentile(values, pct):
    numbers = sorted(_numbers(values))
    if not numbers:
        raise ValueError("percentile() needs at least one number")
    if len(numbers) == 1:
        return numbers[0]
    rank = (len(numbers) - 1) * pct / 100
    low = int(rank)
    high = min(low + 1, len(numbers) - 1)
    return numbers[low] + (numbers[high] - numbers[low]) * (rank - low)


def stdev(values):
    numbers = _numbers(values)
    if len(numbers) < 2:
        raise ValueError("stdev() needs at least two numbers")
    centre = sum(numbers) / len(numbers)
    return (sum((value - centre) ** 2 for value in numbers) / (len(numbers) - 1)) ** 0.5


def group_by(rows, key):
    groups = {}
    for row in rows:
        name = row.get(key) if isinstance(key, str) else key(row)
        groups.setdefault(name, []).append(row)
    return groups


trenova = _Trenova(json.loads(_TRENOVA_INDEX), json.loads(_TRENOVA_INPUTS))
inputs = trenova.inputs
trenova.datasets = {}
_DATASET_ROWS = {}
for _entry in json.loads(_TRENOVA_DATASETS):
    trenova.datasets[_entry["name"]] = {"label": _entry.get("label", ""), "rows": _entry.get("rows", 0), "columns": _entry["columns"]}
    if _entry.get("text") is not None:
        _DATASET_ROWS[_entry["name"]] = _entry["text"]
    else:
        _DATASET_ROWS[_entry["name"]] = _load_rows(_entry)
