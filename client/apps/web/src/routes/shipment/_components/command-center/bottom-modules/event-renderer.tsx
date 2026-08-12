import type { ReactNode } from "react";
import type { ShipmentEvent, ShipmentEventActorType } from "@/types/shipment-event";

const COMMENT_DETAIL_MAX_LEN = 240;
const MENTION_REGEX = /(@[\w.-]+)/g;

const ACTOR_LABEL: Record<ShipmentEventActorType, string> = {
  user: "Someone",
  apikey: "API key",
  system: "System",
  edi: "EDI",
};

export type RenderedEvent = {
  headline: ReactNode;
  detail?: ReactNode;
  actorHandle: string;
};

export function renderEvent(event: ShipmentEvent): RenderedEvent {
  const actor = formatActor(event);
  const target = formatTarget(event);
  const meta = event.metadata ?? {};

  switch (event.type) {
    case "CommentPosted": {
      const body = stringFrom(meta.commentBody) ?? "";
      return {
        headline: composeHeadline(actor, "added a comment to", target),
        detail: body ? withMentions(clamp(body, COMMENT_DETAIL_MAX_LEN)) : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "StatusChanged": {
      const newStatus = stringFrom(meta.newStatus);
      return {
        headline: composeHeadline(
          actor,
          newStatus ? `marked` : "updated",
          target,
          newStatus ? ` as ${newStatus}` : "",
        ),
        actorHandle: actorHandle(event),
      };
    }
    case "ShipmentCreated":
      return {
        headline: composeHeadline(actor, "created", target),
        actorHandle: actorHandle(event),
      };
    case "ShipmentCanceled": {
      const reason = stringFrom(meta.reason);
      return {
        headline: composeHeadline(actor, "canceled", target),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "ShipmentUncanceled":
      return {
        headline: composeHeadline(actor, "reopened", target),
        actorHandle: actorHandle(event),
      };
    case "OwnershipTransferred":
      return {
        headline: composeHeadline(actor, "transferred ownership of", target),
        actorHandle: actorHandle(event),
      };
    case "MoveDeparted":
      return {
        headline: composeHeadline(actor, "dispatched a move on", target),
        actorHandle: actorHandle(event),
      };
    case "MoveArrived":
      return {
        headline: composeHeadline(actor, "completed a move on", target),
        actorHandle: actorHandle(event),
      };
    case "MoveStatusChanged": {
      const prev = stringFrom(meta.previousStatus);
      const next = stringFrom(meta.newStatus);
      const trail = prev && next ? ` (${prev} → ${next})` : "";
      return {
        headline: composeHeadline(actor, "updated a move on", target, trail),
        actorHandle: actorHandle(event),
      };
    }
    case "DriverAssigned": {
      const driver = stringFrom(meta.driverName) ?? "a driver";
      return {
        headline: composeHeadline(actor, `assigned ${driver} to`, target),
        actorHandle: actorHandle(event),
      };
    }
    case "DriverReassigned": {
      const driver = stringFrom(meta.driverName) ?? "a driver";
      return {
        headline: composeHeadline(actor, `reassigned ${driver} on`, target),
        actorHandle: actorHandle(event),
      };
    }
    case "DriverUnassigned":
      return {
        headline: composeHeadline(actor, "unassigned a driver from", target),
        actorHandle: actorHandle(event),
      };
    case "CarrierAssigned": {
      const carrier = carrierFrom(meta);
      return {
        headline: composeHeadline(actor, `assigned ${carrier} to`, target),
        actorHandle: actorHandle(event),
      };
    }
    case "CarrierUnassigned": {
      const carrier = carrierFrom(meta);
      const reason = stringFrom(meta.reason);
      return {
        headline: composeHeadline(actor, `unassigned ${carrier} from`, target),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "TenderOffered": {
      const carrier = carrierFrom(meta);
      const channel = stringFrom(meta.channel);
      return {
        headline: composeHeadline(
          actor,
          "offered",
          target,
          ` to ${carrier}${channel ? ` via ${channel}` : ""}`,
        ),
        actorHandle: actorHandle(event),
      };
    }
    case "TenderAccepted":
      return {
        headline: composeHeadline(carrierFrom(meta), "accepted the tender for", target),
        actorHandle: actorHandle(event),
      };
    case "TenderDeclined": {
      const reason = stringFrom(meta.reason);
      return {
        headline: composeHeadline(carrierFrom(meta), "declined the tender for", target),
        detail: reason ? `"${reason}"` : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "TenderExpired":
      return {
        headline: composeHeadline(`Offer to ${carrierFrom(meta)}`, "expired on", target),
        actorHandle: actorHandle(event),
      };
    case "TenderWithdrawn": {
      const reason = stringFrom(meta.reason);
      return {
        headline: composeHeadline(actor, "withdrew the tender on", target),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "TenderNeedsReview": {
      const reason = stringFrom(meta.reason);
      return {
        headline: composeHeadline(
          actor,
          `flagged ${carrierFrom(meta)}'s acceptance on`,
          target,
          " for review",
        ),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "RoutingGuideExhausted": {
      const noun = stringFrom(meta.mode) === "Waterfall" ? "routing guide" : "spot tender";
      return {
        headline: composeHeadline(actor, `exhausted the ${noun} on`, target),
        actorHandle: actorHandle(event),
      };
    }
    case "TenderLateResponse": {
      const action = stringFrom(meta.action) ?? "response";
      return {
        headline: composeHeadline(
          actor,
          `recorded a late ${action} from ${carrierFrom(meta)} on`,
          target,
        ),
        actorHandle: actorHandle(event),
      };
    }
    case "TenderDeliveryFailed": {
      const deliveryError = stringFrom(meta.error);
      return {
        headline: composeHeadline(
          actor,
          `could not deliver the offer to ${carrierFrom(meta)} for`,
          target,
        ),
        detail: deliveryError
          ? <span className="text-destructive">{deliveryError}</span>
          : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "TenderEntrySkipped": {
      const reasons = stringListFrom(meta.reasons);
      return {
        headline: composeHeadline(
          actor,
          `skipped ${carrierFrom(meta)}${rankSuffix(meta.rank)} on`,
          target,
        ),
        detail: reasons.length > 0 ? reasons.join("; ") : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "TenderEntryWarned": {
      const warnings = stringListFrom(meta.warnings);
      return {
        headline: composeHeadline(
          actor,
          `tendered ${carrierFrom(meta)}${rankSuffix(meta.rank)} with warnings on`,
          target,
        ),
        detail: warnings.length > 0 ? warnings.join("; ") : undefined,
        actorHandle: actorHandle(event),
      };
    }
    case "HoldPlaced": {
      const holdType = stringFrom(meta.holdType);
      const verb = holdType ? `placed a ${holdType} hold on` : "placed a hold on";
      return {
        headline: composeHeadline(actor, verb, target),
        actorHandle: actorHandle(event),
      };
    }
    case "HoldUpdated": {
      const holdType = stringFrom(meta.holdType);
      const verb = holdType ? `updated a ${holdType} hold on` : "updated a hold on";
      return {
        headline: composeHeadline(actor, verb, target),
        actorHandle: actorHandle(event),
      };
    }
    case "HoldReleased": {
      const holdType = stringFrom(meta.holdType);
      const verb = holdType ? `released a ${holdType} hold on` : "released a hold on";
      return {
        headline: composeHeadline(actor, verb, target),
        actorHandle: actorHandle(event),
      };
    }
    case "ShipmentUpdated":
      return {
        headline: composeHeadline(actor, "updated", target),
        actorHandle: actorHandle(event),
      };
    case "StopCompleted":
      return {
        headline: composeHeadline(actor, "completed a stop on", target),
        actorHandle: actorHandle(event),
      };
    default:
      return {
        headline: event.summary,
        actorHandle: actorHandle(event),
      };
  }
}

function composeHeadline(
  actor: string,
  verb: string,
  target: ReactNode,
  trail = "",
): ReactNode {
  return (
    <>
      <span className="font-medium text-foreground">{actor}</span>
      {" "}
      {verb}
      {" "}
      {target}
      {trail}
    </>
  );
}

function formatTarget(event: ShipmentEvent): ReactNode {
  const proNumber = event.shipment?.proNumber;
  if (proNumber) {
    return <span className="font-mono text-foreground">#{proNumber}</span>;
  }
  return "a shipment";
}

function formatActor(event: ShipmentEvent): string {
  if (event.actor?.name) return event.actor.name;
  if (event.actorLabel) return event.actorLabel;
  return ACTOR_LABEL[event.actorType] ?? "Someone";
}

function actorHandle(event: ShipmentEvent): string {
  if (event.actor?.username) return `@${event.actor.username}`;
  if (event.actor?.name) return `@${event.actor.name.toLowerCase().replace(/\s+/g, "-")}`;
  if (event.actorLabel) return event.actorLabel;
  return event.actorType;
}

function carrierFrom(meta: Record<string, unknown>): string {
  return stringFrom(meta.carrierName) ?? "a carrier";
}

function rankSuffix(value: unknown): string {
  const rank = typeof value === "number" && Number.isFinite(value) ? value : undefined;
  return rank === undefined ? "" : ` (rank ${rank})`;
}

function stringListFrom(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((entry): entry is string => typeof entry === "string" && entry.length > 0);
}

function stringFrom(value: unknown): string | undefined {
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

function clamp(value: string, max: number): string {
  if (value.length <= max) return value;
  return value.slice(0, max).trimEnd() + "…";
}

function withMentions(text: string): ReactNode {
  const parts = text.split(MENTION_REGEX);
  return parts.map((part, idx) =>
    MENTION_REGEX.test(part) ? (
      <span key={idx} className="text-brand">
        {part}
      </span>
    ) : (
      part
    ),
  );
}
