import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { statusChoices } from "@/lib/choices";
import {
  fetchActiveWorkerCredentialTypes,
  WORKER_CREDENTIAL_TYPES_KEY,
} from "@/lib/graphql/worker-credential";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { cn } from "@trenova/shared/lib/utils";
import {
  CHECKLIST_ITEM_KIND_LABELS,
  CHECKLIST_KIND_LABELS,
  CHECKLIST_OWNER_LABELS,
  CHECKLIST_TRIGGER_LABELS,
  checklistItemKindSchema,
  checklistKindSchema,
  checklistOwnerSchema,
  checklistTriggerSchema,
  type ChecklistTemplateFormValues,
} from "@trenova/shared/types/worker-checklist";
import { GripVerticalIcon, InfoIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useMemo } from "react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

const KIND_OPTIONS = checklistKindSchema.options.map((value) => ({
  value,
  label: CHECKLIST_KIND_LABELS[value],
}));
const TRIGGER_OPTIONS = checklistTriggerSchema.options.map((value) => ({
  value,
  label: CHECKLIST_TRIGGER_LABELS[value],
}));
const ITEM_KIND_OPTIONS = checklistItemKindSchema.options.map((value) => ({
  value,
  label: CHECKLIST_ITEM_KIND_LABELS[value],
}));
const OWNER_OPTIONS = checklistOwnerSchema.options.map((value) => ({
  value,
  label: CHECKLIST_OWNER_LABELS[value],
}));

const ITEM_KIND_HINT: Record<string, string> = {
  Document: "Completes itself when a worker document of this type is on file.",
  Credential: "Completes itself when the worker holds a valid credential of this type.",
  Task: "Ticked off by the owner when done.",
  Equipment: "Ticked off by the owner when issued or returned.",
  PortalAccess:
    "Completes itself from Dash access — granted for onboarding, revoked for offboarding.",
};

export function ChecklistTemplateForm({
  isEdit,
  openChecklistCount = 0,
}: {
  isEdit: boolean;
  openChecklistCount?: number;
}) {
  const t = useT();

  const { control } = useFormContext<ChecklistTemplateFormValues>();
  const itemsArray = useFieldArray({ control, name: "items" });
  const trigger = useWatch({ control, name: "trigger" });

  const { data: credentialTypes = [] } = useQuery({
    queryKey: [WORKER_CREDENTIAL_TYPES_KEY],
    queryFn: ({ signal }) => fetchActiveWorkerCredentialTypes({ signal }),
    staleTime: 5 * 60 * 1000,
  });
  const credentialTypeOptions = useMemo(
    () => credentialTypes.map((type) => ({ value: type.id, label: type.name })),
    [credentialTypes],
  );

  return (
    <div className="flex flex-col gap-6">
      <section className="flex flex-col gap-3">
        <SectionTitle
          title={t("General")}
          hint={t("Name and code identify the checklist; the trigger decides when it starts on its own.")}
        />
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="code"
              label={t("Code")}
              placeholder={t("e.g. DRIVER-ONBOARDING")}
              rules={{ required: true }}
              description={t("Short unique identifier; saved in uppercase.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="name"
              label={t("Name")}
              placeholder={t("e.g. Driver onboarding")}
              rules={{ required: true }}
              description={t("Shown on the worker's Checklist tab once a checklist is started from this template.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="kind"
              label={t("Kind")}
              options={KIND_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select a kind")}
              description={t("Onboarding completion marks the worker DQF-ready.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="trigger"
              label={t("Starts")}
              options={TRIGGER_OPTIONS}
              rules={{ required: true }}
              placeholder={t("Select an event")}
              description={t("The employment event that starts this checklist for a worker.")}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="status"
              label={t("Status")}
              options={statusChoices}
              rules={{ required: true }}
              placeholder={t("Select a status")}
              description={
                isEdit && openChecklistCount > 0
                  ? `${openChecklistCount} checklist${openChecklistCount === 1 ? " is" : "s are"} in progress from this template; they keep their items either way.`
                  : "Inactive templates cannot be started."
              }
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="isDefault"
              label={t("Default for this trigger")}
              description={
                trigger === "Manual"
                  ? "Manual checklists are started from the worker's Checklist tab and cannot be the default."
                  : "Starts automatically when the matching employment event is recorded."
              }
              position="left"
            />
          </FormControl>
          <FormControl className="col-span-2">
            <TextareaField
              control={control}
              name="description"
              label={t("Description")}
              placeholder={t("Who this checklist is for and what done looks like")}
              maxLength={1000}
              description={t("Optional context for the people working through the checklist.")}
            />
          </FormControl>
        </FormGroup>
      </section>

      <section className="flex flex-col gap-3">
        <div className="flex items-start justify-between gap-2">
          <SectionTitle
            title={t("Items")}
            hint={t("Each line has an owner and a due date counted from the day the checklist starts.")}
          />
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() =>
              itemsArray.append({
                label: "",
                description: null,
                kind: "Task",
                required: true,
                dueOffsetDays: 3,
                owner: "HR",
                credentialTypeId: null,
                documentTypeId: null,
              })
            }
          >
            <PlusIcon className="size-3.5" />
            {t("Add item")}
          </Button>
        </div>
        <Alert variant="default">
          <InfoIcon className="size-4" />
          <AlertTitle>{t("Items are copied when a checklist starts")}</AlertTitle>
          <AlertDescription>
            {t("Changes to the items below only affect checklists started after you save. Checklists already in progress keep the items they were started with.")}
          </AlertDescription>
        </Alert>
        <div className="flex flex-col gap-3">
          {itemsArray.fields.map((field, index) => (
            <ItemRow
              key={field.id}
              index={index}
              credentialTypeOptions={credentialTypeOptions}
              onRemove={itemsArray.fields.length > 1 ? () => itemsArray.remove(index) : undefined}
              onMoveUp={index > 0 ? () => itemsArray.move(index, index - 1) : undefined}
              onMoveDown={
                index < itemsArray.fields.length - 1
                  ? () => itemsArray.move(index, index + 1)
                  : undefined
              }
            />
          ))}
        </div>
      </section>
    </div>
  );
}

function ItemRow({
  index,
  credentialTypeOptions,
  onRemove,
  onMoveUp,
  onMoveDown,
}: {
  index: number;
  credentialTypeOptions: { value: string; label: string }[];
  onRemove?: () => void;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
}) {
  const t = useT();

  const { control } = useFormContext<ChecklistTemplateFormValues>();
  const kind = useWatch({ control, name: `items.${index}.kind` });

  return (
    <div className="border-border bg-muted/20 flex flex-col gap-3 rounded-lg border p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-muted-foreground flex items-center gap-1 text-[11px] font-medium uppercase">
          <GripVerticalIcon className="size-3.5" />
          {t("Item {0}", index + 1)}
        </span>
        <div className="flex items-center gap-0.5">
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className={cn("h-7 px-2 text-xs", !onMoveUp && "invisible")}
            onClick={onMoveUp}
          >
            {t("Up")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className={cn("h-7 px-2 text-xs", !onMoveDown && "invisible")}
            onClick={onMoveDown}
          >
            {t("Down")}
          </Button>
          {onRemove ? (
            <Button
              type="button"
              size="icon"
              variant="ghost"
              className="text-muted-foreground hover:text-destructive size-7"
              aria-label={`Remove item ${index + 1}`}
              onClick={onRemove}
            >
              <Trash2Icon className="size-3.5" />
            </Button>
          ) : null}
        </div>
      </div>
      <FormGroup cols={2}>
        <FormControl className="col-span-2">
          <InputField
            control={control}
            name={`items.${index}.label`}
            label={t("Label")}
            placeholder={t("e.g. Fuel card issued")}
            rules={{ required: true }}
            description={t("Shown as the line the owner ticks off on the worker's checklist.")}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name={`items.${index}.kind`}
            label={t("Kind")}
            options={ITEM_KIND_OPTIONS}
            rules={{ required: true }}
            placeholder={t("Select a kind")}
            description={ITEM_KIND_HINT[kind ?? "Task"]}
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name={`items.${index}.owner`}
            label={t("Owner")}
            options={OWNER_OPTIONS}
            rules={{ required: true }}
            placeholder={t("Select a team")}
            description={t("The team responsible for getting this item done.")}
          />
        </FormControl>
        {kind === "Credential" ? (
          <FormControl className="col-span-2">
            <SelectField
              control={control}
              name={`items.${index}.credentialTypeId`}
              label={t("Credential type")}
              options={credentialTypeOptions}
              rules={{ required: true }}
              placeholder={t("Which credential completes this item")}
              description={t("The item completes itself once the worker holds a valid credential of this type.")}
            />
          </FormControl>
        ) : null}
        {kind === "Document" ? (
          <FormControl className="col-span-2">
            <DocumentTypeAutocompleteField
              control={control}
              name={`items.${index}.documentTypeId`}
              label={t("Document type")}
              placeholder={t("Which document completes this item")}
              description={t("The item completes itself once a worker document of this type is on file.")}
            />
          </FormControl>
        ) : null}
        <FormControl>
          <NumberField
            control={control}
            name={`items.${index}.dueOffsetDays`}
            label={t("Due")}
            sideText={t("days after start")}
            min={0}
            max={365}
            placeholder="3"
            description={t("Days after the checklist starts before this item counts as overdue.")}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={`items.${index}.required`}
            label={t("Required")}
            description={t("Required items must settle before the checklist closes.")}
            position="left"
          />
        </FormControl>
        <FormControl className="col-span-2">
          <TextareaField
            control={control}
            name={`items.${index}.description`}
            label={t("Description")}
            placeholder={t("What done looks like for this item")}
            maxLength={1000}
            description={t("Optional guidance shown with the item to whoever completes it.")}
          />
        </FormControl>
      </FormGroup>
    </div>
  );
}

function SectionTitle({ title, hint }: { title: string; hint: string }) {
  return (
    <div>
      <h3 className="text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground text-xs">{hint}</p>
    </div>
  );
}
