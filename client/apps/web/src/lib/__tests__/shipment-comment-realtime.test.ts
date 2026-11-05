import { QueryClient } from "@tanstack/react-query";
import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import { describe, expect, it, vi } from "vitest";
import { commentKeys } from "../shipment-comment-cache";
import {
  handleShipmentCommentEvent,
  subscribeToCommentInserts,
} from "../shipment-comment-realtime";

function commentEvent(overrides: Partial<ResourceInvalidationEvent>): ResourceInvalidationEvent {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    resource: "shipment_comment",
    action: "created",
    recordId: "shp_1",
    actorUserId: "usr_other",
    entity: { id: "shc_1", shipmentId: "shp_1" },
    ...overrides,
  };
}

describe("handleShipmentCommentEvent", () => {
  it("refetches that shipment's thread and no other", () => {
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    handleShipmentCommentEvent(queryClient, commentEvent({}), "usr_me");

    expect(invalidate).toHaveBeenCalledWith({ queryKey: commentKeys.root("shp_1") });
    expect(invalidate).toHaveBeenCalledWith({ queryKey: commentKeys.count("shp_1") });
    expect(invalidate).toHaveBeenCalledTimes(2);
  });

  it("tells the thread about somebody else's new comment", () => {
    const listener = vi.fn();
    const stop = subscribeToCommentInserts("shp_1", listener);

    handleShipmentCommentEvent(new QueryClient(), commentEvent({}), "usr_me");

    expect(listener).toHaveBeenCalledTimes(1);
    stop();
  });

  it("does not count the person's own comment or a change to an existing one", () => {
    const listener = vi.fn();
    const stop = subscribeToCommentInserts("shp_1", listener);

    handleShipmentCommentEvent(
      new QueryClient(),
      commentEvent({ actorUserId: "usr_me" }),
      "usr_me",
    );
    handleShipmentCommentEvent(new QueryClient(), commentEvent({ action: "pinned" }), "usr_me");

    expect(listener).not.toHaveBeenCalled();
    stop();
  });

  it("refetches every open thread when the event names no shipment", () => {
    const queryClient = new QueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    handleShipmentCommentEvent(queryClient, commentEvent({ recordId: undefined }), "usr_me");

    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["shipment-comments"] });
  });
});
