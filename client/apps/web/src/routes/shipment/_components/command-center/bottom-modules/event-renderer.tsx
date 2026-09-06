import type { ReactNode } from "react";
import type {
  ShipmentAssignmentEvent,
  ShipmentCarrierEvent,
  ShipmentCommentEvent,
  ShipmentEvent,
  ShipmentEventActorType,
  ShipmentHoldEvent,
  ShipmentLifecycleEvent,
  ShipmentMoveEvent,
  ShipmentOwnershipEvent,
  ShipmentTenderEvent,
} from "@/types/shipment-event";

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

type RenderContext = {
  actor: string;
  target: ReactNode;
  actorHandle: string;
};

export function renderEvent(event: ShipmentEvent): RenderedEvent {
  const ctx: RenderContext = {
    actor: formatActor(event),
    target: formatTarget(event),
    actorHandle: actorHandle(event),
  };

  switch (event.__typename) {
    case "ShipmentLifecycleEvent":
      return renderLifecycle(event, ctx);
    case "ShipmentOwnershipEvent":
      return renderOwnership(event, ctx);
    case "ShipmentMoveEvent":
      return renderMove(event, ctx);
    case "ShipmentAssignmentEvent":
      return renderAssignment(event, ctx);
    case "ShipmentCarrierEvent":
      return renderCarrier(event, ctx);
    case "ShipmentTenderEvent":
      return renderTender(event, ctx);
    case "ShipmentHoldEvent":
      return renderHold(event, ctx);
    case "ShipmentCommentEvent":
      return renderComment(event, ctx);
    default:
      return fallback(event, ctx);
  }
}

function renderLifecycle(event: ShipmentLifecycleEvent, ctx: RenderContext): RenderedEvent {
  switch (event.type) {
    case "ShipmentCreated":
      return {
        headline: composeHeadline(ctx.actor, "created", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "ShipmentUpdated":
      return {
        headline: composeHeadline(ctx.actor, "updated", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "StatusChanged": {
      const newStatus = present(event.newStatus);
      return {
        headline: composeHeadline(
          ctx.actor,
          newStatus ? "marked" : "updated",
          ctx.target,
          newStatus ? ` as ${newStatus}` : "",
        ),
        actorHandle: ctx.actorHandle,
      };
    }
    case "ShipmentCanceled": {
      const reason = present(event.reason);
      return {
        headline: composeHeadline(ctx.actor, "canceled", ctx.target),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "ShipmentUncanceled":
      return {
        headline: composeHeadline(ctx.actor, "reopened", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderOwnership(event: ShipmentOwnershipEvent, ctx: RenderContext): RenderedEvent {
  if (event.type !== "OwnershipTransferred") return fallback(event, ctx);
  return {
    headline: composeHeadline(ctx.actor, "transferred ownership of", ctx.target),
    actorHandle: ctx.actorHandle,
  };
}

function renderMove(event: ShipmentMoveEvent, ctx: RenderContext): RenderedEvent {
  switch (event.type) {
    case "MoveDeparted":
      return {
        headline: composeHeadline(ctx.actor, "dispatched a move on", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "MoveArrived":
      return {
        headline: composeHeadline(ctx.actor, "completed a move on", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "MoveStatusChanged": {
      const prev = present(event.previousStatus);
      const next = present(event.newStatus);
      const trail = prev && next ? ` (${prev} → ${next})` : "";
      return {
        headline: composeHeadline(ctx.actor, "updated a move on", ctx.target, trail),
        actorHandle: ctx.actorHandle,
      };
    }
    case "StopCompleted":
      return {
        headline: composeHeadline(ctx.actor, "completed a stop on", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderAssignment(event: ShipmentAssignmentEvent, ctx: RenderContext): RenderedEvent {
  const driver = present(event.driverName) ?? "a driver";
  switch (event.type) {
    case "DriverAssigned":
      return {
        headline: composeHeadline(ctx.actor, `assigned ${driver} to`, ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "DriverReassigned":
      return {
        headline: composeHeadline(ctx.actor, `reassigned ${driver} on`, ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "DriverUnassigned":
      return {
        headline: composeHeadline(ctx.actor, "unassigned a driver from", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderCarrier(event: ShipmentCarrierEvent, ctx: RenderContext): RenderedEvent {
  const carrier = carrierName(event.carrierName);
  switch (event.type) {
    case "CarrierAssigned":
      return {
        headline: composeHeadline(ctx.actor, `assigned ${carrier} to`, ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "CarrierUnassigned": {
      const reason = present(event.reason);
      return {
        headline: composeHeadline(ctx.actor, `unassigned ${carrier} from`, ctx.target),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    default:
      return fallback(event, ctx);
  }
}

function renderTender(event: ShipmentTenderEvent, ctx: RenderContext): RenderedEvent {
  const carrier = carrierName(event.carrierName);
  switch (event.type) {
    case "TenderOffered": {
      const channel = present(event.channel);
      return {
        headline: composeHeadline(
          ctx.actor,
          "offered",
          ctx.target,
          ` to ${carrier}${channel ? ` via ${channel}` : ""}`,
        ),
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderAccepted":
      return {
        headline: composeHeadline(carrier, "accepted the tender for", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "TenderDeclined": {
      const reason = present(event.reason);
      return {
        headline: composeHeadline(carrier, "declined the tender for", ctx.target),
        detail: reason ? `"${reason}"` : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderExpired":
      return {
        headline: composeHeadline(`Offer to ${carrier}`, "expired on", ctx.target),
        actorHandle: ctx.actorHandle,
      };
    case "TenderWithdrawn": {
      const reason = present(event.reason);
      return {
        headline: composeHeadline(ctx.actor, "withdrew the tender on", ctx.target),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderNeedsReview": {
      const reason = present(event.reason);
      return {
        headline: composeHeadline(
          ctx.actor,
          `flagged ${carrier}'s acceptance on`,
          ctx.target,
          " for review",
        ),
        detail: reason ? `Reason: ${reason}` : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "RoutingGuideExhausted": {
      const noun = event.mode === "Waterfall" ? "routing guide" : "spot tender";
      return {
        headline: composeHeadline(ctx.actor, `exhausted the ${noun} on`, ctx.target),
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderLateResponse": {
      const action = present(event.action) ?? "response";
      return {
        headline: composeHeadline(
          ctx.actor,
          `recorded a late ${action} from ${carrier} on`,
          ctx.target,
        ),
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderDeliveryFailed": {
      const deliveryError = present(event.error);
      return {
        headline: composeHeadline(
          ctx.actor,
          `could not deliver the offer to ${carrier} for`,
          ctx.target,
        ),
        detail: deliveryError ? (
          <span className="text-destructive">{deliveryError}</span>
        ) : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderEntrySkipped": {
      const reasons = presentList(event.reasons);
      return {
        headline: composeHeadline(
          ctx.actor,
          `skipped ${carrier}${rankSuffix(event.rank)} on`,
          ctx.target,
        ),
        detail: reasons.length > 0 ? reasons.join("; ") : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderEntryWarned": {
      const warnings = presentList(event.warnings);
      return {
        headline: composeHeadline(
          ctx.actor,
          `tendered ${carrier}${rankSuffix(event.rank)} with warnings on`,
          ctx.target,
        ),
        detail: warnings.length > 0 ? warnings.join("; ") : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    default:
      return fallback(event, ctx);
  }
}

function renderHold(event: ShipmentHoldEvent, ctx: RenderContext): RenderedEvent {
  const holdType = present(event.holdType);
  switch (event.type) {
    case "HoldPlaced":
      return {
        headline: composeHeadline(
          ctx.actor,
          holdType ? `placed a ${holdType} hold on` : "placed a hold on",
          ctx.target,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "HoldUpdated":
      return {
        headline: composeHeadline(
          ctx.actor,
          holdType ? `updated a ${holdType} hold on` : "updated a hold on",
          ctx.target,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "HoldReleased":
      return {
        headline: composeHeadline(
          ctx.actor,
          holdType ? `released a ${holdType} hold on` : "released a hold on",
          ctx.target,
        ),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderComment(event: ShipmentCommentEvent, ctx: RenderContext): RenderedEvent {
  if (event.type !== "CommentPosted") return fallback(event, ctx);
  const body = present(event.commentBody);
  return {
    headline: composeHeadline(ctx.actor, "added a comment to", ctx.target),
    detail: body ? withMentions(clamp(body, COMMENT_DETAIL_MAX_LEN)) : undefined,
    actorHandle: ctx.actorHandle,
  };
}

function fallback(event: ShipmentEvent, ctx: RenderContext): RenderedEvent {
  return {
    headline: event.summary,
    actorHandle: ctx.actorHandle,
  };
}

function composeHeadline(actor: string, verb: string, target: ReactNode, trail = ""): ReactNode {
  return (
    <>
      <span className="text-foreground font-medium">{actor}</span> {verb} {target}
      {trail}
    </>
  );
}

function formatTarget(event: ShipmentEvent): ReactNode {
  const proNumber = event.shipment?.proNumber;
  if (proNumber) {
    return <span className="text-foreground font-mono">#{proNumber}</span>;
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

function carrierName(value: string | undefined): string {
  return present(value) ?? "a carrier";
}

function rankSuffix(rank: number | undefined): string {
  return rank === undefined || !Number.isFinite(rank) ? "" : ` (rank ${rank})`;
}

function presentList(values: readonly string[]): string[] {
  return values.filter((entry) => entry.length > 0);
}

function present<T extends string>(value: T | undefined): T | undefined {
  return value !== undefined && value.length > 0 ? value : undefined;
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
