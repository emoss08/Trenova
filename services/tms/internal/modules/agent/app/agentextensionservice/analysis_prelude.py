# Trenova's preparation for an analysis, run in the sandbox before the
# agent's code. It loads each dataset as a typed DataFrame named after it,
# styles charts so they read in a light or a dark theme, and defines
# show_table and show_figure, which hand tables and headline numbers back as
# values rather than printed text. Each step is guarded: a step that fails
# leaves the rest in place and never fails the analysis.
import datetime as _datetime
import json as _json
import warnings as _warnings

import numpy as np
import pandas as pd
from IPython.display import display as _display

_warnings.filterwarnings("ignore", category=FutureWarning)
_warnings.filterwarnings("ignore", category=DeprecationWarning)

try:
    with open("/home/user/.trenova/analysis.json", encoding="utf-8") as _file:
        _CONFIG = _json.load(_file)
except Exception:
    _CONFIG = {}

TIMEZONE = _CONFIG.get("timezone") or "UTC"
_FORMATS = {}

try:
    pd.set_option("display.width", 200)
    pd.set_option("display.max_columns", 40)
    pd.set_option("display.max_colwidth", 60)
except Exception:
    pass

try:
    import matplotlib
    import matplotlib.pyplot as plt
    import matplotlib.ticker as _ticker

    _INK = "#737373"
    _RULE = "#a3a3a3"
    PALETTE = [
        "#2f6fde",
        "#0f9b8e",
        "#d97706",
        "#8b5cf6",
        "#dc4c4c",
        "#3f9142",
        "#c2468f",
        "#6b7a8f",
    ]
    plt.rcParams.update(
        {
            "figure.figsize": (8, 4.5),
            "figure.dpi": 144,
            "savefig.dpi": 144,
            "figure.facecolor": "none",
            "axes.facecolor": "none",
            "savefig.facecolor": "none",
            "savefig.transparent": True,
            "font.size": 10,
            "axes.titlesize": 12,
            "axes.titleweight": "semibold",
            "axes.titlelocation": "left",
            "axes.titlepad": 12,
            "axes.titlecolor": _INK,
            "axes.labelsize": 10,
            "axes.labelcolor": _INK,
            "text.color": _INK,
            "xtick.color": _INK,
            "ytick.color": _INK,
            "axes.edgecolor": _RULE,
            "axes.spines.top": False,
            "axes.spines.right": False,
            "axes.grid": True,
            "grid.color": _RULE,
            "grid.alpha": 0.35,
            "grid.linewidth": 0.6,
            "axes.axisbelow": True,
            "axes.prop_cycle": matplotlib.cycler(color=PALETTE),
            "axes.formatter.useoffset": False,
            "axes.formatter.limits": (-12, 15),
            "axes.xmargin": 0.04,
            "axes.ymargin": 0.08,
            "legend.frameon": False,
            "lines.linewidth": 2,
        }
    )

    def _compact(value, _position=None):
        magnitude = abs(value)
        for limit, suffix in ((1e12, "T"), (1e9, "B"), (1e6, "M"), (1e3, "K")):
            if magnitude >= limit:
                text = f"{value / limit:,.1f}".rstrip("0").rstrip(".")
                return text + suffix
        if float(value).is_integer():
            return f"{value:,.0f}"
        return f"{value:,.2f}".rstrip("0").rstrip(".")

    def _polish(figure):
        for axes in figure.axes:
            for axis in (axes.xaxis, axes.yaxis):
                if type(axis.get_major_formatter()) is _ticker.ScalarFormatter:
                    axis.set_major_formatter(_ticker.FuncFormatter(_compact))

    def _polish_open():
        for number in plt.get_fignums():
            try:
                _polish(plt.figure(number))
            except Exception:
                pass

    _plain_show = plt.show

    def _show(*args, **kwargs):
        _polish_open()
        return _plain_show(*args, **kwargs)

    plt.show = _show

    try:
        from matplotlib_inline import backend_inline as _inline

        _shell = get_ipython()  # noqa: F821
        _shell.events.unregister("post_execute", _inline.flush_figures)

        def _flush():
            _polish_open()
            _inline.flush_figures()

        _shell.events.register("post_execute", _flush)
    except Exception:
        pass
except Exception:
    pass


def _is_missing(value):
    try:
        return value is None or value is pd.NA or value is pd.NaT or bool(pd.isna(value))
    except (TypeError, ValueError):
        return False


def _cell(value):
    if _is_missing(value):
        return None
    if isinstance(value, pd.Timestamp):
        return value.isoformat()
    if isinstance(value, (_datetime.datetime, _datetime.date)):
        return value.isoformat()
    if isinstance(value, np.generic):
        value = value.item()
    if isinstance(value, bool):
        return value
    if isinstance(value, int):
        return value if abs(value) < 2**53 else float(value)
    if isinstance(value, float):
        return value if np.isfinite(value) else None
    if isinstance(value, str):
        return value
    return str(value)


def _kind_of(series, name, given):
    if given:
        return given, ""
    if name in _FORMATS:
        return _FORMATS[name]
    try:
        if pd.api.types.is_bool_dtype(series):
            return "bool", ""
        if pd.api.types.is_datetime64_any_dtype(series):
            times = series.dropna()
            if len(times) and (times.dt.hour.any() or times.dt.minute.any()):
                return "datetime", ""
            return "date", ""
        if pd.api.types.is_integer_dtype(series):
            return "number", ""
        if pd.api.types.is_numeric_dtype(series):
            return "number", ""
    except Exception:
        pass
    return "text", ""


def show_table(data, title=None, formats=None, max_rows=200):
    """Show a DataFrame or Series as a table beside the reply.

    formats names how a column reads when it is not a dataset column:
    {"Share": "percent", "Margin": "money"}. Kinds are text, number, money,
    count, percent (a fraction: 0.25 is 25%), date, datetime and bool.
    """
    frame = data.to_frame() if isinstance(data, pd.Series) else pd.DataFrame(data)
    if not isinstance(frame.index, pd.RangeIndex) or frame.index.name is not None:
        frame = frame.reset_index()
    formats = formats or {}
    columns = []
    for name in frame.columns:
        kind, currency = _kind_of(frame[name], str(name), formats.get(name))
        columns.append({"name": str(name), "kind": kind, "currency": currency})
    rows = [
        [_cell(value) for value in row]
        for row in frame.head(max_rows).itertuples(index=False, name=None)
    ]
    _display(
        {
            "application/json": {
                "trenova": "table",
                "title": "" if title is None else str(title),
                "columns": columns,
                "rows": rows,
                "totalRows": int(len(frame)),
            }
        },
        raw=True,
    )


def show_figure(label, value, kind=None, note=None, currency=None):
    """Show one headline number beside the reply, such as the total open
    balance. kind is money, count, percent (a fraction), number, date or text."""
    value = _cell(value)
    if kind is None:
        if isinstance(value, bool):
            kind = "bool"
        elif isinstance(value, int):
            kind = "count"
        elif isinstance(value, float):
            kind = "number"
        else:
            kind = "text"
    _display(
        {
            "application/json": {
                "trenova": "figure",
                "label": str(label),
                "value": value,
                "kind": kind,
                "currency": currency or "",
                "note": "" if note is None else str(note),
            }
        },
        raw=True,
    )


def _typed(frame, column):
    source = frame[column["id"]]
    blank = source.replace("", np.nan)
    kind = column.get("type")
    if kind == "int":
        return pd.to_numeric(blank, errors="coerce").astype("Int64")
    if kind == "decimal":
        return pd.to_numeric(blank, errors="coerce").astype("float64")
    if kind == "epoch":
        seconds = pd.to_numeric(blank, errors="coerce")
        return pd.to_datetime(seconds, unit="s", utc=True).dt.tz_convert(TIMEZONE)
    if kind == "bool":
        return source.map({"true": True, "false": False}).astype("boolean")
    return source.astype("string").replace("", pd.NA)


def _load_report(spec):
    frame = pd.read_csv(spec["path"], dtype=str, keep_default_na=False)
    labels = {}
    taken = set()
    for column in spec.get("columns") or []:
        identifier = column.get("id")
        if identifier not in frame.columns:
            continue
        frame[identifier] = _typed(frame, column)
        label = column.get("label") or identifier
        candidate, count = label, 2
        while candidate in taken:
            candidate = f"{label} ({count})"
            count += 1
        taken.add(candidate)
        labels[identifier] = candidate
        _FORMATS[candidate] = (column.get("kind") or "text", column.get("currency") or "")
    return frame.rename(columns=labels)


def _load(spec):
    entry = {"name": spec.get("name", ""), "rows": 0, "columns": [], "error": ""}
    try:
        extension = spec.get("extension")
        if extension == "csv" and spec.get("columns"):
            frame = _load_report(spec)
        elif extension == "csv":
            frame = pd.read_csv(spec["path"])
        elif extension == "xlsx":
            frame = pd.read_excel(spec["path"])
        elif extension == "json":
            frame = pd.read_json(spec["path"])
        else:
            frame = None
        if frame is not None:
            globals()[spec["name"]] = frame
            entry["rows"] = int(len(frame))
            entry["columns"] = [
                {"name": str(name), "dtype": str(dtype)} for name, dtype in frame.dtypes.items()
            ]
    except Exception as error:
        entry["error"] = f"{type(error).__name__}: {error}"
    return entry


_LOADED = [_load(_spec) for _spec in _CONFIG.get("datasets") or []]
_display({"application/json": {"trenova": "datasets", "datasets": _LOADED}}, raw=True)
