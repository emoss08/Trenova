import {
  AGENT_ICONS,
  agentMonogram,
  resolveAgentIdentity,
  type AgentAccentName,
  type AgentIdentityInput,
} from "@/components/agent-identity/agent-identity";
import { queries } from "@/lib/queries";
import type { AIProviderKind, AIProviderPreset } from "@/types/ai-provider";
import { useQuery } from "@tanstack/react-query";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, type CSSProperties } from "react";
import { providerBrandDomain, providerBrandPresetKey } from "../providers/provider-brand";

/** The hue token each accent draws its tile from; slate is the neutral tile. */
export const ACCENT_HUE: Record<AgentAccentName, string | null> = {
  indigo: "var(--hue-brand)",
  teal: "var(--hue-teal)",
  amber: "var(--hue-amber)",
  rose: "var(--hue-rose)",
  emerald: "var(--hue-emerald)",
  sky: "var(--hue-sky)",
  violet: "var(--hue-violet)",
  slate: null,
};

type TileProps = {
  agent: AgentIdentityInput | null | undefined;
  /** Size in pixels. */
  s?: number;
  className?: string;
};

/** An agent's tile: its icon on a tint of its accent, or a neutral tile for slate. */
export function Tile({ agent, s = 30, className }: TileProps) {
  const { icon, accent, iconChosen } = resolveAgentIdentity(agent ?? {});
  const Icon = AGENT_ICONS[icon];
  const hue = ACCENT_HUE[accent];
  const style = {
    "--h": hue ?? 0,
    width: s,
    height: s,
    borderRadius: Math.round(s * 0.28),
  } as CSSProperties;

  return (
    <span aria-hidden className={cn("tile", !hue && "n", className)} style={style}>
      {iconChosen ? (
        <Icon size={Math.round(s * 0.5)} strokeWidth={1.7} />
      ) : (
        <span style={{ fontSize: Math.round(s * 0.36), fontWeight: 600 }}>
          {agentMonogram(agent?.name)}
        </span>
      )}
    </span>
  );
}

type MarkProps = {
  /** A saved provider, or anything named, such as a route preview's choice. */
  provider: { name: string; kind?: AIProviderKind; baseUrl?: string };
  /** A preset, which names its vendor's mark directly. */
  preset?: Pick<AIProviderPreset, "key" | "domain"> | null;
  /** Size in pixels. */
  s?: number;
  className?: string;
};

/**
 * A provider's logo, drawn bare: the vendor's own mark where Trenova ships one or
 * Brandfetch has it, and the provider's initials otherwise.
 */
export function Mark({ provider, preset, s = 28, className }: MarkProps) {
  const catalog = useQuery({ ...queries.aiProvider.catalog(), staleTime: Infinity });
  const domain = useMemo(() => {
    if (preset) return preset.domain || null;
    if (!provider.kind) return null;
    return providerBrandDomain(
      { kind: provider.kind, baseUrl: provider.baseUrl ?? "" },
      catalog.data?.presets ?? [],
    );
  }, [catalog.data?.presets, preset, provider.baseUrl, provider.kind]);
  const presetKey = preset?.key ?? providerBrandPresetKey({ baseUrl: provider.baseUrl ?? "" });

  return (
    <BrandLogo
      domain={domain}
      presetKey={presetKey}
      name={provider.name}
      size={s}
      className={cn("pmk", className)}
    />
  );
}
