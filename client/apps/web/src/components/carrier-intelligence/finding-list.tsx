import { groupFindings } from "@/lib/carrier-intelligence";
import type { CarrierIntelFinding } from "@/lib/graphql/carrier-intelligence";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { InfoIcon, OctagonXIcon, TriangleAlertIcon, type LucideIcon } from "lucide-react";
import { useMemo, type ReactNode } from "react";

export type FindingKind = "blocker" | "advisory" | "notice";

export const ALL_FINDING_KINDS: readonly FindingKind[] = ["blocker", "advisory", "notice"];

const KIND_ICON: Record<FindingKind, { icon: LucideIcon; className: string }> = {
  blocker: { icon: OctagonXIcon, className: "text-red-600 dark:text-red-400" },
  advisory: { icon: TriangleAlertIcon, className: "text-amber-600 dark:text-amber-400" },
  notice: { icon: InfoIcon, className: "text-muted-foreground" },
};

export type FindingListProps = {
  findings: readonly CarrierIntelFinding[];
  ruleLabels?: Readonly<Record<string, string>>;
  kinds?: readonly FindingKind[];
  grouped?: boolean;
  compact?: boolean;
  limit?: number;
  renderActions?: (finding: CarrierIntelFinding) => ReactNode;
  emptyMessage?: ReactNode;
  className?: string;
};

type FindingRowItem = { finding: CarrierIntelFinding; kind: FindingKind };

export function findingTitle(
  finding: Pick<CarrierIntelFinding, "code" | "message">,
  ruleLabels: Readonly<Record<string, string>> | undefined,
): string {
  return ruleLabels?.[finding.code] ?? finding.message;
}

function useKindLabels(): Record<FindingKind, { row: string; group: string }> {
  const t = useT();
  return useMemo(
    () => ({
      blocker: { row: t("Blocks tendering"), group: t("Blocking") },
      advisory: { row: t("Advisory"), group: t("Advisory") },
      notice: { row: t("Notice"), group: t("Notices") },
    }),
    [t],
  );
}

function FindingRow({
  item,
  title,
  kindLabel,
  actions,
}: {
  item: FindingRowItem;
  title: string;
  kindLabel: string;
  actions: ReactNode;
}) {
  const t = useT();
  const { finding, kind } = item;
  const { icon: Icon, className } = KIND_ICON[kind];
  const notes = [
    title !== finding.message ? finding.message : null,
    finding.unverifiable ? t("Couldn't be verified") : null,
    finding.unconfirmed ? t("Waiting for a confirming refresh") : null,
    finding.overridden
      ? finding.overrideExpiresAt
        ? t("Overridden until {0}", formatUnixDateMedium(finding.overrideExpiresAt))
        : t("Overridden")
      : null,
  ].filter((note): note is string => note !== null && note !== "");

  return (
    <li
      className="flex items-start gap-2.5 py-2"
      data-finding-kind={kind}
      data-overridden={finding.overridden ? "true" : undefined}
    >
      <Tooltip>
        <TooltipTrigger
          render={<span className="mt-0.5 flex shrink-0 cursor-default" aria-label={kindLabel} />}
        >
          <Icon
            className={cn("size-3.5", finding.overridden ? "text-muted-foreground" : className)}
            aria-hidden
          />
        </TooltipTrigger>
        <TooltipContent>
          <span className="flex flex-col gap-0.5 text-xs">
            <span>{kindLabel}</span>
            <span className="font-mono opacity-70">{finding.code}</span>
          </span>
        </TooltipContent>
      </Tooltip>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className={cn("text-sm", finding.overridden && "text-muted-foreground")}>
          {title}
        </span>
        {notes.length > 0 ? (
          <span className="text-muted-foreground text-xs">{notes.join(" · ")}</span>
        ) : null}
      </div>
      {actions ? <div className="flex shrink-0 items-center gap-1">{actions}</div> : null}
    </li>
  );
}

function CompactRow({ item, title }: { item: FindingRowItem; title: string }) {
  const { icon: Icon, className } = KIND_ICON[item.kind];
  return (
    <li className="flex min-w-0 items-center gap-2 text-xs" data-finding-kind={item.kind}>
      <Icon className={cn("size-3 shrink-0", className)} aria-hidden />
      <span className="truncate">{title}</span>
    </li>
  );
}

export function FindingList({
  findings,
  ruleLabels,
  kinds = ALL_FINDING_KINDS,
  grouped = false,
  compact = false,
  limit,
  renderActions,
  emptyMessage,
  className,
}: FindingListProps) {
  const t = useT();
  const kindLabels = useKindLabels();

  const groups = useMemo(() => {
    const byKind = groupFindings(findings);
    const lists: Record<FindingKind, CarrierIntelFinding[]> = {
      blocker: byKind.blockers,
      advisory: byKind.advisories,
      notice: byKind.notices,
    };
    return kinds.map((kind) => ({
      kind,
      items: lists[kind].map((finding) => ({ finding, kind })),
    }));
  }, [findings, kinds]);

  const ordered = groups.flatMap((group) => group.items);

  if (ordered.length === 0) {
    if (emptyMessage === null) {
      return null;
    }
    return (
      <p className={cn("text-muted-foreground text-xs", className)}>
        {emptyMessage ?? t("Every enabled vetting rule passed.")}
      </p>
    );
  }

  const shown = limit === undefined ? ordered : ordered.slice(0, limit);
  const hidden = ordered.length - shown.length;
  const moreRow =
    hidden > 0 ? (
      <li className={cn("text-muted-foreground text-xs", compact ? "pl-5" : "py-2 pl-6")}>
        {t("{0, plural, one {# more finding} other {# more findings}}", hidden)}
      </li>
    ) : null;

  if (compact) {
    return (
      <ul className={cn("flex flex-col gap-1", className)}>
        {shown.map((item) => (
          <CompactRow
            key={`${item.finding.action}-${item.finding.code}`}
            item={item}
            title={findingTitle(item.finding, ruleLabels)}
          />
        ))}
        {moreRow}
      </ul>
    );
  }

  const renderRow = (item: FindingRowItem) => (
    <FindingRow
      key={`${item.finding.action}-${item.finding.code}`}
      item={item}
      title={findingTitle(item.finding, ruleLabels)}
      kindLabel={kindLabels[item.kind].row}
      actions={renderActions?.(item.finding)}
    />
  );

  if (!grouped) {
    return (
      <ul className={cn("divide-border divide-y", className)}>
        {shown.map(renderRow)}
        {moreRow}
      </ul>
    );
  }

  const visible = new Set(shown);
  return (
    <div className={cn("flex flex-col gap-4", className)}>
      {groups
        .map((group) => ({ ...group, items: group.items.filter((item) => visible.has(item)) }))
        .filter((group) => group.items.length > 0)
        .map((group) => (
          <section
            key={group.kind}
            aria-label={kindLabels[group.kind].group}
            data-finding-group={group.kind}
            className="flex flex-col"
          >
            <h3 className="text-muted-foreground flex items-center gap-1.5 text-xs font-medium">
              {kindLabels[group.kind].group}
              <span className="tabular-nums">
                {groups.find((candidate) => candidate.kind === group.kind)?.items.length}
              </span>
            </h3>
            <ul className="divide-border divide-y">{group.items.map(renderRow)}</ul>
          </section>
        ))}
      {hidden > 0 ? <ul>{moreRow}</ul> : null}
    </div>
  );
}
