import type { InboundMessageRow } from "@/lib/graphql/inbox";
import { useTriageStore } from "@/stores/triage-store";
import {
  ArchiveIcon,
  CheckIcon,
  ClockIcon,
  LayersThree01Icon,
  User01Icon,
  XCloseIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { useTriageActions } from "../use-triage-actions";

function BarIcon({
  label,
  icon: Icon,
  disabled,
  onClick,
}: {
  label: string;
  icon: IconComponent;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            size="icon-sm"
            variant="ghost"
            aria-label={label}
            disabled={disabled}
            onClick={onClick}
          />
        }
      >
        <Icon aria-hidden className="size-3.5" />
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

/**
 * What can be done to everything selected at once, floating over the foot of
 * the list while anything is: clear, select all, snooze until tomorrow,
 * assign to me, ignore, and mark handled. Like every floating surface it is
 * inverted and opaque. Settled messages in the selection are left alone.
 */
export function BulkBar({ messages }: { messages: readonly InboundMessageRow[] }) {
  const t = useT();
  const actions = useTriageActions();
  const selected = useTriageStore((state) => state.selected);
  const setSelected = useTriageStore((state) => state.setSelected);
  const clear = useTriageStore((state) => state.clearSelection);

  if (selected.size === 0) {
    return null;
  }

  const waiting = messages
    .filter((message) => selected.has(message.id) && message.needsReview)
    .map((message) => message.id);
  const none = waiting.length === 0;
  const run = (act: (ids: string[]) => void) => () => {
    act(waiting);
    clear();
  };

  return (
    <div className="dark pointer-events-none absolute inset-x-3 bottom-3 z-20 flex justify-center">
      <div
        role="toolbar"
        aria-label={t("Selected messages")}
        className="bg-popover text-popover-foreground ui-lift-float animate-pop-in pointer-events-auto flex w-full max-w-sm items-center gap-0.5 rounded-lg p-1 ring-1 ring-foreground/10"
      >
        <BarIcon label={t("Clear")} icon={XCloseIcon} onClick={clear} />
        <span className="px-1.5 text-sm font-medium tabular-nums">
          {t("{0, plural, one {# selected} other {# selected}}", selected.size)}
        </span>
        <span className="flex-1" />
        <BarIcon
          label={t("Select all")}
          icon={LayersThree01Icon}
          onClick={() => setSelected(messages.map((message) => message.id))}
        />
        <BarIcon
          label={t("Snooze until tomorrow")}
          icon={ClockIcon}
          disabled={none}
          onClick={run(actions.snoozeUntilTomorrow)}
        />
        <BarIcon
          label={t("Assign to me")}
          icon={User01Icon}
          disabled={none}
          onClick={run(actions.assignToMe)}
        />
        <BarIcon
          label={t("Ignore")}
          icon={ArchiveIcon}
          disabled={none}
          onClick={run(actions.ignore)}
        />
        <Button size="sm" disabled={none} onClick={run(actions.handle)}>
          <CheckIcon aria-hidden className="size-3.5" strokeWidth={2.4} />
          {t("Handled")}
        </Button>
      </div>
    </div>
  );
}
