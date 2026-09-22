"use no memo";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FieldFilter, SortField } from "@trenova/shared/types/data-table";
import { useMutation, useQuery } from "@tanstack/react-query";
import { apiService } from "@/services/api";
import { TABLE_CATALOGUE_KEY } from "@/services/table-query";
import type { ComposedTableQuery } from "@/types/table-query";
import { CornerDownLeftIcon, TriangleAlertIcon, XIcon } from "lucide-react";
import { useCallback, useRef, useState } from "react";

/**
 * Ask the table for what you want, in the words you would use.
 *
 * "In transit for Acme that haven't shipped yet" is a question every
 * dispatcher can ask and almost nobody can build, because building it means
 * knowing the column is billingTransferStatus, that the value is spelled
 * InTransit, and that "haven't shipped yet" is a null check rather than a date
 * in the future.
 *
 * What comes back is filters, not an answer: they land in the same builder and
 * the same chips as a hand-built filter, and the person can see and change
 * every one before the table moves. That is the whole reason this is safe to
 * put on a data table — nothing is hidden, and nothing is applied that is not
 * shown.
 */
export function DataTableAsk({
  resource,
  query,
  fieldFilters,
  sort,
  onApply,
  className,
}: {
  resource: string;
  query: string;
  fieldFilters: FieldFilter[];
  sort: SortField[];
  onApply: (composed: ComposedTableQuery) => void;
  className?: string;
}) {
  const t = useT();
  // The catalogue is curated on the server — a vetted entity per tool, not one
  // per permission resource — so most tables cannot be narrowed by description
  // and showing the input would be a promise the request refuses. It changes
  // only when the server ships, so one cached request answers for every table.
  const catalogue = useQuery({
    queryKey: TABLE_CATALOGUE_KEY,
    queryFn: ({ signal }) => apiService.tableQueryService.catalogue({ signal }),
    staleTime: Number.POSITIVE_INFINITY,
  });
  const askable = (catalogue.data?.results ?? []).some((entry) => entry.resource === resource);
  const [prompt, setPrompt] = useState("");
  const [result, setResult] = useState<ComposedTableQuery | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const {
    mutate: composeQuery,
    reset: resetCompose,
    isPending,
  } = useMutation({
    mutationFn: (asked: string) =>
      apiService.tableQueryService.compose(resource, {
        prompt: asked,
        current: { query, fieldFilters, sort },
      }),
    onSuccess: (composed) => {
      setResult(composed);
      // Nothing compiled means there is nothing to apply and nothing to
      // review — the reasons are the whole answer, so the panel stays open on
      // them rather than closing on a table that did not move.
      if (composed.fieldFilters.length > 0 || composed.sort.length > 0) {
        onApply(composed);
      }
    },
  });

  const ask = useCallback(() => {
    const asked = prompt.trim();
    if (asked === "" || isPending) {
      return;
    }
    composeQuery(asked);
  }, [composeQuery, isPending, prompt]);

  const clear = useCallback(() => {
    setPrompt("");
    setResult(null);
    resetCompose();
    inputRef.current?.focus();
  }, [resetCompose]);

  const answered = result !== null;

  if (!askable) {
    return null;
  }

  return (
    <div className={cn("flex items-center gap-1.5", className)}>
      <div className="relative">
        <AssistMark className="text-muted-foreground pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2" />
        <Input
          ref={inputRef}
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              ask();
            }
            if (event.key === "Escape" && answered) {
              clear();
            }
          }}
          disabled={isPending}
          placeholder={t("Ask for what you want")}
          aria-label={t("Ask for what you want")}
          className="h-7 w-56 pr-7 pl-7 text-xs"
        />
        {prompt !== "" && (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("Clear")}
            onClick={clear}
            className="text-muted-foreground hover:text-foreground absolute top-1/2 right-0.5 size-6 -translate-y-1/2"
          >
            <XIcon className="size-3" />
          </Button>
        )}
      </div>

      {prompt !== "" && !answered && (
        <Button size="sm" variant="outline" onClick={ask} isLoading={isPending}>
          <CornerDownLeftIcon className="size-3" />
          {t("Ask")}
        </Button>
      )}

      {answered && <AskOutcome result={result} onDismiss={clear} />}
    </div>
  );
}

/**
 * What the question became, and what it did not.
 *
 * Both halves are shown because a filter that silently did not apply is the
 * worst outcome available: the table comes back looking answered, and the one
 * condition the person cared about is the one that went missing.
 */
function AskOutcome({ result, onDismiss }: { result: ComposedTableQuery; onDismiss: () => void }) {
  const t = useT();
  const missed = result.unresolved.length > 0;
  const applied = result.fieldFilters.length > 0 || result.sort.length > 0;

  return (
    <Popover defaultOpen={missed}>
      <PopoverTrigger
        render={
          <Button
            variant={missed ? "outline" : "ghost"}
            size="sm"
            className={cn("gap-1.5 text-xs", missed && "text-warning")}
          />
        }
      >
        {missed && <TriangleAlertIcon className="size-3" />}
        {applied
          ? t("{0, plural, one {# filter} other {# filters}}", result.fieldFilters.length)
          : t("Nothing applied")}
        {missed && ` · ${t("{0} missed", result.unresolved.length)}`}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80 p-0">
        <div className="space-y-1 p-3">
          <p className="text-xs font-medium">{t("What was applied")}</p>
          <p className="text-xs">{result.explanation}</p>
        </div>

        {missed && (
          <div className="border-border space-y-2 border-t p-3">
            <p className="text-xs font-medium">{t("What could not be")}</p>
            <ul className="space-y-1.5">
              {result.unresolved.map((entry) => (
                <li key={entry.phrase + entry.reason} className="text-xs">
                  <span className="block">{entry.phrase}</span>
                  <span className="text-muted-foreground block">{entry.reason}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        <div className="border-border flex justify-end border-t p-2">
          <Button variant="ghost" size="sm" onClick={onDismiss}>
            {t("Ask something else")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
