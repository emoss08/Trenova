import type { StreamedSegment } from "@/components/streamed-text";
import type { AIControlSegment } from "@/lib/graphql/ai-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type NovaTarget =
  | { kind: "watchtower" }
  | { kind: "providers" }
  | { kind: "routing" }
  | { kind: "agents"; filter: "shadow" | "waiting" }
  | { kind: "provider"; providerId: string };

/** Where a linked part of a sentence leads, or null for a target this client does not know. */
export function novaTarget(segment: AIControlSegment): NovaTarget | null {
  switch (segment.target) {
    case "watchtower":
      return { kind: "watchtower" };
    case "providers":
      return { kind: "providers" };
    case "routing":
      return { kind: "routing" };
    case "agents:shadow":
      return { kind: "agents", filter: "shadow" };
    case "agents:waiting":
      return { kind: "agents", filter: "waiting" };
    case "provider":
      return segment.providerId ? { kind: "provider", providerId: segment.providerId } : null;
    default:
      return null;
  }
}

/** The server's sentence as the streamed text draws it, each link wired to where it leads. */
export function useNovaSegments(
  segments: readonly AIControlSegment[] | undefined,
  onTarget: (target: NovaTarget) => void,
): StreamedSegment[] {
  const t = useT();

  return useMemo(
    () =>
      (segments ?? []).map((segment) => {
        const target = novaTarget(segment);
        const tone =
          segment.tone === "danger" ? "danger" : segment.tone === "warn" ? "warning" : undefined;
        return {
          text: segment.text,
          emphasis: segment.strong,
          tone,
          ...(target
            ? { onActivate: () => onTarget(target), label: t("Open {0}", segment.text.trim()) }
            : {}),
        };
      }),
    [onTarget, segments, t],
  );
}
