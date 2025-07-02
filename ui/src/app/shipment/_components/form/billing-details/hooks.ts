import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { useEffect } from "react";
import {
  Control,
  UseFormGetValues,
  UseFormSetValue,
  useFormState,
  useWatch,
} from "react-hook-form";
import { calculateTotalChargeAmount } from "./utils";

type WatchOptions = {
  control: Control<ShipmentSchema>;
  setValue: UseFormSetValue<ShipmentSchema>;
  getValues: UseFormGetValues<ShipmentSchema>;
};

/**
 * Base hook that handles watching changes for additional charges
 */
export function useAdditionalChargesWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const additionalCharges = useWatch({
    name: "additionalCharges",
    control,
  });

  useEffect(() => {
    if (dirtyFields.additionalCharges) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [additionalCharges, setValue, getValues, dirtyFields.additionalCharges]);
}

/**
 * Hook for watching rating method changes
 */
export function useRatingMethodWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const ratingMethod = useWatch({
    name: "ratingMethod",
    control,
  });

  useEffect(() => {
    if (dirtyFields.ratingMethod) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [ratingMethod, setValue, getValues, dirtyFields.ratingMethod]);

  return ratingMethod;
}

/**
 * Hook for watching flat rate specific fields
 */
export function useFlatRateWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const freightChargeAmount = useWatch({
    name: "freightChargeAmount",
    control,
  });

  useEffect(() => {
    if (dirtyFields.freightChargeAmount) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    freightChargeAmount,
    setValue,
    getValues,
    dirtyFields.freightChargeAmount,
  ]);
}

/**
 * Hook for watching per mile specific fields
 */
export function usePerMileWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const freightChargeAmount = useWatch({
    name: "freightChargeAmount",
    control,
  });

  const ratingUnit = useWatch({
    name: "ratingUnit",
    control,
  });

  useEffect(() => {
    if (dirtyFields.freightChargeAmount || dirtyFields.ratingUnit) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    freightChargeAmount,
    ratingUnit,
    setValue,
    getValues,
    dirtyFields.freightChargeAmount,
    dirtyFields.ratingUnit,
  ]);
}

/**
 * Hook for watching per stop specific fields
 */
export function usePerStopWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const ratingUnit = useWatch({
    name: "ratingUnit",
    control,
  });

  const moves = useWatch({
    name: "moves",
    control,
  });

  useEffect(() => {
    if (dirtyFields.ratingUnit || dirtyFields.moves) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    ratingUnit,
    moves,
    setValue,
    getValues,
    dirtyFields.ratingUnit,
    dirtyFields.moves,
  ]);
}

/**
 * Hook for watching per pound specific fields
 */
export function usePerPoundWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const ratingUnit = useWatch({
    name: "ratingUnit",
    control,
  });

  const weight = useWatch({
    name: "weight",
    control,
  });

  useEffect(() => {
    if (dirtyFields.ratingUnit || dirtyFields.weight) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    ratingUnit,
    weight,
    setValue,
    getValues,
    dirtyFields.ratingUnit,
    dirtyFields.weight,
  ]);
}

/**
 * Hook for watching per pallet specific fields
 */
export function usePerPalletWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const ratingUnit = useWatch({
    name: "ratingUnit",
    control,
  });

  const pieces = useWatch({
    name: "pieces",
    control,
  });

  useEffect(() => {
    if (dirtyFields.ratingUnit || dirtyFields.pieces) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    ratingUnit,
    pieces,
    setValue,
    getValues,
    dirtyFields.ratingUnit,
    dirtyFields.pieces,
  ]);
}

/**
 * Hook for watching per linear foot specific fields
 */
export function usePerLinearFootWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const ratingUnit = useWatch({
    name: "ratingUnit",
    control,
  });

  const commodities = useWatch({
    name: "commodities",
    control,
  });

  useEffect(() => {
    if (dirtyFields.ratingUnit || dirtyFields.commodities) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    ratingUnit,
    commodities,
    setValue,
    getValues,
    dirtyFields.ratingUnit,
    dirtyFields.commodities,
  ]);
}

/**
 * Hook for watching other rating method specific fields
 */
export function useOtherRatingWatcher({
  control,
  setValue,
  getValues,
}: WatchOptions) {
  const { dirtyFields } = useFormState({ control });
  const freightChargeAmount = useWatch({
    name: "freightChargeAmount",
    control,
  });

  const ratingUnit = useWatch({
    name: "ratingUnit",
    control,
  });

  useEffect(() => {
    if (dirtyFields.freightChargeAmount || dirtyFields.ratingUnit) {
      const shipment = getValues() as ShipmentSchema;
      setValue("totalChargeAmount", calculateTotalChargeAmount(shipment), {
        shouldValidate: true,
        shouldDirty: false,
      });
    }
  }, [
    freightChargeAmount,
    ratingUnit,
    setValue,
    getValues,
    dirtyFields.freightChargeAmount,
    dirtyFields.ratingUnit,
  ]);
}
