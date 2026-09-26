"""Command line for the extraction fine-tuning pipeline."""

from __future__ import annotations

import argparse
import json
import logging
import sys
from pathlib import Path

from . import __version__, dataset
from .config import ConfigError, load_config
from .lengths import LengthBudgetError
from .models import HardwareError
from .run import RunError

USER_ERRORS = (
    ConfigError,
    dataset.DatasetError,
    RunError,
    LengthBudgetError,
    HardwareError,
    FileExistsError,
)
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

    return parser


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


HANDLERS = {
    "verify": _verify,
    "targets": _targets,
    "run": _run,
    "merge": _merge,
    "predict": _predict,
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
