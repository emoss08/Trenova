import { useT } from "@trenova/shared/i18n/use-t";
import {
  CommentBodyRenderer,
  type CommentBodyEntityRefAttrs,
} from "@trenova/shared/components/comment-body/comment-body";
import { useNavigate } from "@tanstack/react-router";
import { useCallback } from "react";
import type { LocalShipmentComment } from "@/lib/shipment-comment-cache";
import { isRecordEntityType, recordPath, type RecordEntityType } from "@/config/record-links";

const COMMENT_REF_KINDS: ReadonlySet<RecordEntityType> = new Set([
  "shipment",
  "worker",
  "customer",
]);

export function CommentBody({ comment }: { comment: LocalShipmentComment }) {
  const t = useT();

  const navigate = useNavigate();

  const handleEntityRefClick = useCallback(
    (attrs: CommentBodyEntityRefAttrs) => {
      const kind = attrs.entityType;
      if (!attrs.entityId || !isRecordEntityType(kind) || !COMMENT_REF_KINDS.has(kind)) return;
      void navigate({ to: recordPath(kind, attrs.entityId) });
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
          className="cursor-pointer rounded-md bg-accent-violet/10 px-0.5 font-medium text-accent-violet-on-subtle hover:bg-accent-violet/20"
          onClick={() => handleEntityRefClick(attrs)}
        >
          #{t(attrs.label)}
        </button>
      )}
    />
  );
}
