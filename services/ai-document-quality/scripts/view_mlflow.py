#!/usr/bin/env python3
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md#
"""
Script to view MLflow experiments

Usage:
    python view_mlflow.py                    # Launches MLflow UI
    python view_mlflow.py --list            # Lists experiments and runs
    python view_mlflow.py --best            # Shows best run details
"""

import argparse
import subprocess
from pathlib import Path

import mlflow
import pandas as pd


def launch_ui(tracking_uri="mlruns", port=5000):
    """Launch MLflow UI"""
    print(f"Launching MLflow UI at http://localhost:{port}")
    print("Press Ctrl+C to stop the server")
    subprocess.run(
        ["mlflow", "ui", "--backend-store-uri", tracking_uri, "--port", str(port)]
    )


def list_experiments(tracking_uri="mlruns"):
    """List all experiments and their runs"""
    mlflow.set_tracking_uri(tracking_uri)

    experiments = mlflow.search_experiments()

    for exp in experiments:
        print(f"\nExperiment: {exp.name} (ID: {exp.experiment_id})")
        print(f"  Artifact Location: {exp.artifact_location}")

        # Get runs for this experiment
        runs = mlflow.search_runs(experiment_ids=[exp.experiment_id])

        if isinstance(runs, pd.DataFrame) and not runs.empty:
            print(f"  Number of runs: {len(runs)}")
            print("\n  Recent runs:")
            # Only show columns that exist
            cols = ["run_id", "status", "start_time"]
            if "metrics.val_loss" in runs.columns:
                cols.append("metrics.val_loss")
            if "metrics.test_mae" in runs.columns:
                cols.append("metrics.test_mae")
            print(runs[cols].head())


def show_best_run(tracking_uri="mlruns", metric="val_loss", ascending=True):
    """Show details of the best run"""
    mlflow.set_tracking_uri(tracking_uri)

    # Search all runs
    runs = mlflow.search_runs(search_all_experiments=True)

    if isinstance(runs, pd.DataFrame) and runs.empty:
        print("No runs found!")
        return
    elif not isinstance(runs, pd.DataFrame):
        print("No runs found!")
        return

    # Find best run
    metric_col = f"metrics.{metric}"
    if metric_col not in runs.columns:
        print(f"Metric '{metric}' not found. Available metrics:")
        metric_cols = [col for col in runs.columns if col.startswith("metrics.")]
        for col in metric_cols:
            print(f"  - {col.replace('metrics.', '')}")
        return

    runs_with_metric = runs.dropna(subset=[metric_col])
    if runs_with_metric.empty:
        print(f"No runs with metric '{metric}' found!")
        return

    best_idx = (
        runs_with_metric[metric_col].idxmin()
        if ascending
        else runs_with_metric[metric_col].idxmax()
    )
    best_run = runs_with_metric.loc[best_idx]

    print(
        f"\nBest run based on {metric} ({'lower' if ascending else 'higher'} is better):"
    )
    print(f"  Run ID: {best_run['run_id']}")
    print(f"  {metric}: {best_run[metric_col]:.4f}")

    # Show all metrics
    print("\n  All metrics:")
    for col in runs.columns:
        if col.startswith("metrics."):
            value = best_run[col]
            if pd.notna(value):
                print(f"    {col.replace('metrics.', '')}: {value:.4f}")

    # Show parameters
    print("\n  Key parameters:")
    param_cols = [
        "params.backbone",
        "params.learning_rate",
        "params.batch_size",
        "params.hidden_dim",
        "params.dropout_rate",
    ]
    for col in param_cols:
        if col in runs.columns:
            value = best_run[col]
            if pd.notna(value):
                print(f"    {col.replace('params.', '')}: {value}")


def main():
    parser = argparse.ArgumentParser(description="View MLflow experiments")
    parser.add_argument("--tracking-uri", default="mlruns", help="MLflow tracking URI")
    parser.add_argument("--port", type=int, default=5000, help="Port for MLflow UI")
    parser.add_argument("--list", action="store_true", help="List experiments and runs")
    parser.add_argument("--best", action="store_true", help="Show best run details")
    parser.add_argument(
        "--metric", default="val_loss", help="Metric to use for best run"
    )
    parser.add_argument(
        "--maximize", action="store_true", help="Maximize metric instead of minimize"
    )

    args = parser.parse_args()

    if args.list:
        list_experiments(args.tracking_uri)
    elif args.best:
        show_best_run(args.tracking_uri, args.metric, not args.maximize)
    else:
        launch_ui(args.tracking_uri, args.port)


if __name__ == "__main__":
    main()
