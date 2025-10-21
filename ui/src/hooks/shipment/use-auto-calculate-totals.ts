import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import { useFormContext } from "react-hook-form";

interface UseAutoCalculateTotalsOptions {
  /**
   * Enable/disable automatic calculation. Default: true
   */
  enabled?: boolean;
  /**
   * Debounce delay in milliseconds before triggering calculation.
   * Default: 3000ms (3 seconds) - balances responsiveness with avoiding
   * interruptions during active typing/editing.
   *
   * Adjust based on your use case:
   * - Shorter (1000-2000ms): More responsive, but may interrupt typing
   * - Longer (4000-5000ms): Less interruption, but slower feedback
   */
  debounceMs?: number;
  /**
   * Callback when calculation succeeds
   */
  onSuccess?: (data: {
    baseCharge: string | number;
    otherChargeAmount: string | number;
    totalChargeAmount: string | number;
  }) => void;
  /**
   * Callback when calculation fails
   */
  onError?: (error: Error) => void;
}

export function useAutoCalculateTotals({
  enabled = true,
  debounceMs = 3000, // 3 second default - gives user time to finish typing without interruption
  onSuccess,
  onError,
}: UseAutoCalculateTotalsOptions = {}) {
  const { setValue, getValues, subscribe } = useFormContext<ShipmentSchema>();
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastCalculatedHash = useRef<string | null>(null);
  const isCalculatingRef = useRef(false);

  const createFieldHash = useCallback((values: Partial<ShipmentSchema>) => {
    const relevantData = {
      ratingMethod: values.ratingMethod,
      ratingUnit: values.ratingUnit,
      freightChargeAmount: values.freightChargeAmount,
      additionalCharges: values.additionalCharges,
      moves: values.moves,
      commodities: values.commodities,
      serviceTypeId: values.serviceTypeId,
      shipmentTypeId: values.shipmentTypeId,
      customerId: values.customerId,
      weight: values.weight,
      pieces: values.pieces,
    };
    return JSON.stringify(relevantData);
  }, []);

  const { mutate: calculateTotals, isPending } = useMutation({
    mutationFn: async (values: Partial<ShipmentSchema>) => {
      return await api.shipments.calculateTotals(values);
    },
    onSuccess: (data) => {
      isCalculatingRef.current = false;

      const currentOtherCharge = getValues("otherChargeAmount");
      const currentTotalCharge = getValues("totalChargeAmount");
      const newOtherCharge = Number(data.otherChargeAmount);
      const newTotalCharge = Number(data.totalChargeAmount);

      // Batch updates to minimize re-renders and avoid focus issues
      const hasOtherChargeChanged = currentOtherCharge !== newOtherCharge;
      const hasTotalChargeChanged = currentTotalCharge !== newTotalCharge;

      if (hasOtherChargeChanged || hasTotalChargeChanged) {
        // Use requestAnimationFrame to batch updates and avoid interrupting user input
        requestAnimationFrame(() => {
          if (hasOtherChargeChanged) {
            setValue("otherChargeAmount", newOtherCharge, {
              shouldValidate: false,
              shouldDirty: true,
              shouldTouch: false, // Don't mark as touched to avoid validation triggers
            });
          }

          if (hasTotalChargeChanged) {
            setValue("totalChargeAmount", newTotalCharge, {
              shouldValidate: false,
              shouldDirty: true,
              shouldTouch: false, // Don't mark as touched to avoid validation triggers
            });
          }
        });
      }

      onSuccess?.(data);
    },
    onError: (error) => {
      isCalculatingRef.current = false;
      console.error("Failed to calculate totals:", error);
      onError?.(error as Error);
    },
  });

  const calculateTotalsRef = useRef(calculateTotals);
  const enabledRef = useRef(enabled);
  const debounceMsRef = useRef(debounceMs);

  useEffect(() => {
    calculateTotalsRef.current = calculateTotals;
    enabledRef.current = enabled;
    debounceMsRef.current = debounceMs;
  });

  useEffect(() => {
    const unsubscribe = subscribe({
      formState: {
        values: true,
      },
      callback: ({ values }) => {
        if (!enabledRef.current || isCalculatingRef.current) {
          return;
        }

        // Create a hash of relevant fields to detect actual changes
        const currentHash = createFieldHash(values as Partial<ShipmentSchema>);

        // Skip if nothing actually changed (prevents unnecessary calculations)
        if (currentHash === lastCalculatedHash.current) {
          return;
        }

        // Clear any pending calculation (implements debounce)
        if (debounceTimer.current) {
          clearTimeout(debounceTimer.current);
        }

        // Update hash immediately to prevent duplicate calculations during debounce
        // This ensures only the final value after user stops typing gets calculated
        lastCalculatedHash.current = currentHash;

        // Start debounce timer - calculation fires after user stops making changes
        debounceTimer.current = setTimeout(() => {
          const currentValues = values as Partial<ShipmentSchema>;
          const calculationPayload: Partial<ShipmentSchema> = {
            ratingMethod: currentValues.ratingMethod,
            ratingUnit: currentValues.ratingUnit,
            freightChargeAmount: currentValues.freightChargeAmount,
            otherChargeAmount: currentValues.otherChargeAmount,
            additionalCharges: currentValues.additionalCharges ?? undefined,
            moves: currentValues.moves,
            commodities: currentValues.commodities,
            serviceTypeId: currentValues.serviceTypeId,
            shipmentTypeId: currentValues.shipmentTypeId,
            customerId: currentValues.customerId,
            weight: currentValues.weight,
            pieces: currentValues.pieces,
            formulaTemplateId: currentValues.formulaTemplateId,
          };

          isCalculatingRef.current = true;
          calculateTotalsRef.current(calculationPayload);
        }, debounceMsRef.current);
      },
    });

    return () => {
      unsubscribe();
      if (debounceTimer.current) {
        clearTimeout(debounceTimer.current);
      }
    };
  }, [subscribe, createFieldHash]);

  const triggerCalculation = useCallback(() => {
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
    }

    if (isCalculatingRef.current) {
      return;
    }

    const currentValues = getValues();
    const calculationPayload: Partial<ShipmentSchema> = {
      ratingMethod: currentValues.ratingMethod,
      ratingUnit: currentValues.ratingUnit,
      freightChargeAmount: currentValues.freightChargeAmount,
      otherChargeAmount: currentValues.otherChargeAmount,
      additionalCharges: currentValues.additionalCharges ?? undefined,
      moves: currentValues.moves,
      commodities: currentValues.commodities,
      serviceTypeId: currentValues.serviceTypeId,
      shipmentTypeId: currentValues.shipmentTypeId,
      customerId: currentValues.customerId,
      weight: currentValues.weight,
      pieces: currentValues.pieces,
      formulaTemplateId: currentValues.formulaTemplateId,
    };

    isCalculatingRef.current = true;
    calculateTotalsRef.current(calculationPayload);
  }, [getValues]);

  return {
    isCalculating: isPending,
    triggerCalculation,
  };
}
