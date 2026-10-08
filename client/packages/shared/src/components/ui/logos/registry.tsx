import type { ComponentType } from "react";
import { AnthropicLogo } from "./anthropic";
import { AWSLogo } from "./aws";
import { AzureLogo } from "./azure";
import { BedrockLogo } from "./bedrock";
import { DeepInfraLogo } from "./deepinfra";
import { DeepSeekLogo } from "./deepseek";
import { ExaLogo } from "./exa";
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
import { VoyageLogo } from "./voyage";

export type BrandMark = ComponentType<{ className?: string }>;

/**
 * Marks we ship ourselves, keyed by the provider preset they belong to. Bundling
 * them means a logo renders with no API key, no network request and no layout
 * shift, which the Brandfetch CDN cannot promise.
 *
 * Each keeps its vendor's own colours, and is used to name that vendor in a
 * list of endpoints and nothing else. A vendor whose mark is black (Anthropic,
 * OpenAI, OpenRouter, Ollama, Groq, LM Studio) draws it in the text colour, so it stays
 * legible in either theme. The coloured marks are taken from LobeHub's icon
 * set (@lobehub/icons-static-svg 1.95.1, MIT, Copyright (c) LobeHub).
 */
const MARKS_BY_PRESET: Record<string, BrandMark> = {
  anthropic: AnthropicLogo,
  openai: OpenAILogo,
  openrouter: OpenRouterLogo,
  bedrock: BedrockLogo,
  deepinfra: DeepInfraLogo,
  voyage: VoyageLogo,
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
  "azure.com": AzureLogo,
  "azure.microsoft.com": AzureLogo,
  "deepinfra.com": DeepInfraLogo,
  "voyageai.com": VoyageLogo,
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
  "exa.ai": ExaLogo,
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
