import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";

const DAY = 24 * 60 * 60;

/** The hour a case put away "until tomorrow" comes back: the start of the working day. */
const MORNING_HOUR = 8;

/** Tomorrow morning in the reader's own clock, in Unix seconds. */
export function tomorrowMorning(now: number): number {
  const at = new Date(now * 1000);
  at.setDate(at.getDate() + 1);
  at.setHours(MORNING_HOUR, 0, 0, 0);

  return Math.floor(at.getTime() / 1000);
}

/** How long a case can be put away for, shortest first. */
export function putAwayChoices(
  t: TranslateFn,
  now: number,
): readonly { key: string; label: string; until: number }[] {
  return [
    { key: "morning", label: t("Until tomorrow morning"), until: tomorrowMorning(now) },
    { key: "3d", label: t("For 3 days"), until: now + 3 * DAY },
    { key: "7d", label: t("For a week"), until: now + 7 * DAY },
  ];
}

/**
 * Puts the case away once the person has done what they can and the problem
 * is still running: it leaves Needs you and the rail until the time chosen,
 * and comes back sooner if anything new is raised about the record. Nothing
 * is closed by it; that is what accepting the risk is for.
 */
export function CasePutAway({
  now,
  pending,
  onPutAway,
}: {
  now: number;
  pending: boolean;
  onPutAway: (until: number) => void;
}) {
  const t = useT();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline" isLoading={pending} />}>
        <DeskIcon name="clock" size={13} />
        {t("Put away")}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" sideOffset={6} className="w-56">
        {putAwayChoices(t, now).map((choice) => (
          <DropdownMenuItem
            key={choice.key}
            title={choice.label}
            onClick={() => onPutAway(choice.until)}
          />
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/**
 * Says a case is put away and until when, with the way to bring it back. It
 * stands at the top of the reading, since everything under it is as it was
 * left.
 */
export function CaseAwayNotice({
  until,
  timezone,
  canBringBack,
  pending,
  onBringBack,
}: {
  until: number;
  timezone: string;
  canBringBack: boolean;
  pending: boolean;
  onBringBack: () => void;
}) {
  const t = useT();

  return (
    <div
      role="status"
      className="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-[8px] bg-dsk-sunken px-4 py-3"
    >
      <div className="flex min-w-0 flex-auto items-start gap-2 text-sm text-dsk-fg2">
        <DeskIcon name="clock" size={14} className="mt-0.5 flex-none text-dsk-subtle" />
        <span>
          {t(
            "Put away until {0}. It comes back sooner if anything new is raised about this record.",
            formatUnixDateTime(until, { timezone }),
          )}
        </span>
      </div>
      {canBringBack && (
        <Button variant="secondary" size="sm" isLoading={pending} onClick={onBringBack}>
          {t("Bring it back")}
        </Button>
      )}
    </div>
  );
}
