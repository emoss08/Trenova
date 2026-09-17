import type { AIProviderKind } from "@/types/ai-provider";

export type BrandPreset = {
  kind: AIProviderKind;
  baseUrl: string;
  domain: string;
};

type BrandSource = {
  kind: AIProviderKind;
  baseUrl: string;
};

const KIND_VENDOR_DOMAIN: Partial<Record<AIProviderKind, string>> = {
  AnthropicMessages: "anthropic.com",
  OpenAIResponses: "openai.com",
  Ollama: "ollama.com",
};

const PRIVATE_HOST =
  /^(localhost|127\.|10\.|192\.168\.|172\.(1[6-9]|2\d|3[01])\.|0\.0\.0\.0|\[?::1\]?)/;

function hostOf(url: string): string | null {
  try {
    return new URL(url.trim()).hostname.toLowerCase();
  } catch {
    return null;
  }
}

function isPrivateHost(host: string): boolean {
  return (
    PRIVATE_HOST.test(host) ||
    !host.includes(".") ||
    host.endsWith(".local") ||
    host.endsWith(".internal")
  );
}

function registrableDomain(host: string): string {
  const parts = host.split(".");
  return parts.length <= 2 ? host : parts.slice(-2).join(".");
}

/**
 * The vendor domain a saved provider's logo should come from, or null when
 * the endpoint is private or generic and a monogram is the honest choice.
 */
export function providerBrandDomain(
  provider: BrandSource,
  presets: readonly BrandPreset[],
): string | null {
  const baseUrl = provider.baseUrl.trim();
  if (baseUrl === "") {
    return KIND_VENDOR_DOMAIN[provider.kind] ?? null;
  }

  const host = hostOf(baseUrl);
  if (!host) {
    return null;
  }
  if (isPrivateHost(host)) {
    return null;
  }

  const preset = presets.find(
    (candidate) =>
      candidate.kind === provider.kind &&
      candidate.domain !== "" &&
      hostOf(candidate.baseUrl) === host,
  );
  if (preset) {
    return preset.domain;
  }

  return registrableDomain(host);
}
