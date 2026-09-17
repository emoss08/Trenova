import type { ComponentType } from "react";
import { AnthropicLogo } from "./anthropic";
import { AWSLogo } from "./aws";
import { DeepSeekLogo } from "./deepseek";
import { FireworksLogo } from "./fireworks";
import { GeminiLogo } from "./gemini";
import { GroqLogo } from "./groq";
import { HuggingFaceLogo } from "./huggingface";
import { LMStudioLogo } from "./lmstudio";
import { MetaLogo } from "./meta";
import { MistralLogo } from "./mistral";
import { OllamaLogo } from "./ollama";
import { OpenAILogo } from "./openai";
import { OpenRouterLogo } from "./openrouter";
import { TogetherLogo } from "./together";
import { VLLMLogo } from "./vllm";

export type BrandMark = ComponentType<{ className?: string }>;

/**
 * Marks we ship ourselves, keyed by the provider preset they belong to. Bundling
 * them means a logo renders with no API key, no network request and no layout
 * shift, which the Brandfetch CDN cannot promise.
 *
 * Each is redrawn flat in one colour from the vendor's own geometry, used to
 * name that vendor in a list of endpoints and nothing else.
 */
const MARKS_BY_PRESET: Record<string, BrandMark> = {
  anthropic: AnthropicLogo,
  openai: OpenAILogo,
  openrouter: OpenRouterLogo,
  bedrock: AWSLogo,
  ollama: OllamaLogo,
  vllm: VLLMLogo,
  lmstudio: LMStudioLogo,
  groq: GroqLogo,
  together: TogetherLogo,
  fireworks: FireworksLogo,
};

/**
 * The same marks keyed by vendor domain, for a saved provider whose endpoint we
 * recognise but whose preset we no longer know.
 */
const MARKS_BY_DOMAIN: Record<string, BrandMark> = {
  "anthropic.com": AnthropicLogo,
  "openai.com": OpenAILogo,
  "openrouter.ai": OpenRouterLogo,
  "aws.amazon.com": AWSLogo,
  "amazonaws.com": AWSLogo,
  "ollama.com": OllamaLogo,
  "vllm.ai": VLLMLogo,
  "lmstudio.ai": LMStudioLogo,
  "mistral.ai": MistralLogo,
  "deepseek.com": DeepSeekLogo,
  "huggingface.co": HuggingFaceLogo,
  "google.com": GeminiLogo,
  "ai.google.dev": GeminiLogo,
  "googleapis.com": GeminiLogo,
  "meta.com": MetaLogo,
  "llama.com": MetaLogo,
  "groq.com": GroqLogo,
  "together.ai": TogetherLogo,
  "together.xyz": TogetherLogo,
  "fireworks.ai": FireworksLogo,
};

export type BrandMarkLookup = {
  presetKey?: string | null;
  domain?: string | null;
};

/** The bundled mark for a provider, or null when we do not ship one. */
export function brandMarkFor({ presetKey, domain }: BrandMarkLookup): BrandMark | null {
  const key = presetKey?.trim().toLowerCase();
  if (key && MARKS_BY_PRESET[key]) {
    return MARKS_BY_PRESET[key];
  }

  const host = domain
    ?.trim()
    .toLowerCase()
    .replace(/^www\./, "");
  if (!host) {
    return null;
  }
  if (MARKS_BY_DOMAIN[host]) {
    return MARKS_BY_DOMAIN[host];
  }

  // A regional or sub-domain endpoint still belongs to its vendor.
  for (const [known, mark] of Object.entries(MARKS_BY_DOMAIN)) {
    if (host.endsWith(`.${known}`)) {
      return mark;
    }
  }

  return null;
}

export function hasBundledBrandMark(lookup: BrandMarkLookup): boolean {
  return brandMarkFor(lookup) !== null;
}
