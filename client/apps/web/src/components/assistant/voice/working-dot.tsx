import { cn } from "@trenova/shared/lib/utils";

/**
 * The live indicator: one dot in the agent's accent that breathes while work
 * is running and is simply absent when it is not.
 *
 * This is the only looping animation in the product, and it earns the
 * exception the way a heartbeat monitor does — it loops because the thing it
 * describes is still going, and it stops the moment that stops.
 *
 * It is one shape in one place so that "something is happening" reads the
 * same in the Desk's header, on a tool step and anywhere else it turns up.
 * A spinner here and a pulse there is two vocabularies for one fact.
 *
 * `still` keeps the shape and drops the breath. A list marks many rows at
 * once and stays on screen while the person works elsewhere in it, so a dot
 * breathing on each would be a column of pulses on a working screen; there
 * the dot says "under way" by being there, the way the launcher does.
 */
export function WorkingDot({
  working,
  still = false,
  className,
}: {
  working: boolean;
  still?: boolean;
  className?: string;
}) {
  if (!working) {
    return null;
  }

  return (
    <span
      aria-hidden
      className={cn(
        !still && "animate-breathe",
        "size-1.5 shrink-0 rounded-full",
        "bg-[var(--agent-accent,var(--brand))]",
        className,
      )}
    />
  );
}
