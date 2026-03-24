import { useDebounce } from "@/hooks/use-debounce";
import { apiService } from "@/services/api";
import type { Shipment } from "@/types/shipment";
import { useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export function useShipmentTotalsPreview() {
  const { control, getValues, setValue } = useFormContext<Shipment>();
  const [isCalculating, setIsCalculating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const lastHashRef = useRef<string>("");
  const abortRef = useRef<AbortController | null>(null);

  const formulaTemplateId = useWatch({ control, name: "formulaTemplateId" });
  const freightChargeAmount = useWatch({ control, name: "freightChargeAmount" });
  const additionalCharges = useWatch({ control, name: "additionalCharges" });
  const commodities = useWatch({ control, name: "commodities" });
  const moves = useWatch({ control, name: "moves" });
  const ratingUnit = useWatch({ control, name: "ratingUnit" });
  const weight = useWatch({ control, name: "weight" });
  const pieces = useWatch({ control, name: "pieces" });

  const completeCharges = additionalCharges?.filter((c) => c.accessorialChargeId);

  const watchedFieldsHash = JSON.stringify({
    formulaTemplateId,
    freightChargeAmount,
    additionalCharges: completeCharges,
    commodities,
    moves,
    ratingUnit,
    weight,
    pieces,
  });

  const debouncedHash = useDebounce(watchedFieldsHash, 500);

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
        setValue("otherChargeAmount", result.otherChargeAmount ?? null, {
          shouldDirty: false,
        });
        setValue("totalChargeAmount", result.totalChargeAmount ?? null, {
          shouldDirty: false,
        });
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
  }, [debouncedHash, getValues, setValue]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  return { isCalculating, error };
}
