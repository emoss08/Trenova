import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, XIcon } from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";

/**
 * The batch, once something is marked: how many, and the two things that
 * can be done to all of them. A change applies to one proposal from its
 * own detail, so there is no batch modify.
 */
export function BatchBar({
  count,
  mixedTools,
  busy,
  onAccept,
  onReject,
  onClear,
}: {
  count: number;
  /** The marked rows span more than one tool, which the server refuses as one batch. */
  mixedTools: boolean;
  busy: boolean;
  onAccept: () => void;
  onReject: () => void;
  onClear: () => void;
}) {
  const t = useT();
  const reduceMotion = useReducedMotion();

  return (
    <AnimatePresence initial={false}>
      {count > 0 && (
        <m.div
          key="batch"
          initial={reduceMotion ? false : { opacity: 0, y: 8 }}
          animate={{ opacity: 1, y: 0 }}
          exit={reduceMotion ? undefined : { opacity: 0, y: 8 }}
          transition={{ duration: 0.16 }}
          className="bg-popover text-popover-foreground ring-foreground/10 dark absolute inset-x-3 bottom-3 z-10 flex items-center gap-3 rounded-lg px-3 py-2 ring-1"
        >
          <span className="text-sm font-medium tabular-nums">
            {t("{0, plural, one {# marked} other {# marked}}", count)}
          </span>
          {mixedTools ? (
            <span className="text-muted-foreground text-xs">
              {t("A batch decides one kind of change at a time.")}
            </span>
          ) : (
            <span className="text-muted-foreground hidden items-center gap-1 text-xs sm:flex">
              <KbdGroup>
                <Kbd>⇧</Kbd>
                <Kbd>A</Kbd>
              </KbdGroup>
              {t("approve all")}
            </span>
          )}
          <div className="ml-auto flex items-center gap-1.5">
            <Button
              size="sm"
              variant="outline"
              disabled={busy || mixedTools}
              onClick={onReject}
            >
              <XIcon className="size-3.5" />
              {t("Reject all")}
            </Button>
            <Button size="sm" disabled={busy || mixedTools} onClick={onAccept}>
              <CheckIcon className="size-3.5" />
              {t("Approve all")}
            </Button>
            <Button
              size="icon-sm"
              variant="ghost"
              aria-label={t("Clear the batch")}
              onClick={onClear}
            >
              <XIcon className="size-4" />
            </Button>
          </div>
        </m.div>
      )}
    </AnimatePresence>
  );
}
