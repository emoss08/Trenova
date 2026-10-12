import { type RichTags, translateRich } from "@trenova/shared/i18n/rich";
import { translate } from "@trenova/shared/i18n/runtime";
import type { ReactNode } from "react";
import type {
  ShipmentAssignmentEvent,
  ShipmentCarrierEvent,
  ShipmentCommentEvent,
  ShipmentEvent,
  ShipmentHoldEvent,
  ShipmentLifecycleEvent,
  ShipmentMoveEvent,
  ShipmentOwnershipEvent,
  ShipmentTenderEvent,
} from "@/types/shipment-event";

const COMMENT_DETAIL_MAX_LEN = 240;
const MENTION_REGEX = /(@[\w.-]+)/g;

export type RenderedEvent = {
  headline: ReactNode;
  detail?: ReactNode;
  actorHandle: string;
};

type RenderContext = {
  actor: string;
  tags: RichTags;
  actorHandle: string;
};

export function renderEvent(event: ShipmentEvent): RenderedEvent {
  const target = formatTarget(event);
  const ctx: RenderContext = {
    actor: formatActor(event),
    tags: {
      actor: (children) => <span className="text-foreground font-medium">{children}</span>,
      target: () => target,
    },
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

function reasonDetail(reason: string | undefined): string | undefined {
  return reason ? translate("Reason: {0}", reason) : undefined;
}

function renderLifecycle(event: ShipmentLifecycleEvent, ctx: RenderContext): RenderedEvent {
  switch (event.type) {
    case "ShipmentCreated":
      return {
        headline: translateRich("<actor>{0}</actor> created <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    case "ShipmentUpdated":
      return {
        headline: translateRich("<actor>{0}</actor> updated <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    case "StatusChanged": {
      const newStatus = present(event.newStatus);
      return {
        headline: newStatus
          ? translateRich(
              "<actor>{0}</actor> marked <target/> as {1}",
              ctx.tags,
              ctx.actor,
              newStatus,
            )
          : translateRich("<actor>{0}</actor> updated <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    }
    case "ShipmentCanceled":
      return {
        headline: translateRich("<actor>{0}</actor> canceled <target/>", ctx.tags, ctx.actor),
        detail: reasonDetail(present(event.reason)),
        actorHandle: ctx.actorHandle,
      };
    case "ShipmentUncanceled":
      return {
        headline: translateRich("<actor>{0}</actor> reopened <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderOwnership(event: ShipmentOwnershipEvent, ctx: RenderContext): RenderedEvent {
  if (event.type !== "OwnershipTransferred") return fallback(event, ctx);
  return {
    headline: translateRich(
      "<actor>{0}</actor> transferred ownership of <target/>",
      ctx.tags,
      ctx.actor,
    ),
    actorHandle: ctx.actorHandle,
  };
}

function renderMove(event: ShipmentMoveEvent, ctx: RenderContext): RenderedEvent {
  switch (event.type) {
    case "MoveDeparted":
      return {
        headline: translateRich(
          "<actor>{0}</actor> dispatched a move on <target/>",
          ctx.tags,
          ctx.actor,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "MoveArrived":
      return {
        headline: translateRich(
          "<actor>{0}</actor> completed a move on <target/>",
          ctx.tags,
          ctx.actor,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "MoveStatusChanged": {
      const prev = present(event.previousStatus);
      const next = present(event.newStatus);
      return {
        headline:
          prev && next
            ? translateRich(
                "<actor>{0}</actor> updated a move on <target/> ({1} → {2})",
                ctx.tags,
                ctx.actor,
                prev,
                next,
              )
            : translateRich("<actor>{0}</actor> updated a move on <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    }
    case "StopCompleted":
      return {
        headline: translateRich(
          "<actor>{0}</actor> completed a stop on <target/>",
          ctx.tags,
          ctx.actor,
        ),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderAssignment(event: ShipmentAssignmentEvent, ctx: RenderContext): RenderedEvent {
  const driver = present(event.driverName);
  switch (event.type) {
    case "DriverAssigned":
      return {
        headline: driver
          ? translateRich(
              "<actor>{0}</actor> assigned {1} to <target/>",
              ctx.tags,
              ctx.actor,
              driver,
            )
          : translateRich("<actor>{0}</actor> assigned a driver to <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    case "DriverReassigned":
      return {
        headline: driver
          ? translateRich(
              "<actor>{0}</actor> reassigned {1} on <target/>",
              ctx.tags,
              ctx.actor,
              driver,
            )
          : translateRich(
              "<actor>{0}</actor> reassigned a driver on <target/>",
              ctx.tags,
              ctx.actor,
            ),
        actorHandle: ctx.actorHandle,
      };
    case "DriverUnassigned":
      return {
        headline: translateRich(
          "<actor>{0}</actor> unassigned a driver from <target/>",
          ctx.tags,
          ctx.actor,
        ),
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
        headline: translateRich(
          "<actor>{0}</actor> assigned {1} to <target/>",
          ctx.tags,
          ctx.actor,
          carrier,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "CarrierUnassigned":
      return {
        headline: translateRich(
          "<actor>{0}</actor> unassigned {1} from <target/>",
          ctx.tags,
          ctx.actor,
          carrier,
        ),
        detail: reasonDetail(present(event.reason)),
        actorHandle: ctx.actorHandle,
      };
    default:
      return fallback(event, ctx);
  }
}

function renderTender(event: ShipmentTenderEvent, ctx: RenderContext): RenderedEvent {
  const carrier = carrierName(event.carrierName);
  const rank = knownRank(event.rank);
  switch (event.type) {
    case "TenderOffered": {
      const channel = present(event.channel);
      return {
        headline: channel
          ? translateRich(
              "<actor>{0}</actor> offered <target/> to {1} via {2}",
              ctx.tags,
              ctx.actor,
              carrier,
              channel,
            )
          : translateRich(
              "<actor>{0}</actor> offered <target/> to {1}",
              ctx.tags,
              ctx.actor,
              carrier,
            ),
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderAccepted":
      return {
        headline: translateRich(
          "<actor>{0}</actor> accepted the tender for <target/>",
          ctx.tags,
          carrier,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "TenderDeclined": {
      const reason = present(event.reason);
      return {
        headline: translateRich(
          "<actor>{0}</actor> declined the tender for <target/>",
          ctx.tags,
          carrier,
        ),
        detail: reason ? `"${reason}"` : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderExpired":
      return {
        headline: translateRich(
          "<actor>Offer to {0}</actor> expired on <target/>",
          ctx.tags,
          carrier,
        ),
        actorHandle: ctx.actorHandle,
      };
    case "TenderWithdrawn":
      return {
        headline: translateRich(
          "<actor>{0}</actor> withdrew the tender on <target/>",
          ctx.tags,
          ctx.actor,
        ),
        detail: reasonDetail(present(event.reason)),
        actorHandle: ctx.actorHandle,
      };
    case "TenderNeedsReview":
      return {
        headline: translateRich(
          "<actor>{0}</actor> flagged {1}'s acceptance on <target/> for review",
          ctx.tags,
          ctx.actor,
          carrier,
        ),
        detail: reasonDetail(present(event.reason)),
        actorHandle: ctx.actorHandle,
      };
    case "RoutingGuideExhausted":
      return {
        headline:
          event.mode === "Waterfall"
            ? translateRich(
                "<actor>{0}</actor> exhausted the routing guide on <target/>",
                ctx.tags,
                ctx.actor,
              )
            : translateRich(
                "<actor>{0}</actor> exhausted the spot tender on <target/>",
                ctx.tags,
                ctx.actor,
              ),
        actorHandle: ctx.actorHandle,
      };
    case "TenderLateResponse": {
      const action = present(event.action);
      return {
        headline: action
          ? translateRich(
              "<actor>{0}</actor> recorded a late {1} from {2} on <target/>",
              ctx.tags,
              ctx.actor,
              action,
              carrier,
            )
          : translateRich(
              "<actor>{0}</actor> recorded a late response from {1} on <target/>",
              ctx.tags,
              ctx.actor,
              carrier,
            ),
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderDeliveryFailed": {
      const deliveryError = present(event.error);
      return {
        headline: translateRich(
          "<actor>{0}</actor> could not deliver the offer to {1} for <target/>",
          ctx.tags,
          ctx.actor,
          carrier,
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
        headline:
          rank === undefined
            ? translateRich(
                "<actor>{0}</actor> skipped {1} on <target/>",
                ctx.tags,
                ctx.actor,
                carrier,
              )
            : translateRich(
                "<actor>{0}</actor> skipped {1} (rank {2}) on <target/>",
                ctx.tags,
                ctx.actor,
                carrier,
                rank,
              ),
        detail: reasons.length > 0 ? reasons.join("; ") : undefined,
        actorHandle: ctx.actorHandle,
      };
    }
    case "TenderEntryWarned": {
      const warnings = presentList(event.warnings);
      return {
        headline:
          rank === undefined
            ? translateRich(
                "<actor>{0}</actor> tendered {1} with warnings on <target/>",
                ctx.tags,
                ctx.actor,
                carrier,
              )
            : translateRich(
                "<actor>{0}</actor> tendered {1} (rank {2}) with warnings on <target/>",
                ctx.tags,
                ctx.actor,
                carrier,
                rank,
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
        headline: holdType
          ? translateRich(
              "<actor>{0}</actor> placed a {1} hold on <target/>",
              ctx.tags,
              ctx.actor,
              holdType,
            )
          : translateRich("<actor>{0}</actor> placed a hold on <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    case "HoldUpdated":
      return {
        headline: holdType
          ? translateRich(
              "<actor>{0}</actor> updated a {1} hold on <target/>",
              ctx.tags,
              ctx.actor,
              holdType,
            )
          : translateRich("<actor>{0}</actor> updated a hold on <target/>", ctx.tags, ctx.actor),
        actorHandle: ctx.actorHandle,
      };
    case "HoldReleased":
      return {
        headline: holdType
          ? translateRich(
              "<actor>{0}</actor> released a {1} hold on <target/>",
              ctx.tags,
              ctx.actor,
              holdType,
            )
          : translateRich("<actor>{0}</actor> released a hold on <target/>", ctx.tags, ctx.actor),
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
    headline: translateRich("<actor>{0}</actor> added a comment to <target/>", ctx.tags, ctx.actor),
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

function formatTarget(event: ShipmentEvent): ReactNode {
  const proNumber = event.shipment?.proNumber;
  if (proNumber) {
    return <span className="text-foreground font-mono">#{proNumber}</span>;
  }
  return translate("a shipment");
}

function formatActor(event: ShipmentEvent): string {
  if (event.actor?.name) return event.actor.name;
  if (event.actorLabel) return event.actorLabel;
  switch (event.actorType) {
    case "apikey":
      return translate("API key");
    case "system":
      return translate("System");
    case "edi":
      return translate("EDI");
    default:
      return translate("Someone");
  }
}

function actorHandle(event: ShipmentEvent): string {
  if (event.actor?.username) return `@${event.actor.username}`;
  if (event.actor?.name) return `@${event.actor.name.toLowerCase().replace(/\s+/g, "-")}`;
  if (event.actorLabel) return event.actorLabel;
  return event.actorType;
}

function carrierName(value: string | undefined): string {
  return present(value) ?? translate("a carrier");
}

function knownRank(rank: number | undefined): number | undefined {
  return rank === undefined || !Number.isFinite(rank) ? undefined : rank;
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
