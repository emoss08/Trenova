"""Command line for the extraction fine-tuning pipeline."""

from __future__ import annotations

import argparse
import json
import logging
import os
import sys
from pathlib import Path

from . import __version__, dataset
from .bench import BenchError
from .config import PRODUCTION_MAX_TOKENS, ConfigError, load_config
from .lengths import LengthBudgetError
from .models import HardwareError
from .run import RunError
from .serve import API_KEY_ENV, ServeError

USER_ERRORS = (
    ConfigError,
    dataset.DatasetError,
    RunError,
    LengthBudgetError,
    HardwareError,
    BenchError,
    ServeError,
    FileExistsError,
)
DEFAULT_CONCURRENCY = "1,4,16"
EXIT_USER_ERROR = 2


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="trenova-finetune",
        description="Fine-tune an open model for Trenova's document extraction.",
    )
    parser.add_argument("--version", action="version", version=__version__)
    parser.add_argument("--verbose", action="store_true", help="log debug detail")
    commands = parser.add_subparsers(dest="command", required=True)

    verify = commands.add_parser("verify", help="check a rendered dataset against its manifest")
    verify.add_argument("--dataset", type=Path, required=True)

    build = commands.add_parser("targets", help="build SFT and preference data from a recipe")
    build.add_argument("--config", type=Path, required=True)
    build.add_argument("--dataset", type=Path, required=True)
    build.add_argument("--out", type=Path, required=True)

    run = commands.add_parser("run", help="train, merge and predict in one resumable run")
    run.add_argument("--config", type=Path, required=True)
    run.add_argument("--dataset", type=Path, required=True)
    run.add_argument("--out", type=Path, required=True)
    run.add_argument("--resume", action="store_true", help="continue a run that stopped")
    run.add_argument("--skip-predict", action="store_true", help="stop after the final merge")

    merge = commands.add_parser("merge", help="fold a LoRA adapter into its base model")
    merge.add_argument("--config", type=Path, required=True)
    merge.add_argument("--base", required=True, help="a Hugging Face id or a local model")
    merge.add_argument("--adapter", type=Path, required=True)
    merge.add_argument("--out", type=Path, required=True)
    merge.add_argument("--revision")

    predict = commands.add_parser("predict", help="answer the validation prompts with vLLM")
    predict.add_argument("--config", type=Path, required=True)
    predict.add_argument("--dataset", type=Path, required=True)
    predict.add_argument("--model", type=Path, required=True)
    predict.add_argument("--out", type=Path, required=True)

    serve = commands.add_parser(
        "serve",
        help="serve a model with vLLM using the config's serve section",
        description="Arguments after -- are passed to vllm serve unchanged.",
    )
    serve.add_argument("--config", type=Path, required=True)
    serve.add_argument("--model", type=Path, required=True)
    serve.add_argument("--name", required=True, help="the model name Trenova's provider sends")
    serve.add_argument("--dry-run", action="store_true", help="print the command, do not run it")
    serve.add_argument("vllm_args", nargs=argparse.REMAINDER)

    bench = commands.add_parser("bench", help="time extraction requests against a served model")
    bench.add_argument("--dataset", type=Path, required=True)
    bench.add_argument("--base-url", required=True, help="e.g. http://gpu-1:8000/v1")
    bench.add_argument("--model", required=True, help="the served model name")
    bench.add_argument(
        "--api-key-env",
        default=API_KEY_ENV,
        help="environment variable holding the API key (default: %(default)s)",
    )
    bench.add_argument(
        "--concurrency",
        type=_levels,
        default=_levels(DEFAULT_CONCURRENCY),
        help="comma-separated requests in flight per level (default: %(default)s)",
    )
    bench.add_argument("--requests", type=int, help="requests per level (default: every prompt)")
    bench.add_argument("--warmup", type=int, default=2, help="untimed requests sent first")
    bench.add_argument(
        "--max-tokens",
        type=int,
        default=PRODUCTION_MAX_TOKENS,
        help="match Trenova's ai.extractionMaxTokens (default: %(default)s)",
    )
    bench.add_argument("--timeout", type=float, default=300.0, help="seconds per request")
    bench.add_argument("--label", default="", help="names the serving setup in the report")
    bench.add_argument("--out", type=Path, required=True, help="where to write the JSON report")
    bench.add_argument(
        "--predictions-dir",
        type=Path,
        help="also write each level's replies for trenova ai fine-tune score",
    )

    compare = commands.add_parser("bench-compare", help="compare two benchmark reports")
    compare.add_argument("--baseline", type=Path, required=True)
    compare.add_argument("--candidate", type=Path, required=True)
    compare.add_argument("--json", action="store_true", help="print the rows as JSON")

    return parser


def _levels(value: str) -> tuple[int, ...]:
    try:
        levels = tuple(int(part) for part in value.split(",") if part.strip())
    except ValueError as err:
        raise argparse.ArgumentTypeError(f"{value!r} is not a list of numbers") from err
    if not levels:
        raise argparse.ArgumentTypeError("give at least one concurrency level")
    return levels


def _verify(args: argparse.Namespace) -> int:
    manifest = dataset.load_manifest(args.dataset)
    dataset.verify(args.dataset, manifest)
    print(json.dumps({"exportId": manifest.export_id, "counts": manifest.counts}, indent=2))
    return 0


def _targets(args: argparse.Namespace) -> int:
    from .targets import build

    config = load_config(args.config)
    manifest = dataset.load_manifest(args.dataset)
    dataset.verify(args.dataset, manifest)
    data = build(args.dataset, args.out, manifest, config.targets)
    print(json.dumps({"out": str(data.directory), "counts": data.counts}, indent=2))
    return 0


def _run(args: argparse.Namespace) -> int:
    from .pipeline import default_stages, execute, open_run

    config = load_config(args.config)
    run, manifest = open_run(config, args.dataset, args.out, resume=args.resume)
    final = execute(
        config,
        args.dataset,
        run,
        manifest,
        default_stages(),
        skip_predict=args.skip_predict,
    )
    print(f"Model: {final}")
    if not args.skip_predict:
        print(
            "Score it with: trenova ai fine-tune score "
            f"--eval {args.dataset / dataset.EVALUATION_FILE} "
            f"--predictions {run.path('predictions.jsonl')}"
        )
    return 0


def _merge(args: argparse.Namespace) -> int:
    from .merge import merge_adapter

    config = load_config(args.config)
    output = merge_adapter(config, args.base, args.adapter, args.out, revision=args.revision)
    print(f"Merged model: {output}")
    return 0


def _predict(args: argparse.Namespace) -> int:
    from .predict import predict

    config = load_config(args.config)
    manifest = dataset.load_manifest(args.dataset)
    dataset.verify(args.dataset, manifest)
    metrics = predict(config, args.model, args.dataset, args.out)
    print(json.dumps(metrics, indent=2))
    return 0


def _serve(args: argparse.Namespace) -> int:
    from .serve import serve

    extra = args.vllm_args[1:] if args.vllm_args[:1] == ["--"] else args.vllm_args
    command = serve(load_config(args.config), args.model, args.name, extra, dry_run=args.dry_run)
    print(command)
    return 0


def _bench(args: argparse.Namespace) -> int:
    from .bench import BenchPlan, Client, Endpoint, Workload, bench
    from .files import write_json

    endpoint = Endpoint(
        base_url=args.base_url,
        model=args.model,
        api_key=os.environ.get(args.api_key_env) or None,
        timeout=args.timeout,
    )
    plan = BenchPlan(
        concurrency=args.concurrency,
        requests=args.requests,
        warmup=args.warmup,
        label=args.label,
        predictions_dir=args.predictions_dir,
    )
    workload = Workload.load(args.dataset, args.max_tokens)
    report = bench(
        workload,
        Client(endpoint),
        endpoint,
        plan,
        progress=lambda message: print(message, file=sys.stderr),
    )
    write_json(args.out, report)
    print(json.dumps(report["levels"], indent=2))
    return 0


def _bench_compare(args: argparse.Namespace) -> int:
    from .bench import compare, load_report, render_comparison

    rows = compare(load_report(args.baseline), load_report(args.candidate))
    print(json.dumps(rows, indent=2) if args.json else render_comparison(rows))
    return 0


HANDLERS = {
    "verify": _verify,
    "targets": _targets,
    "run": _run,
    "merge": _merge,
    "predict": _predict,
    "serve": _serve,
    "bench": _bench,
    "bench-compare": _bench_compare,
}


def main(argv: list[str] | None = None) -> int:
    args = build_parser().parse_args(argv)
    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )
    try:
        return HANDLERS[args.command](args)
    except USER_ERRORS as err:
        print(f"error: {err}", file=sys.stderr)
        return EXIT_USER_ERROR


if __name__ == "__main__":
    sys.exit(main())
