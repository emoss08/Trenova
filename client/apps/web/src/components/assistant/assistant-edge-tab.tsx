import { dockPositionClass, isLeftDock, type AssistantDock } from "@/lib/assistant-dock";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { m, useReducedMotion } from "motion/react";
import { AssistantOrb } from "./assistant-orb";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";
import { launcherLabel, type BeaconState } from "./beacon-state";

type AssistantEdgeTabProps = {
  dock: AssistantDock;
  beacon: BeaconState;
  onClick: () => void;
};

/**
 * What is left of the beacon once someone has hidden it: a narrow tab on the
 * edge of the screen, in the same corner, that covers nothing a page puts
 * there. Rested on, it opens to show the orb.
 *
 * It still carries the one signal the beacon exists for. A change waiting on
 * the person keeps it open with the orb turned warm, because hiding the
 * button is a choice about space, not about being told nothing.
 */
export function AssistantEdgeTab({ dock, beacon, onClick }: AssistantEdgeTabProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const left = isLeftDock(dock);
  const hasPending = beacon.mode === "pending";

  return (
    <m.button
      type="button"
      onClick={onClick}
      layoutId={ASSISTANT_SURFACE_ID}
      style={{
        borderTopLeftRadius: left ? 0 : 10,
        borderBottomLeftRadius: left ? 0 : 10,
        borderTopRightRadius: left ? 10 : 0,
        borderBottomRightRadius: left ? 10 : 0,
      }}
      transition={reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 34 }}
      aria-label={launcherLabel(t, beacon)}
      aria-keyshortcuts="Meta+J"
      title={t("Assistant · ⌘J")}
      data-pending={hasPending || undefined}
      data-mode={beacon.mode}
      className={cn("as-tab ui-focus-ring", dockPositionClass(dock, "tab"))}
    >
      <AssistantOrb mode={beacon.mode} />
    </m.button>
  );
}
