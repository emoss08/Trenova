"""Time real extraction requests against a served model, the way Trenova sends them.

Each request is the production chat call: the rendered prompt, the reply schema as a
strict `json_schema` response format (or `json_object`, or nothing, following the
dataset's structured output mode), the dataset's sampling, and Trenova's output token
ceiling, streamed with usage reporting. Timing is taken client side:

- time to first token: request sent to the first generated token arriving;
- time per output token: the rest of the reply divided by the tokens after the first;
- end-to-end latency: request sent to the stream closing.

Levels run the same prompts at increasing concurrency, so a report shows both the
latency one document sees and the throughput the server sustains. Replies can be
written in the scorer's format, so a serving change is judged on accuracy as well as
speed: quantization and speculative decoding can both change what the model says.
"""

from __future__ import annotations

import json
import math
import time
import urllib.error
import urllib.parse
import urllib.request
from collections import Counter
from collections.abc import Callable, Iterable, Sequence
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, field
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from jsonschema.validators import validator_for

from . import dataset
from .predict import ERROR_EMPTY, ERROR_TRUNCATED, structured_kind, write_predictions

BENCH_FORMAT = "trenova.extraction-bench/v1"
CHAT_PATH = "/chat/completions"
SSE_DATA = "data:"
SSE_DONE = "[DONE]"
ERROR_BODY_LIMIT = 300
TOP_ERRORS = 5
PERCENTILES = (50, 90, 99)
MILLISECONDS = 1000.0


class BenchError(ValueError):
    """The benchmark cannot run as asked, or its reports cannot be compared."""


@dataclass(frozen=True)
class Endpoint:
    base_url: str
    model: str
    api_key: str | None = field(default=None, repr=False)
    timeout: float = 300.0

    def __post_init__(self) -> None:
        parsed = urllib.parse.urlsplit(self.base_url)
        if parsed.scheme not in ("http", "https") or not parsed.hostname:
            raise BenchError(f"{self.base_url!r} is not an http(s) base URL")
        if parsed.username or parsed.password:
            raise BenchError("put the API key in its environment variable, not the URL")
        if parsed.query or parsed.fragment:
            raise BenchError(f"{self.base_url!r} must not carry a query or fragment")
        if not self.model.strip():
            raise BenchError("the served model name is empty")
        if self.timeout <= 0:
            raise BenchError("the request timeout must be positive")

    @property
    def url(self) -> str:
        return self.base_url.rstrip("/") + CHAT_PATH


@dataclass(frozen=True)
class Workload:
    """What every request carries, read from the rendered dataset."""

    prompts: tuple[tuple[str, list], ...]
    schema: dict[str, Any]
    manifest: dataset.DatasetManifest
    max_tokens: int

    @classmethod
    def load(cls, dataset_dir: Path, max_tokens: int) -> Workload:
        if max_tokens < 1:
            raise BenchError("max_tokens must be positive")
        manifest = dataset.load_manifest(dataset_dir)
        dataset.verify(dataset_dir, manifest)
        prompts = tuple(dataset.load_evaluation_prompts(dataset_dir))
        if not prompts:
            raise BenchError(f"{dataset_dir} has no validation prompts to send")
        return cls(prompts, dataset.load_schema(dataset_dir), manifest, max_tokens)

    def body(self, model: str, prompt: list) -> bytes:
        """The chat request Trenova's OpenAIChat adapter sends for one document."""
        request: dict[str, Any] = {
            "model": model,
            "messages": prompt,
            "max_tokens": self.max_tokens,
            "stream": True,
            "stream_options": {"include_usage": True},
        }
        response_format = self.response_format()
        if response_format is not None:
            request["response_format"] = response_format
        if self.manifest.temperature is not None:
            request["temperature"] = self.manifest.temperature
        if self.manifest.top_p is not None:
            request["top_p"] = self.manifest.top_p
        return json.dumps(request, ensure_ascii=False).encode("utf-8")

    def response_format(self) -> dict[str, Any] | None:
        kind = structured_kind(self.manifest.structured_output_mode)
        if kind == "json_schema":
            return {
                "type": "json_schema",
                "json_schema": {
                    "name": self.manifest.schema_name,
                    "schema": self.schema,
                    "strict": True,
                },
            }
        if kind == "json_object":
            return {"type": "json_object"}
        return None


@dataclass
class Sample:
    """One timed request."""

    id: str
    reply: str = ""
    error: str | None = None
    finish_reason: str | None = None
    ttft: float | None = None
    e2e: float | None = None
    prompt_tokens: int | None = None
    completion_tokens: int | None = None
    cached_tokens: int | None = None
    valid: bool = False

    @property
    def tpot(self) -> float | None:
        if self.ttft is None or self.e2e is None or not self.completion_tokens:
            return None
        if self.completion_tokens < 2:
            return None
        return (self.e2e - self.ttft) / (self.completion_tokens - 1)

    def prediction(self) -> dict[str, str]:
        """The line `trenova ai fine-tune score` reads for this request."""
        text = self.reply.strip()
        if self.error is not None:
            return {"id": self.id, "reply": text, "error": self.error}
        if self.finish_reason == "length":
            return {"id": self.id, "reply": text, "error": ERROR_TRUNCATED}
        if not text:
            return {"id": self.id, "reply": "", "error": ERROR_EMPTY}
        return {"id": self.id, "reply": text}


Opener = Callable[[urllib.request.Request, float], Any]
Clock = Callable[[], float]


def _open(request: urllib.request.Request, timeout: float) -> Any:
    return urllib.request.urlopen(request, timeout=timeout)


class Client:
    """Sends one streamed chat request and times it."""

    def __init__(
        self,
        endpoint: Endpoint,
        opener: Opener = _open,
        clock: Clock = time.perf_counter,
    ) -> None:
        self._endpoint = endpoint
        self._opener = opener
        self._clock = clock
        self._headers = {
            "Content-Type": "application/json",
            "Accept": "text/event-stream",
        }
        if endpoint.api_key:
            self._headers["Authorization"] = f"Bearer {endpoint.api_key}"

    def send(self, identifier: str, body: bytes) -> Sample:
        sample = Sample(id=identifier)
        request = urllib.request.Request(
            self._endpoint.url, data=body, headers=self._headers, method="POST"
        )
        started = self._clock()
        try:
            with self._opener(request, self._endpoint.timeout) as response:
                self._read(response, sample, started)
        except urllib.error.HTTPError as err:
            sample.error = f"HTTP {err.code}: {_error_body(err)}"
        except (urllib.error.URLError, OSError) as err:
            sample.error = f"request failed: {getattr(err, 'reason', err)}"
        sample.e2e = self._clock() - started
        return sample

    def _read(self, response: Iterable[bytes], sample: Sample, started: float) -> None:
        parts: list[str] = []
        for raw in response:
            line = raw.decode("utf-8", errors="replace").strip()
            if not line.startswith(SSE_DATA):
                continue
            data = line[len(SSE_DATA) :].strip()
            if data == SSE_DONE:
                break
            try:
                chunk = json.loads(data)
            except json.JSONDecodeError:
                sample.error = "the server sent a stream chunk that is not JSON"
                break
            if not isinstance(chunk, dict):
                continue
            if "error" in chunk:
                sample.error = f"stream error: {_message(chunk['error'])}"
                break
            if self._absorb(chunk, parts) and sample.ttft is None:
                sample.ttft = self._clock() - started
            _usage(chunk.get("usage"), sample)
            _finish(chunk, sample)
        sample.reply = "".join(parts)

    @staticmethod
    def _absorb(chunk: dict[str, Any], parts: list[str]) -> bool:
        generated = False
        for choice in chunk.get("choices") or ():
            delta = choice.get("delta") or {}
            content = delta.get("content")
            if content:
                parts.append(content)
                generated = True
            if delta.get("reasoning_content") or delta.get("reasoning"):
                generated = True
        return generated


def _usage(usage: Any, sample: Sample) -> None:
    if not isinstance(usage, dict):
        return
    sample.prompt_tokens = _int(usage.get("prompt_tokens"), sample.prompt_tokens)
    sample.completion_tokens = _int(usage.get("completion_tokens"), sample.completion_tokens)
    details = usage.get("prompt_tokens_details")
    if isinstance(details, dict):
        sample.cached_tokens = _int(details.get("cached_tokens"), sample.cached_tokens)


def _finish(chunk: dict[str, Any], sample: Sample) -> None:
    for choice in chunk.get("choices") or ():
        reason = choice.get("finish_reason")
        if reason:
            sample.finish_reason = str(reason)


def _int(value: Any, fallback: int | None) -> int | None:
    return value if isinstance(value, int) and not isinstance(value, bool) else fallback


def _message(error: Any) -> str:
    if isinstance(error, dict):
        return str(error.get("message") or error)[:ERROR_BODY_LIMIT]
    return str(error)[:ERROR_BODY_LIMIT]


def _error_body(err: urllib.error.HTTPError) -> str:
    try:
        raw = err.read(ERROR_BODY_LIMIT * 4).decode("utf-8", errors="replace")
    except OSError:
        return err.reason or "no body"
    try:
        return _message(json.loads(raw).get("error") or raw)
    except (json.JSONDecodeError, AttributeError):
        return raw[:ERROR_BODY_LIMIT] or str(err.reason)


class ReplyChecker:
    """Whether a reply is JSON that satisfies the reply schema."""

    def __init__(self, schema: dict[str, Any]) -> None:
        cls = validator_for(schema)
        cls.check_schema(schema)
        self._validator = cls(schema)

    def valid(self, reply: str) -> bool:
        try:
            value = json.loads(reply)
        except json.JSONDecodeError:
            return False
        return self._validator.is_valid(value)


@dataclass(frozen=True)
class Level:
    concurrency: int
    samples: tuple[Sample, ...]
    wall_seconds: float


def run_level(
    client: Client,
    checker: ReplyChecker,
    plan: Sequence[tuple[str, bytes]],
    concurrency: int,
    clock: Clock = time.perf_counter,
) -> Level:
    """Send every request in `plan` with at most `concurrency` in flight."""
    if concurrency < 1:
        raise BenchError("concurrency must be at least 1")
    if not plan:
        raise BenchError("there are no requests to send")

    started = clock()
    with ThreadPoolExecutor(max_workers=min(concurrency, len(plan))) as pool:
        samples = tuple(pool.map(lambda item: client.send(*item), plan))
    wall = clock() - started
    for sample in samples:
        if sample.error is None:
            sample.valid = checker.valid(sample.reply)
    return Level(concurrency=concurrency, samples=samples, wall_seconds=wall)


def plan_requests(workload: Workload, model: str, count: int) -> list[tuple[str, bytes]]:
    """`count` requests, cycling through the prompts in order when there are fewer of them."""
    if count < 1:
        raise BenchError("the number of requests must be at least 1")
    prompts = workload.prompts
    return [
        (prompts[index % len(prompts)][0], workload.body(model, prompts[index % len(prompts)][1]))
        for index in range(count)
    ]


def distribution(values: Iterable[float | None]) -> dict[str, float] | None:
    """Mean, percentiles and maximum in milliseconds, or None when nothing was measured."""
    ordered = sorted(value for value in values if value is not None)
    if not ordered:
        return None
    summary = {"mean": sum(ordered) / len(ordered) * MILLISECONDS}
    for percentile in PERCENTILES:
        summary[f"p{percentile}"] = _percentile(ordered, percentile) * MILLISECONDS
    summary["max"] = ordered[-1] * MILLISECONDS
    return {key: round(value, 3) for key, value in summary.items()}


def _percentile(ordered: Sequence[float], percentile: float) -> float:
    if len(ordered) == 1:
        return ordered[0]
    rank = (len(ordered) - 1) * percentile / 100.0
    low = math.floor(rank)
    high = math.ceil(rank)
    return ordered[low] + (ordered[high] - ordered[low]) * (rank - low)


def summarize(level: Level) -> dict[str, Any]:
    samples = level.samples
    succeeded = [sample for sample in samples if sample.error is None]
    completion = sum(sample.completion_tokens or 0 for sample in succeeded)
    prompt = sum(sample.prompt_tokens or 0 for sample in succeeded)
    cached = [sample.cached_tokens for sample in succeeded if sample.cached_tokens is not None]
    wall = level.wall_seconds
    errors = Counter(sample.error for sample in samples if sample.error is not None)
    return {
        "concurrency": level.concurrency,
        "requests": len(samples),
        "succeeded": len(succeeded),
        "failed": len(samples) - len(succeeded),
        "truncated": sum(1 for sample in succeeded if sample.finish_reason == "length"),
        "invalidReplies": sum(1 for sample in succeeded if not sample.valid),
        "wallSeconds": round(wall, 3),
        "requestsPerSecond": round(len(succeeded) / wall, 3) if wall > 0 else None,
        "outputTokensPerSecond": round(completion / wall, 1) if wall > 0 else None,
        "promptTokensMean": round(prompt / len(succeeded), 1) if succeeded else None,
        "completionTokensMean": round(completion / len(succeeded), 1) if succeeded else None,
        "cachedPromptFraction": (
            round(sum(cached) / prompt, 4)
            if cached and prompt and len(cached) == len(succeeded)
            else None
        ),
        "ttftMs": distribution(sample.ttft for sample in succeeded),
        "tpotMs": distribution(sample.tpot for sample in succeeded),
        "e2eMs": distribution(sample.e2e for sample in succeeded),
        "errors": [
            {"error": message, "count": count} for message, count in errors.most_common(TOP_ERRORS)
        ],
    }


@dataclass(frozen=True)
class BenchPlan:
    concurrency: tuple[int, ...]
    requests: int | None = None
    warmup: int = 2
    label: str = ""
    predictions_dir: Path | None = None

    def __post_init__(self) -> None:
        if not self.concurrency or any(level < 1 for level in self.concurrency):
            raise BenchError("give at least one concurrency level, each at least 1")
        if len(set(self.concurrency)) != len(self.concurrency):
            raise BenchError("each concurrency level may appear once")
        if self.requests is not None and self.requests < 1:
            raise BenchError("the number of requests must be at least 1")
        if self.warmup < 0:
            raise BenchError("warmup cannot be negative")


def bench(
    workload: Workload,
    client: Client,
    endpoint: Endpoint,
    plan: BenchPlan,
    progress: Callable[[str], None] = lambda _message: None,
) -> dict[str, Any]:
    """Warm the server, run each level, and return the report."""
    checker = ReplyChecker(workload.schema)
    count = plan.requests or len(workload.prompts)
    requests = plan_requests(workload, endpoint.model, count)

    started_at = datetime.now(UTC).isoformat(timespec="seconds")
    if plan.warmup:
        warm = run_level(client, checker, requests[: plan.warmup], 1)
        failures = [sample.error for sample in warm.samples if sample.error is not None]
        if len(failures) == len(warm.samples):
            raise BenchError(f"every warmup request failed; first error: {failures[0]}")

    levels = []
    for concurrency in plan.concurrency:
        progress(f"concurrency {concurrency}: {count} requests")
        level = run_level(client, checker, requests, concurrency)
        if plan.predictions_dir is not None:
            write_predictions(
                plan.predictions_dir / f"predictions-c{concurrency}.jsonl", _first_replies(level)
            )
        levels.append(summarize(level))

    return {
        "format": BENCH_FORMAT,
        "label": plan.label,
        "startedAt": started_at,
        "endpoint": {"baseUrl": endpoint.base_url, "model": endpoint.model},
        "dataset": {
            "exportId": workload.manifest.export_id,
            "datasetManifestSha256": workload.manifest.sha256,
            "promptSha256": workload.manifest.prompt_sha256,
            "structuredOutputMode": workload.manifest.structured_output_mode,
            "schemaName": workload.manifest.schema_name,
        },
        "request": {
            "maxTokens": workload.max_tokens,
            "temperature": workload.manifest.temperature,
            "topP": workload.manifest.top_p,
        },
        "prompts": len(workload.prompts),
        "requestsPerLevel": count,
        "promptsRepeat": count > len(workload.prompts)
        or len(plan.concurrency) > 1
        or plan.warmup > 0,
        "warmup": plan.warmup,
        "levels": levels,
    }


def _first_replies(level: Level) -> list[dict[str, str]]:
    seen: set[str] = set()
    records = []
    for sample in level.samples:
        if sample.id in seen:
            continue
        seen.add(sample.id)
        records.append(sample.prediction())
    return records


COMPARED_METRICS = (
    ("e2eMs", "p50", "lower"),
    ("e2eMs", "p90", "lower"),
    ("ttftMs", "p50", "lower"),
    ("tpotMs", "p50", "lower"),
    ("outputTokensPerSecond", None, "higher"),
    ("requestsPerSecond", None, "higher"),
)
COMPARABLE_KEYS = (
    ("dataset", "promptSha256"),
    ("dataset", "datasetManifestSha256"),
    ("dataset", "structuredOutputMode"),
    ("request", "maxTokens"),
    ("request", "temperature"),
    ("request", "topP"),
)


def load_report(path: Path) -> dict[str, Any]:
    try:
        report = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise BenchError(f"cannot read the report {path}: {err}") from err
    if not isinstance(report, dict) or report.get("format") != BENCH_FORMAT:
        raise BenchError(f"{path} is not a {BENCH_FORMAT} report")
    return report


def compare(baseline: dict[str, Any], candidate: dict[str, Any]) -> list[dict[str, Any]]:
    """Per shared concurrency level, each metric on both sides and the relative change."""
    for section, key in COMPARABLE_KEYS:
        before = (baseline.get(section) or {}).get(key)
        after = (candidate.get(section) or {}).get(key)
        if before != after:
            raise BenchError(
                f"the reports ran different workloads ({section}.{key}: {before!r} vs {after!r});"
                " benchmark both against the same rendered dataset and settings"
            )
    by_level = {level["concurrency"]: level for level in candidate.get("levels") or ()}
    rows = []
    for level in baseline.get("levels") or ():
        other = by_level.get(level["concurrency"])
        if other is None:
            continue
        for metric, statistic, better in COMPARED_METRICS:
            before = _metric(level, metric, statistic)
            after = _metric(other, metric, statistic)
            change = None
            if before is not None and after is not None and before != 0:
                change = round((after - before) / before, 4)
            rows.append(
                {
                    "concurrency": level["concurrency"],
                    "metric": metric if statistic is None else f"{metric}.{statistic}",
                    "better": better,
                    "baseline": before,
                    "candidate": after,
                    "change": change,
                }
            )
    if not rows:
        raise BenchError("the reports share no concurrency level")
    return rows


def _metric(level: dict[str, Any], metric: str, statistic: str | None) -> float | None:
    value = level.get(metric)
    if statistic is None:
        return value if isinstance(value, int | float) else None
    if not isinstance(value, dict):
        return None
    inner = value.get(statistic)
    return inner if isinstance(inner, int | float) else None


def render_comparison(rows: Sequence[dict[str, Any]]) -> str:
    header = f"{'conc':>5}  {'metric':<24} {'baseline':>12} {'candidate':>12} {'change':>9}"
    lines = [header, "-" * len(header)]
    for row in rows:
        change = row["change"]
        verdict = ""
        if change is not None and change != 0:
            improved = (change < 0) == (row["better"] == "lower")
            verdict = " better" if improved else " worse"
        lines.append(
            f"{row['concurrency']:>5}  {row['metric']:<24} {_cell(row['baseline']):>12} "
            f"{_cell(row['candidate']):>12} {_percent(change):>9}{verdict}"
        )
    return "\n".join(lines)


def _cell(value: float | None) -> str:
    return "-" if value is None else f"{value:,.1f}"


def _percent(change: float | None) -> str:
    return "-" if change is None else f"{change * 100:+.1f}%"
