-- Vendor request fields an OpenAI-compatible endpoint takes that the
-- protocol does not define: NVIDIA's NIM reads chat_template_kwargs and
-- reasoning_budget to steer a model's thinking and max_tokens where this
-- system sends max_completion_tokens, vLLM reads top_k, OpenRouter reads
-- provider routing. They are merged under the fields this system sets, so a
-- provider can configure its endpoint without redirecting the call.
ALTER TABLE "ai_providers"
    ADD COLUMN IF NOT EXISTS "extra_body" jsonb;
