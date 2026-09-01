import { DEFAULT_TEST_VALUES } from "@/components/formula-editor/test-data-editor";
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

  const [testValues, setTestValues] = useState<Record<string, unknown>>({
    ...DEFAULT_TEST_VALUES,
  });
  const [useRealShipment, setUseRealShipment] = useState(false);
  const [shipmentId, setShipmentId] = useState("");

  const { mutate, data, isPending, reset } = useMutation<
    TestExpressionResponse,
    Error,
    TestExpressionRequest
  >({
    mutationFn: (request) => apiService.formulaTemplateService.test(request),
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
    };
  }, [
    expression,
    schemaId,
    variableDefinitions,
    breakdownDefinitions,
    minCharge,
    maxCharge,
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
