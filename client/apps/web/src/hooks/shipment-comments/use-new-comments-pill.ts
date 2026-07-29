import { useCallback, useEffect, useRef, useState } from "react";
import { subscribeToCommentInserts } from "@/lib/shipment-comment-realtime";

export function useNewCommentsPill(shipmentId: string, isAtEnd: boolean) {
  const [unseenCount, setUnseenCount] = useState(0);
  const isAtEndRef = useRef(isAtEnd);
  isAtEndRef.current = isAtEnd;

  useEffect(() => {
    if (!shipmentId) return;
    return subscribeToCommentInserts(shipmentId, () => {
      if (!isAtEndRef.current) {
        setUnseenCount((count) => count + 1);
      }
    });
  }, [shipmentId]);

  useEffect(() => {
    if (isAtEnd) setUnseenCount(0);
  }, [isAtEnd]);

  const clearUnseen = useCallback(() => setUnseenCount(0), []);

  return { unseenCount, clearUnseen };
}
