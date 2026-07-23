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
