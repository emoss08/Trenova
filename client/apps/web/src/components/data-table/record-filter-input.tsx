import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { MultiSelectAutocomplete } from "@/components/fields/multi-select-field";
import {
  fetchGraphQLSelectOptions,
  type GraphQLSelectOptionsConfig,
  type SelectOption,
} from "@/lib/graphql/select-options";
import { recordOptionLabel } from "@/lib/select-option-meta";
import type { SelectOptionResource } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FilterOperator } from "@trenova/shared/types/data-table";

const MULTI_OPERATORS: readonly FilterOperator[] = ["in", "notin"];
/** How many names a chip spells out before it counts the rest. */
const CHIP_NAMES_SHOWN = 2;

function sourceFor(record: string): GraphQLSelectOptionsConfig {
  return { resource: record as SelectOptionResource };
}

function recordValue(option: SelectOption): string {
  return option.id;
}

function renderRecordOption(option: SelectOption) {
  return (
    <div className="flex min-w-0 flex-1 flex-col" title={option.description ?? undefined}>
      <span className="block w-full truncate">{recordOptionLabel(option)}</span>
      {option.description ? (
        <span className="text-muted-foreground block w-full truncate text-xs">
          {option.description}
        </span>
      ) : null}
    </div>
  );
}

export function isMultiRecordOperator(operator: FilterOperator): boolean {
  return MULTI_OPERATORS.includes(operator);
}

export function recordFilterIds(value: unknown): string[] {
  if (Array.isArray(value)) return value.filter((id): id is string => typeof id === "string");
  return typeof value === "string" && value !== "" ? [value] : [];
}

type RecordFilterInputProps = {
  /** The select-option resource the filter's values name, such as `CUSTOMER`. */
  record: string;
  operator: FilterOperator;
  value: unknown;
  onChange: (value: unknown) => void;
};

/**
 * Picks the records a filter matches, searching as the person types and loading
 * further pages as they scroll. "Is any of" and "is none of" take several records;
 * every other operator takes one.
 */
export function RecordFilterInput({ record, operator, value, onChange }: RecordFilterInputProps) {
  const t = useT();
  const source = sourceFor(record);

  if (isMultiRecordOperator(operator)) {
    return (
      <MultiSelectAutocomplete<SelectOption>
        link=""
        graphql={source}
        values={recordFilterIds(value)}
        onChange={(next) =>
          onChange(next.map((entry) => (typeof entry === "string" ? entry : entry.id)))
        }
        getOptionValue={recordValue}
        getDisplayValue={recordOptionLabel}
        renderOption={renderRecordOption}
        placeholder={t("Select records")}
        maxCount={2}
      />
    );
  }

  const single = recordFilterIds(value)[0] ?? null;
  return (
    <Autocomplete<SelectOption, Record<string, never>>
      graphql={source}
      value={single}
      onChange={(next: string | null) => onChange(next ?? null)}
      getOptionValue={recordValue}
      getDisplayValue={recordOptionLabel}
      renderOption={renderRecordOption}
      placeholder={t("Select a record")}
      clearable
    />
  );
}

/**
 * The names of records, looked up in one request and shared with every other reader
 * asking for the same ones. A record that no longer exists is left out of the map.
 */
export function useRecordNames(record: string | undefined, ids: readonly string[]) {
  const { data, isPending } = useQuery({
    queryKey: ["selectOptions", "byIds", record, ids],
    queryFn: ({ signal }) =>
      fetchGraphQLSelectOptions(
        { resource: record as SelectOptionResource, ids: [...ids], initialLimit: ids.length },
        { signal },
      ),
    enabled: !!record && ids.length > 0,
    staleTime: 5 * 60_000,
  });

  const names = new Map<string, string>();
  for (const option of data?.results ?? []) names.set(option.id, recordOptionLabel(option));
  return { names, isPending: !!record && ids.length > 0 && isPending };
}

/**
 * The names of the records a filter chip matches. Every name in a chip is looked up
 * in one request, and the answer is shared with every other chip naming the same
 * records.
 */
export function RecordFilterValue({ record, value }: { record: string; value: unknown }) {
  const t = useT();
  const ids = recordFilterIds(value);
  const { names: byId, isPending } = useRecordNames(record, ids);

  if (ids.length === 0) return null;
  if (isPending) return <Skeleton className="inline-block h-3 w-16 align-middle" />;

  const names = ids.map((id) => byId.get(id) ?? t("Removed record"));

  if (names.length <= CHIP_NAMES_SHOWN) return <>{names.join(", ")}</>;
  return (
    <>
      {t(
        "{0} +{1}",
        names.slice(0, CHIP_NAMES_SHOWN).join(", "),
        names.length - CHIP_NAMES_SHOWN,
      )}
    </>
  );
}
