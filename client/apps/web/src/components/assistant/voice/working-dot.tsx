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
 */
export function WorkingDot({ working, className }: { working: boolean; className?: string }) {
  if (!working) {
    return null;
  }

  return (
    <span
      aria-hidden
      className={cn(
        "animate-breathe size-1.5 shrink-0 rounded-full",
        "bg-[var(--agent-accent,var(--brand))]",
        className,
      )}
    />
  );
}
