import { ComponentLoader } from "@/components/component-loader";
import { EDIPartnerReadinessBadge } from "@/components/status-badge";
import { Button } from "@/components/ui/button";
import { apiService } from "@/services/api";
import type { EDIPartner } from "@/types/edi";
import { useQuery } from "@tanstack/react-query";
import { CheckCircle2Icon, CircleIcon, TriangleAlertIcon } from "lucide-react";
import { Link } from "react-router";

const READINESS_LINKS: Record<string, { label: string; to: string } | undefined> = {
  "communication-profile": {
    label: "Create communication profile",
    to: "/edi/communication-profiles?panelType=create",
  },
  "inbound-document-profile": {
    label: "Open designer",
    to: "/edi/designer",
  },
  "outbound-document-profile": {
    label: "Open designer",
    to: "/edi/designer",
  },
  "test-case": {
    label: "Open test cases",
    to: "/edi/test-cases?panelType=create",
  },
};

const READINESS_HINTS: Record<string, string> = {
  details: "Fill in the contact email and timezone on the Details tab.",
  mappings: "Add entity mappings on the Mappings tab.",
};

export function PartnerReadinessChecklist({ partner }: { partner: EDIPartner }) {
  const partnerId = partner.id ?? "";
  const { data, isPending, isError } = useQuery({
    queryKey: ["edi-partner-readiness-detail", partnerId],
    queryFn: () => apiService.ediService.getPartnerReadiness(partnerId),
    enabled: partnerId !== "",
  });

  if (isPending) {
    return <ComponentLoader message="Checking partner readiness" />;
  }
  if (isError || !data) {
    return (
      <div className="rounded-md border bg-background p-6 text-sm text-muted-foreground">
        The readiness checklist could not be loaded.
      </div>
    );
  }

  const exchangeEnabled = partner.enabledForInbound || partner.enabledForOutbound;

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between rounded-md border bg-muted/20 p-3">
        <div className="flex items-center gap-3">
          <EDIPartnerReadinessBadge
            ready={data.ready}
            completedCount={data.completedCount}
            totalCount={data.totalCount}
          />
          <p className="text-sm text-muted-foreground">
            {data.ready
              ? "This partner has completed every onboarding step."
              : `${data.totalCount - data.completedCount} onboarding step(s) remaining before this partner is production-ready.`}
          </p>
        </div>
      </div>
      {!data.ready && exchangeEnabled && (
        <div className="flex items-start gap-2 rounded-md border border-yellow-600/30 bg-yellow-600/10 p-3 text-sm text-yellow-800 dark:text-yellow-300">
          <TriangleAlertIcon className="mt-0.5 size-4 shrink-0" />
          <p>
            This partner is enabled for {partner.enabledForInbound ? "inbound" : ""}
            {partner.enabledForInbound && partner.enabledForOutbound ? " and " : ""}
            {partner.enabledForOutbound ? "outbound" : ""} exchange while the checklist is
            incomplete. Documents may fail to generate, deliver, or map until the remaining steps
            are finished.
          </p>
        </div>
      )}
      <div className="flex flex-col divide-y rounded-md border bg-background">
        {data.items.map((item) => {
          const link = item.complete ? undefined : READINESS_LINKS[item.key];
          const hint = item.complete ? undefined : READINESS_HINTS[item.key];
          return (
            <div key={item.key} className="flex items-center gap-3 p-3">
              {item.complete ? (
                <CheckCircle2Icon className="size-4 shrink-0 text-green-600 dark:text-green-400" />
              ) : (
                <CircleIcon className="size-4 shrink-0 text-muted-foreground" />
              )}
              <div className="min-w-0 flex-1">
                <p className="text-sm">{item.label}</p>
                {hint && <p className="text-xs text-muted-foreground">{hint}</p>}
              </div>
              {link && (
                <Button
                  variant="outline"
                  size="sm"
                  className="shrink-0"
                  render={<Link to={link.to} />}
                >
                  {link.label}
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
