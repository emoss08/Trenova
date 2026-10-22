import {
  isReadyFor,
  ReadinessView,
  type ReadinessStep,
} from "@/components/tariff-studio/readiness-view";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";


/** A template's review gate, from the server's readiness. */
export function ReadinessPanel({
  templateId,
  step,
  onReadinessChange,
}: {
  templateId: string;
  step: ReadinessStep;
  onReadinessChange?: (ready: boolean | null) => void;
}) {
  const { data, isLoading, isError } = useQuery({
    ...formulaTemplate.readiness(templateId),
    enabled: !!templateId,
    staleTime: 0,
  });

  const ready = data ? isReadyFor(step, data) : null;
  const effectiveReady = isError ? true : ready;
  useEffect(() => {
    onReadinessChange?.(effectiveReady);
  }, [effectiveReady, onReadinessChange]);

  return <ReadinessView data={data} isLoading={isLoading} isError={isError} step={step} />;
}
