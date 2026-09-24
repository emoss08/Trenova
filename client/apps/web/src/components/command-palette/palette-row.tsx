import { resolveNotificationDescriptor } from "@/components/notification-center/notification-registry";
import { formatTimestamp } from "@/lib/notification-helpers";
import { ShipmentStatusBadge, StatusBadge } from "@trenova/shared/components/status-badge";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Badge } from "@trenova/shared/components/ui/badge";
import { CommandItem } from "@trenova/shared/components/ui/command";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { shipmentStatusSchema } from "@trenova/shared/types/shipment";
import { CheckIcon, ChevronRightIcon, CircleAlertIcon, PinIcon } from "lucide-react";
import { memo, useMemo } from "react";
import { BRAND_TILE_CLASS, NEUTRAL_TILE_CLASS, PALETTE_ENTITIES } from "./palette-entities";
import type { PaletteItem, PaletteRecord } from "./palette-model";
import { PaletteTile } from "./palette-tile";

const ATTENTION_TILE: Record<string, string> = {
  info: "bg-info-subtle text-info-subtle-foreground ring-info-border",
  warning: "bg-warning-subtle text-warning-subtle-foreground ring-warning-border",
  danger: "bg-danger-subtle text-danger-subtle-foreground ring-danger-border",
};

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * The text with every typed term underlined where it landed. A quiet mark,
 * not a fill: the row's own selection is the loud thing on screen.
 */
export const MatchText = memo(function MatchText({
  text,
  query,
  className,
}: {
  text: string;
  query: string;
  className?: string;
}) {
  const parts = useMemo(() => {
    const terms = query
      .trim()
      .split(/\s+/)
      .filter((term) => term.length > 0)
      .sort((a, b) => b.length - a.length)
      .map(escapeRegExp);
    if (terms.length === 0 || text === "") {
      return null;
    }
    return text.split(new RegExp(`(${terms.join("|")})`, "gi"));
  }, [query, text]);

  if (!parts) {
    return <span className={className}>{text}</span>;
  }

  return (
    <span className={className}>
      {parts.map((part, index) =>
        index % 2 === 1 ? (
          <mark
            key={index}
            className="decoration-highlight bg-transparent text-inherit underline decoration-2 underline-offset-[3px]"
          >
            {part}
          </mark>
        ) : (
          part
        ),
      )}
    </span>
  );
});

function Dot() {
  return (
    <span aria-hidden className="text-border-strong">
      ·
    </span>
  );
}

function Meta({ parts }: { parts: (React.ReactNode | null | undefined | false)[] }) {
  const present = parts.filter((part): part is React.ReactNode => Boolean(part));
  if (present.length === 0) {
    return null;
  }
  return (
    <span className="text-foreground-subtle flex min-w-0 items-center gap-1.5 truncate text-xs">
      {present.map((part, index) => (
        <span key={index} className="flex min-w-0 items-center gap-1.5">
          {index > 0 && <Dot />}
          <span className="truncate">{part}</span>
        </span>
      ))}
    </span>
  );
}

function RecordStatus({ record }: { record: PaletteRecord }) {
  const status = record.metadata.status;
  if (!status) {
    return null;
  }
  if (record.entityType === "shipment") {
    const parsed = shipmentStatusSchema.safeParse(status);
    return parsed.success ? <ShipmentStatusBadge status={parsed.data} /> : null;
  }
  if (record.entityType === "customer" || record.entityType === "worker") {
    return <StatusBadge status={status} />;
  }
  return null;
}

function RecordBody({ record, query }: { record: PaletteRecord; query: string }) {
  const t = useT();
  const entity = PALETTE_ENTITIES[record.entityType];
  const meta = record.metadata;

  const details: React.ReactNode[] =
    record.entityType === "shipment"
      ? [
          meta.customerName && (
            <MatchText
              key="customer"
              text={
                meta.customerCode
                  ? `${meta.customerName} (${meta.customerCode})`
                  : meta.customerName
              }
              query={query}
            />
          ),
          meta.bol && <MatchText key="bol" text={t("BOL {0}", meta.bol)} query={query} />,
          meta.serviceTypeCode,
        ]
      : [record.subtitle && <MatchText key="subtitle" text={record.subtitle} query={query} />];

  return (
    <>
      <PaletteTile
        icon={entity.icon}
        initials={record.entityType === "worker" ? getNameInitials(record.title) : undefined}
        className={entity.tileClass}
      />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="flex min-w-0 items-center gap-2">
          <MatchText text={record.title} query={query} className="truncate font-medium" />
          <span className="text-2xs text-foreground-subtle shrink-0">{t(entity.label)}</span>
        </span>
        <Meta parts={details} />
      </span>
      <span className="flex shrink-0 items-center gap-2">
        <RecordStatus record={record} />
      </span>
    </>
  );
}

function Trail({ trail, query }: { trail: string; query: string }) {
  const segments = trail.split(" > ");
  return (
    <span className="text-foreground-subtle flex min-w-0 items-center gap-0.5 truncate text-xs">
      {segments.map((segment, index) => (
        <span key={`${segment}-${index}`} className="flex min-w-0 items-center gap-0.5">
          {index > 0 && <ChevronRightIcon aria-hidden className="size-3 shrink-0 opacity-60" />}
          <MatchText text={segment} query={query} className="truncate" />
        </span>
      ))}
    </span>
  );
}

export function ShortcutKeys({ keys, className }: { keys: readonly string[]; className?: string }) {
  return (
    <KbdGroup className={className}>
      {keys.map((key) => (
        <Kbd key={key}>{key}</Kbd>
      ))}
    </KbdGroup>
  );
}

function RowBody({ item, query }: { item: PaletteItem; query: string }) {
  const t = useT();

  switch (item.kind) {
    case "record":
      return <RecordBody record={item.record} query={query} />;
    case "page":
      return (
        <>
          <PaletteTile icon={item.page.icon} className={NEUTRAL_TILE_CLASS} />
          <span className="flex min-w-0 flex-1 flex-col">
            <MatchText text={t(item.page.title)} query={query} className="truncate font-medium" />
            <Trail trail={item.page.trail} query={query} />
          </span>
          {item.pinned && (
            <PinIcon
              aria-label={t("Pinned")}
              className="text-foreground-subtle size-3.5 shrink-0"
            />
          )}
        </>
      );
    case "command": {
      const { command } = item;
      return (
        <>
          <PaletteTile
            icon={command.icon}
            className={
              command.group === "create" || command.group === "assistant"
                ? BRAND_TILE_CLASS
                : NEUTRAL_TILE_CLASS
            }
          />
          <span className="flex min-w-0 flex-1 flex-col">
            <MatchText
              text={command.label}
              query={query}
              className={cn("truncate font-medium", command.destructive && "text-danger")}
            />
            <span className="text-foreground-subtle truncate text-xs">{command.description}</span>
          </span>
          {command.checked && (
            <CheckIcon aria-label={t("Current")} className="text-brand size-4 shrink-0" />
          )}
          {command.shortcut && <ShortcutKeys keys={command.shortcut} className="shrink-0" />}
        </>
      );
    }
    case "attention":
      return (
        <>
          <PaletteTile
            icon={CircleAlertIcon}
            className={ATTENTION_TILE[item.attention.tone] ?? NEUTRAL_TILE_CLASS}
          />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate font-medium">{item.attention.label}</span>
            <span className="text-foreground-subtle truncate text-xs">{item.attention.module}</span>
          </span>
          <Badge variant={item.attention.tone} className="tabular-nums">
            {item.attention.count.toLocaleString()}
          </Badge>
        </>
      );
    case "notification": {
      const descriptor = resolveNotificationDescriptor(item.notification);
      return (
        <>
          <PaletteTile
            icon={descriptor.icon}
            className={cn(NEUTRAL_TILE_CLASS, descriptor.tileClass, descriptor.iconClass)}
          />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate font-medium">{item.notification.title}</span>
            {!descriptor.hideMessage && item.notification.message && (
              <span className="text-foreground-subtle truncate text-xs">
                {item.notification.message}
              </span>
            )}
          </span>
          <span className="text-2xs text-foreground-subtle shrink-0 tabular-nums">
            {formatTimestamp(item.notification.createdAt)}
          </span>
        </>
      );
    }
    case "ask":
      return (
        <>
          <PaletteTile icon={AssistMark} className={BRAND_TILE_CLASS} />
          <span className="flex min-w-0 flex-1 flex-col">
            <span className="truncate font-medium">{t("Ask the assistant")}</span>
            <span className="text-foreground-subtle truncate text-xs">{item.question}</span>
          </span>
          <ShortcutKeys keys={["↵"]} className="shrink-0" />
        </>
      );
  }
}

/**
 * One row, whatever it stands for. Every kind shares the anatomy — a tile
 * naming what it is, a title, a line of context, and whatever it needs on
 * the right — so a list that mixes records, pages and commands still reads
 * as one list.
 */
export const PaletteRow = memo(function PaletteRow({
  item,
  query,
  onSelect,
  onHover,
}: {
  item: PaletteItem;
  query: string;
  onSelect: (item: PaletteItem) => void;
  onHover?: (item: PaletteItem) => void;
}) {
  return (
    <CommandItem
      value={item.key}
      onSelect={() => onSelect(item)}
      onPointerMove={onHover ? () => onHover(item) : undefined}
      data-kind={item.kind}
      className={cn(
        "group/row rounded-control relative h-auto min-h-11 gap-3 px-2.5 py-1.5 text-sm",
        "ease-swift transition-colors duration-100",
        "data-[selected=true]:bg-surface-selected",
        "before:bg-brand before:absolute before:inset-y-2 before:left-0 before:w-0.5 before:rounded-full before:opacity-0 before:transition-opacity",
        "data-[selected=true]:before:opacity-100",
      )}
    >
      <RowBody item={item} query={query} />
    </CommandItem>
  );
});
