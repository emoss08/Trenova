import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import type { ChecklistKind, CustomCheck } from "@/types/case-checklist";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { useFormContext, useFormState, useWatch } from "react-hook-form";
import { onRadioArrows } from "../../use-modal-focus";
import type { ChecklistEditorValues } from "./editor-schema";

const NAME_MAX = 80;
const BUTTON_MAX = 40;
const PROMPT_MAX = 500;

/**
 * A step the organization added, opened under its row: what it is called,
 * what ticks it, and what its button asks the case's agent when it is the
 * next step, beside a preview of that button as the case draws it.
 */
export function CustomStepForm({
  index,
  kind,
  onDocumentPicked,
  onDone,
  onRemove,
}: {
  index: number;
  kind: ChecklistKind;
  onDocumentPicked: (id: string, name: string) => void;
  onDone: () => void;
  onRemove: () => void;
}) {
  const t = useT();
  const { control, register, setValue } = useFormContext<ChecklistEditorValues>();
  const base = `items.${index}.custom` as const;
  const custom = useWatch({ control, name: base });
  const { errors, isSubmitted } = useFormState({ control, name: base });
  const fieldErrors = isSubmitted ? errors.items?.[index]?.custom : undefined;

  const label = custom?.label ?? "";
  const stepLabel = custom?.stepLabel ?? "";
  const prompt = custom?.prompt ?? "";
  const check = custom?.check ?? "Manual";
  const button = stepLabel.trim() || label.trim() || t("Name the step");

  const ways: Array<[CustomCheck, DeskIconName, string, string]> = [
    [
      "Manual",
      "check",
      t("A person on the case"),
      t("Anyone working the case ticks it. Everyone sees who did."),
    ],
    ...(kind === "ReadyToBill"
      ? ([
          [
            "Document",
            "copy",
            t("A document on file"),
            t("Ticks itself once an accepted copy is attached to the shipment."),
          ],
        ] as Array<[CustomCheck, DeskIconName, string, string]>)
      : []),
  ];
  const chooseCheck = (next: CustomCheck) => {
    setValue(`${base}.check`, next, { shouldDirty: true, shouldValidate: isSubmitted });
    if (next === "Manual") {
      setValue(`${base}.documentTypeId`, "", { shouldDirty: true });
    }
  };

  return (
    <div className="dk-ck-form">
      <div className="dk-ck-fsec">
        <Field label={t("Name")} error={fieldErrors?.label?.message} count={label.length} max={NAME_MAX}>
          <input
            {...register(`${base}.label`)}
            // oxlint-disable-next-line jsx-a11y/no-autofocus -- a new step opens to be named
            autoFocus={label === ""}
            placeholder={t("Lumper receipt checked")}
          />
        </Field>
      </div>

      <div className="dk-ck-fsec">
        <div className="dk-ck-fsh">
          <b>{t("What ticks it")}</b>
        </div>
        <div className="dk-ck-ways" role="radiogroup" aria-label={t("What ticks it")}>
          {ways.map(([value, icon, title, detail]) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={check === value}
              tabIndex={check === value ? 0 : -1}
              className={cn("dk-ck-way", check === value && "dk-on")}
              onClick={() => chooseCheck(value)}
              onKeyDown={(event) =>
                onRadioArrows(
                  event,
                  ways.map(([option]) => option),
                  check,
                  chooseCheck,
                )
              }
            >
              <span className="dk-ck-wr" />
              <span className="dk-ck-wx">
                <b>
                  <DeskIcon name={icon} size={13} />
                  {title}
                </b>
                <span>{detail}</span>
              </span>
            </button>
          ))}
        </div>
        {check === "Document" && (
          <div className={cn("dk-ck-doc", fieldErrors?.documentTypeId && "dk-bad")}>
            <DocumentTypeAutocompleteField
              control={control}
              name={`${base}.documentTypeId`}
              label={t("Document type")}
              rules={{ required: true }}
              onOptionChange={(option) => {
                if (option?.id) {
                  onDocumentPicked(option.id, option.label);
                }
              }}
            />
          </div>
        )}
      </div>

      <div className="dk-ck-fsec">
        <div className="dk-ck-fsh">
          <b>{t("When it's the next step")}</b>
          <span>{t("The case offers a button that asks its agent to help. Both are optional.")}</span>
        </div>
        <div className="dk-ck-nx">
          <div className="dk-ck-nxf">
            <Field
              label={t("Button")}
              error={fieldErrors?.stepLabel?.message}
              count={stepLabel.length}
              max={BUTTON_MAX}
            >
              <input
                {...register(`${base}.stepLabel`)}
                placeholder={label || t("Check the lumper receipt")}
              />
            </Field>
            <Field
              label={t("What it asks the agent")}
              error={fieldErrors?.prompt?.message}
              count={prompt.length}
              max={PROMPT_MAX}
            >
              <textarea
                {...register(`${base}.prompt`)}
                rows={3}
                placeholder={t("Find the lumper receipt and check it against the charge.")}
              />
            </Field>
          </div>
          <div className="dk-ck-nxp" aria-hidden>
            <span className="dk-ck-nxk">{t("On the case")}</span>
            <span className="dk-ck-nxb">
              <DeskIcon name="handoff" size={12} />
              {button}
            </span>
            <span className="dk-ck-nxa">{t("Sends as your message")}</span>
            <span className="dk-ck-nxm">
              {prompt.trim() || button}
              <i>
                {" "}
                {kind === "ReadyToBill" ? t("— for the shipment") : t("— for the invoice")}
              </i>
            </span>
          </div>
        </div>
      </div>

      <div className="dk-ck-ff">
        <button type="button" className="dk-ck-rm" onClick={onRemove}>
          <DeskIcon name="trash" size={13} />
          {t("Remove step")}
        </button>
        <span className="flex-1" />
        <button type="button" className="dk-ck-btn" onClick={onDone}>
          {t("Done")}
        </button>
      </div>
    </div>
  );
}

/** A labelled field whose counter shows once it is focused or nearly full. */
function Field({
  label,
  error,
  count,
  max,
  children,
}: {
  label: string;
  error?: string;
  count: number;
  max: number;
  children: ReactNode;
}) {
  const t = useT();
  return (
    <label className={cn("dk-ck-f", error && "dk-bad")}>
      <span className="dk-ck-fl">
        <b>{label}</b>
        <em className={cn(count > max ? "dk-over" : count > max * 0.8 && "dk-near")}>
          {count}/{max}
        </em>
      </span>
      {children}
      {error && <span className="dk-ck-fh">{t(error)}</span>}
    </label>
  );
}
