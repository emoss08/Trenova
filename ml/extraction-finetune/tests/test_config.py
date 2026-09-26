from pathlib import Path

import pytest

from trenova_finetune.config import ConfigError, PipelineConfig, load_config

CONFIGS = Path(__file__).resolve().parent.parent / "configs"


def test_shipped_configs_are_valid() -> None:
    names = sorted(path.name for path in CONFIGS.glob("*.yaml"))
    assert names
    for path in CONFIGS.glob("*.yaml"):
        config = load_config(path)
        assert config.base_model


def test_defaults_fill_every_section(config_path: Path) -> None:
    config = load_config(config_path)
    assert config.sft.max_length == 4096
    assert config.lora.r == 16
    assert config.dpo.enabled
    assert config.predict.max_model_len == 8192


@pytest.mark.parametrize(
    ("text", "message"),
    [
        ("base_model: x\nunknown: 1\n", "unknown"),
        ("base_model: x\nquantization: 8bit\n", "quantization"),
        ("base_model: x\ndpo:\n  beta: 0\n", "beta"),
        ("base_model: x\nsft:\n  max_length: 8192\npredict:\n  max_model_len: 4096\n", "max_model"),
        ("- not a mapping\n", "mapping"),
        ("base_model: [unclosed\n", "YAML"),
    ],
)
def test_invalid_configs_are_rejected(tmp_path: Path, text: str, message: str) -> None:
    path = tmp_path / "bad.yaml"
    path.write_text(text)
    with pytest.raises(ConfigError, match=message):
        load_config(path)


def test_missing_config_is_a_config_error(tmp_path: Path) -> None:
    with pytest.raises(ConfigError, match="cannot read"):
        load_config(tmp_path / "missing.yaml")


def test_config_is_immutable() -> None:
    config = PipelineConfig(base_model="x")
    with pytest.raises(Exception):  # noqa: B017
        config.base_model = "y"  # type: ignore[misc]
