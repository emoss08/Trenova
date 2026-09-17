import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export function useCarrierIntelRuleLabels(enabled: boolean): Readonly<Record<string, string>> {
  const settingsQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    enabled,
  });

  return useMemo(
    () =>
      Object.fromEntries(
        (settingsQuery.data?.carrierIntelRuleCatalog ?? []).map((rule) => [rule.code, rule.label]),
      ),
    [settingsQuery.data],
  );
}
