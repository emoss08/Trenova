import { useCallback, useEffect, useRef, useState } from "react";
import { subscribeToCommentInserts } from "@/lib/shipment-comment-realtime";

export function useNewCommentsPill(shipmentId: string, isAtEnd: boolean) {
  const [unseenCount, setUnseenCount] = useState(0);
  const [prevIsAtEnd, setPrevIsAtEnd] = useState(isAtEnd);
  const isAtEndRef = useRef(isAtEnd);

  if (isAtEnd !== prevIsAtEnd) {
    setPrevIsAtEnd(isAtEnd);
    if (isAtEnd) setUnseenCount(0);
  }

  useEffect(() => {
    isAtEndRef.current = isAtEnd;
  }, [isAtEnd]);

  useEffect(() => {
    if (!shipmentId) return;
    return subscribeToCommentInserts(shipmentId, () => {
      if (!isAtEndRef.current) {
        setUnseenCount((count) => count + 1);
      }
    });
  }, [shipmentId]);

  const clearUnseen = useCallback(() => setUnseenCount(0), []);

  return { unseenCount, clearUnseen };
}
