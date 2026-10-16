import { useState } from "react";

/**
 * A value as the person just set it, shown until the value it stands for
 * changes. A control that writes to a cache draws the new value from the
 * click on, rather than its old one until the cache's readers hear of the
 * change (TanStack Query tells them a task later). Whatever the source next
 * becomes takes over: the value asked for, the server's own form of it, or
 * the old value back when the change is refused.
 */
export function usePendingValue<T>(value: T): [T, (pending: T) => void] {
  const [pending, setPending] = useState<{ value: T } | null>(null);
  const [seen, setSeen] = useState(value);
  if (!Object.is(seen, value)) {
    setSeen(value);
    setPending(null);
  }

  return [pending ? pending.value : value, (next: T) => setPending({ value: next })];
}
