import { isPlainList, listDelta, previewValue } from "@/lib/edit-diff";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import {
  type FieldPath,
  type FieldValues,
  type UseFormReturn,
  get,
  useWatch,
} from "react-hook-form";
import { Ic } from "../kit/ic";
import type { EditFlow } from "./use-edit-flow";
import { Button } from "@trenova/shared/components/ui/button";

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
  const values: unknown[] = useWatch({
    control: form.control,
    name: flow.changed as FieldPath<T>[],
  });

  return (
    <div className="es-rv" role="dialog" aria-label={t("Unsaved changes")}>
      <div className="es-rvh">
        <b>{count === 1 ? t("Review 1 change") : t("Review {0} changes", count)}</b>
        <span>{t("Undo any one before saving")}</span>
      </div>
      {flow.changed.map((key, index) => (
        <ChangeRow
          key={key}
          field={fields[key] ?? { label: key }}
          before={get(form.formState.defaultValues, key)}
          after={values[index]}
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
    <div className="cr">
      <span className="cr-l">{field.label}</span>
      <ChangeValue field={field} before={before} after={after} />
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="text-muted-foreground hover:text-foreground"
        title={t("Undo this change")}
        aria-label={t("Undo the change to {0}", field.label)}
        onClick={onUndo}
      >
        <Ic n="undo" s={11} />
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
      <span className="cr-v">
        {added.map((value) => (
          <em key={`+${value}`} className="ad">
            + {name(value)}
          </em>
        ))}
        {removed.map((value) => (
          <em key={`-${value}`} className="rm">
            − {name(value)}
          </em>
        ))}
      </span>
    );
  }

  const write = field.format ?? previewValue;
  return (
    <span className="cr-v">
      <s>{write(before)}</s>
      <Ic n="arrowR" s={11} />
      <b>{write(after)}</b>
    </span>
  );
}
