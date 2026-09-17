import { useT } from "@trenova/shared/i18n/use-t";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { cn } from "@trenova/shared/lib/utils";
import { SparklesIcon } from "lucide-react";
import { m } from "motion/react";

type AssistantLauncherProps = {
  pendingCount: number;
  onClick: () => void;
};

/**
 * The corner button. It pulses only when a change is waiting on the
 * person's decision, because that is the one thing worth pulling them in
 * for; an idle assistant stays quiet.
 */
export function AssistantLauncher({ pendingCount, onClick }: AssistantLauncherProps) {
  const t = useT();
  const hasPending = pendingCount > 0;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <m.button
            type="button"
            onClick={onClick}
            initial={{ opacity: 0, scale: 0.8, y: 8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.8, y: 8 }}
            whileHover={{ scale: 1.06 }}
            whileTap={{ scale: 0.94 }}
            transition={{ type: "spring", stiffness: 420, damping: 28 }}
            aria-label={
              hasPending
                ? t("Open the assistant, {0} changes await your decision", pendingCount)
                : t("Open the assistant")
            }
            className={cn(
              "bg-primary text-primary-foreground focus-visible:ring-ring/50 fixed right-5 bottom-5 z-50 flex size-11 items-center justify-center rounded-full shadow-lg shadow-black/20 outline-none focus-visible:ring-[3px]",
              hasPending && "assistant-pulse",
            )}
          />
        }
      >
        <SparklesIcon className="size-5" strokeWidth={2} />
        {hasPending && (
          <span className="bg-warning text-warning-foreground ring-background absolute -top-1 -right-1 flex h-5 min-w-5 items-center justify-center rounded-full px-1 text-[11px] font-semibold tabular-nums ring-2">
            {pendingCount > 99 ? "99+" : pendingCount}
          </span>
        )}
      </TooltipTrigger>
      <TooltipContent side="left" sideOffset={8} className="flex items-center gap-2">
        {t("Assistant")}
        <KbdGroup>
          <Kbd>⌘</Kbd>
          <Kbd>J</Kbd>
        </KbdGroup>
      </TooltipContent>
    </Tooltip>
  );
}
