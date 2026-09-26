import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { ProposalField } from "@/types/assistant";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { CircleAlertIcon, SearchIcon } from "lucide-react";
import { useId, useMemo, useState } from "react";
import {
  SUBSET_FILTER_THRESHOLD,
  filterSubsetChoices,
  keptSubsetIds,
  proposedSubsetIds,
  subsetChoices,
  subsetCountLabel,
  subsetDraft,
  type SubsetChoice,
} from "./record-subset";

/**
 * The records a write is over, as rows a person unticks.
 *
 * The agent proposed a transfer of forty shipments and the approver wants
 * thirty-seven of them. Every record the proposal names is listed by its
 * label, ticked, with what the preview says happens to it where the preview
 * reaches it; unticking one takes it out of the value approval sends. A
 * record can only be dropped, never added, and at least one has to stay:
 * the server refuses an empty set, so the form says so first.
 */
export function RecordSubsetField({
  labelId,
  field,
  proposed,
  value,
  outcomes,
  readOnly,
  onChange,
}: {
  /** The field's label, which names the list. */
  labelId: string;
  field: ProposalField;
  /** The parameter as the agent proposed it. */
  proposed: unknown;
  /** The draft: the kept ids as the form holds them. */
  value: string;
  /** What the preview says happens to each record, by id. */
  outcomes: ReadonlyMap<string, string>;
  readOnly: boolean;
  onChange: (value: string) => void;
}) {
  const t = useT();
  const [filter, setFilter] = useState("");

  const proposedIds = useMemo(() => proposedSubsetIds(proposed), [proposed]);
  const choices = useMemo(() => subsetChoices(field, proposedIds), [field, proposedIds]);
  const base = useMemo(
    () => (proposedIds.length > 0 ? proposedIds : choices.map((choice) => choice.id)),
    [choices, proposedIds],
  );
  const everyID = useMemo(() => new Set(base.map((id) => id.trim())), [base]);
  const kept = useMemo(() => keptSubsetIds(value), [value]);
  const keptCount = useMemo(() => {
    let count = 0;
    for (const id of everyID) {
      if (kept.has(id)) count++;
    }
    return count;
  }, [everyID, kept]);

  const visible = useMemo(() => filterSubsetChoices(choices, filter), [choices, filter]);
  const filterable = choices.length > SUBSET_FILTER_THRESHOLD;
  const fieldName = toSentenceFragment(field.label);

  const rows = useMemo<VirtualRow[]>(() => {
    const toggle = (id: string, on: boolean) => {
      const next = new Set(kept);
      if (on) {
        next.add(id);
      } else {
        next.delete(id);
      }
      onChange(subsetDraft(base, next));
    };

    return visible.map((choice) => ({
      key: choice.id,
      render: () => (
        <SubsetRow
          choice={choice}
          checked={kept.has(choice.id)}
          outcome={outcomes.get(choice.id)}
          disabled={readOnly}
          onToggle={(on) => toggle(choice.id, on)}
        />
      ),
    }));
  }, [base, kept, onChange, outcomes, readOnly, visible]);

  return (
    <div role="group" aria-labelledby={labelId} className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        {filterable && (
          <Input
            type="search"
            inputContainerClassName="w-full max-w-60"
            className="h-7"
            value={filter}
            onChange={(event) => setFilter(event.target.value)}
            placeholder={t("Filter by name")}
            aria-label={t("Filter {0}", fieldName)}
            leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          />
        )}
        <span className="text-muted-foreground text-xs tabular-nums" aria-live="polite">
          {subsetCountLabel(field.resource, keptCount, everyID.size, t)}
        </span>
        <div className="ml-auto flex gap-1.5">
          <Button
            type="button"
            size="xs"
            variant="outline"
            disabled={readOnly || keptCount === everyID.size}
            onClick={() => onChange(subsetDraft(base, everyID))}
          >
            {t("Select all")}
          </Button>
          <Button
            type="button"
            size="xs"
            variant="ghost"
            disabled={readOnly || keptCount === 0}
            onClick={() => onChange("")}
          >
            {t("Clear")}
          </Button>
        </div>
      </div>

      <VirtualRows
        rows={rows}
        estimateSize={36}
        initialHeight={288}
        aria-label={field.label}
        className="border-border max-h-72 rounded-md border p-1"
        empty={
          <p className="border-border text-muted-foreground rounded-md border px-3 py-6 text-center text-xs">
            {t("Nothing matches that filter.")}
          </p>
        }
      />

      {keptCount === 0 && (
        <Alert size="sm" variant="destructive">
          <CircleAlertIcon />
          <AlertDescription>
            {t("Keep at least one, or reject the proposal instead.")}
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}

function SubsetRow({
  choice,
  checked,
  outcome,
  disabled,
  onToggle,
}: {
  choice: SubsetChoice;
  checked: boolean;
  outcome: string | undefined;
  disabled: boolean;
  onToggle: (on: boolean) => void;
}) {
  const id = useId();

  return (
    <div
      data-slot="subset-row"
      className="hover:bg-surface-hover flex items-start gap-2.5 rounded-md px-2 py-1.5 transition-colors"
    >
      <Checkbox
        id={id}
        checked={checked}
        disabled={disabled}
        onCheckedChange={(value) => onToggle(value === true)}
        aria-label={choice.label}
        className="mt-0.5"
      />
      <div className="flex min-w-0 flex-1 flex-col">
        <label
          htmlFor={id}
          className={cn(
            "cursor-pointer truncate text-sm",
            !checked && "text-foreground-muted",
            disabled && "cursor-default",
          )}
        >
          {choice.label}
        </label>
        {outcome && <p className="text-muted-foreground text-xs">{outcome}</p>}
      </div>
    </div>
  );
}
