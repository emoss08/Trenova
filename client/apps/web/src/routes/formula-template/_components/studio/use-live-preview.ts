import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import type {
  FormulaPreviewRequest,
  FormulaPreviewResponse,
} from "@trenova/shared/types/formula-template";
import type { TariffInputValues } from "@trenova/shared/types/tariff";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

const PREVIEW_DEBOUNCE_MS = 600;

export type PreviewSource = "sample" | "shipment";

export type LivePreviewState = {
  result: FormulaPreviewResponse | undefined;
  isPending: boolean;
  /** The request failed (network, server), as opposed to the tariff refusing or erroring. */
  requestError: string | null;
  /** Wall-clock time of the last successful run. */
  lastRunAt: number | null;
  source: PreviewSource;
  setSource: (source: PreviewSource) => void;
  shipmentId: string;
  setShipmentId: (value: string) => void;
  /** The request the preview last rated, so a result is never shown against other text. */
  lastRequest: FormulaPreviewRequest | null;
  /** True when the source checks with errors, so nothing is sent until they are fixed. */
  blocked: boolean;
  runNow: () => void;
};

/** Drops inputs left unset, so the tariff's own defaults apply to them. */
export function presentInputs(values: TariffInputValues | undefined): TariffInputValues {
  return Object.fromEntries(
    Object.entries(values ?? {}).filter(([, value]) => value !== undefined),
  ) as TariffInputValues;
}

/**
 * Rates what is in the editor on the server (`POST /formula-templates/test`), again once the
 * source or the sample inputs settle: the sample record with the inputs the person set, or a
 * saved shipment's record. It watches only the source and those inputs, never the name or the
 * description, and sends nothing while the source has errors, which the editor already shows.
 */
export function useLivePreview({
  source: text,
  inputs,
  errors,
}: {
  source: string;
  inputs: TariffInputValues | undefined;
  errors: number;
}): LivePreviewState {
  const [mode, setModeState] = useState<PreviewSource>("sample");
  const [shipmentId, setShipmentIdState] = useState("");
  const [lastRunAt, setLastRunAt] = useState<number | null>(null);
  const [lastRequest, setLastRequest] = useState<FormulaPreviewRequest | null>(null);

  const { mutate, data, error, isPending, reset } = useMutation<
    FormulaPreviewResponse,
    Error,
    FormulaPreviewRequest
  >({
    mutationFn: (request) => apiService.formulaTemplateService.preview(request),
    onSuccess: (_response, request) => {
      setLastRunAt(Date.now());
      setLastRequest(request);
    },
  });

  const usingShipment = mode === "shipment" && shipmentId !== "";
  const blocked = errors > 0 || text.trim() === "";
  const inputsKey = JSON.stringify(presentInputs(inputs));

  const request = useMemo((): FormulaPreviewRequest | null => {
    if (blocked) return null;
    const values = JSON.parse(inputsKey) as TariffInputValues;
    return {
      source: text,
      ...(Object.keys(values).length > 0 ? { inputs: values } : {}),
      ...(usingShipment ? { shipmentId } : {}),
    };
  }, [blocked, text, inputsKey, usingShipment, shipmentId]);

  const requestRef = useRef(request);
  useEffect(() => {
    requestRef.current = request;
  }, [request]);

  const runNow = useCallback(() => {
    if (requestRef.current) mutate(requestRef.current);
  }, [mutate]);

  useEffect(() => {
    if (!request || (mode === "shipment" && !usingShipment)) {
      reset();
      return;
    }
    const timer = window.setTimeout(() => mutate(request), PREVIEW_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [request, mode, usingShipment, mutate, reset]);

  const setSource = useCallback((next: PreviewSource) => {
    setModeState(next);
    if (next === "sample") setShipmentIdState("");
  }, []);

  return {
    result: data,
    isPending,
    requestError: error ? describeApiError(error, "The preview request failed.") : null,
    lastRunAt,
    source: mode,
    setSource,
    shipmentId,
    setShipmentId: setShipmentIdState,
    lastRequest,
    blocked,
    runNow,
  };
}
