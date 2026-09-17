import { useT } from "@trenova/shared/i18n/use-t";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  availableVetDepths,
  carrierIntelProviderLabel,
  carrierIntelVetCost,
} from "@/lib/carrier-intelligence";
import type { CarrierIntelProviderInfo } from "@/lib/graphql/carrier-intel-settings";
import { vetCarrier, type CarrierIntelVetResult } from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelDepth } from "@trenova/graphql/generated/graphql";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Label } from "@trenova/shared/components/ui/label";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Switch } from "@trenova/shared/components/ui/switch";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { CoinsIcon } from "lucide-react";
import { useEffect, useId, useMemo, useState } from "react";
import { toast } from "sonner";

export type VetCarrierDialogProps = {
  carrierId: string;
  provider: CarrierIntelProviderInfo;
  lastDepth: CarrierIntelDepth | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onVetted: (result: CarrierIntelVetResult) => void;
};

export function VetCarrierDialog({
  carrierId,
  provider,
  lastDepth,
  open,
  onOpenChange,
  onVetted,
}: VetCarrierDialogProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const forceId = useId();
  const depths = useMemo(() => availableVetDepths(provider.capabilities), [provider.capabilities]);
  const preferredDepth = lastDepth && depths.includes(lastDepth) ? lastDepth : (depths[0] ?? null);
  const [depth, setDepth] = useState<CarrierIntelDepth | null>(preferredDepth);
  const [force, setForce] = useState(false);
  const providerName = carrierIntelProviderLabel(provider.provider);

  useEffect(() => {
    if (open) {
      setDepth(preferredDepth);
      setForce(false);
    }
  }, [open, preferredDepth]);

  const cost = depth ? carrierIntelVetCost(provider.provider, depth) : null;

  const vet = useApiMutation<CarrierIntelVetResult, void>({
    resourceName: "Carrier vetting",
    mutationFn: () => vetCarrier({ carrierId, depth, force }),
    onSuccess: (result) => {
      const details: string[] = [];
      if (result.fromCache) {
        details.push(t("A fresh snapshot was already on file, so no provider call was made."));
      }
      if (result.usedFallback) {
        details.push(
          t("{0} was unavailable; the fallback provider answered instead.", providerName),
        );
      }
      if (result.changeCount > 0 || result.raisedCount > 0) {
        details.push(
          t(
            "{0, plural, one {# change} other {# changes}} detected, {1, plural, one {# event} other {# events}} raised.",
            result.changeCount,
            result.raisedCount,
          ),
        );
      }
      toast.success(result.fromCache ? t("Vetting is current") : t("Carrier vetted"), {
        description: details.length > 0 ? details.join(" ") : undefined,
      });
      onVetted(result);
      onOpenChange(false);
    },
  });

  const depthItems = depths.map((value) => ({
    value,
    label: labels.depth[value],
  }));

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Vet carrier")}</DialogTitle>
          <DialogDescription>
            {t(
              "Pulls the carrier's profile from {0}, re-evaluates every rule and records what changed.",
              providerName,
            )}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          {depths.length > 0 && depth ? (
            <div className="flex flex-col gap-1.5">
              <Label>{t("Depth")}</Label>
              <SegmentedControl<CarrierIntelDepth>
                items={depthItems}
                value={depth}
                onValueChange={setDepth}
                fullWidth
                aria-label={t("Vetting depth")}
              />
              <p className="text-muted-foreground text-xs">{labels.depthHint[depth]}</p>
            </div>
          ) : (
            <p className="text-muted-foreground text-xs">
              {t(
                "{0} does not advertise a profile lookup. The default depth is used.",
                providerName,
              )}
            </p>
          )}
          <div className="flex items-start justify-between gap-3">
            <div className="flex flex-col gap-0.5">
              <Label htmlFor={forceId}>{t("Force a fresh pull")}</Label>
              <p className="text-muted-foreground text-xs">
                {t(
                  "Skips the snapshot cache and always calls the provider, even when the current snapshot is still fresh.",
                )}
              </p>
            </div>
            <Switch id={forceId} checked={force} onCheckedChange={setForce} />
          </div>
          {cost ? (
            <div className="text-muted-foreground flex items-start gap-2 border-t pt-3 text-xs">
              <CoinsIcon className="mt-0.5 size-3.5 shrink-0" aria-hidden />
              <span>
                {cost.basis === "Free"
                  ? t("{0} does not charge per lookup.", providerName)
                  : cost.basis === "PerDOTMonth"
                    ? t(
                        "About {0} per carrier per month at this depth. Pulling the same carrier again this month is not billed twice.",
                        formatCurrency(cost.amount),
                      )
                    : t("About {0} per matched lookup at this depth.", formatCurrency(cost.amount))}
                {cost.basis !== "Free" && force
                  ? ` ${t("Forcing a pull may incur a charge even when the cache would have answered.")}`
                  : null}
              </span>
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button type="button" isLoading={vet.isPending} onClick={() => vet.mutate()}>
            {t("Vet now")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
