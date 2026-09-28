from typing import Any

from trenova_finetune.models import conversation_tokens


class DictDefaultTokenizer:
    """Mimics transformers 5, whose apply_chat_template returns a dict unless told otherwise."""

    def apply_chat_template(self, messages: list, **kwargs: Any) -> Any:
        ids = list(range(sum(len(message["content"]) for message in messages)))
        if kwargs.get("return_dict", True):
            return {"input_ids": ids, "attention_mask": [1] * len(ids)}
        return ids


def test_token_counts_are_ids_not_dictionary_keys() -> None:
    messages = [{"role": "user", "content": "x" * 40}, {"role": "assistant", "content": "y" * 2}]
    assert conversation_tokens(DictDefaultTokenizer(), messages) == 42
