import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { useNow } from "@trenova/shared/hooks/use-now";
import { CommandItem } from "@trenova/shared/components/ui/command";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatCompactAge, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { cn, formatCurrency, getNameInitials } from "@trenova/shared/lib/utils";
import {
  AlertCircleIcon,
  ArrowRightIcon,
  CheckIcon,
  Pin01Icon,
} from "@trenova/shared/components/icons";
import { memo } from "react";
import { DANGER_TILE_CLASS, PALETTE_ENTITY_ICONS } from "./palette-entities";
import {
  documentKind,
  MATCH_SOURCE_LABELS,
  shipmentBol,
  shipmentLane,
  TRAIL_SEPARATOR,
  workerRole,
} from "./palette-fields";
import type { PaletteItem, PaletteMatch, PaletteRecord } from "./palette-model";
import {
  ContextLine,
  CountChip,
  MatchText,
  PaletteTile,
  ShortcutKeys,
  StatusLabel,
} from "./palette-parts";
import { recordStatus } from "./palette-status";

function ranges(match: PaletteMatch | undefined, key: string) {
  return match?.highlights[key];
}

function Title({
  text,
  match,
  mono = false,
  className,
}: {
  text: string;
  match?: PaletteMatch;
  mono?: boolean;
  className?: string;
}) {
  return (
    <MatchText
      text={text}
      ranges={ranges(match, "title")}
      className={cn(
        "text-foreground truncate text-base font-medium",
        mono && "font-mono",
        className,
      )}
    />
  );
}

/** "Keyword assistant" or "Driver Leon Park": why a row matched when the field is not on screen. */
function MatchSource({ match }: { match?: PaletteMatch }) {
  const via = match?.via;
  if (!via) {
    return null;
  }
  const label = Object.hasOwn(MATCH_SOURCE_LABELS, via.label)
    ? MATCH_SOURCE_LABELS[via.label as keyof typeof MATCH_SOURCE_LABELS]
    : via.label;
  return (
    <span className="border-border text-foreground-subtle inline-flex h-4 max-w-44 shrink-0 items-center gap-1 truncate rounded-[4px] border px-1 text-2xs">
      {label}
      <MatchText text={via.text} ranges={via.ranges} className="text-foreground-muted truncate" />
    </span>
  );
}

function openBalance(record: PaletteRecord): string | null {
  const balance = record.metadata.openBalance;
  if (balance === undefined) {
    return null;
  }
  const amount = Number(balance);
  return Number.isFinite(amount)
    ? formatCurrency(amount, record.metadata.currencyCode || "USD")
    : null;
}

function RecordContext({ record, match }: { record: PaletteRecord; match?: PaletteMatch }) {
  const t = useT();
  const meta = record.metadata;
  switch (record.entityType) {
    case "shipment": {
      const bolRanges = ranges(match, "bol");
      return (
        <ContextLine
          parts={[
            meta.customerName && (
              <MatchText
                key="customer"
                text={meta.customerName}
                ranges={ranges(match, "customer")}
                className="text-foreground-muted"
              />
            ),
            shipmentLane(record) && (
              <MatchText key="lane" text={shipmentLane(record)} ranges={ranges(match, "lane")} />
            ),
            bolRanges && (
              <MatchText
                key="bol"
                text={shipmentBol(record)}
                ranges={bolRanges}
                className="text-foreground-muted"
              />
            ),
          ]}
        />
      );
    }
    case "customer": {
      const balance = openBalance(record);
      return (
        <ContextLine
          parts={[
            (meta.code ?? record.subtitle) && (
              <MatchText
                key="code"
                text={meta.code ?? record.subtitle ?? ""}
                ranges={ranges(match, "code")}
                className="text-foreground-muted font-mono"
              />
            ),
            meta.city && <MatchText key="city" text={meta.city} ranges={ranges(match, "city")} />,
            balance && (
              <span key="balance" className="text-foreground-muted">
                {t("{0} open", balance)}
              </span>
            ),
          ]}
        />
      );
    }
    case "worker":
      return (
        <ContextLine
          parts={[
            meta.externalId && (
              <MatchText
                key="id"
                text={meta.externalId}
                ranges={ranges(match, "id")}
                className="text-foreground-muted font-mono"
              />
            ),
            workerRole(record) && (
              <MatchText key="role" text={workerRole(record)} ranges={ranges(match, "role")} />
            ),
            meta.city && <MatchText key="home" text={meta.city} ranges={ranges(match, "home")} />,
            meta.currentLoad && (
              <span key="load" className="text-foreground-muted font-mono">
                {meta.currentLoad}
              </span>
            ),
          ]}
        />
      );
    case "document": {
      const received = Number(meta.receivedAt);
      return (
        <ContextLine
          parts={[
            documentKind(record) && (
              <MatchText
                key="type"
                text={documentKind(record)}
                ranges={ranges(match, "type")}
                className="text-foreground-muted"
              />
            ),
            meta.attachedLabel && (
              <MatchText
                key="record"
                text={meta.attachedLabel}
                ranges={ranges(match, "record")}
                className="font-mono"
              />
            ),
            Number.isFinite(received) && received > 0 && (
              <span key="received">{formatUnixDateTimeShort(received)}</span>
            ),
          ]}
        />
      );
    }
  }
}

function TwoLines({ children }: { children: React.ReactNode }) {
  return <span className="flex min-w-0 flex-1 flex-col gap-px">{children}</span>;
}

function End({ children }: { children: React.ReactNode }) {
  return <span className="flex shrink-0 items-center gap-2">{children}</span>;
}

function SelectedKey({ show }: { show: boolean }) {
  if (!show) {
    return null;
  }
  return (
    <ShortcutKeys
      keys={["↵"]}
      className="hidden shrink-0 group-data-[selected=true]/row:inline-flex"
    />
  );
}

function RecordRow({ item }: { item: Extract<PaletteItem, { kind: "record" }> }) {
  const t = useT();
  const now = useNow();
  const { record, match } = item;
  const status = recordStatus(record, t);
  const mono = record.entityType === "shipment" || record.entityType === "document";
  return (
    <>
      <PaletteTile
        icon={PALETTE_ENTITY_ICONS[record.entityType]}
        initials={record.entityType === "worker" ? getNameInitials(record.title) : undefined}
      />
      <TwoLines>
        <Title text={record.title} match={match} mono={mono} />
        <RecordContext record={record} match={match} />
      </TwoLines>
      <End>
        <MatchSource match={match} />
        {item.visitedAt !== undefined && (
          <span className="text-foreground-faint font-mono text-xs tabular-nums">
            {formatCompactAge(now - item.visitedAt / 1000)}
          </span>
        )}
        {status && <StatusLabel label={status.label} state={status.state} />}
        <SelectedKey show />
      </End>
    </>
  );
}

function RowBody({ item }: { item: PaletteItem }) {
  const t = useT();

  switch (item.kind) {
    case "record":
      return <RecordRow item={item} />;
    case "page": {
      const { page, match } = item;
      const trail = page.trail.map((segment) => t(segment)).join(TRAIL_SEPARATOR);
      return (
        <>
          <PaletteTile icon={page.icon} />
          <TwoLines>
            <Title text={t(page.title)} match={match} />
            <ContextLine
              parts={[
                trail ? (
                  <MatchText key="trail" text={trail} ranges={ranges(match, "trail")} />
                ) : (
                  t("Workspace")
                ),
              ]}
            />
          </TwoLines>
          <End>
            <MatchSource match={match} />
            {item.pinned && (
              <Pin01Icon aria-label={t("Pinned")} className="text-foreground-subtle size-3.5" />
            )}
            <SelectedKey show />
          </End>
        </>
      );
    }
    case "command": {
      const { command, match } = item;
      return (
        <>
          <PaletteTile
            icon={command.icon}
            className={command.destructive ? DANGER_TILE_CLASS : undefined}
          />
          <TwoLines>
            <Title
              text={command.label}
              match={match}
              className={command.destructive ? "text-danger" : undefined}
            />
            <ContextLine
              parts={[
                <MatchText
                  key="description"
                  text={command.description}
                  ranges={ranges(match, "description")}
                />,
              ]}
            />
          </TwoLines>
          <End>
            <MatchSource match={match} />
            {command.checked && (
              <CheckIcon aria-label={t("Current")} className="text-foreground size-4" />
            )}
            {command.shortcut ? <ShortcutKeys keys={command.shortcut} /> : <SelectedKey show />}
          </End>
        </>
      );
    }
    case "attention":
      return (
        <>
          <PaletteTile icon={AlertCircleIcon} />
          <TwoLines>
            <span className="text-foreground truncate text-base font-medium">
              {item.attention.label}
            </span>
            <ContextLine parts={[item.attention.module]} />
          </TwoLines>
          <End>
            <CountChip value={item.attention.count} />
            <SelectedKey show />
          </End>
        </>
      );
    case "ask":
      return (
        <>
          <PaletteTile mark={<NovaMark className="size-3.5" />} />
          <TwoLines>
            <span className="text-foreground truncate text-base font-medium">
              {item.subject ? item.question : t("Ask Nova")}
            </span>
            <ContextLine
              parts={[
                item.subject ? t("About {0}", item.subject.label) : t("“{0}”", item.question),
              ]}
            />
          </TwoLines>
          <End>
            <SelectedKey show />
          </End>
        </>
      );
    case "widen":
      return (
        <>
          <PaletteTile icon={ArrowRightIcon} />
          <span className="text-foreground min-w-0 flex-1 truncate text-base font-medium">
            {t("Search everything for “{0}”", item.query)}
          </span>
          <End>
            <SelectedKey show />
          </End>
        </>
      );
  }
}

function isSingleLine(item: PaletteItem): boolean {
  return item.kind === "widen";
}

/**
 * One row, whatever it stands for. Every kind shares the anatomy — a neutral
 * tile, a title, one line of context, and what it needs on the right — so a
 * list that mixes records, pages and commands still reads as one list. The
 * selection is drawn by the list as one sliding fill, so a row draws no
 * selected background or bar of its own.
 */
export const PaletteRow = memo(function PaletteRow({
  item,
  onSelect,
}: {
  item: PaletteItem;
  onSelect: (item: PaletteItem) => void;
}) {
  return (
    <CommandItem
      value={item.key}
      onSelect={() => onSelect(item)}
      data-kind={item.kind}
      data-palette-row=""
      className={cn(
        "group/row rounded-surface relative z-1 h-auto scroll-mt-9 scroll-mb-2 gap-3 px-2.5 text-base",
        "data-[selected=true]:bg-transparent",
        isSingleLine(item) ? "min-h-9 py-1" : "min-h-11.5 py-1.5",
      )}
    >
      <RowBody item={item} />
    </CommandItem>
  );
});
