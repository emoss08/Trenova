import { iftaJurisdictionsQuery, type IftaJurisdiction } from "@/lib/graphql/ifta-jurisdiction";
import { useQuery } from "@tanstack/react-query";
import type {
  FormControlProps,
  GenericSelectOption,
  SelectOption,
  SelectOptionGroup,
} from "@trenova/shared/types/fields";
import { useMemo } from "react";
import type { FieldValues } from "react-hook-form";
import { SelectField, type BaseSelectFieldProps } from "./select-field";

const COUNTRY_LABELS: Record<string, string> = {
  US: "United States",
  CA: "Canada",
  MX: "Mexico",
};

const COUNTRY_ORDER = ["US", "CA", "MX"];

export function jurisdictionLabel(jurisdiction: Pick<IftaJurisdiction, "code" | "name">): string {
  return `${jurisdiction.code} — ${jurisdiction.name}`;
}

function countryLabel(countryCode: string): string {
  return COUNTRY_LABELS[countryCode] ?? countryCode;
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

export function jurisdictionOptionGroups(
  jurisdictions: readonly IftaJurisdiction[],
): SelectOptionGroup[] {
  const groups = new Map<string, SelectOption[]>();
  for (const jurisdiction of sortJurisdictions(jurisdictions)) {
    const options = groups.get(jurisdiction.countryCode) ?? [];
    options.push({
      value: jurisdiction.id,
      label: jurisdictionLabel(jurisdiction),
      description: jurisdiction.isIftaMember ? undefined : "Not an IFTA member",
    });
    groups.set(jurisdiction.countryCode, options);
  }
  return [...groups.entries()].map(([countryCode, options]) => ({
    label: countryLabel(countryCode),
    options,
  }));
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
  const groups = useMemo(() => jurisdictionOptionGroups(jurisdictions), [jurisdictions]);
  const byId = useMemo(
    () => new Map(jurisdictions.map((jurisdiction) => [jurisdiction.id, jurisdiction])),
    [jurisdictions],
  );
  return {
    jurisdictions,
    groups,
    byId,
    isLoading: query.isLoading,
    isError: query.isError,
  };
}

type IftaJurisdictionSelectFieldProps<T extends FieldValues> = Omit<
  BaseSelectFieldProps,
  "options" | "groups"
> &
  FormControlProps<T> & {
    membersOnly?: boolean;
  };

export function IftaJurisdictionSelectField<T extends FieldValues>({
  membersOnly,
  placeholder,
  ...props
}: IftaJurisdictionSelectFieldProps<T>) {
  const { groups, isLoading } = useIftaJurisdictionOptions({ membersOnly });
  return (
    <SelectField<T>
      {...props}
      groups={groups}
      placeholder={isLoading ? "Loading jurisdictions…" : (placeholder ?? "Select a jurisdiction")}
    />
  );
}
