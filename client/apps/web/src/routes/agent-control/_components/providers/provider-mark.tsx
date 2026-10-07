import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo } from "react";
import { providerBrandDomain, type BrandPreset } from "./provider-brand";
import type { AIProviderKind } from "@/types/ai-provider";

type ProviderMarkProps = {
  provider: { name: string; kind: AIProviderKind; baseUrl: string; enabled?: boolean };
  presets: readonly BrandPreset[];
  size?: number;
  className?: string;
};

/** A provider's logo, from its vendor where one is known, its initials otherwise. */
export function ProviderMark({ provider, presets, size = 28, className }: ProviderMarkProps) {
  const domain = useMemo(() => providerBrandDomain(provider, presets), [provider, presets]);

  return (
    <BrandLogo
      domain={domain}
      name={provider.name}
      size={size}
      className={cn(provider.enabled === false && "opacity-60 grayscale", className)}
    />
  );
}
