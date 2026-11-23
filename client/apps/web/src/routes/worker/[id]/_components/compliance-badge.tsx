import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ComplianceStatus } from "@trenova/shared/types/worker";

const COMPLIANCE_TONE: Readonly<Record<ComplianceStatus, BadgeVariant>> = {
  Compliant: "success",
  Pending: "warning",
  NonCompliant: "danger",
};

/** A worker's compliance in words. */
export function complianceLabel(status: ComplianceStatus, t: TranslateFn): string {
  switch (status) {
    case "Compliant":
      return t("Compliant");
    case "Pending":
      return t("Pending review");
    case "NonCompliant":
      return t("Non-compliant");
  }
}

/** A worker's compliance as a state badge, the same on every tab. */
export function ComplianceBadge({ status }: { status: ComplianceStatus }) {
  const t = useT();
  return (
    <Badge variant={COMPLIANCE_TONE[status]} dot>
      {complianceLabel(status, t)}
    </Badge>
  );
}
