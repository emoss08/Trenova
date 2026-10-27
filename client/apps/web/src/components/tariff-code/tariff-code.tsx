import {
  HighlightedCode,
  type HighlightedCodeProps,
} from "@trenova/shared/components/ui/highlighted-code";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { tariffCodeSpans } from "@trenova/shared/lib/highlight/tariff";
import { tariffTokens } from "@trenova/shared/rules/tariff";
import { useMemo } from "react";

export type TariffCodeProps = Omit<HighlightedCodeProps, "language" | "spans">;

/**
 * Tariff source, read-only, coloured from the rules module's tokens. It suspends until the
 * module has loaded, so it sits under the screen's Suspense boundary and skeleton.
 */
export function TariffCode({ code, ...props }: TariffCodeProps) {
  const rules = useRules();
  const spans = useMemo(() => tariffCodeSpans(tariffTokens(rules, code)), [rules, code]);
  return <HighlightedCode {...props} code={code} language="tariff" spans={spans} />;
}
