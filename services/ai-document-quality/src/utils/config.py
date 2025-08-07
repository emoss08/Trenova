#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""Configuration management utilities"""

import os
from pathlib import Path
from typing import Any, Dict, Optional

import yaml


class Config:
    """Configuration manager for the Document Quality Assessment System"""

    def __init__(self, config_path: Optional[str] = None):
        """
        Initialize configuration

        Args:
            config_path: Path to configuration file. If None, uses default.
        """
        self.config_path = config_path or self._get_default_config_path()
        self.config = self._load_config()

    def _get_default_config_path(self) -> str:
        """Get path to default configuration file"""
        # Look for config in multiple locations
        possible_paths = [
            Path("config/default.yaml"),
            Path(__file__).parent.parent.parent / "config" / "default.yaml",
            Path.home() / ".document_quality" / "config.yaml",
        ]

        for path in possible_paths:
            if path.exists():
                return str(path)

        raise FileNotFoundError(
            "No configuration file found. Please create config/default.yaml"
        )

    def _load_config(self) -> Dict[str, Any]:
        """Load configuration from YAML file"""
        with open(self.config_path, "r") as f:
            config = yaml.safe_load(f)

        # Override with environment variables if present
        config = self._override_with_env(config)

        return config

    def _override_with_env(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Override configuration with environment variables"""
        # Model settings
        if os.getenv("DOC_QUALITY_MODEL_BACKBONE"):
            config["model"]["backbone"] = os.getenv("DOC_QUALITY_MODEL_BACKBONE")

        # API settings
        if os.getenv("DOC_QUALITY_API_HOST"):
            config["api"]["host"] = os.getenv("DOC_QUALITY_API_HOST")
        if os.getenv("DOC_QUALITY_API_PORT"):
            config["api"]["port"] = int(os.getenv("DOC_QUALITY_API_PORT"))

        # Paths
        if os.getenv("DOC_QUALITY_MODELS_DIR"):
            config["paths"]["models_dir"] = os.getenv("DOC_QUALITY_MODELS_DIR")
        if os.getenv("DOC_QUALITY_DATASETS_DIR"):
            config["paths"]["datasets_dir"] = os.getenv("DOC_QUALITY_DATASETS_DIR")

        return config

    def get(self, key: str, default: Any = None) -> Any:
        """
        Get configuration value by dot-separated key

        Args:
            key: Dot-separated configuration key (e.g., 'model.backbone')
            default: Default value if key not found

        Returns:
            Configuration value
        """
        keys = key.split(".")
        value = self.config

        for k in keys:
            if isinstance(value, dict) and k in value:
                value = value[k]
            else:
                return default

        return value

    def set(self, key: str, value: Any):
        """
        Set configuration value

        Args:
            key: Dot-separated configuration key
            value: Value to set
        """
        keys = key.split(".")
        config = self.config

        for k in keys[:-1]:
            if k not in config:
                config[k] = {}
            config = config[k]

        config[keys[-1]] = value

    def save(self, path: Optional[str] = None):
        """Save configuration to file"""
        save_path = path or self.config_path
        with open(save_path, "w") as f:
            yaml.dump(self.config, f, default_flow_style=False)

    def to_dict(self) -> Dict[str, Any]:
        """Return configuration as dictionary"""
        return self.config.copy()


# Global configuration instance
_config = None


def get_config(config_path: Optional[str] = None) -> Config:
    """Get global configuration instance"""
    global _config

    if _config is None:
        _config = Config(config_path)

    return _config


def reset_config():
    """Reset global configuration instance"""
    global _config
    _config = None
