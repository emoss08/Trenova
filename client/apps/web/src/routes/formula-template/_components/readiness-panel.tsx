import {
  isReadyFor,
  ReadinessView,
  type ReadinessSourceLink,
  type ReadinessStep,
} from "@/components/tariff-studio/readiness-view";
import type { ReadinessCheck } from "@trenova/shared/types/formula-template";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

/** A template's review gate, from the server's readiness. */
export function ReadinessPanel({
  templateId,
  step,
  onReadinessChange,
  liveChecks,
  link,
}: {
  templateId: string;
  step: ReadinessStep;
  onReadinessChange?: (ready: boolean | null) => void;
  liveChecks?: readonly ReadinessCheck[];
  link?: ReadinessSourceLink;
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

  return (
    <ReadinessView
      data={data}
      isLoading={isLoading}
      isError={isError}
      step={step}
      liveChecks={liveChecks}
      link={link}
    />
  );
}
