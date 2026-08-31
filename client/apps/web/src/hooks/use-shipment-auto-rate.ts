import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { canResolveContract, toRatingPreviewPayload } from "@/lib/shipment-rating-payload";
import { apiService } from "@/services/api";
import type { AccessorialCharge } from "@trenova/shared/types/accessorial-charge";
import type { AdditionalCharge, ContractRate, Shipment } from "@trenova/shared/types/shipment";
import { useCallback, useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The lane a contract is resolved by. Nothing else about a shipment changes
 * which agreement covers it, so a second lookup for the same lane would return
 * the same contract and overwrite whatever the rater has typed since.
 */
function laneSignature(values: Shipment): string {
  const stops = (values.moves ?? []).flatMap((move) => move.stops ?? []);

  return JSON.stringify({
    customerId: values.customerId,
    serviceTypeId: values.serviceTypeId,
    shipmentTypeId: values.shipmentTypeId,
    locations: stops.map((stop) => stop.locationId),
  });
}

/**
 * Applies a rate agreement to a new shipment, once.
 *
 * The moment a shipment being typed holds enough to be priced — a customer, a
 * service and shipment type, and both ends of the lane — the contracts are
 * asked what they charge, and what they answer is written into the shipment's
 * own rating method, base rate and accessorial rows. From then on those are
 * ordinary fields: the rater can change any of them, and nothing asks the
 * contract again unless they press Auto-rate.
 *
 * It runs on the create form only. An existing shipment already carries a rate
 * that somebody may have negotiated, and quietly replacing it on open is the
 * behaviour this whole design exists to remove.
 */
export function useShipmentAutoRate({ enabled }: { enabled: boolean }) {
  const { control, getValues, setValue } = useFormContext<Shipment>();
  const [applied, setApplied] = useState<ContractRate | null>(null);
  const [isRating, setIsRating] = useState(false);
  const attemptedRef = useRef<string>("");
  const abortRef = useRef<AbortController | null>(null);

  const customerId = useWatch({ control, name: "customerId" });
  const serviceTypeId = useWatch({ control, name: "serviceTypeId" });
  const shipmentTypeId = useWatch({ control, name: "shipmentTypeId" });
  const moves = useWatch({ control, name: "moves" });

  const lane = JSON.stringify({
    customerId,
    serviceTypeId,
    shipmentTypeId,
    locations: (moves ?? []).flatMap((move) => (move.stops ?? []).map((stop) => stop.locationId)),
  });

  const debouncedLane = useDebounce(lane, 500);

  const applyContractRate = useCallback(
    (rate: ContractRate) => {
      if (!rate.applied) {
        return;
      }

      if (rate.formulaTemplateId) {
        setValue("formulaTemplateId", rate.formulaTemplateId, { shouldDirty: true });
      }

      // A rule that binds no rate of its own prices through whatever the
      // shipment already carries, so there is nothing to seat.
      if (rate.baseRate) {
        setValue("baseRate", Number(rate.baseRate), { shouldDirty: true });
      }

      const current = getValues("additionalCharges") ?? [];
      const contractRows: AdditionalCharge[] = rate.accessorials.map((accessorial) => ({
        accessorialChargeId: accessorial.accessorialChargeId,
        accessorialCharge: {
          description: accessorial.description,
        } as AccessorialCharge,
        isSystemGenerated: true,
        method: accessorial.method,
        amount: Number(accessorial.amount) || 0,
        unit: accessorial.unit || 1,
      })) as AdditionalCharge[];

      // Rows the rater added themselves are kept. Only the contract's own are
      // replaced, and on a new shipment there are none to replace yet.
      const keep = current.filter(
        (charge) =>
          !contractRows.some((row) => row.accessorialChargeId === charge.accessorialChargeId),
      );

      if (contractRows.length > 0) {
        setValue("additionalCharges", [...keep, ...contractRows], { shouldDirty: true });
      }

      setValue("autoRated", true, { shouldDirty: false });
      setValue("rateAgreementId", rate.agreementId ?? null, { shouldDirty: false });
      setValue("rateAgreementRuleId", rate.ruleId ?? null, { shouldDirty: false });

      setApplied(rate);
    },
    [getValues, setValue],
  );

  useEffect(() => {
    if (!enabled) {
      return;
    }

    // Both ends of the lane decide the contract, so nothing is asked until both
    // are known — a contract chosen from half a lane is the wrong contract.
    const values = getValues();
    if (!canResolveContract(values)) {
      return;
    }

    const signature = laneSignature(values);
    if (signature === attemptedRef.current) {
      return;
    }
    attemptedRef.current = signature;

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setIsRating(true);

    apiService.shipmentService
      .previewContractRate(toRatingPreviewPayload(values))
      .then((rate) => {
        if (controller.signal.aborted) return;
        applyContractRate(rate);
      })
      .catch(() => {
        // A lane no contract covers is the normal case for an organization
        // that has not written its agreements yet, and it is not something to
        // interrupt somebody typing a shipment with. The rater picks a rating
        // method by hand, exactly as before.
        attemptedRef.current = "";
      })
      .finally(() => {
        if (abortRef.current === controller) {
          setIsRating(false);
        }
      });
  }, [enabled, debouncedLane, getValues, applyContractRate]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return {
    isRating,
    appliedRate: applied,
    dismissAppliedRate: useCallback(() => setApplied(null), []),
  };
}
