import {
  isLeftDock,
  nearestDock,
  dockPositionClass,
  type AssistantDock,
} from "@/lib/assistant-dock";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { EyeOffIcon } from "@trenova/shared/components/icons";
import { m, useMotionValue, useReducedMotion, type PanInfo } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { AssistantDockTargets } from "./assistant-dock-targets";
import { AssistantOrb } from "./assistant-orb";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";
import { cappedCount, launcherLabel, type BeaconState } from "./beacon-state";

type AssistantLauncherProps = {
  beacon: BeaconState;
  dock?: AssistantDock;
  onClick: () => void;
  /** Moves the beacon, and the panel it opens, to another corner. */
  onMove?: (dock: AssistantDock) => void;
  /** Tucks the beacon into a tab at the edge of the screen. */
  onHide?: () => void;
};

/**
 * The beacon in the corner: a pill holding an orb, the Desk's colours turning
 * slowly round a point, with a label that slides out of it.
 *
 * At rest only the orb shows; on hover the label offers the agent last asked
 * and ⌘J. While a reply is written the ring turns fast and the point goes
 * out, and the label says what is being answered and for how long. A change
 * waiting on the person turns the orb warm and says how many, with a way to
 * review them; that leads, because only it needs the person. A reply that
 * arrived while the panel was closed swells the point and names who replied
 * until the panel is opened. With reduced motion nothing turns or breathes.
 *
 * It sits over whatever page is open, so it can be moved out of the way:
 * dragged to any corner, or tucked into a tab at the edge of the screen.
 */
export function AssistantLauncher({
  beacon,
  dock = "bottom-right",
  onClick,
  onMove,
  onHide,
}: AssistantLauncherProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const dragged = useRef(false);
  const [dragTarget, setDragTarget] = useState<AssistantDock | null>(null);
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const movable = onMove !== undefined;
  const side = isLeftDock(dock) ? "left" : "right";

  const pointerDock = (info: PanInfo): AssistantDock =>
    nearestDock(
      { x: info.point.x - window.scrollX, y: info.point.y - window.scrollY },
      { width: window.innerWidth, height: window.innerHeight },
    );

  const handleDrag = (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
    const next = pointerDock(info);
    setDragTarget((current) => (current === next ? current : next));
  };

  const handleDragEnd = (_event: MouseEvent | TouchEvent | PointerEvent, info: PanInfo) => {
    const next = pointerDock(info);
    setDragTarget(null);
    x.set(0);
    y.set(0);
    if (next !== dock) {
      onMove?.(next);
    }
  };

  return (
    <m.div
      layout
      drag={movable}
      dragMomentum={false}
      style={{ x, y }}
      onPointerDown={() => {
        dragged.current = false;
      }}
      onDragStart={() => {
        dragged.current = true;
        setDragTarget(dock);
      }}
      onDrag={handleDrag}
      onDragEnd={handleDragEnd}
      transition={reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 34 }}
      className={cn("as-beacon", dockPositionClass(dock, "launcher"))}
      data-mode={beacon.mode}
      data-side={side}
      data-dragging={dragTarget !== null || undefined}
    >
      {dragTarget && <AssistantDockTargets active={dragTarget} />}
      <m.button
        type="button"
        layoutId={ASSISTANT_SURFACE_ID}
        style={{ borderRadius: 19 }}
        transition={
          reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 420, damping: 30 }
        }
        onClick={() => {
          // A drag ends with the pointer let go over the beacon, which the
          // browser reports as a click. Moving it is not asking for it.
          if (dragged.current) {
            dragged.current = false;
            return;
          }
          onClick();
        }}
        aria-label={launcherLabel(t, beacon)}
        aria-keyshortcuts="Meta+J"
        data-writing={beacon.writingCount > 0 || undefined}
        className={cn("as-beacon-b", movable && "cursor-grab")}
      >
        <AssistantOrb mode={beacon.mode} />
        <span className="as-beacon-l" aria-hidden>
          <div>
            <span className="as-beacon-in">
              <BeaconLabel beacon={beacon} dragging={dragTarget !== null} />
            </span>
          </div>
        </span>
      </m.button>
      {onHide && (
        <button
          type="button"
          aria-label={t("Hide the assistant button")}
          title={t("Hide the button. Open the assistant from the edge tab or with ⌘J.")}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={onHide}
          className="as-beacon-hide ui-focus-ring"
        >
          <EyeOffIcon className="size-3" />
        </button>
      )}
    </m.div>
  );
}

/** What the beacon's label says, for the mode it is in. */
function BeaconLabel({ beacon, dragging }: { beacon: BeaconState; dragging: boolean }) {
  const t = useT();

  if (dragging) {
    return <span>{t("Drop it in any corner")}</span>;
  }
  switch (beacon.mode) {
    case "pending":
      return (
        <>
          <span>
            <b>
              {beacon.pendingCount > 99
                ? t("{0} changes", cappedCount(beacon.pendingCount))
                : t("{0, plural, one {# change} other {# changes}}", beacon.pendingCount)}
            </b>{" "}
            {beacon.pendingCount === 1 ? t("needs your approval") : t("need your approval")}
          </span>
          <span className="as-beacon-review">{t("Review")}</span>
        </>
      );
    case "writing":
      return beacon.writing ? (
        <>
          <span className="as-beacon-t as-shim">{beacon.writing.title || t("Writing")}</span>
          <Elapsed since={beacon.writing.startedAt} />
        </>
      ) : (
        <span>
          <b>{cappedCount(beacon.writingCount)}</b> {t("writing")}
        </span>
      );
    case "replied":
      return (
        <span>
          <b>{beacon.agentName}</b> {t("replied")}
        </span>
      );
    default:
      return (
        <>
          <span>{beacon.agentName ? t("Ask {0}", beacon.agentName) : t("Ask the assistant")}</span>
          <span className="as-beacon-keys">
            <span className="dk-kbd">⌘</span>
            <span className="dk-kbd">J</span>
          </span>
        </>
      );
  }
}

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/** Seconds since a reply started, counting while it is written. */
function Elapsed({ since }: { since: number }) {
  const [now, setNow] = useState(nowInSeconds);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(nowInSeconds()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  return <em>{`${Math.max(0, now - since)}s`}</em>;
}
