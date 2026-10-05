import { cn } from "@trenova/shared/lib/utils";
import type { BeaconMode } from "./beacon-state";

/**
 * The assistant's orb: a thin ring in the Desk's colours turning round a
 * point. How it turns says what the assistant is doing; see assistant.css.
 */
export function AssistantOrb({ mode, className }: { mode: BeaconMode; className?: string }) {
  return <span aria-hidden className={cn("as-orb", className)} data-mode={mode} />;
}
