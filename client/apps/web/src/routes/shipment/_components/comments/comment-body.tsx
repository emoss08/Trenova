import {
  CommentBodyRenderer,
  type CommentBodyEntityRefAttrs,
} from "@trenova/shared/components/comment-body/comment-body";
import { useNavigate } from "react-router";
import { useCallback } from "react";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";

const ENTITY_REF_BASE_PATHS: Record<string, string> = {
  shipment: "/shipment-management/shipments",
  worker: "/dispatch/workers",
  customer: "/billing/configuration-files/customers",
};

export function CommentBody({ comment }: { comment: LocalShipmentComment }) {
  const navigate = useNavigate();

  const handleEntityRefClick = useCallback(
    (attrs: CommentBodyEntityRefAttrs) => {
      const basePath = ENTITY_REF_BASE_PATHS[attrs.entityType];
      if (!basePath || !attrs.entityId) return;
      const params = new URLSearchParams({ panelType: "edit", panelEntityId: attrs.entityId });
      void navigate(`${basePath}?${params.toString()}`);
    },
    [navigate],
  );

  return (
    <CommentBodyRenderer
      body={comment.body ?? null}
      comment={comment.comment}
      renderEntityRef={(attrs, key) => (
        <button
          key={key}
          type="button"
          className="cursor-pointer rounded bg-violet-500/10 px-0.5 font-medium text-violet-500 hover:bg-violet-500/20"
          onClick={() => handleEntityRefClick(attrs)}
        >
          #{attrs.label}
        </button>
      )}
    />
  );
}
