import json
import shlex
from pathlib import Path

import pytest
from pydantic import ValidationError

from trenova_finetune import cli
from trenova_finetune.config import (
    PipelineConfig,
    ServeSettings,
    SpeculativeSettings,
    load_config,
)
from trenova_finetune.pipeline import CARD_FILE, CARD_FORMAT
from trenova_finetune.serve import (
    API_KEY_ENV,
    ServeError,
    read_card,
    serve,
    serve_arguments,
    serve_command,
)

CONFIGS = Path(__file__).resolve().parent.parent / "configs"


def _model(tmp_path: Path) -> Path:
    model = tmp_path / "model"
    model.mkdir()
    (model / CARD_FILE).write_text(json.dumps({"format": CARD_FORMAT}))
    return model


def _flag(args: list[str], name: str) -> str:
    return args[args.index(name) + 1]


def test_defaults_turn_on_prefix_caching_and_report_cached_tokens() -> None:
    args = serve_arguments(ServeSettings())
    assert "--enable-prefix-caching" in args
    assert "--enable-chunked-prefill" in args
    assert "--enable-prompt-tokens-details" in args
    assert _flag(args, "--kv-cache-dtype") == "auto"
    assert json.loads(_flag(args, "--structured-outputs-config")) == {"backend": "auto"}
    for absent in ("--quantization", "--speculative-config", "--max-num-seqs"):
        assert absent not in args


def test_every_tuning_setting_reaches_vllm() -> None:
    settings = ServeSettings(
        port=9000,
        enable_prefix_caching=False,
        enable_chunked_prefill=False,
        max_num_seqs=32,
        max_num_batched_tokens=16384,
        kv_cache_dtype="fp8",
        quantization="fp8",
        structured_outputs_backend="xgrammar",
        speculative=SpeculativeSettings(
            method="ngram", num_speculative_tokens=4, prompt_lookup_min=2, prompt_lookup_max=5
        ),
        report_cached_tokens=False,
    )
    args = serve_arguments(settings)
    assert _flag(args, "--port") == "9000"
    assert "--no-enable-prefix-caching" in args
    assert "--no-enable-chunked-prefill" in args
    assert _flag(args, "--max-num-seqs") == "32"
    assert _flag(args, "--max-num-batched-tokens") == "16384"
    assert _flag(args, "--kv-cache-dtype") == "fp8"
    assert _flag(args, "--quantization") == "fp8"
    assert json.loads(_flag(args, "--structured-outputs-config")) == {"backend": "xgrammar"}
    assert json.loads(_flag(args, "--speculative-config")) == {
        "method": "ngram",
        "num_speculative_tokens": 4,
        "prompt_lookup_min": 2,
        "prompt_lookup_max": 5,
    }
    assert "--enable-prompt-tokens-details" not in args


@pytest.mark.parametrize(
    ("arguments", "message"),
    [
        ({"method": "eagle3", "num_speculative_tokens": 3}, "draft model"),
        ({"method": "ngram", "num_speculative_tokens": 3}, "prompt_lookup"),
        (
            {"method": "ngram", "num_speculative_tokens": 3, "prompt_lookup_max": 4, "model": "x"},
            "drop model",
        ),
        (
            {
                "method": "draft_model",
                "num_speculative_tokens": 3,
                "model": "x",
                "prompt_lookup_max": 4,
            },
            "only apply to ngram",
        ),
        (
            {
                "method": "ngram",
                "num_speculative_tokens": 3,
                "prompt_lookup_min": 5,
                "prompt_lookup_max": 2,
            },
            "must not exceed",
        ),
        ({"method": "ngram", "num_speculative_tokens": 0, "prompt_lookup_max": 4}, "greater"),
    ],
)
def test_speculative_settings_must_fit_their_method(arguments: dict, message: str) -> None:
    with pytest.raises(ValidationError, match=message):
        SpeculativeSettings(**arguments)


def test_a_draft_model_is_named_in_the_speculative_config() -> None:
    settings = SpeculativeSettings(
        method="eagle3", num_speculative_tokens=3, model="org/qwen2.5-7b-eagle3"
    )
    assert settings.as_vllm() == {
        "method": "eagle3",
        "num_speculative_tokens": 3,
        "model": "org/qwen2.5-7b-eagle3",
    }


def test_without_chunked_prefill_a_batch_must_hold_a_whole_prompt() -> None:
    with pytest.raises(ValidationError, match="chunked prefill"):
        ServeSettings(enable_chunked_prefill=False, max_num_batched_tokens=4096)
    ServeSettings(enable_chunked_prefill=True, max_num_batched_tokens=4096)


def test_the_served_context_must_hold_a_training_example() -> None:
    with pytest.raises(ValidationError, match=r"serve\.max_model_len"):
        PipelineConfig(base_model="m", serve={"max_model_len": 4096})


@pytest.mark.parametrize("name", sorted(path.name for path in CONFIGS.glob("*.yaml")))
def test_shipped_configs_serve_with_prefix_caching(name: str) -> None:
    config = load_config(CONFIGS / name)
    assert config.serve.enable_prefix_caching
    assert config.serve.max_model_len >= config.sft.max_length


def test_the_command_names_the_model_and_keeps_the_key_off_it(
    tmp_path: Path, config_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv(API_KEY_ENV, "sk-serve-secret")
    config = load_config(config_path)
    command = serve_command(config, _model(tmp_path), "trenova-extract", ["--max-num-seqs", "8"])
    assert command[:3] == ["vllm", "serve", str(tmp_path / "model")]
    assert _flag(command, "--served-model-name") == "trenova-extract"
    assert _flag(command, "--generation-config") == "vllm"
    assert _flag(command, "--seed") == "42"
    assert command[-2:] == ["--max-num-seqs", "8"]
    assert "--trust-remote-code" not in command
    assert "sk-serve-secret" not in " ".join(command)


@pytest.mark.parametrize("extra", [["--api-key", "k"], ["--api-key=k"]])
def test_the_key_cannot_be_passed_on_the_command_line(
    tmp_path: Path, config_path: Path, extra: list[str]
) -> None:
    with pytest.raises(ServeError, match=API_KEY_ENV):
        serve_command(load_config(config_path), _model(tmp_path), "m", extra)


def test_a_blank_served_name_is_refused(tmp_path: Path, config_path: Path) -> None:
    with pytest.raises(ServeError, match="name"):
        serve_command(load_config(config_path), _model(tmp_path), " ")


def test_only_models_this_pipeline_produced_are_served(tmp_path: Path) -> None:
    with pytest.raises(ServeError, match=r"no trenova-model\.json"):
        read_card(tmp_path)
    (tmp_path / CARD_FILE).write_text(json.dumps({"format": "something-else"}))
    with pytest.raises(ServeError, match=CARD_FORMAT):
        read_card(tmp_path)
    (tmp_path / CARD_FILE).write_text("{")
    with pytest.raises(ServeError, match="cannot read"):
        read_card(tmp_path)


def test_serving_needs_the_api_key(
    tmp_path: Path, config_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.delenv(API_KEY_ENV, raising=False)
    with pytest.raises(ServeError, match=API_KEY_ENV):
        serve(load_config(config_path), _model(tmp_path), "m")


def test_serving_replaces_the_process_with_vllm(
    tmp_path: Path, config_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    calls: list[tuple[str, list[str]]] = []
    monkeypatch.setenv(API_KEY_ENV, "sk")
    monkeypatch.setattr("trenova_finetune.serve.shutil.which", lambda _name: "/opt/bin/vllm")
    monkeypatch.setattr(
        "trenova_finetune.serve.os.execv", lambda path, argv: calls.append((path, argv))
    )
    serve(load_config(config_path), _model(tmp_path), "m")
    assert calls[0][0] == "/opt/bin/vllm"
    assert calls[0][1][:2] == ["vllm", "serve"]


def test_serving_without_vllm_installed_says_so(
    tmp_path: Path, config_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    monkeypatch.setenv(API_KEY_ENV, "sk")
    monkeypatch.setattr("trenova_finetune.serve.shutil.which", lambda _name: None)
    with pytest.raises(ServeError, match="not installed"):
        serve(load_config(config_path), _model(tmp_path), "m")


def test_the_serve_command_prints_a_dry_run(
    tmp_path: Path, config_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    model = _model(tmp_path)
    code = cli.main(
        [
            "serve",
            "--config",
            str(config_path),
            "--model",
            str(model),
            "--name",
            "trenova-extract",
            "--dry-run",
            "--",
            "--max-num-seqs",
            "8",
        ]
    )
    assert code == 0
    printed = shlex.split(capsys.readouterr().out)
    assert printed[:3] == ["vllm", "serve", str(model)]
    assert printed[-2:] == ["--max-num-seqs", "8"]


def test_the_serve_command_reports_user_errors(
    tmp_path: Path, config_path: Path, capsys: pytest.CaptureFixture[str]
) -> None:
    code = cli.main(
        ["serve", "--config", str(config_path), "--model", str(tmp_path), "--name", "m"]
    )
    assert code == cli.EXIT_USER_ERROR
    assert "trenova-model.json" in capsys.readouterr().err
