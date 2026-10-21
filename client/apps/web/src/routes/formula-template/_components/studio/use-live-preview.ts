import { flattenResolvedVariables } from "@/components/formula-editor/resolved-variables";
import { DEFAULT_TEST_VALUES } from "@/components/formula-editor/test-data-editor";
import { describeApiError } from "@/lib/api-error-message";
import { formulaTestRequest } from "@/lib/formula-test-request";
import { apiService } from "@/services/api";
import type {
  FormulaTemplateFormValues,
  TestExpressionRequest,
  TestExpressionResponse,
} from "@trenova/shared/types/formula-template";
import { useMutation } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const PREVIEW_DEBOUNCE_MS = 600;

export type PreviewSource = "sample" | "shipment";

export type LivePreviewState = {
  result: TestExpressionResponse | undefined;
  isPending: boolean;
  /** Set when the preview request itself failed (network, server), as opposed to the expression being invalid. */
  requestError: string | null;
  /** Wall-clock time of the last successful preview run. */
  lastRunAt: number | null;
  source: PreviewSource;
  setSource: (source: PreviewSource) => void;
  shipmentId: string;
  setShipmentId: (value: string) => void;
  /** The sample load's own values, before anything the person changed. */
  sampleValues: Readonly<Record<string, unknown>>;
  /** Values the person set over the load, by variable name. */
  overrides: Readonly<Record<string, unknown>>;
  setOverride: (name: string, value: unknown) => void;
  resetOverrides: () => void;
  /** What the load holds for a name: the person's override, the template default, the sample. */
  valueOf: (name: string) => unknown;
  /** The request the preview last priced. */
  lastRequest: TestExpressionRequest | null;
  /** Every value the load holds, as a scenario pinned from it records them. */
  loadValues: () => Record<string, unknown>;
  runNow: () => void;
};

/**
 * Prices what is in the editor on one load, again after every edit: the
 * sample load with whatever the person changed over it, or a saved shipment.
 * Every amount the studio shows for the preview comes from this response.
 */
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

  const [source, setSourceState] = useState<PreviewSource>("sample");
  const [shipmentId, setShipmentIdState] = useState("");
  const [overrides, setOverrides] = useState<Record<string, unknown>>({});
  const [lastRunAt, setLastRunAt] = useState<number | null>(null);
  const [lastRequest, setLastRequest] = useState<TestExpressionRequest | null>(null);

  const { mutate, data, error, isPending, reset } = useMutation<
    TestExpressionResponse,
    Error,
    TestExpressionRequest
  >({
    mutationFn: (request) => apiService.formulaTemplateService.test(request),
    onSuccess: (_response, request) => {
      setLastRunAt(Date.now());
      setLastRequest(request);
    },
  });

  const usingShipment = source === "shipment" && shipmentId !== "";

  const request = useMemo(
    (): TestExpressionRequest | null =>
      formulaTestRequest(
        {
          expression,
          schemaId,
          variableDefinitions,
          breakdownDefinitions,
          minCharge,
          maxCharge,
          roundingMode,
          roundingPrecision,
        },
        {
          sampleValues: DEFAULT_TEST_VALUES,
          overrides,
          shipmentId: usingShipment ? shipmentId : undefined,
          traceStatements: true,
        },
      ),
    [
      expression,
      schemaId,
      variableDefinitions,
      breakdownDefinitions,
      minCharge,
      maxCharge,
      roundingMode,
      roundingPrecision,
      overrides,
      usingShipment,
      shipmentId,
    ],
  );

  const requestRef = useRef(request);
  useEffect(() => {
    requestRef.current = request;
  }, [request]);

  const runNow = useCallback(() => {
    if (requestRef.current) {
      mutate(requestRef.current);
    }
  }, [mutate]);

  useEffect(() => {
    if (!request || (source === "shipment" && !usingShipment)) {
      reset();
      return;
    }
    const timer = window.setTimeout(() => mutate(request), PREVIEW_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [request, source, usingShipment, mutate, reset]);

  const setSource = useCallback((next: PreviewSource) => {
    setSourceState(next);
    setOverrides({});
    if (next === "sample") setShipmentIdState("");
  }, []);

  const setShipmentId = useCallback((value: string) => {
    setShipmentIdState(value);
    setOverrides({});
  }, []);

  const setOverride = useCallback((name: string, value: unknown) => {
    setOverrides((current) => {
      if (value !== undefined) return { ...current, [name]: value };
      return Object.fromEntries(Object.entries(current).filter(([key]) => key !== name));
    });
  }, []);

  const resetOverrides = useCallback(() => setOverrides({}), []);

  const resolvedVariables = data?.resolvedVariables;
  const resolved = useMemo(
    () => (resolvedVariables ? flattenResolvedVariables(resolvedVariables) : undefined),
    [resolvedVariables],
  );
  const receiptValues = data?.receipt?.variables;
  const valueOf = useCallback(
    (name: string): unknown => {
      if (name in overrides) return overrides[name];
      const read = receiptValues?.find((variable) => variable.name === name);
      if (read) return read.value;
      const definition = (variableDefinitions ?? []).find((variable) => variable.name === name);
      if (definition && definition.defaultValue !== undefined && definition.defaultValue !== null) {
        return definition.defaultValue;
      }
      if (usingShipment) return resolved?.[name];
      return DEFAULT_TEST_VALUES[name];
    },
    [overrides, receiptValues, variableDefinitions, usingShipment, resolved],
  );

  const loadValues = useCallback(
    (): Record<string, unknown> => ({
      ...(usingShipment ? (resolved ?? {}) : DEFAULT_TEST_VALUES),
      ...overrides,
    }),
    [usingShipment, resolved, overrides],
  );

  return {
    result: data,
    isPending,
    requestError: error ? describeApiError(error, "The preview request failed.") : null,
    lastRunAt,
    source,
    setSource,
    shipmentId,
    setShipmentId,
    sampleValues: DEFAULT_TEST_VALUES,
    overrides,
    setOverride,
    resetOverrides,
    valueOf,
    lastRequest,
    loadValues,
    runNow,
  };
}
