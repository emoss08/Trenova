import {
  HoverCard,
  HoverCardContent,
  HoverCardPortal,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { formatToUserTimezone, toDate } from "@/lib/date";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth-store";
import { UTCDate } from "@date-fns/utc";
import { format, formatDistanceToNowStrict } from "date-fns";
import { CheckIcon, CopyIcon } from "lucide-react";
import type { ComponentPropsWithoutRef } from "react";

type HoverCardContentProps = ComponentPropsWithoutRef<typeof HoverCardContent>;

type HoverCardTimestampProps = {
  timestamp?: number;
  side?: HoverCardContentProps["side"];
  sideOffset?: HoverCardContentProps["sideOffset"];
  align?: HoverCardContentProps["align"];
  alignOffset?: HoverCardContentProps["alignOffset"];
  className?: string;
  showTime?: boolean;
};

export function HoverCardTimestamp({
  timestamp,
  side = "right",
  align = "start",
  alignOffset = -4,
  sideOffset,
  className,
  showTime = true,
}: HoverCardTimestampProps) {
  const user = useAuthStore((state) => state.user);
  const userTimezone = user?.timezone || "auto";
  const userTimeFormat = user?.timeFormat || "24-hour";

  const effectiveTimezone =
    userTimezone === "auto"
      ? Intl.DateTimeFormat().resolvedOptions().timeZone
      : userTimezone;

  const date = toDate(timestamp);

  if (!timestamp || !date) {
    return <span>-</span>;
  }

  return (
    <HoverCard>
      <HoverCardTrigger
        delay={0}
        render={
          <div
            className={cn(
              "max-w-[150px] cursor-help truncate font-mono whitespace-nowrap underline decoration-muted-foreground decoration-dashed hover:decoration-primary",
              className,
            )}
          >
            {formatToUserTimezone(
              timestamp,
              {
                timeFormat: userTimeFormat,
                showSeconds: false,
                showTimeZone: false,
                showTime: showTime,
                showDate: true,
              },
              userTimezone,
            )}
          </div>
        }
      />
      <HoverCardPortal>
        <HoverCardContent
          className="w-auto p-2"
          {...{ side, align, alignOffset, sideOffset }}
        >
          <dl className="flex flex-col gap-1">
            <Row value={String(date.getTime())} label="Timestamp" />
            <Row
              value={format(new UTCDate(date), "LLL dd, y HH:mm:ss")}
              label="UTC"
            />
            <Row
              value={formatToUserTimezone(
                timestamp,
                {
                  timeFormat: userTimeFormat,
                  showSeconds: showTime,
                  showTimeZone: false,
                  showDate: true,
                },
                userTimezone,
              )}
              label={userTimezone === "auto" ? "Local" : effectiveTimezone}
            />
            <Row
              value={formatDistanceToNowStrict(date, {
                addSuffix: true,
              })}
              label="Relative"
            />
          </dl>
        </HoverCardContent>
      </HoverCardPortal>
    </HoverCard>
  );
}

function Row({ value, label }: { value: string; label: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div
      className="group flex items-center justify-between gap-4 text-sm"
      onClick={(e) => {
        e.stopPropagation();
        void copy(value);
      }}
    >
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="flex items-center gap-1 truncate font-mono">
        <span className="invisible group-hover:visible">
          {!isCopied ? (
            <CopyIcon className="size-3" />
          ) : (
            <CheckIcon className="size-3" />
          )}
        </span>
        {value}
      </dd>
    </div>
  );
}
