import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { AnimatePresence, m } from "motion/react";
import type { PresenceViewer } from "@/hooks/use-presence-viewers";

const MAX_VISIBLE_VIEWERS = 4;
/** How many people the tooltip names before it counts the rest. */
const MAX_NAMED_VIEWERS = 8;

const SIZES = {
  md: { avatar: "size-6", overlap: "-space-x-2", more: "size-6 text-2xs" },
  sm: { avatar: "size-5", overlap: "-space-x-1.5", more: "h-5 min-w-5 px-1 text-2xs" },
} as const;

/**
 * The people in a presence scope as one group of overlapped avatars. It takes
 * the same room for five people as for fifty: the first few are drawn, the
 * rest are a count, and the tooltip names them up to a limit. People keep
 * their place as others come and go, so the group does not reshuffle.
 */
export function ViewersStack({
  viewers,
  label,
  size = "md",
  max = MAX_VISIBLE_VIEWERS,
}: {
  viewers: readonly PresenceViewer[];
  /** What the people are doing, such as "Viewing now: {0}". */
  label?: (names: string) => string;
  /** `sm` for a bar the group shares with other controls. */
  size?: keyof typeof SIZES;
  /** How many avatars are drawn before the rest become a count. */
  max?: number;
}) {
  const t = useT();

  if (viewers.length === 0) return null;

  const ordered = [...viewers].sort(
    (a, b) => a.name.localeCompare(b.name) || a.userId.localeCompare(b.userId),
  );
  const visible = ordered.slice(0, Math.max(max, 1));
  const overflow = ordered.length - visible.length;
  const sizing = SIZES[size];
  const named = ordered.slice(0, MAX_NAMED_VIEWERS).map((viewer) => viewer.name);
  const unnamed = ordered.length - named.length;
  const names =
    unnamed > 0
      ? t("{0} and {1, plural, one {# other} other {# others}}", named.join(", "), unnamed)
      : named.join(", ");

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div
            className={cn("flex shrink-0 items-center", sizing.overlap)}
            role="group"
            aria-label={label ? label(names) : t("Viewing now: {0}", names)}
          >
            <AnimatePresence initial={false}>
              {visible.map((viewer) => (
                <m.span
                  key={viewer.userId}
                  initial={{ scale: 0.5, opacity: 0 }}
                  animate={{ scale: 1, opacity: 1 }}
                  exit={{ scale: 0.5, opacity: 0 }}
                  transition={{ type: "spring", stiffness: 500, damping: 30 }}
                  className="ring-background block rounded-full ring-2"
                >
                  <ResolvedUserAvatar
                    userId={viewer.userId}
                    name={viewer.name}
                    className={cn(sizing.avatar, "rounded-full after:rounded-full")}
                    imageClassName="rounded-full"
                    fallbackClassName="rounded-full text-2xs"
                  />
                </m.span>
              ))}
            </AnimatePresence>
            {overflow > 0 && (
              <span
                className={cn(
                  "bg-muted text-muted-foreground border-border ring-background z-10 flex items-center justify-center rounded-full border font-medium tabular-nums ring-2",
                  sizing.more,
                )}
              >
                +{overflow}
              </span>
            )}
          </div>
        }
      />
      <TooltipContent side="top">
        {label ? label(names) : t("Viewing now: {0}", names)}
      </TooltipContent>
    </Tooltip>
  );
}
