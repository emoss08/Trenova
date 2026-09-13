import { iftaJurisdictionsQuery, type IftaJurisdiction } from "@/lib/graphql/ifta-jurisdiction";
import { useQuery } from "@tanstack/react-query";
import type { GenericSelectOption } from "@trenova/shared/types/fields";
import { useMemo } from "react";

const COUNTRY_ORDER = ["US", "CA", "MX"];

export function jurisdictionLabel(jurisdiction: Pick<IftaJurisdiction, "code" | "name">): string {
  return `${jurisdiction.code} — ${jurisdiction.name}`;
}

function countryRank(countryCode: string): number {
  const index = COUNTRY_ORDER.indexOf(countryCode);
  return index === -1 ? COUNTRY_ORDER.length : index;
}

export function sortJurisdictions(jurisdictions: readonly IftaJurisdiction[]): IftaJurisdiction[] {
  return [...jurisdictions].sort((a, b) => {
    const byCountry = countryRank(a.countryCode) - countryRank(b.countryCode);
    if (byCountry !== 0) return byCountry;
    if (a.sortOrder !== b.sortOrder) return a.sortOrder - b.sortOrder;
    return a.code.localeCompare(b.code);
  });
}

export function jurisdictionFilterOptions(
  jurisdictions: readonly IftaJurisdiction[],
): GenericSelectOption<string>[] {
  return sortJurisdictions(jurisdictions).map((jurisdiction) => ({
    value: jurisdiction.id,
    label: jurisdictionLabel(jurisdiction),
  }));
}

export function useIftaJurisdictionOptions(options?: { membersOnly?: boolean }) {
  const membersOnly = options?.membersOnly ?? false;
  const query = useQuery(iftaJurisdictionsQuery(membersOnly));
  const jurisdictions = useMemo(() => query.data ?? [], [query.data]);
  const byId = useMemo(
    () => new Map(jurisdictions.map((jurisdiction) => [jurisdiction.id, jurisdiction])),
    [jurisdictions],
  );
  return {
    jurisdictions,
    byId,
    isLoading: query.isLoading,
    isError: query.isError,
  };
}
