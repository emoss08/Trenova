import { audit } from "@/lib/queries/audit";
import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  ShieldAlertIcon,
  ShieldTickIcon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { AuditChainStatusSkeleton } from "./audit-chain-status-skeleton";

export default function AuditChainStatus() {
  const t = useT();
  const { data, isPending, isError } = useQuery(audit.chainStatus());

  if (isPending) {
    return <AuditChainStatusSkeleton />;
  }
  if (isError || !data) {
    return null;
  }

  const sealedThrough = data.sealedThrough
    ? formatToUserTimezone(data.sealedThrough, { showTimeZone: true })
    : null;
  const verifiedAt = data.lastVerifiedAt
    ? formatToUserTimezone(data.lastVerifiedAt, { showTimeZone: true })
    : null;

  if (data.lastVerificationStatus === "Broken") {
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

  if (data.lastVerificationStatus === "KeyMissing") {
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

  return (
    <Alert size="sm" variant={data.signed ? "success" : "warning"}>
      {data.signed ? <ShieldTickIcon /> : <AlertTriangleIcon />}
      <AlertTitle>
        {data.signed
          ? t("Entries are sealed with a signing key")
          : t("Entries are sealed without a signing key")}
      </AlertTitle>
      <AlertDescription>
        {sealedThrough && verifiedAt
          ? t("Sealed through {0}; last verified {1}.", sealedThrough, verifiedAt)
          : sealedThrough
            ? t("Sealed through {0}; not verified yet.", sealedThrough)
            : t("Entries are sealed every hour; nothing has been sealed yet.")}
      </AlertDescription>
    </Alert>
  );
}
