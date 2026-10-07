import type { AIControlSegment } from "@/lib/graphql/ai-control";

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
