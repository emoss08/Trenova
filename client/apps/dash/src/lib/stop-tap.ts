import { dateToUnixTimestamp } from "@trenova/shared/lib/date";
import { newIdempotencyKey } from "@trenova/shared/lib/idempotency";
import type { PortalStopAction } from "@trenova/graphql/generated/graphql";

export type StopTap = {
  stopId: string;
  action: PortalStopAction;
  occurredAt: number;
  clientEventId: string;
};

// A tap is stamped when the driver makes it, not when the request lands, and
// carries one id for its life so a resend after a dropped connection is
// recorded once.
export function stopTap(stopId: string, action: PortalStopAction, at = new Date()): StopTap {
  return {
    stopId,
    action,
    occurredAt: dateToUnixTimestamp(at),
    clientEventId: newIdempotencyKey(),
  };
}
