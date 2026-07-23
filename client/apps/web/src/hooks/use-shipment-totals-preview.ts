import { useDebounce } from "@/hooks/use-debounce";
import { apiService } from "@/services/api";
import type {
  AdditionalCharge,
  Shipment,
  ShipmentTotalsFuelSurcharge,
  ShipmentTotalsResponse,
} from "@trenova/shared/types/shipment";
import { useCallback, useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

function isGeneratedFuelCharge(charge: AdditionalCharge | undefined | null): boolean {
  return !!charge?.isSystemGenerated && !!charge?.fuelSurchargeProgramId;
}

export type FuelSurchargeChange = {
  previousAmount: number;
  nextAmount: number;
};

type PendingFuelChange = FuelSurchargeChange & {
  totals: ShipmentTotalsResponse;
};

export function useShipmentTotalsPreview() {
  const { control, getValues, setValue } = useFormContext<Shipment>();
  const [isCalculating, setIsCalculating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pendingFuelChange, setPendingFuelChange] = useState<PendingFuelChange | null>(null);
  const lastHashRef = useRef<string>("");
  const abortRef = useRef<AbortController | null>(null);

  const customerId = useWatch({ control, name: "customerId" });
  const formulaTemplateId = useWatch({ control, name: "formulaTemplateId" });
  const baseRate = useWatch({ control, name: "baseRate" });
  const additionalCharges = useWatch({ control, name: "additionalCharges" });
  const commodities = useWatch({ control, name: "commodities" });
  const moves = useWatch({ control, name: "moves" });
  const ratingUnit = useWatch({ control, name: "ratingUnit" });
  const weight = useWatch({ control, name: "weight" });
  const pieces = useWatch({ control, name: "pieces" });
  const fuelSurchargeLocked = useWatch({ control, name: "fuelSurchargeLocked" });

  // The generated fuel surcharge row is our own output — hashing it would
  // retrigger a fetch every time we sync it back into the form.
  const completeCharges = additionalCharges?.filter(
    (charge) => charge.accessorialChargeId && !isGeneratedFuelCharge(charge),
  );

  const watchedFieldsHash = JSON.stringify({
    customerId,
    formulaTemplateId,
    baseRate,
    additionalCharges: completeCharges,
    commodities,
    moves,
    ratingUnit,
    weight,
    pieces,
    fuelSurchargeLocked,
  });

  const debouncedHash = useDebounce(watchedFieldsHash, 500);

  const applyTotals = useCallback(
    (result: ShipmentTotalsResponse) => {
      setValue("freightChargeAmount", result.freightChargeAmount ?? null, {
        shouldDirty: false,
      });
      setValue("otherChargeAmount", result.otherChargeAmount ?? null, {
        shouldDirty: false,
      });
      setValue("totalChargeAmount", result.totalChargeAmount ?? null, {
        shouldDirty: false,
      });
    },
    [setValue],
  );

  const applyFuelSurchargeRow = useCallback(
    (fuelSurcharge: ShipmentTotalsFuelSurcharge | null | undefined) => {
      const current = getValues("additionalCharges") ?? [];
      const existing = current.find((charge) => isGeneratedFuelCharge(charge));

      if (!fuelSurcharge) {
        if (existing) {
          setValue(
            "additionalCharges",
            current.filter((charge) => !isGeneratedFuelCharge(charge)),
            { shouldDirty: false },
          );
        }
        return;
      }

      const amount = Number(fuelSurcharge.amount) || 0;
      const unchanged =
        existing &&
        existing.accessorialChargeId === fuelSurcharge.accessorialChargeId &&
        (existing.fuelSurchargeProgramId ?? null) ===
          (fuelSurcharge.fuelSurchargeProgramId ?? null) &&
        Number(existing.amount) === amount;
      if (unchanged) return;

      const row: AdditionalCharge = {
        ...existing,
        accessorialChargeId: fuelSurcharge.accessorialChargeId,
        isSystemGenerated: true,
        method: fuelSurcharge.method,
        amount,
        unit: fuelSurcharge.unit ?? 1,
        fuelSurchargeProgramId: fuelSurcharge.fuelSurchargeProgramId ?? null,
        fuelSurchargeDetail: fuelSurcharge.fuelSurchargeDetail ?? null,
      };

      setValue(
        "additionalCharges",
        [...current.filter((charge) => !isGeneratedFuelCharge(charge)), row],
        { shouldDirty: false },
      );
    },
    [getValues, setValue],
  );

  const handleResult = useCallback(
    (result: ShipmentTotalsResponse) => {
      const locked = getValues("fuelSurchargeLocked");
      if (locked) {
        // The server keeps the locked row as-is; only totals can move.
        applyTotals(result);
        setPendingFuelChange(null);
        return;
      }

      const current = getValues("additionalCharges") ?? [];
      const existing = current.find((charge) => isGeneratedFuelCharge(charge));
      const nextAmount = result.fuelSurcharge ? Number(result.fuelSurcharge.amount) || 0 : 0;
      const previousAmount = existing ? Number(existing.amount) || 0 : 0;

      // A previously saved fuel surcharge re-rated to a different amount —
      // let the user decide instead of silently replacing it.
      const needsDecision =
        !!existing?.id && !!result.fuelSurcharge && previousAmount !== nextAmount;

      if (needsDecision) {
        setPendingFuelChange({ previousAmount, nextAmount, totals: result });
        return;
      }

      setPendingFuelChange(null);
      applyTotals(result);
      applyFuelSurchargeRow(result.fuelSurcharge);
    },
    [getValues, applyTotals, applyFuelSurchargeRow],
  );

  const resolveFuelSurchargeChange = useCallback(
    (action: "replace" | "keep" | "dismiss") => {
      setPendingFuelChange((pending) => {
        if (!pending) return null;

        switch (action) {
          case "replace":
            applyTotals(pending.totals);
            applyFuelSurchargeRow(pending.totals.fuelSurcharge);
            break;
          case "keep":
            // Locking re-runs the preview (the lock is part of the request
            // hash), which restores totals based on the kept amount.
            setValue("fuelSurchargeLocked", true, { shouldDirty: true });
            break;
          default:
            // Decide later — leave the form untouched and allow the next
            // change to prompt again.
            lastHashRef.current = "";
            break;
        }
        return null;
      });
    },
    [applyTotals, applyFuelSurchargeRow, setValue],
  );

  useEffect(() => {
    const parsed = JSON.parse(debouncedHash);
    if (!parsed.formulaTemplateId) {
      lastHashRef.current = "";
      setIsCalculating(false);
      setError(null);
      return;
    }

    if (debouncedHash === lastHashRef.current) return;
    lastHashRef.current = debouncedHash;

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    setIsCalculating(true);
    setError(null);

    const values = getValues();
    const payload = {
      ...values,
      additionalCharges: values.additionalCharges?.map(
        // oxlint-disable-next-line no-unused-vars
        ({ accessorialCharge, ...rest }) => rest,
      ),
      commodities: values.commodities?.map(
        // oxlint-disable-next-line no-unused-vars
        ({ commodity, ...rest }) => rest,
      ),
    };

    apiService.shipmentService
      .calculateTotals(payload as Shipment, controller.signal)
      .then((result) => {
        if (controller.signal.aborted) return;
        handleResult(result);
        setError(null);
      })
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        lastHashRef.current = "";
        setError(err instanceof Error ? err.message : "Failed to calculate totals");
      })
      .finally(() => {
        if (abortRef.current === controller) {
          setIsCalculating(false);
        }
      });
  }, [debouncedHash, getValues, handleResult]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return {
    isCalculating,
    error,
    fuelSurchargeChange: pendingFuelChange
      ? {
          previousAmount: pendingFuelChange.previousAmount,
          nextAmount: pendingFuelChange.nextAmount,
        }
      : null,
    resolveFuelSurchargeChange,
  };
}
