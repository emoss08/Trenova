import type { SelectOption as GraphQLSelectOption } from "@/lib/graphql/select-options";
import type { SelectOption } from "@trenova/shared/types/fields";

export const EDI_TRANSACTION_SET_OPTIONS_LIMIT = 50;

export function toEDITransactionSetFilterOptions(
  results: readonly GraphQLSelectOption[],
  fallback: readonly SelectOption[],
): SelectOption[] {
  const options: SelectOption[] = [];
  const seen = new Set<string>();

  for (const option of results) {
    const code = option.meta?.code;
    if (typeof code !== "string" || code === "" || seen.has(code)) {
      continue;
    }
    seen.add(code);
    options.push({
      value: code,
      label: option.label,
      ...(option.description ? { description: option.description } : {}),
    });
  }

  return options.length > 0 ? options : [...fallback];
}
