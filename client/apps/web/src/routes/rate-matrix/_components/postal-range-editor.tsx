import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { PlusIcon } from "@trenova/shared/components/icons";
import {
  axisLabel,
  normalizePostal,
  postalRangeLabel,
  postalRangeProblem,
  type PostalRange,
} from "@trenova/shared/lib/rate-matrix";
import type { RateMatrixDimension } from "@trenova/shared/types/rate";
import { useState } from "react";

type PostalRangeEditorProps = {
  readonly dimension: RateMatrixDimension;
  /** The axis's ranges, each once, in order. */
  readonly ranges: readonly PostalRange[];
  readonly onAdd: (range: PostalRange) => void;
  readonly onChange: (from: PostalRange, to: PostalRange) => void;
  readonly onRemove: (range: PostalRange) => void;
  /** Why no range can be added yet, when one cannot. */
  readonly addBlocked?: string | null;
};

function RangeInputs({
  from,
  to,
  onFrom,
  onTo,
  label,
}: {
  from: string;
  to: string;
  onFrom: (value: string) => void;
  onTo: (value: string) => void;
  label: string;
}) {
  const t = useT();
  return (
    <>
      <Input
        className="h-8 w-28 font-mono text-xs"
        aria-label={t("First postal code of {0}", label)}
        placeholder={t("From")}
        value={from}
        onChange={(event) => onFrom(event.target.value)}
      />
      <span className="text-muted-foreground text-xs">{t("through")}</span>
      <Input
        className="h-8 w-28 font-mono text-xs"
        aria-label={t("Last postal code of {0}", label)}
        placeholder={t("To")}
        value={to}
        onChange={(event) => onTo(event.target.value)}
      />
    </>
  );
}

function RangeRow({
  range,
  others,
  label,
  onChange,
  onRemove,
}: {
  range: PostalRange;
  others: readonly PostalRange[];
  label: string;
  onChange: (from: PostalRange, to: PostalRange) => void;
  onRemove: (range: PostalRange) => void;
}) {
  const t = useT();
  const [editing, setEditing] = useState(false);
  const [from, setFrom] = useState(range.from);
  const [to, setTo] = useState(range.to);

  const next = { from: normalizePostal(from), to: normalizePostal(to) };
  const problem = postalRangeProblem(next, others);

  if (!editing) {
    return (
      <li className="flex items-center gap-2 py-1">
        <span className="min-w-36 font-mono text-xs">{postalRangeLabel(range)}</span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-6 text-xs"
          onClick={() => {
            setFrom(range.from);
            setTo(range.to);
            setEditing(true);
          }}
        >
          {t("Edit")}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-6 text-xs"
          aria-label={t("Remove {0}", postalRangeLabel(range))}
          onClick={() => onRemove(range)}
        >
          {t("Remove")}
        </Button>
      </li>
    );
  }

  return (
    <li className="flex flex-col gap-1 py-1">
      <div className="flex flex-wrap items-center gap-2">
        <RangeInputs from={from} to={to} onFrom={setFrom} onTo={setTo} label={label} />
        <Button
          type="button"
          size="sm"
          className="h-7 text-xs"
          disabled={problem !== null}
          onClick={() => {
            onChange(range, next);
            setEditing(false);
          }}
        >
          {t("Apply")}
        </Button>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 text-xs"
          onClick={() => setEditing(false)}
        >
          {t("Cancel")}
        </Button>
      </div>
      {problem && (
        <p role="alert" className="text-destructive text-xs">
          {problem}
        </p>
      )}
    </li>
  );
}

/**
 * The postal ranges of one `ZipRange` axis: each range is a row (or column, or sheet) of the
 * grid, so adding one adds its cells across the other axes, and editing or removing one changes
 * every sheet. Ranges are inclusive at both ends, one length per axis, and never overlap.
 */
export function PostalRangeEditor({
  dimension,
  ranges,
  onAdd,
  onChange,
  onRemove,
  addBlocked = null,
}: PostalRangeEditorProps) {
  const t = useT();
  const label = axisLabel(dimension);
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const candidate = { from: normalizePostal(from), to: normalizePostal(to) };
  const touched = from.trim() !== "" || to.trim() !== "";
  const problem = postalRangeProblem(candidate, ranges);

  return (
    <div className="bg-card flex flex-col gap-2 rounded-md border p-3">
      <div>
        <p className="text-sm font-medium">{t("{0}: postal ranges", label)}</p>
        <p className="text-muted-foreground text-xs">
          {t(
            "A postal code matches a range when its first characters fall between the two ends, both included. Codes are compared character by character, after trimming and upper-casing.",
          )}
        </p>
      </div>

      {ranges.length > 0 ? (
        <ul aria-label={t("{0}: postal ranges", label)} className="divide-y">
          {ranges.map((range) => (
            <RangeRow
              key={`${range.from}..${range.to}`}
              range={range}
              others={ranges.filter((other) => other !== range)}
              label={label}
              onChange={onChange}
              onRemove={onRemove}
            />
          ))}
        </ul>
      ) : (
        <p className="text-muted-foreground text-xs">
          {t("No ranges yet. A postal code no range holds finds no rate.")}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <RangeInputs from={from} to={to} onFrom={setFrom} onTo={setTo} label={label} />
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="h-7 text-xs"
          disabled={problem !== null || addBlocked !== null}
          onClick={() => {
            onAdd(candidate);
            setFrom("");
            setTo("");
          }}
        >
          <PlusIcon className="mr-1 size-3.5" />
          {t("Add range")}
        </Button>
      </div>
      {addBlocked && <p className="text-muted-foreground text-xs">{addBlocked}</p>}
      {touched && problem && (
        <p role="alert" className="text-destructive text-xs">
          {problem}
        </p>
      )}
    </div>
  );
}
