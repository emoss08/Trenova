import { keepsIdempotencyKey, newIdempotencyKey } from "@trenova/shared/lib/idempotency";
import { useCallback, useRef } from "react";

type PendingKey = {
  key: string;
  payload: string;
};

// useIdempotencyKey returns a runner that sends one Idempotency-Key for a write until
// the server settles it. A resubmit of the same payload after a lost response reuses
// the key, so the server replays the first result instead of writing twice; a success,
// a definite failure or a changed payload starts a new key.
export function useIdempotencyKey() {
  const pending = useRef<PendingKey | null>(null);

  return useCallback(
    async <TResult>(payload: unknown, run: (idempotencyKey: string) => Promise<TResult>) => {
      const serialized = JSON.stringify(payload) ?? "";
      if (!pending.current || pending.current.payload !== serialized) {
        pending.current = { key: newIdempotencyKey(), payload: serialized };
      }
      const { key } = pending.current;

      try {
        const result = await run(key);
        if (pending.current?.key === key) {
          pending.current = null;
        }
        return result;
      } catch (error) {
        if (!keepsIdempotencyKey(error) && pending.current?.key === key) {
          pending.current = null;
        }
        throw error;
      }
    },
    [],
  );
}
