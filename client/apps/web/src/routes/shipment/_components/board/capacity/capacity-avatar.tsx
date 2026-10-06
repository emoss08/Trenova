import { cn } from "@trenova/shared/lib/utils";
import type { CapacityUnit } from "@/lib/graphql/shipment-board";
import type { CapacityProvider } from "@/lib/shipment-board/capacity-providers";
import { forwardRef, type ComponentPropsWithoutRef } from "react";
import { IdentityAvatar } from "../identity-avatar";

const RING_RADIUS = 17;
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

function Ring({ value, max, low }: { value: number; max: number; low: boolean }) {
  const fraction = max > 0 ? Math.max(0, Math.min(1, value / max)) : 0;
  return (
    <svg viewBox="0 0 40 40" className="absolute inset-0 size-10 -rotate-90" aria-hidden>
      <circle cx="20" cy="20" r={RING_RADIUS} fill="none" strokeWidth="2" className="stroke-border" />
      <circle
        cx="20"
        cy="20"
        r={RING_RADIUS}
        fill="none"
        strokeWidth="2"
        strokeLinecap="round"
        strokeDasharray={RING_CIRCUMFERENCE}
        strokeDashoffset={RING_CIRCUMFERENCE * (1 - fraction)}
        className={cn("transition-[stroke-dashoffset] duration-500", low ? "stroke-warning" : "stroke-success")}
      />
    </svg>
  );
}

type CapacityAvatarProps = ComponentPropsWithoutRef<"button"> & {
  unit: CapacityUnit;
  provider: CapacityProvider;
  caption: string;
  showRing: boolean;
  dim: boolean;
  leaving: boolean;
  active: boolean;
};

/**
 * One driver or carrier in the dock: the avatar inside its ring, a ready dot
 * or a truck count, and a short caption. Initials sit dead centre in the ring.
 */
export const CapacityAvatar = forwardRef<HTMLButtonElement, CapacityAvatarProps>(function CapacityAvatar(
  { unit, provider, caption, showRing, dim, leaving, active, className, ...props },
  ref,
) {
  const badge = provider.badge(unit);
  return (
    <button
      ref={ref}
      type="button"
      data-active={active || undefined}
      aria-label={`${unit.name}, ${caption}`}
      className={cn(
        "ui-focus-ring group/unit flex w-16 shrink-0 flex-col items-center gap-1 rounded-md py-1",
        "hover:bg-surface-hover data-active:bg-surface-active",
        dim && "opacity-75",
        leaving && "animate-leave pointer-events-none",
        className,
      )}
      {...props}
    >
      <span className="relative grid size-10 place-items-center">
        {showRing && unit.ring ? <Ring value={unit.ring.value} max={unit.ring.max} low={unit.ring.low} /> : null}
        <IdentityAvatar
          id={unit.id}
          initials={unit.initials}
          shape={provider.avatarShape}
          className="size-7.5 text-xs"
        />
        {badge.show ? (
          <span
            className={cn(
              "bg-success ring-canvas text-foreground-on-solid absolute right-0.5 bottom-0.5 grid place-items-center rounded-full font-mono text-3xs leading-none ring-2",
              badge.count ? "h-3.5 min-w-3.5 px-0.5" : "size-2.5",
            )}
          >
            {badge.count ?? ""}
          </span>
        ) : null}
      </span>
      <span className="max-w-full truncate text-xs font-medium">{unit.name.split(" ")[0]?.replace(/[.,]/g, "")}</span>
      <span className="text-muted-foreground max-w-full truncate font-mono text-2xs">{caption}</span>
    </button>
  );
});
