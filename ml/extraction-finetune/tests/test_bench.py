import json
import threading
from collections.abc import Iterator
from dataclasses import dataclass, field
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import pytest

from trenova_finetune import cli
from trenova_finetune.bench import (
    BENCH_FORMAT,
    BenchError,
    BenchPlan,
    Client,
    Endpoint,
    Level,
    ReplyChecker,
    Sample,
    Workload,
    bench,
    compare,
    distribution,
    plan_requests,
    render_comparison,
    run_level,
    summarize,
)
from trenova_finetune.predict import ERROR_EMPTY, ERROR_TRUNCATED

from .conftest import write_dataset

REPLY = json.dumps(
    {
        "documentKind": "RateConfirmation",
        "overallConfidence": 0.9,
        "reviewStatus": "Ready",
        "missingFields": [],
        "signals": [],
        "fields": [],
        "stops": [],
        "conflicts": [],
    }
)
API_KEY = "sk-bench-secret"


@dataclass
class Behaviour:
    status: int = 200
    pieces: tuple[str, ...] = (REPLY[:20], REPLY[20:])
    finish_reason: str = "stop"
    usage: dict = field(
        default_factory=lambda: {
            "prompt_tokens": 1000,
            "completion_tokens": 41,
            "prompt_tokens_details": {"cached_tokens": 800},
        }
    )
    stream_error: str | None = None
    requests: list[dict] = field(default_factory=list)
    headers: list[dict] = field(default_factory=list)


class _Handler(BaseHTTPRequestHandler):
    behaviour: Behaviour

    def log_message(self, *_: object) -> None:
        return

    def do_POST(self) -> None:
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        self.behaviour.requests.append(body)
        self.behaviour.headers.append(dict(self.headers))
        if self.path != "/v1/chat/completions":
            self._fail(404, "no route")
            return
        if self.behaviour.status != 200:
            self._fail(self.behaviour.status, "prompt too long for max_model_len")
            return
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()
        self._event({"choices": [{"index": 0, "delta": {"role": "assistant"}}]})
        for piece in self.behaviour.pieces:
            self._event({"choices": [{"index": 0, "delta": {"content": piece}}]})
        if self.behaviour.stream_error:
            self._event({"error": {"message": self.behaviour.stream_error}})
            return
        self._event(
            {"choices": [{"index": 0, "delta": {}, "finish_reason": self.behaviour.finish_reason}]}
        )
        self._event({"choices": [], "usage": self.behaviour.usage})
        self.wfile.write(b"data: [DONE]\n\n")

    def _event(self, payload: dict) -> None:
        self.wfile.write(b"data: " + json.dumps(payload).encode() + b"\n\n")
        self.wfile.flush()

    def _fail(self, status: int, message: str) -> None:
        payload = json.dumps({"error": {"message": message}}).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)


@pytest.fixture
def server() -> Iterator[tuple[str, Behaviour]]:
    behaviour = Behaviour()
    handler = type("Handler", (_Handler,), {"behaviour": behaviour})
    httpd = ThreadingHTTPServer(("127.0.0.1", 0), handler)
    thread = threading.Thread(target=httpd.serve_forever, daemon=True)
    thread.start()
    try:
        yield f"http://127.0.0.1:{httpd.server_address[1]}/v1", behaviour
    finally:
        httpd.shutdown()
        httpd.server_close()


def _endpoint(base_url: str, api_key: str | None = API_KEY) -> Endpoint:
    return Endpoint(base_url=base_url, model="trenova-extract", api_key=api_key, timeout=10)


def test_requests_are_the_call_trenova_makes(dataset_dir: Path) -> None:
    workload = Workload.load(dataset_dir, 5000)
    body = json.loads(workload.body("trenova-extract", workload.prompts[0][1]))
    assert body["model"] == "trenova-extract"
    assert body["messages"] == workload.prompts[0][1]
    assert body["max_tokens"] == 5000
    assert body["temperature"] == pytest.approx(0.1)
    assert body["top_p"] == pytest.approx(0.95)
    assert body["stream"] is True
    assert body["stream_options"] == {"include_usage": True}
    assert body["response_format"] == {
        "type": "json_schema",
        "json_schema": {
            "name": "rate_confirmation_extract",
            "schema": workload.schema,
            "strict": True,
        },
    }


@pytest.mark.parametrize(
    ("mode", "expected"),
    [("JSONMode", {"type": "json_object"}), ("Prompted", None)],
)
def test_response_format_follows_the_structured_output_mode(
    tmp_path: Path, mode: str, expected: dict | None
) -> None:
    root = write_dataset(tmp_path / "dataset")
    manifest = json.loads((root / "dataset-manifest.json").read_text())
    manifest["structuredOutputMode"] = mode
    (root / "dataset-manifest.json").write_text(json.dumps(manifest))
    workload = Workload.load(root, 5000)
    body = json.loads(workload.body("m", workload.prompts[0][1]))
    assert body.get("response_format") == expected


def test_endpoints_refuse_urls_that_would_leak_or_mislead() -> None:
    with pytest.raises(BenchError, match="http"):
        Endpoint(base_url="ftp://gpu-1/v1", model="m")
    with pytest.raises(BenchError, match="API key"):
        Endpoint(base_url="http://user:key@gpu-1/v1", model="m")
    with pytest.raises(BenchError, match="query"):
        Endpoint(base_url="http://gpu-1/v1?key=1", model="m")
    with pytest.raises(BenchError, match="model"):
        Endpoint(base_url="http://gpu-1/v1", model=" ")
    assert API_KEY not in repr(_endpoint("http://gpu-1/v1"))
    assert _endpoint("http://gpu-1/v1/").url == "http://gpu-1/v1/chat/completions"


def test_a_streamed_reply_is_timed_and_counted(server: tuple[str, Behaviour]) -> None:
    base_url, behaviour = server
    sample = Client(_endpoint(base_url)).send("v0", b'{"model": "m"}')
    assert sample.error is None
    assert sample.reply == REPLY
    assert sample.finish_reason == "stop"
    assert (sample.prompt_tokens, sample.completion_tokens, sample.cached_tokens) == (1000, 41, 800)
    assert sample.ttft is not None and sample.e2e is not None
    assert 0 <= sample.ttft <= sample.e2e
    assert behaviour.headers[0]["Authorization"] == f"Bearer {API_KEY}"
    assert behaviour.headers[0]["Accept"] == "text/event-stream"


def test_no_key_sends_no_authorization(server: tuple[str, Behaviour]) -> None:
    base_url, behaviour = server
    Client(_endpoint(base_url, api_key=None)).send("v0", b"{}")
    assert "Authorization" not in behaviour.headers[0]


def test_time_to_first_token_is_taken_at_the_first_generated_text() -> None:
    ticks = iter([0.0, 0.25, 2.0])
    stream = [
        b'data: {"choices": [{"delta": {"role": "assistant"}}]}\n',
        b"\n",
        b'data: {"choices": [{"delta": {"content": "{"}}]}\n',
        b'data: {"choices": [{"delta": {"content": "}"}, "finish_reason": "stop"}]}\n',
        b'data: {"choices": [], "usage": {"prompt_tokens": 10, "completion_tokens": 5}}\n',
        b"data: [DONE]\n",
    ]

    class _Response:
        def __enter__(self) -> list[bytes]:
            return stream

        def __exit__(self, *_: object) -> None:
            return None

    client = Client(
        _endpoint("http://gpu-1/v1"),
        opener=lambda _request, _timeout: _Response(),
        clock=lambda: next(ticks),
    )
    sample = client.send("v0", b"{}")
    assert sample.ttft == pytest.approx(0.25)
    assert sample.e2e == pytest.approx(2.0)
    assert sample.tpot == pytest.approx((2.0 - 0.25) / 4)
    assert sample.reply == "{}"


def test_http_errors_are_recorded_not_raised(server: tuple[str, Behaviour]) -> None:
    base_url, behaviour = server
    behaviour.status = 400
    sample = Client(_endpoint(base_url)).send("v0", b"{}")
    assert sample.error == "HTTP 400: prompt too long for max_model_len"
    assert sample.prediction()["error"] == sample.error


def test_errors_inside_the_stream_are_recorded(server: tuple[str, Behaviour]) -> None:
    base_url, behaviour = server
    behaviour.stream_error = "engine died"
    sample = Client(_endpoint(base_url)).send("v0", b"{}")
    assert sample.error == "stream error: engine died"


def test_unreachable_servers_are_recorded() -> None:
    sample = Client(_endpoint("http://127.0.0.1:9/v1")).send("v0", b"{}")
    assert sample.error is not None and sample.error.startswith("request failed")


def test_predictions_flag_what_the_scorer_cannot_use() -> None:
    assert Sample(id="a", reply=f" {REPLY} ", finish_reason="stop").prediction() == {
        "id": "a",
        "reply": REPLY,
    }
    assert Sample(id="a", reply="{", finish_reason="length").prediction()["error"] == (
        ERROR_TRUNCATED
    )
    assert Sample(id="a", reply="", finish_reason="stop").prediction()["error"] == ERROR_EMPTY


def test_replies_are_checked_against_the_schema(dataset_dir: Path) -> None:
    checker = ReplyChecker(Workload.load(dataset_dir, 5000).schema)
    assert checker.valid(REPLY)
    assert not checker.valid('{"fields": []}')
    assert not checker.valid("not json")


def test_distributions_interpolate_percentiles_in_milliseconds() -> None:
    summary = distribution([0.1, 0.2, 0.3, 0.4, None])
    assert summary == {
        "mean": pytest.approx(250.0),
        "p50": pytest.approx(250.0),
        "p90": pytest.approx(370.0),
        "p99": pytest.approx(397.0),
        "max": pytest.approx(400.0),
    }
    assert distribution([None]) is None
    assert distribution([0.5])["p99"] == pytest.approx(500.0)


def test_levels_summarize_speed_and_failures() -> None:
    samples = (
        Sample(
            id="a",
            finish_reason="stop",
            ttft=0.1,
            e2e=1.1,
            prompt_tokens=100,
            completion_tokens=11,
            cached_tokens=50,
            valid=True,
        ),
        Sample(
            id="b",
            finish_reason="length",
            ttft=0.3,
            e2e=2.3,
            prompt_tokens=100,
            completion_tokens=21,
            cached_tokens=0,
        ),
        Sample(id="c", error="HTTP 500: boom", e2e=0.01),
    )
    summary = summarize(Level(concurrency=4, samples=samples, wall_seconds=2.0))
    assert summary["requests"] == 3
    assert summary["succeeded"] == 2
    assert summary["failed"] == 1
    assert summary["truncated"] == 1
    assert summary["invalidReplies"] == 1
    assert summary["requestsPerSecond"] == pytest.approx(1.0)
    assert summary["outputTokensPerSecond"] == pytest.approx(16.0)
    assert summary["cachedPromptFraction"] == pytest.approx(0.25)
    assert summary["tpotMs"]["p50"] == pytest.approx(100.0)
    assert summary["e2eMs"]["max"] == pytest.approx(2300.0)
    assert summary["errors"] == [{"error": "HTTP 500: boom", "count": 1}]


def test_cached_fraction_is_unknown_when_the_server_does_not_report_it() -> None:
    samples = (Sample(id="a", prompt_tokens=10, completion_tokens=2, ttft=0.1, e2e=0.2),)
    assert summarize(Level(1, samples, 1.0))["cachedPromptFraction"] is None


def test_requests_cycle_through_the_prompts(dataset_dir: Path) -> None:
    workload = Workload.load(dataset_dir, 5000)
    assert [identifier for identifier, _ in plan_requests(workload, "m", 5)] == [
        "v0",
        "v1",
        "v0",
        "v1",
        "v0",
    ]
    with pytest.raises(BenchError):
        plan_requests(workload, "m", 0)


@pytest.mark.parametrize(
    "arguments",
    [
        {"concurrency": ()},
        {"concurrency": (0,)},
        {"concurrency": (4, 4)},
        {"concurrency": (1,), "requests": 0},
        {"concurrency": (1,), "warmup": -1},
    ],
)
def test_bench_plans_are_validated(arguments: dict) -> None:
    with pytest.raises(BenchError):
        BenchPlan(**arguments)


def test_bench_runs_every_level_and_writes_scorable_replies(
    tmp_path: Path, dataset_dir: Path, server: tuple[str, Behaviour]
) -> None:
    base_url, behaviour = server
    endpoint = _endpoint(base_url)
    plan = BenchPlan(
        concurrency=(1, 2), requests=3, warmup=1, label="fp8", predictions_dir=tmp_path / "p"
    )
    report = bench(Workload.load(dataset_dir, 5000), Client(endpoint), endpoint, plan)

    assert report["format"] == BENCH_FORMAT
    assert report["label"] == "fp8"
    assert report["endpoint"] == {"baseUrl": base_url, "model": "trenova-extract"}
    assert report["dataset"]["exportId"] == "aitx_1"
    assert report["request"] == {"maxTokens": 5000, "temperature": 0.1, "topP": 0.95}
    assert report["requestsPerLevel"] == 3
    assert report["promptsRepeat"] is True
    assert [level["concurrency"] for level in report["levels"]] == [1, 2]
    assert all(
        level["succeeded"] == 3 and level["invalidReplies"] == 0 for level in report["levels"]
    )
    assert len(behaviour.requests) == 1 + 3 + 3
    assert API_KEY not in json.dumps(report)

    lines = (tmp_path / "p" / "predictions-c2.jsonl").read_text().splitlines()
    assert [json.loads(line) for line in lines] == [
        {"id": "v0", "reply": REPLY},
        {"id": "v1", "reply": REPLY},
    ]


def test_bench_stops_when_the_server_cannot_be_reached(
    dataset_dir: Path, server: tuple[str, Behaviour]
) -> None:
    base_url, behaviour = server
    behaviour.status = 401
    endpoint = _endpoint(base_url)
    with pytest.raises(BenchError, match="warmup"):
        bench(Workload.load(dataset_dir, 5000), Client(endpoint), endpoint, BenchPlan((1,)))


def _report(p50: float, tokens_per_second: float, **overrides: object) -> dict:
    report = {
        "format": BENCH_FORMAT,
        "dataset": {
            "promptSha256": "p",
            "datasetManifestSha256": "d",
            "structuredOutputMode": "JSONSchema",
        },
        "request": {"maxTokens": 5000, "temperature": 0.1, "topP": 0.95},
        "levels": [
            {
                "concurrency": 1,
                "e2eMs": {"p50": p50, "p90": p50 * 2},
                "ttftMs": {"p50": 100.0},
                "tpotMs": {"p50": 20.0},
                "outputTokensPerSecond": tokens_per_second,
                "requestsPerSecond": 1.0,
            },
            {"concurrency": 64, "e2eMs": None},
        ],
    }
    report.update(overrides)
    return report


def test_comparisons_line_up_matching_levels() -> None:
    rows = compare(_report(1000.0, 50.0), _report(800.0, 75.0))
    by_metric = {row["metric"]: row for row in rows if row["concurrency"] == 1}
    assert by_metric["e2eMs.p50"]["change"] == pytest.approx(-0.2)
    assert by_metric["outputTokensPerSecond"]["change"] == pytest.approx(0.5)
    assert by_metric["ttftMs.p50"]["change"] == 0
    assert {row["candidate"] for row in rows if row["concurrency"] == 64} == {None}

    table = render_comparison(rows)
    assert "e2eMs.p50" in table
    assert "-20.0% better" in table
    assert "+50.0% better" in table


def test_comparisons_refuse_different_workloads() -> None:
    other = _report(800.0, 75.0, request={"maxTokens": 2048, "temperature": 0.1, "topP": 0.95})
    with pytest.raises(BenchError, match="maxTokens"):
        compare(_report(1000.0, 50.0), other)
    with pytest.raises(BenchError, match="no concurrency level"):
        compare(_report(1000.0, 50.0), _report(800.0, 75.0, levels=[{"concurrency": 8}]))


def test_the_bench_command_writes_a_report(
    tmp_path: Path,
    dataset_dir: Path,
    server: tuple[str, Behaviour],
    monkeypatch: pytest.MonkeyPatch,
    capsys: pytest.CaptureFixture[str],
) -> None:
    base_url, behaviour = server
    monkeypatch.setenv("BENCH_TEST_KEY", API_KEY)
    out = tmp_path / "reports" / "base.json"
    code = cli.main(
        [
            "bench",
            "--dataset",
            str(dataset_dir),
            "--base-url",
            base_url,
            "--model",
            "trenova-extract",
            "--api-key-env",
            "BENCH_TEST_KEY",
            "--concurrency",
            "1,2",
            "--warmup",
            "0",
            "--out",
            str(out),
        ]
    )
    assert code == 0
    report = json.loads(out.read_text())
    assert [level["concurrency"] for level in report["levels"]] == [1, 2]
    assert behaviour.headers[0]["Authorization"] == f"Bearer {API_KEY}"
    assert API_KEY not in out.read_text()
    assert json.loads(capsys.readouterr().out)[0]["succeeded"] == 2

    assert cli.main(["bench-compare", "--baseline", str(out), "--candidate", str(out)]) == 0
    assert "e2eMs.p50" in capsys.readouterr().out


def test_the_bench_command_rejects_bad_levels(dataset_dir: Path) -> None:
    with pytest.raises(SystemExit):
        cli.main(
            [
                "bench",
                "--dataset",
                str(dataset_dir),
                "--base-url",
                "http://gpu-1/v1",
                "--model",
                "m",
                "--concurrency",
                "one",
                "--out",
                "x.json",
            ]
        )


def test_a_level_needs_requests_and_concurrency(dataset_dir: Path) -> None:
    client = Client(_endpoint("http://gpu-1/v1"))
    checker = ReplyChecker(Workload.load(dataset_dir, 5000).schema)
    with pytest.raises(BenchError, match="no requests"):
        run_level(client, checker, [], 1)
    with pytest.raises(BenchError, match="concurrency"):
        run_level(client, checker, [("v0", b"{}")], 0)
