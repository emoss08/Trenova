/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { api } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import { useFormContext, useWatch } from "react-hook-form";

// Fields that trigger recalculation when changed
// Note: otherChargeAmount and totalChargeAmount are not included as they are outputs
const CALCULATION_TRIGGER_FIELDS = [
  "ratingMethod",
  "ratingUnit",
  "freightChargeAmount", // User input that affects calculation
  "additionalCharges",
  "moves",
  "commodities",
  "serviceTypeId",
  "shipmentTypeId",
  "customerId",
  "weight",
  "pieces",
  "distance",
] as const;

interface UseAutoCalculateTotalsOptions {
  enabled?: boolean;
  debounceMs?: number;
  onSuccess?: (data: {
    baseCharge: string | number;
    otherChargeAmount: string | number;
    totalChargeAmount: string | number;
  }) => void;
  onError?: (error: Error) => void;
}

export function useAutoCalculateTotals({
  enabled = true,
  debounceMs = 1000,
  onSuccess,
  onError,
}: UseAutoCalculateTotalsOptions = {}) {
  const { setValue, getValues } = useFormContext<ShipmentSchema>();
  const debounceTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastCalculatedHash = useRef<string | null>(null);
  const hasInitialized = useRef(false);

  // Watch relevant fields
  const watchedFields = useWatch({
    name: CALCULATION_TRIGGER_FIELDS as any,
  });

  // Create mutation for calculating totals
  const { mutate: calculateTotals, isPending } = useMutation({
    mutationFn: async (values: Partial<ShipmentSchema>) => {
      return await api.shipments.calculateTotals(values);
    },
    onSuccess: (data) => {
      // Only update otherChargeAmount and totalChargeAmount from API
      // freightChargeAmount is set by the user and shouldn't be overwritten
      setValue("otherChargeAmount", Number(data.otherChargeAmount), {
        shouldValidate: false,
        shouldDirty: true, // Mark as dirty so the save dock appears
      });
      setValue("totalChargeAmount", Number(data.totalChargeAmount), {
        shouldValidate: false,
        shouldDirty: true, // Mark as dirty so the save dock appears
      });

      onSuccess?.(data);
    },
    onError: (error) => {
      console.error("Failed to calculate totals:", error);
      // Don't show toast for every calculation error, only log
      // toast.error("Failed to calculate totals", {
      //   description: "Please check your input values",
      // });
      onError?.(error as Error);
    },
  });

  // Create a hash of the watched fields to detect actual changes
  const createFieldHash = useCallback((fields: any) => {
    return JSON.stringify(fields);
  }, []);

  // Debounced calculation function
  const performCalculation = useCallback(() => {
    if (!enabled) return;

    const currentValues = getValues();

    // Only include fields that are needed for calculation
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

    calculateTotals(calculationPayload);
  }, [enabled, getValues, calculateTotals]);

  // Effect to trigger calculation on field changes
  useEffect(() => {
    if (!enabled) return;

    // Create hash of current fields
    const currentHash = createFieldHash(watchedFields);

    // On first run, just store the initial hash without triggering calculation
    if (!hasInitialized.current) {
      hasInitialized.current = true;
      lastCalculatedHash.current = currentHash;
      return;
    }

    // Skip if nothing changed
    if (currentHash === lastCalculatedHash.current) {
      return;
    }

    // Clear existing timer
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
    }

    // Set new timer for debounced calculation
    debounceTimer.current = setTimeout(() => {
      lastCalculatedHash.current = currentHash;
      performCalculation();
    }, debounceMs);

    // Cleanup function
    return () => {
      if (debounceTimer.current) {
        clearTimeout(debounceTimer.current);
      }
    };
  }, [watchedFields, enabled, debounceMs, createFieldHash, performCalculation]);

  // Manual trigger function
  const triggerCalculation = useCallback(() => {
    // Clear any pending debounced calculation
    if (debounceTimer.current) {
      clearTimeout(debounceTimer.current);
    }
    performCalculation();
  }, [performCalculation]);

  return {
    isCalculating: isPending,
    triggerCalculation,
  };
}
