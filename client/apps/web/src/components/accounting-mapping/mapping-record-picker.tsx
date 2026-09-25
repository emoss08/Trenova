import { accountingReferenceDetail } from "@/lib/accounting-sync";
import type { AccountingReferenceObject } from "@/lib/graphql/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingReferenceKind, AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { SearchIcon } from "lucide-react";
import { useState } from "react";

const SEARCH_DELAY_MS = 250;

export function MappingRecordPicker({
  system,
  kind,
  providerName,
  disabled,
  isChoosing,
  onChoose,
}: {
  system: AccountingSystem;
  kind: AccountingReferenceKind;
  providerName: string;
  disabled: boolean;
  isChoosing: boolean;
  onChoose: (ref: AccountingReferenceObject) => void;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const search = useDebounce(query.trim(), SEARCH_DELAY_MS);
  const results = useQuery({
    ...queries.accountingSync.referenceSearch({ integrationType: system, kind, query: search }),
    enabled: !disabled,
  });

  return (
    <div className="space-y-2">
      <Input
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={t("Search {0} records", providerName)}
        aria-label={t("Search {0} records", providerName)}
        leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
        disabled={disabled}
      />
      {results.isLoading ? (
        <div className="space-y-1.5">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
        </div>
      ) : results.isError ? (
        <p className="text-danger-foreground text-sm">
          {t("The {0} records could not be searched.", providerName)}
        </p>
      ) : (results.data ?? []).length === 0 ? (
        <p className="text-foreground-muted text-sm">
          {search === ""
            ? t("{0} has no usable records of this kind yet.", providerName)
            : t("No usable {0} record matches that.", providerName)}
        </p>
      ) : (
        <ul className="divide-border-subtle max-h-64 divide-y overflow-y-auto rounded-md border">
          {(results.data ?? []).map((ref) => (
            <li key={ref.id} className="flex items-center justify-between gap-3 px-3 py-2">
              <div className="min-w-0">
                <div className="truncate text-sm">{ref.label}</div>
                {accountingReferenceDetail(ref) ? (
                  <div className="text-foreground-muted truncate text-xs">
                    {accountingReferenceDetail(ref)}
                  </div>
                ) : null}
              </div>
              <Button
                type="button"
                size="xs"
                variant="outline"
                disabled={disabled || isChoosing}
                onClick={() => onChoose(ref)}
              >
                {t("Use")}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
