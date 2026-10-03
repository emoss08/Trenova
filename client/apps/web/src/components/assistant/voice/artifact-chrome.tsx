import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { ArtifactKind, ArtifactStatus } from "@/types/assistant";
import {
  Compass03Icon,
  File06Icon,
  GavelIcon,
  GitCompareArrowsIcon,
  type IconComponent,
  IdCardIcon,
  Inbox01Icon,
  LayoutAlt04Icon,
  ListChecksIcon,
  Mail01Icon,
  NewspaperIcon,
  PencilLineIcon,
  Pin01Icon,
  PinOffIcon,
  ReceiptTextIcon,
  ScrollTextIcon,
  TableIcon,
  ScanTextIcon,
} from "@trenova/shared/components/icons";
import { useId, useState, type ReactNode } from "react";

/**
 * What each kind is called and drawn as, once, for every surface that names
 * one, and where in the product it came from when that is somewhere other
 * than the agent's own writing: a table from search, a run from the report
 * builder. A draft, a plan or a document is the agent's own work and names no
 * source, because "from the assistant" says nothing a person did not know.
 */
export const ARTIFACT_KINDS: Record<
  ArtifactKind,
  { label: string; icon: IconComponent; source?: string }
> = {
  report_preview: { label: "Preview", icon: TableIcon, source: "Report builder" },
  report_run: { label: "Report", icon: File06Icon, source: "Report builder" },
  email_draft: { label: "Draft", icon: Mail01Icon },
  plan: { label: "Plan", icon: ListChecksIcon },
  entity_card: { label: "Record", icon: IdCardIcon, source: "Records" },
  table_view: { label: "Table", icon: TableIcon, source: "Search" },
  rate_explanation: { label: "Rate", icon: ReceiptTextIcon, source: "Rating" },
  dashboard_ref: { label: "Dashboard", icon: LayoutAlt04Icon, source: "Dashboards" },
  briefing: { label: "Briefing", icon: NewspaperIcon },
  inbound_message: { label: "Message", icon: Inbox01Icon, source: "Inbox" },
  run_diff: { label: "Changes", icon: GitCompareArrowsIcon, source: "Report builder" },
  document: { label: "Document", icon: ScrollTextIcon },
  navigation: { label: "Page", icon: Compass03Icon },
  draft_edit: { label: "Draft change", icon: PencilLineIcon },
  decision_request: { label: "Decision", icon: GavelIcon },
  extraction: { label: "Extraction", icon: ScanTextIcon },
};

/** Where an artifact is, as a tone: severity, not category. */
export const ARTIFACT_STATUS_TONE: Record<
  ArtifactStatus,
  "neutral" | "info" | "success" | "danger"
> = {
  Pending: "info",
  Ready: "neutral",
  Failed: "danger",
  Sent: "success",
};

export function artifactStatusLabel(status: ArtifactStatus, t: (s: string) => string): string {
  switch (status) {
    case "Pending":
      return t("Waiting");
    case "Failed":
      return t("Failed");
    case "Sent":
      return t("Sent");
    default:
      return t("Ready");
  }
}

/** The status badge as every artifact surface draws it: only when the status is not Ready. */
export function ArtifactStatusBadge({
  status,
  className,
}: {
  status: ArtifactStatus;
  className?: string;
}) {
  const t = useT();
  if (status === "Ready") {
    return null;
  }

  return (
    <Badge
      variant={ARTIFACT_STATUS_TONE[status]}
      className={cn("animate-rise h-4 shrink-0 px-1.5 text-2xs", className)}
    >
      {artifactStatusLabel(status, t)}
    </Badge>
  );
}

export function ArtifactKindIcon({ kind, className }: { kind: ArtifactKind; className?: string }) {
  const Icon = ARTIFACT_KINDS[kind].icon;

  return <Icon className={className} aria-hidden />;
}

/**
 * Where an artifact came from and when, as one quiet line: "Report · from
 * Report builder · 2 minutes ago". The kind says what it is, the source says
 * which part of the product made it, and the time says how fresh it is —
 * the three things a person weighs before trusting a table.
 */
export function ArtifactProvenance({
  kind,
  createdAt,
  note,
  className,
}: {
  kind: ArtifactKind;
  /** Unix seconds; zero or absent leaves the time out. */
  createdAt?: number;
  /** How it was made, when the kind alone does not say: "Read together from 11 calls". */
  note?: string;
  className?: string;
}) {
  const t = useT();
  const now = useNowSeconds();
  const meta = ARTIFACT_KINDS[kind];
  const stamped = createdAt !== undefined && createdAt > 0;

  return (
    <p className={cn("text-foreground-subtle flex min-w-0 items-center text-2xs", className)}>
      <span className="truncate">
        {t(meta.label)}
        {meta.source ? ` · ${t("from {0}", t(meta.source))}` : ""}
      </span>
      {stamped && (
        <>
          <span aria-hidden className="shrink-0 px-1">
            ·
          </span>
          <time
            dateTime={new Date(createdAt * 1000).toISOString()}
            title={formatUnixDateTimeMedium(createdAt)}
            className="shrink-0 tabular-nums"
          >
            {formatRelativeTime(Math.min(0, createdAt - now))}
          </time>
        </>
      )}
      {note !== undefined && note !== "" && (
        <>
          <span aria-hidden className="shrink-0 px-1">
            ·
          </span>
          <span className="truncate">{note}</span>
        </>
      )}
    </p>
  );
}

/**
 * What an artifact body shows when it has nothing to show: a run with no id,
 * a page with no link, an empty document. One shape for all of them, centred
 * in the frame with the kind's own mark, so an empty artifact reads as
 * deliberate rather than as a body that failed to render.
 */
export function ArtifactNotice({ kind, children }: { kind: ArtifactKind; children: ReactNode }) {
  return (
    <div className="animate-rise flex min-h-32 flex-1 flex-col items-center justify-center gap-2 px-6 py-8 text-center">
      <span className="bg-sunken text-foreground-subtle flex size-8 items-center justify-center rounded-md">
        <ArtifactKindIcon kind={kind} className="size-4" />
      </span>
      <p className="text-muted-foreground max-w-72 text-sm">{children}</p>
    </div>
  );
}

/**
 * The frame every artifact sits in: its mark, its title with where it came
 * from beneath, its status only when it is not Ready, and the pin. Chrome is
 * the surface radius and a hairline; no shadow, so the table or the draft
 * inside is what a person looks at.
 */
export function ArtifactChrome({
  kind,
  title,
  status,
  createdAt,
  pinned,
  onPin,
  actions,
  children,
  className,
}: {
  kind: ArtifactKind;
  title: string;
  status: ArtifactStatus;
  /** Unix seconds, for the provenance line. */
  createdAt?: number;
  pinned?: boolean;
  onPin?: (pinned: boolean) => void;
  /** Kind-specific actions on the right of the header: Download, Open, Send. */
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  const t = useT();
  const titleId = useId();
  // The pin settles with a small spring when it is clicked, and only then:
  // opening an artifact that is already pinned is not an action to confirm.
  const [pinTouched, setPinTouched] = useState(false);

  return (
    <section
      data-slot="artifact"
      aria-labelledby={titleId}
      className={cn(
        "border-border bg-card flex min-h-0 min-w-0 flex-col overflow-hidden rounded-lg border",
        className,
      )}
    >
      <header className="border-border-subtle flex h-12 shrink-0 items-center gap-2.5 border-b pr-1.5 pl-3">
        <span className="bg-sunken text-foreground-muted flex size-7 shrink-0 items-center justify-center rounded-md">
          <ArtifactKindIcon kind={kind} className="size-3.5" />
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <h3 id={titleId} className="truncate text-sm leading-tight font-semibold">
            {title}
          </h3>
          <ArtifactProvenance kind={kind} createdAt={createdAt} />
        </div>
        <ArtifactStatusBadge status={status} />
        {actions}
        {onPin && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={pinned ? t("Unpin") : t("Pin")}
                  aria-pressed={pinned}
                  className={cn(
                    "shrink-0 rounded-full",
                    pinned ? "text-foreground" : "text-foreground-subtle hover:text-foreground",
                  )}
                  onClick={() => {
                    setPinTouched(true);
                    onPin(!pinned);
                  }}
                />
              }
            >
              <span
                key={pinned ? "pinned" : "loose"}
                className={cn("flex", pinTouched && "animate-confirm")}
              >
                {pinned ? <PinOffIcon className="size-3.5" /> : <Pin01Icon className="size-3.5" />}
              </span>
            </TooltipTrigger>
            <TooltipContent>{pinned ? t("Unpin") : t("Pin to the top")}</TooltipContent>
          </Tooltip>
        )}
      </header>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">{children}</div>
    </section>
  );
}
