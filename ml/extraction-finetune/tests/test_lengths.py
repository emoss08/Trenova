import pytest

from trenova_finetune.lengths import LengthBudgetError, enforce_budget, partition_by_length


def test_long_examples_are_dropped_not_truncated() -> None:
    partition = partition_by_length(["aa", "aaaa", "a", "aaaaaa"], len, 4)
    assert partition.kept == ["aa", "aaaa", "a"]
    assert partition.dropped == 1
    assert partition.longest == 6
    assert partition.dropped_fraction == pytest.approx(0.25)


def test_the_budget_stops_a_run_that_drops_too_much() -> None:
    partition = partition_by_length(["aaaa", "aaaa", "a"], len, 2)
    with pytest.raises(LengthBudgetError, match="longer than max_length"):
        enforce_budget(partition, 0.5, "training")
    enforce_budget(partition, 0.7, "training")


def test_nothing_fitting_is_an_error() -> None:
    with pytest.raises(LengthBudgetError, match="no training example"):
        enforce_budget(partition_by_length(["aaa"], len, 1), 1.0, "training")
    with pytest.raises(LengthBudgetError, match="no training example"):
        enforce_budget(partition_by_length([], len, 1), 1.0, "training")
