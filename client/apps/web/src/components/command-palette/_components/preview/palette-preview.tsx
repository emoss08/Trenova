import { useT } from "@trenova/shared/i18n/use-t";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatCompactAge } from "@trenova/shared/lib/date";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { usePageGlance } from "../../page-glance";
import { commandGroupLabel } from "../../palette-commands";
import type { ActionContext } from "../../palette-actions";
import type {
  PaletteAction,
  PaletteAttention,
  PaletteCommand,
  PaletteIntent,
  PaletteItem,
  PalettePage,
  PaletteRecord,
} from "../../palette-model";
import { ShortcutKeys } from "../../palette-parts";
import type { RecentPaletteRecord } from "../../palette-sections";
import { CustomerPreview } from "./customer-preview";
import { DocumentPreview } from "./document-preview";
import {
  PreviewFields,
  PreviewFrame,
  PreviewHead,
  PreviewSection,
  PreviewSource,
  PreviewStats,
  RelatedRow,
  StatePill,
} from "./preview-frame";
import { RecordLink } from "./record-link";
import { ShipmentPreview } from "./shipment-preview";
import { SOURCE_DETAILS_SHOWN, SourceRecord } from "./source-records";
import { WorkerPreview } from "./worker-preview";

const RECENT_ON_PAGE = 3;

export interface PreviewContext {
  onRun: (intent: PaletteIntent) => void;
  onNavigate: (href: string) => void;
  onOpenRecord: (record: PaletteRecord) => void;
  onMore: () => void;
  recentRecords: readonly RecentPaletteRecord[];
  relatedCommands: ActionContext["relatedCommands"];
  pinnedUrls: ReadonlySet<string>;
}

function PagePreview({ page, context }: { page: PalettePage; context: PreviewContext }) {
  const t = useT();
  const glance = usePageGlance(page.glance);
  const related = context.relatedCommands(page.href);
  const recent = page.recordType
    ? context.recentRecords
        .filter(({ record }) => record.entityType === page.recordType)
        .slice(0, RECENT_ON_PAGE)
    : [];
  const pinned = context.pinnedUrls.has(page.href);
  const now = useNow();
  const trail = page.trail.map((segment) => t(segment));

  return (
    <>
      <PreviewSource kind="page" facts={[page.module ? t(page.module) : null]} />
      <PreviewHead
        title={t(page.title)}
        subtitle={trail.length > 0 ? trail.join(" / ") : t("Workspace")}
        status={pinned ? <StatePill label={t("Pinned")} /> : undefined}
      />
      <PreviewStats items={glance.stats} />
      {related.length > 0 && (
        <PreviewSection label={t("Start something here")}>
          <div className="flex flex-col">
            {related.map((command) => (
              <RelatedRow
                key={command.id}
                icon={command.icon}
                title={command.label}
                note={command.shortcut ? <ShortcutKeys keys={command.shortcut} /> : undefined}
                onOpen={() => context.onRun(command.intent)}
              />
            ))}
          </div>
        </PreviewSection>
      )}
      {recent.length > 0 && (
        <PreviewSection label={t("You were just in")}>
          <div className="flex flex-col">
            {recent.map(({ record, visitedAt }) => (
              <RecordLink
                key={record.id}
                record={record}
                note={t("{0} ago", formatCompactAge(now - visitedAt / 1000))}
                onOpen={context.onOpenRecord}
              />
            ))}
          </div>
        </PreviewSection>
      )}
      <PreviewFields
        rows={[
          {
            label: t("Address"),
            value: <span className="font-mono text-sm break-all">{page.href}</span>,
          },
        ]}
      />
    </>
  );
}

function CommandPreview({ command }: { command: PaletteCommand }) {
  const t = useT();
  return (
    <>
      <PreviewSource kind="command" facts={[commandGroupLabel(command.group, t)]} />
      <PreviewHead
        title={command.label}
        subtitle={command.description}
        status={command.checked ? <StatePill label={t("Current")} state="ok" /> : undefined}
      />
      {command.shortcut && (
        <PreviewFields
          rows={[{ label: t("Shortcut"), value: <ShortcutKeys keys={command.shortcut} /> }]}
        />
      )}
    </>
  );
}

function AttentionPreview({ attention }: { attention: PaletteAttention }) {
  const t = useT();
  return (
    <>
      <PreviewSource kind="queue" facts={[attention.module, t("Refreshes every minute")]} />
      <PreviewHead title={attention.label} subtitle={attention.reason || undefined} />
      <PreviewStats
        items={[
          {
            label: t("Waiting on you"),
            value: attention.count.toLocaleString(),
            state:
              attention.tone === "danger"
                ? "bad"
                : attention.tone === "warning"
                  ? "warn"
                  : "neutral",
          },
        ]}
      />
    </>
  );
}

function AskPreview({ item }: { item: Extract<PaletteItem, { kind: "ask" }> }) {
  const t = useT();
  return (
    <>
      <PreviewSource kind="nova" facts={[t("Answers from your records")]} />
      <PreviewHead
        title={t("Ask Nova")}
        subtitle={item.subject ? t("About {0}", item.subject.label) : undefined}
      />
      <blockquote className="bg-sunken rounded-surface text-foreground px-3.5 py-3 text-base">
        {item.question}
      </blockquote>
      <p className="text-foreground-muted text-base leading-relaxed">
        {t(
          "The answer streams in above your results, with every record it used linked. Hover one to look at it here; continue in the Desk to keep going.",
        )}
      </p>
    </>
  );
}

function SourcesPreview({
  records,
  onOpen,
}: {
  records: readonly PaletteRecord[];
  onOpen: (record: PaletteRecord) => void;
}) {
  const t = useT();
  return (
    <>
      <PreviewSource
        kind="nova"
        facts={[t("{0, plural, one {# record used} other {# records used}}", records.length)]}
      />
      <PreviewHead title={t("What Nova read")} subtitle={t("Every record linked in the answer")} />
      <PreviewSection label={t("Sources")}>
        <div className="flex flex-col">
          {records.map((record, index) => (
            <SourceRecord
              key={`${record.entityType}:${record.id}`}
              record={record}
              detailed={index < SOURCE_DETAILS_SHOWN}
              onOpen={onOpen}
            />
          ))}
        </div>
      </PreviewSection>
    </>
  );
}

function Tips({ mac }: { mac: boolean }) {
  const t = useT();
  const tips: { keys: string[]; label: string }[] = [
    { keys: ["@"], label: t("Narrow to one kind: @shipment, @customer, @worker, @doc") },
    { keys: [">"], label: t("Start with > to run a command") },
    { keys: ["?"], label: t("End with a question mark to ask Nova") },
    { keys: [mac ? "⌘" : "Ctrl", "K"], label: t("Everything you can do with the selected row") },
  ];
  return (
    <div className="flex flex-col gap-4 pt-2">
      <p className="text-foreground-faint flex items-center gap-1.5 font-mono text-2xs uppercase">
        <AssistMark className="size-3" />
        {t("Get around faster")}
      </p>
      <ul className="flex flex-col gap-3.5">
        {tips.map((tip) => (
          <li key={tip.label} className="text-foreground-muted flex items-start gap-3 text-base">
            <ShortcutKeys keys={tip.keys} className="w-14 shrink-0" />
            <span>{tip.label}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function RecordBody({ record, context }: { record: PaletteRecord; context: PreviewContext }) {
  const props = { record, onNavigate: context.onNavigate };
  switch (record.entityType) {
    case "shipment":
      return <ShipmentPreview {...props} />;
    case "customer":
      return <CustomerPreview {...props} />;
    case "worker":
      return <WorkerPreview {...props} />;
    case "document":
      return <DocumentPreview {...props} />;
  }
}

function PreviewBody({
  item,
  sources,
  context,
  mac,
}: {
  item: PaletteItem | null;
  sources: readonly PaletteRecord[];
  context: PreviewContext;
  mac: boolean;
}) {
  if (!item) {
    return sources.length > 0 ? (
      <SourcesPreview records={sources} onOpen={context.onOpenRecord} />
    ) : (
      <Tips mac={mac} />
    );
  }
  switch (item.kind) {
    case "record":
      return <RecordBody record={item.record} context={context} />;
    case "page":
      return <PagePreview page={item.page} context={context} />;
    case "command":
      return <CommandPreview command={item.command} />;
    case "attention":
      return <AttentionPreview attention={item.attention} />;
    case "ask":
      return <AskPreview item={item} />;
    case "widen":
      return <Tips mac={mac} />;
  }
}

function bodyKey(item: PaletteItem | null, sources: readonly PaletteRecord[]): string {
  if (item) {
    return item.kind === "record" ? `record:${item.record.entityType}:${item.record.id}` : item.key;
  }
  return sources.length > 0 ? "sources" : "tips";
}

/**
 * The right-hand pane: a live look at whatever row is selected, a record
 * hovered in an answer, or, after an answer with nothing selected, the
 * records it was built from. Records load their own data; everything else is
 * described from what the palette already holds.
 */
export function PalettePreview({
  item,
  sources,
  actions,
  context,
  mac,
}: {
  item: PaletteItem | null;
  sources: readonly PaletteRecord[];
  actions: readonly PaletteAction[];
  context: PreviewContext;
  mac: boolean;
}) {
  return (
    <PreviewFrame
      key={bodyKey(item, sources)}
      actions={actions}
      onRun={context.onRun}
      onMore={context.onMore}
    >
      <PreviewBody item={item} sources={sources} context={context} mac={mac} />
    </PreviewFrame>
  );
}
