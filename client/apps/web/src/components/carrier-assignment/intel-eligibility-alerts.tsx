import { useT } from "@trenova/shared/i18n/use-t";
import { GrantOverrideDialog } from "@/components/carrier-intelligence/grant-override-dialog";
import { usePermission } from "@/hooks/use-permission";
import { isOverridableIntelBlocker, type EligibilityItem } from "@/lib/carrier-eligibility";
import { carrierPanelPath } from "@/lib/carrier-links";
import { canOverrideFinding } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTELLIGENCE_KEY,
  fetchCarrierIntelligence,
  type CarrierIntelFinding,
} from "@/lib/graphql/carrier-intelligence";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  ArrowUpRightIcon,
  InfoIcon,
  OctagonXIcon,
  TriangleAlertIcon,
  type LucideIcon,
} from "lucide-react";
import { useCallback, useState, type ReactNode } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

export type EligibilityTone = "blocker" | "warning" | "advisory";

const TONE_ICON: Record<EligibilityTone, { icon: LucideIcon; className: string }> = {
  blocker: { icon: OctagonXIcon, className: "text-danger-foreground" },
  warning: { icon: TriangleAlertIcon, className: "text-warning-foreground" },
  advisory: { icon: InfoIcon, className: "text-muted-foreground" },
};

export type EligibilityRow = {
  key: string;
  message: string;
  action?: ReactNode;
};

export type EligibilityGroupProps = {
  group: string;
  tone: EligibilityTone;
  title: string;
  rows: readonly EligibilityRow[];
  role?: "alert" | "note";
  footer?: ReactNode;
  children?: ReactNode;
  className?: string;
};

export function EligibilityGroup({
  group,
  tone,
  title,
  rows,
  role = "alert",
  footer,
  children,
  className,
}: EligibilityGroupProps) {
  const { icon: Icon, className: iconClassName } = TONE_ICON[tone];

  return (
    <section
      role={role}
      aria-label={title}
      data-eligibility-group={group}
      className={cn("flex flex-col gap-1 border-b pb-3 last:border-b-0 last:pb-0", className)}
    >
      <div className="flex items-center gap-2 text-sm font-medium">
        <Icon className={cn("size-3.5 shrink-0", iconClassName)} aria-hidden />
        {title}
      </div>
      <ul className="flex flex-col pl-5.5">
        {rows.map((row) => (
          <li key={row.key} className="flex min-h-7 items-center justify-between gap-2 text-sm">
            <span
              className={cn("min-w-0", tone === "advisory" ? "text-muted-foreground text-xs" : "")}
            >
              {row.message}
            </span>
            {row.action ? <span className="shrink-0">{row.action}</span> : null}
          </li>
        ))}
      </ul>
      {children ? <div className="pl-5.5">{children}</div> : null}
      {footer ? (
        <div className="text-muted-foreground flex flex-wrap items-center gap-x-3 gap-y-1 pl-5.5 text-xs">
          {footer}
        </div>
      ) : null}
    </section>
  );
}

function IntelligenceLink({ carrierId }: { carrierId: string | undefined }) {
  const t = useT();

  if (!carrierId) {
    return null;
  }

  return (
    <Link
      to={carrierPanelPath(carrierId, "intelligence")}
      target="_blank"
      rel="noopener noreferrer"
      className="text-foreground inline-flex items-center gap-0.5 underline-offset-2 hover:underline"
    >
      {t("Open carrier intelligence")}
      <ArrowUpRightIcon className="size-3" aria-hidden />
    </Link>
  );
}

export function IntelBlockersAlert({
  items,
  carrierId,
  onEligibilityChanged,
}: {
  items: readonly EligibilityItem[];
  carrierId: string | undefined;
  onEligibilityChanged?: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canApprove } = usePermission(Resource.CarrierIntelligence, Operation.Approve);
  const [resolvingCode, setResolvingCode] = useState<string | null>(null);
  const [granting, setGranting] = useState<CarrierIntelFinding | null>(null);

  const requestOverride = useCallback(
    async (code: string) => {
      if (!carrierId) {
        return;
      }
      setResolvingCode(code);
      try {
        const result = await queryClient.fetchQuery({
          queryKey: [CARRIER_INTELLIGENCE_KEY, carrierId],
          queryFn: ({ signal }) => fetchCarrierIntelligence(carrierId, { signal }),
        });
        const finding = result.carrier?.intelligence?.findings.find(
          (candidate) => candidate.code === code && canOverrideFinding(candidate),
        );
        if (!finding) {
          toast.error(t("This finding is no longer on the carrier's latest vetting"), {
            description: t("The eligibility check has been refreshed."),
          });
          onEligibilityChanged?.();
          return;
        }
        setGranting(finding);
      } catch (error) {
        toast.error(t("Carrier intelligence could not be loaded"), {
          description: error instanceof Error ? error.message : undefined,
        });
      } finally {
        setResolvingCode(null);
      }
    },
    [carrierId, onEligibilityChanged, queryClient, t],
  );

  const handleGranted = useCallback(() => {
    if (carrierId) {
      void queryClient.invalidateQueries({ queryKey: [CARRIER_INTELLIGENCE_KEY, carrierId] });
    }
    onEligibilityChanged?.();
  }, [carrierId, onEligibilityChanged, queryClient]);

  if (items.length === 0) {
    return null;
  }

  const showOverride = canApprove && !!carrierId;

  return (
    <>
      <EligibilityGroup
        group="intel-blockers"
        tone="blocker"
        title={t("Carrier intelligence blocks this assignment")}
        rows={items.map((item) => {
          const code = item.code;
          return {
            key: item.key,
            message: item.message,
            action:
              showOverride && code && isOverridableIntelBlocker(item) ? (
                <Button
                  type="button"
                  size="xs"
                  variant="ghost"
                  isLoading={resolvingCode === code}
                  disabled={resolvingCode !== null}
                  onClick={() => void requestOverride(code)}
                >
                  {t("Grant override")}
                </Button>
              ) : null,
          };
        })}
        footer={
          <>
            <span>
              {canApprove
                ? t("Resolve the finding on the carrier or grant a time-boxed override to proceed.")
                : t(
                    "Resolve the finding on the carrier, or ask someone who can approve carrier intelligence overrides.",
                  )}
            </span>
            <IntelligenceLink carrierId={carrierId} />
          </>
        }
      />
      {carrierId ? (
        <GrantOverrideDialog
          carrierId={carrierId}
          finding={granting}
          open={granting !== null}
          onOpenChange={(open) => {
            if (!open) {
              setGranting(null);
            }
          }}
          onGranted={handleGranted}
        />
      ) : null}
    </>
  );
}

export function IntelAdvisoriesCallout({
  items,
  carrierId,
}: {
  items: readonly EligibilityItem[];
  carrierId: string | undefined;
}) {
  const t = useT();

  if (items.length === 0) {
    return null;
  }

  return (
    <EligibilityGroup
      group="advisories"
      tone="advisory"
      role="note"
      title={t("Carrier intelligence advisories")}
      rows={items.map((item) => ({ key: item.key, message: item.message }))}
      footer={
        <>
          <span>{t("For awareness only. These do not stop the assignment.")}</span>
          <IntelligenceLink carrierId={carrierId} />
        </>
      }
    />
  );
}
