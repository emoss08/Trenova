import { resolveNotificationDescriptor } from "@/components/notification-center/notification-registry";
import { formatTimestamp } from "@/lib/notification-helpers";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Badge } from "@trenova/shared/components/ui/badge";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { CircleAlertIcon } from "lucide-react";
import { lazy, Suspense } from "react";
import { commandGroupLabel } from "../../palette-commands";
import { BRAND_TILE_CLASS, NEUTRAL_TILE_CLASS } from "../../palette-entities";
import type {
  PaletteAction,
  PaletteCommand,
  PaletteIntent,
  PaletteItem,
} from "../../palette-model";
import { ShortcutKeys } from "../../palette-row";
import { CustomerPreview } from "./customer-preview";
import { DocumentPreview } from "./document-preview";
import { PreviewFrame, PreviewMessage, PreviewSection, PreviewSkeleton } from "./preview-frame";
import { WorkerPreview } from "./worker-preview";

// The shipment preview draws a route map; Google Maps loads only when a
// shipment is actually previewed.
const ShipmentPreview = lazy(() =>
  import("../shipment/shipment-preview").then((module) => ({ default: module.ShipmentPreview })),
);

interface PreviewProps {
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
}

function Tips() {
  const t = useT();
  const tips: { keys: string[]; label: string }[] = [
    {
      keys: ["@"],
      label: t("Search one kind of record: @shipment, @customer, @worker, @document"),
    },
    { keys: ["?"], label: t("End with a question mark to ask the assistant") },
    { keys: ["Tab"], label: t("Switch between records, pages and commands") },
    { keys: ["→"], label: t("See everything you can do with the selected row") },
  ];
  return (
    <div className="flex h-full flex-col justify-center gap-4 px-6">
      <p className="text-foreground text-sm font-medium">{t("Get around faster")}</p>
      <ul className="flex flex-col gap-3">
        {tips.map((tip) => (
          <li key={tip.label} className="text-foreground-muted flex items-start gap-3 text-xs">
            <ShortcutKeys keys={tip.keys} className="shrink-0" />
            <span>{tip.label}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function CommandPreview({ command, actions, onRun }: PreviewProps & { command: PaletteCommand }) {
  const t = useT();
  const tile =
    command.group === "create" || command.group === "assistant"
      ? BRAND_TILE_CLASS
      : NEUTRAL_TILE_CLASS;
  return (
    <PreviewFrame
      icon={command.icon}
      tileClass={tile}
      title={command.label}
      subtitle={commandGroupLabel(command.group, t)}
      badge={command.checked ? <Badge variant="brand">{t("Current")}</Badge> : undefined}
      actions={actions}
      onRun={onRun}
    >
      <p className="text-foreground-muted text-sm">{command.description}</p>
      {command.shortcut && (
        <PreviewSection title={t("Shortcut")}>
          <ShortcutKeys keys={command.shortcut} />
        </PreviewSection>
      )}
    </PreviewFrame>
  );
}

/**
 * The right-hand pane: a live look at whatever row is selected. Records load
 * their own data; everything else is described from what the palette
 * already holds.
 */
export function PalettePreview({
  item,
  actions,
  onRun,
  relatedCommands,
}: PreviewProps & {
  item: PaletteItem | null;
  relatedCommands: (href: string) => readonly PaletteCommand[];
}) {
  const t = useT();

  if (!item) {
    return <Tips />;
  }

  switch (item.kind) {
    case "record": {
      const props = { record: item.record, actions, onRun };
      switch (item.record.entityType) {
        case "shipment":
          return (
            <Suspense fallback={<PreviewSkeleton />}>
              <ShipmentPreview key={item.record.id} {...props} />
            </Suspense>
          );
        case "customer":
          return <CustomerPreview key={item.record.id} {...props} />;
        case "worker":
          return <WorkerPreview key={item.record.id} {...props} />;
        case "document":
          return <DocumentPreview key={item.record.id} {...props} />;
      }
      return null;
    }
    case "page": {
      const related = relatedCommands(item.page.href);
      return (
        <PreviewFrame
          key={item.key}
          icon={item.page.icon}
          tileClass={NEUTRAL_TILE_CLASS}
          title={t(item.page.title)}
          subtitle={item.page.module ? t(item.page.module) : item.page.href}
          badge={item.pinned ? <Badge variant="neutral">{t("Pinned")}</Badge> : undefined}
          actions={actions}
          onRun={onRun}
        >
          <DescriptionList layout="inline">
            <DescriptionItem label={t("Location")}>
              {item.page.trail.split(" > ").join(" / ")}
            </DescriptionItem>
            <DescriptionItem label={t("Address")}>
              <span className="font-mono text-xs break-all">{item.page.href}</span>
            </DescriptionItem>
          </DescriptionList>
          {related.length > 0 && (
            <PreviewSection title={t("Start something here")}>
              <ul className="flex flex-col gap-1">
                {related.map((command) => (
                  <li key={command.id}>
                    <button
                      type="button"
                      tabIndex={-1}
                      onMouseDown={(event) => event.preventDefault()}
                      onClick={() => onRun(command.intent)}
                      className="ui-press rounded-control hover:bg-surface-hover flex w-full items-center gap-2 px-2 py-1.5 text-left text-sm transition-colors"
                    >
                      <command.icon className="text-foreground-subtle size-3.5" />
                      {command.label}
                    </button>
                  </li>
                ))}
              </ul>
            </PreviewSection>
          )}
        </PreviewFrame>
      );
    }
    case "command":
      return (
        <CommandPreview key={item.key} command={item.command} actions={actions} onRun={onRun} />
      );
    case "attention":
      return (
        <PreviewFrame
          key={item.key}
          icon={CircleAlertIcon}
          tileClass={NEUTRAL_TILE_CLASS}
          title={item.attention.label}
          subtitle={item.attention.module}
          actions={actions}
          onRun={onRun}
        >
          <div className="flex items-baseline gap-2">
            <span className="text-foreground text-3xl font-semibold tabular-nums">
              {item.attention.count.toLocaleString()}
            </span>
            <Badge variant={item.attention.tone}>
              {t("{0, plural, one {# item waiting} other {# items waiting}}", item.attention.count)}
            </Badge>
          </div>
          <p className="text-foreground-muted text-sm">
            {t("Counts refresh every minute. Open the queue to work through them.")}
          </p>
        </PreviewFrame>
      );
    case "notification": {
      const descriptor = resolveNotificationDescriptor(item.notification);
      return (
        <PreviewFrame
          key={item.key}
          icon={descriptor.icon}
          tileClass={cn(NEUTRAL_TILE_CLASS, descriptor.tileClass, descriptor.iconClass)}
          title={item.notification.title}
          subtitle={`${t(descriptor.category)} · ${formatTimestamp(item.notification.createdAt)}`}
          badge={
            item.notification.readAt === null ? (
              <Badge variant="brand">{t("Unread")}</Badge>
            ) : undefined
          }
          actions={actions}
          onRun={onRun}
        >
          {item.notification.message ? (
            <p className="text-foreground text-sm whitespace-pre-line">
              {item.notification.message}
            </p>
          ) : (
            <PreviewMessage>{t("No further detail.")}</PreviewMessage>
          )}
        </PreviewFrame>
      );
    }
    case "ask":
      return (
        <PreviewFrame
          key={item.key}
          icon={AssistMark}
          tileClass={BRAND_TILE_CLASS}
          title={t("Ask the assistant")}
          subtitle={t("Answers from your own records, right here")}
          actions={actions}
          onRun={onRun}
        >
          <p className="text-foreground-muted text-sm">
            {t(
              "Press Enter to ask. The answer streams in above the results; open it in the Desk to keep the conversation going.",
            )}
          </p>
          <blockquote className="rounded-surface bg-sunken text-foreground ring-border-subtle px-3 py-2 text-sm ring-1">
            {item.question}
          </blockquote>
        </PreviewFrame>
      );
  }
}
