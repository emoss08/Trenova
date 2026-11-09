import { audit } from "@/lib/queries/audit";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  ShieldAlertIcon,
  ShieldTickIcon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";

type SealTone = "success" | "warning" | "danger";

const TONE_CLASS: Record<SealTone, string> = {
  success: "border-success-border bg-success-subtle text-success-foreground",
  warning: "border-warning-border bg-warning-subtle text-warning-foreground",
  danger: "border-danger-border bg-danger-subtle text-danger-foreground",
};

const TONE_ICON: Record<SealTone, IconComponent> = {
  success: ShieldTickIcon,
  warning: AlertTriangleIcon,
  danger: ShieldAlertIcon,
};

/**
 * Whether the trail is intact, beside the page's title: a quiet chip while the seals verify, which
 * turns red or amber when they do not. The full story is in its tooltip; a failure also raises
 * AuditSealAlert above the page.
 */
export function AuditSealChip() {
  const t = useT();
  const now = useNow("minute");
  const { data, isPending, isError } = useQuery(audit.chainStatus());

  if (isPending) {
    return (
      <span
        aria-busy
        aria-label={t("Checking the audit log's seals")}
        className="h-6 w-40 ui-shimmer rounded-full"
      />
    );
  }
  if (isError || !data) {
    return null;
  }

  const verifiedAgo = data.lastVerifiedAt ? formatRelativeTime(data.lastVerifiedAt - now) : null;
  const sealedThrough = data.sealedThrough
    ? formatToUserTimezone(data.sealedThrough, { showTimeZone: true })
    : null;

  let tone: SealTone = data.signed ? "success" : "warning";
  let label = verifiedAgo
    ? t("Sealed · verified {0}", verifiedAgo)
    : t("Sealed · not verified yet");
  if (!data.signed && data.lastVerificationStatus !== "Broken") {
    label = t("Sealed without a signing key");
  }
  if (data.lastVerificationStatus === "Broken") {
    tone = "danger";
    label = t("Verification failed");
  } else if (data.lastVerificationStatus === "KeyMissing") {
    tone = "warning";
    label = t("A sealing key is missing");
  }
  const Icon = TONE_ICON[tone];

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            tabIndex={0}
            className={cn(
              "inline-flex h-6 items-center gap-1.5 rounded-full border ui-focus-ring px-2 text-xs font-medium",
              TONE_CLASS[tone],
            )}
          />
        }
      >
        <Icon aria-hidden className="size-3" />
        {label}
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">
        {sealedThrough
          ? t(
              "Entries are sealed every hour. Everything written before {0} is sealed, and the seals are checked every day.",
              sealedThrough,
            )
          : t("Entries are sealed every hour; nothing has been sealed yet.")}
      </TooltipContent>
    </Tooltip>
  );
}

/** A failed verification or a missing key, above the page, so no one relies on the trail unawares. */
export function AuditSealAlert() {
  const t = useT();
  const { data } = useQuery(audit.chainStatus());

  if (data?.lastVerificationStatus === "Broken") {
    return (
      <Alert size="sm" variant="destructive">
        <ShieldAlertIcon />
        <AlertTitle>{t("The audit log failed verification")}</AlertTitle>
        <AlertDescription>
          {data.lastVerificationDetail ||
            t(
              "An entry or seal no longer matches what was sealed. Investigate before relying on it.",
            )}
        </AlertDescription>
      </Alert>
    );
  }

  if (data?.lastVerificationStatus === "KeyMissing") {
    return (
      <Alert size="sm" variant="warning">
        <AlertTriangleIcon />
        <AlertTitle>{t("A sealing key is missing")}</AlertTitle>
        <AlertDescription>
          {data.lastVerificationDetail ||
            t(
              "Seals were signed with a key this deployment no longer has, so they cannot be checked.",
            )}
        </AlertDescription>
      </Alert>
    );
  }

  return null;
}
