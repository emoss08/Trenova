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
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { InfoIcon, RadarIcon, ShieldAlertIcon, ShieldPlusIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

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
      className="inline-flex items-center gap-1 text-xs font-medium underline underline-offset-2"
    >
      <RadarIcon className="size-3" aria-hidden />
      {t("Open carrier intelligence")}
    </Link>
  );
}

function ItemText({ item }: { item: EligibilityItem }) {
  return (
    <div className="flex min-w-0 flex-col">
      <span>{item.message}</span>
      {item.code ? (
        <span className="text-muted-foreground font-mono text-2xs">{item.code}</span>
      ) : null}
    </div>
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
    <Alert variant="destructive" data-eligibility-group="intel-blockers">
      <ShieldAlertIcon />
      <AlertTitle>{t("Carrier intelligence blocks this assignment")}</AlertTitle>
      <AlertDescription>
        <ul className="flex flex-col gap-1.5">
          {items.map((item) => {
            const code = item.code;
            return (
              <li key={item.key} className="flex items-start justify-between gap-2">
                <ItemText item={item} />
                {showOverride && code && isOverridableIntelBlocker(item) ? (
                  <Button
                    type="button"
                    size="xs"
                    variant="outline"
                    className="shrink-0"
                    isLoading={resolvingCode === code}
                    disabled={resolvingCode !== null}
                    onClick={() => void requestOverride(code)}
                  >
                    <ShieldPlusIcon />
                    {t("Grant override")}
                  </Button>
                ) : null}
              </li>
            );
          })}
        </ul>
        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1">
          <span className="text-xs">
            {canApprove
              ? t("Resolve the finding on the carrier or grant a time-boxed override to proceed.")
              : t(
                  "Resolve the finding on the carrier, or ask someone who can approve carrier intelligence overrides.",
                )}
          </span>
          <IntelligenceLink carrierId={carrierId} />
        </div>
      </AlertDescription>
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
    </Alert>
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
    <div
      role="note"
      data-eligibility-group="advisories"
      aria-label={t("Carrier intelligence advisories")}
      className="bg-muted/40 text-muted-foreground flex flex-col gap-1.5 rounded-lg border px-3 py-2 text-xs"
    >
      <div className="text-foreground flex items-center gap-1.5 font-medium">
        <InfoIcon className="size-3.5" aria-hidden />
        {t("Carrier intelligence advisories")}
      </div>
      <ul className="flex flex-col gap-1 pl-5">
        {items.map((item) => (
          <li key={item.key} className="list-disc">
            <ItemText item={item} />
          </li>
        ))}
      </ul>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span>{t("For awareness only. These do not stop the assignment.")}</span>
        <IntelligenceLink carrierId={carrierId} />
      </div>
    </div>
  );
}
