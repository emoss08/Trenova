import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ArtifactKind, ArtifactStatus } from "@/types/assistant";
import {
  FileTextIcon,
  GitCompareArrowsIcon,
  IdCardIcon,
  InboxIcon,
  LayoutDashboardIcon,
  ListChecksIcon,
  MailIcon,
  NewspaperIcon,
  PinIcon,
  PinOffIcon,
  ReceiptTextIcon,
  ScrollTextIcon,
  TableIcon,
  type LucideIcon,
} from "lucide-react";
import type { ReactNode } from "react";

/** What each kind is called and drawn as, once, for every surface that names one. */
export const ARTIFACT_KINDS: Record<ArtifactKind, { label: string; icon: LucideIcon }> = {
  report_preview: { label: "Preview", icon: TableIcon },
  report_run: { label: "Report", icon: FileTextIcon },
  email_draft: { label: "Draft", icon: MailIcon },
  plan: { label: "Plan", icon: ListChecksIcon },
  entity_card: { label: "Record", icon: IdCardIcon },
  table_view: { label: "Table", icon: TableIcon },
  rate_explanation: { label: "Rate", icon: ReceiptTextIcon },
  dashboard_ref: { label: "Dashboard", icon: LayoutDashboardIcon },
  briefing: { label: "Briefing", icon: NewspaperIcon },
  inbound_message: { label: "Message", icon: InboxIcon },
  run_diff: { label: "Changes", icon: GitCompareArrowsIcon },
  document: { label: "Document", icon: ScrollTextIcon },
};

/** Where an artifact is, as a tone: severity, not category. */
const STATUS_TONE: Record<ArtifactStatus, "neutral" | "info" | "success" | "danger"> = {
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

export function ArtifactKindIcon({ kind, className }: { kind: ArtifactKind; className?: string }) {
  const Icon = ARTIFACT_KINDS[kind].icon;

  return <Icon className={className} aria-hidden />;
}

/**
 * The frame every artifact sits in: its kind and title on one line with
 * its status and the pin, then the body. Chrome is the surface radius and
 * a hairline; no shadow, no fill of its own, so the table or the draft
 * inside is what a person looks at.
 */
export function ArtifactChrome({
  kind,
  title,
  status,
  pinned,
  onPin,
  actions,
  children,
  className,
}: {
  kind: ArtifactKind;
  title: string;
  status: ArtifactStatus;
  pinned?: boolean;
  onPin?: (pinned: boolean) => void;
  /** Kind-specific actions on the right of the header: Download, Open, Send. */
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  const t = useT();
  const meta = ARTIFACT_KINDS[kind];

  return (
    <section
      data-slot="artifact"
      className={cn(
        "border-border bg-card flex min-h-0 min-w-0 flex-col overflow-hidden rounded-lg border",
        className,
      )}
    >
      <header className="border-border flex h-10 shrink-0 items-center gap-2 border-b pr-1.5 pl-3">
        <ArtifactKindIcon kind={kind} className="text-muted-foreground size-3.5 shrink-0" />
        <span className="text-muted-foreground shrink-0 text-xs">{t(meta.label)}</span>
        <span className="min-w-0 flex-1 truncate text-sm font-medium">{title}</span>
        {status !== "Ready" && (
          <Badge variant={STATUS_TONE[status]} className="h-4 shrink-0 px-1.5 text-2xs">
            {artifactStatusLabel(status, t)}
          </Badge>
        )}
        {actions}
        {onPin && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label={pinned ? t("Unpin") : t("Pin")}
                  aria-pressed={pinned}
                  className={cn(
                    "shrink-0",
                    pinned ? "text-foreground" : "text-muted-foreground hover:text-foreground",
                  )}
                  onClick={() => onPin(!pinned)}
                />
              }
            >
              {pinned ? <PinOffIcon className="size-3.5" /> : <PinIcon className="size-3.5" />}
            </TooltipTrigger>
            <TooltipContent>{pinned ? t("Unpin") : t("Pin to the top")}</TooltipContent>
          </Tooltip>
        )}
      </header>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">{children}</div>
    </section>
  );
}
