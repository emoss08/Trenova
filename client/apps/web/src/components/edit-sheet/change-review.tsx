import { isPlainList, listDelta, previewValue } from "@/lib/edit-diff";
import { Button } from "@trenova/shared/components/ui/button";
import { ArrowRightIcon, ReverseLeftIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { type FieldValues, type UseFormReturn, get } from "react-hook-form";
import type { EditFlow } from "./use-edit-flow";

/** How an editor names and shows one of its fields in a change review. */
export type EditField = {
  label: string;
  /** Writes a value; the default handles text, numbers, switches and lists. */
  format?: (value: unknown) => ReactNode;
  /** Names one item of a list field, such as a tool by its title. */
  item?: (value: string | number) => string;
};

export type EditFields = Record<string, EditField>;

type ChangeReviewProps<T extends FieldValues> = {
  form: UseFormReturn<T>;
  flow: EditFlow;
  fields: EditFields;
};

/** Every unsaved change, before and after, each with its own undo. */
export function ChangeReview<T extends FieldValues>({ form, flow, fields }: ChangeReviewProps<T>) {
  const t = useT();
  const count = flow.changed.length;

  return (
    <div className="flex max-h-80 flex-col gap-1 overflow-y-auto p-3">
      <div className="flex items-baseline gap-2 pb-1">
        <span className="font-medium">
          {count === 1 ? t("Review 1 change") : t("Review {0} changes", count)}
        </span>
        <span className="text-xs text-muted-foreground">{t("Undo any one before saving")}</span>
      </div>
      {flow.changed.map((key) => (
        <ChangeRow
          key={key}
          field={fields[key] ?? { label: key }}
          before={get(form.formState.defaultValues, key)}
          after={form.getValues(key as never)}
          onUndo={() => flow.undo(key)}
        />
      ))}
    </div>
  );
}

function ChangeRow({
  field,
  before,
  after,
  onUndo,
}: {
  field: EditField;
  before: unknown;
  after: unknown;
  onUndo: () => void;
}) {
  const t = useT();

  return (
    <div className="flex items-center gap-3 rounded-control px-2 py-1.5 hover:bg-muted">
      <span className="w-40 shrink-0 truncate text-xs text-muted-foreground">{field.label}</span>
      <span className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5 text-xs">
        <ChangeValue field={field} before={before} after={after} />
      </span>
      <Button
        size="icon-xs"
        variant="ghost"
        onClick={onUndo}
        aria-label={t("Undo the change to {0}", field.label)}
      >
        <ReverseLeftIcon className="size-3" />
      </Button>
    </div>
  );
}

function ChangeValue({
  field,
  before,
  after,
}: {
  field: EditField;
  before: unknown;
  after: unknown;
}) {
  if (!field.format && isPlainList(before) && isPlainList(after)) {
    const { added, removed } = listDelta(before, after);
    const name = field.item ?? String;
    return (
      <>
        {added.map((value) => (
          <span key={`+${value}`} className="text-success">
            + {name(value)}
          </span>
        ))}
        {removed.map((value) => (
          <span key={`-${value}`} className="text-danger line-through">
            − {name(value)}
          </span>
        ))}
      </>
    );
  }

  const write = field.format ?? previewValue;
  return (
    <>
      <span className="text-muted-foreground line-through">{write(before)}</span>
      <ArrowRightIcon className="size-3 text-muted-foreground" />
      <span className="font-medium">{write(after)}</span>
    </>
  );
}
