import { DEFAULT_TEST_VALUES } from "@/components/formula-editor/test-data-editor";
import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import type {
  FormulaTemplateFormValues,
  TestExpressionRequest,
  TestExpressionResponse,
} from "@trenova/shared/types/formula-template";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const PREVIEW_DEBOUNCE_MS = 600;

export type LivePreviewState = {
  result: TestExpressionResponse | undefined;
  isPending: boolean;
  /** Set when the preview request itself failed (network, server), as opposed to the expression being invalid. */
  requestError: string | null;
  /** Wall-clock time of the last successful preview run. */
  lastRunAt: number | null;
  testValues: Record<string, unknown>;
  setTestValues: (values: Record<string, unknown>) => void;
  useRealShipment: boolean;
  setUseRealShipment: (value: boolean) => void;
  shipmentId: string;
  setShipmentId: (value: string) => void;
  runNow: () => void;
  clear: () => void;
};

export function useLivePreview(): LivePreviewState {
  const { control } = useFormContext<FormulaTemplateFormValues>();

  const expression = useWatch({ control, name: "expression" });
  const schemaId = useWatch({ control, name: "schemaId" });
  const variableDefinitions = useWatch({ control, name: "variableDefinitions" });
  const breakdownDefinitions = useWatch({ control, name: "breakdownDefinitions" });
  const minCharge = useWatch({ control, name: "minCharge" });
  const maxCharge = useWatch({ control, name: "maxCharge" });
  const roundingMode = useWatch({ control, name: "roundingMode" });
  const roundingPrecision = useWatch({ control, name: "roundingPrecision" });

  const [testValues, setTestValues] = useState<Record<string, unknown>>({
    ...DEFAULT_TEST_VALUES,
  });
  const [useRealShipment, setUseRealShipment] = useState(false);
  const [shipmentId, setShipmentId] = useState("");
  const [lastRunAt, setLastRunAt] = useState<number | null>(null);

  const { mutate, data, error, isPending, reset } = useMutation<
    TestExpressionResponse,
    Error,
    TestExpressionRequest
  >({
    mutationFn: (request) => apiService.formulaTemplateService.test(request),
    onSuccess: () => setLastRunAt(Date.now()),
  });

  const buildRequest = useCallback((): TestExpressionRequest | null => {
    if (!expression?.trim()) return null;

    const usingShipment = useRealShipment && !!shipmentId;
    const variables: Record<string, unknown> = usingShipment ? {} : { ...testValues };

    for (const variable of variableDefinitions ?? []) {
      if (variable.name && variable.defaultValue !== undefined && variable.defaultValue !== null) {
        variables[variable.name] = variable.defaultValue;
      }
    }

    const validBreakdowns = (breakdownDefinitions ?? []).filter(
      (item) => item.name?.trim() && item.expression?.trim(),
    );

    return {
      expression,
      schemaId: schemaId || "shipment",
      variables,
      ...(usingShipment ? { shipmentId } : {}),
      ...(validBreakdowns.length > 0 ? { breakdowns: validBreakdowns } : {}),
      ...(minCharge != null ? { minCharge: String(minCharge) } : {}),
      ...(maxCharge != null ? { maxCharge: String(maxCharge) } : {}),
      ...(roundingMode ? { roundingMode } : {}),
      ...(roundingPrecision != null ? { roundingPrecision } : {}),
    };
  }, [
    expression,
    schemaId,
    variableDefinitions,
    breakdownDefinitions,
    minCharge,
    maxCharge,
    roundingMode,
    roundingPrecision,
    testValues,
    useRealShipment,
    shipmentId,
  ]);

  const buildRequestRef = useRef(buildRequest);
  buildRequestRef.current = buildRequest;

  const runNow = useCallback(() => {
    const request = buildRequestRef.current();
    if (request) {
      mutate(request);
    }
  }, [mutate]);

  useEffect(() => {
    const request = buildRequest();
    if (!request) {
      reset();
      return;
    }

    const timer = window.setTimeout(() => {
      mutate(request);
    }, PREVIEW_DEBOUNCE_MS);

    return () => window.clearTimeout(timer);
  }, [buildRequest, mutate, reset]);

  return {
    result: data,
    isPending,
    requestError: error ? describeApiError(error, "The preview request failed.") : null,
    lastRunAt,
    testValues,
    setTestValues,
    useRealShipment,
    setUseRealShipment,
    shipmentId,
    setShipmentId,
    runNow,
    clear: reset,
  };
}
