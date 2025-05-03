import { Badge } from "@/components/ui/badge";
import {
  HoverCard,
  HoverCardContent,
  HoverCardTrigger,
} from "@/components/ui/hover-card";
import { Icon } from "@/components/ui/icons";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import {
  generateDateOnlyString,
  generateDateTimeStringFromUnixTimestamp,
  toDate,
} from "@/lib/date";
import { cn, truncateText } from "@/lib/utils";
import { UTCDate } from "@date-fns/utc";
import { faCheck, faCopy } from "@fortawesome/pro-solid-svg-icons";
import { HoverCardPortal } from "@radix-ui/react-hover-card";
import { format, formatDistanceToNowStrict } from "date-fns";
import type { ComponentPropsWithoutRef } from "react";

type DataTableDescriptionProps = {
  description?: string;
  truncateLength?: number;
};

export function DataTableDescription({
  description,
  truncateLength = 50,
}: DataTableDescriptionProps) {
  if (!description) {
    return <span>-</span>;
  }

  return <span>{truncateText(description, truncateLength)}</span>;
}

export function DataTableColorColumn({
  color,
  text,
  className,
}: {
  color?: string;
  text: string;
  className?: string;
}) {
  const isColor = !!color;
  return isColor ? (
    <div
      className={cn(
        "flex items-center gap-x-1.5 text-sm font-normal text-foreground",
        className,
      )}
    >
      <div
        className="size-2 rounded-full"
        style={{
          backgroundColor: color,
        }}
      />
      <p>{text}</p>
    </div>
  ) : (
    <p>{text}</p>
  );
}

export function LastInspectionDateBadge({
  value,
}: {
  value: number | null | undefined;
}) {
  const inspectionDate = toDate(value ?? undefined);

  if (!inspectionDate)
    return <Badge variant="inactive">No inspection date</Badge>;

  return (
    <Badge variant="active">{generateDateOnlyString(inspectionDate)}</Badge>
  );
}

export function BooleanBadge({ value }: { value: boolean }) {
  return (
    <Badge variant={value ? "active" : "inactive"}>
      {value ? "Yes" : "No"}
    </Badge>
  );
}

type HoverCardContentProps = ComponentPropsWithoutRef<typeof HoverCardContent>;

interface HoverCardTimestampProps {
  timestamp?: number;
  side?: HoverCardContentProps["side"];
  sideOffset?: HoverCardContentProps["sideOffset"];
  align?: HoverCardContentProps["align"];
  alignOffset?: HoverCardContentProps["alignOffset"];
  className?: string;
}

// * Credit: https://github.com/openstatusHQ/data-table-filters/blob/main/src/app/infinite/_components/hover-card-timestamp.tsx#L28
export function HoverCardTimestamp({
  timestamp,
  side = "right",
  align = "start",
  alignOffset = -4,
  sideOffset,
  className,
}: HoverCardTimestampProps) {
  const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const date = toDate(timestamp);

  if (!timestamp || !date) {
    return <span>-</span>;
  }

  return (
    <HoverCard openDelay={0} closeDelay={0}>
      <HoverCardTrigger asChild>
        <div
          className={cn(
            "font-mono whitespace-nowrap max-w-[120px] truncate",
            className,
          )}
        >
          {generateDateTimeStringFromUnixTimestamp(timestamp)}
        </div>
      </HoverCardTrigger>
      <HoverCardPortal>
        <HoverCardContent
          className="p-2 w-auto z-10"
          {...{ side, align, alignOffset, sideOffset }}
        >
          <dl className="flex flex-col gap-1">
            <Row value={String(date.getTime())} label="Timestamp" />
            <Row
              value={format(new UTCDate(date), "LLL dd, y HH:mm:ss")}
              label="UTC"
            />
            <Row value={format(date, "LLL dd, y HH:mm:ss")} label={timezone} />
            <Row
              value={formatDistanceToNowStrict(date, { addSuffix: true })}
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
      className="group flex gap-4 text-sm justify-between items-center"
      onClick={(e) => {
        e.stopPropagation();
        copy(value);
      }}
    >
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-mono truncate flex items-center gap-1">
        <span className="invisible group-hover:visible">
          {!isCopied ? (
            <Icon icon={faCopy} className="size-3" />
          ) : (
            <Icon icon={faCheck} className="size-3" />
          )}
        </span>
        {value}
      </dd>
    </div>
  );
}
