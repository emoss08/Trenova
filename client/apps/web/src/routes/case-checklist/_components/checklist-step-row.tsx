import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { builtinStepLabel, builtinStepRule } from "@/lib/case-checklist-labels";
import { isCustomStep } from "@/lib/case-checklist-steps";
import type { ChecklistItemMode, ChecklistKind, TemplateItem } from "@/types/case-checklist";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  ChevronDownIcon,
  GripVerticalIcon,
  Lock01Icon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Controller, useFormContext, useWatch } from "react-hook-form";

/**
 * One step of a checklist being laid out: a handle to drag it by, its name
 * and what it checks, and whether it is required, optional or off. A step
 * that follows a rule kept elsewhere says where and stays required; a step
 * the organization added opens to its name, what ticks it, and what its
 * button asks the case's agent.
 */
export function ChecklistStepRow({
  sortId,
  index,
  kind,
  stepKey,
  locked,
  expanded,
  onToggle,
  onRemove,
}: {
  sortId: string;
  index: number;
  kind: ChecklistKind;
  stepKey: string;
  locked: boolean;
  expanded: boolean;
  onToggle: () => void;
  onRemove: () => void;
}) {
  const t = useT();
  const { control } = useFormContext<{ items: TemplateItem[] }>();
  const custom = useWatch({ control, name: `items.${index}.custom` });
  const added = isCustomStep(stepKey);
  const { attributes, listeners, setNodeRef, setActivatorNodeRef, transform, transition, isDragging } =
    useSortable({ id: sortId });

  const title = added ? custom?.label || t("New step") : builtinStepLabel(stepKey, t);
  const note = added
    ? custom?.check === "Document"
      ? t("Ticked when the document type is on file")
      : t("Ticked by a person on the case")
    : builtinStepRule(stepKey, t);

  return (
    <li
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn("bg-card flex flex-col", isDragging && "relative z-10 opacity-80")}
    >
      <div className="flex items-center gap-3 px-3 py-2.5">
        <button
          type="button"
          ref={setActivatorNodeRef}
          className="ui-focus-ring text-muted-foreground hover:text-foreground cursor-grab rounded-md p-1 active:cursor-grabbing"
          aria-label={t("Move {0}", title)}
          {...attributes}
          {...listeners}
        >
          <GripVerticalIcon className="size-4" />
        </button>
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm font-medium">{title}</span>
          <span className="text-muted-foreground truncate text-xs">{note}</span>
        </div>
        {locked ? (
          <span className="text-muted-foreground inline-flex items-center gap-1.5 text-xs">
            <Lock01Icon className="size-3.5" />
            {t("Always required")}
          </span>
        ) : (
          <Controller
            control={control}
            name={`items.${index}.mode`}
            render={({ field }) => (
              <SegmentedControl<ChecklistItemMode>
                aria-label={t("How {0} counts", title)}
                value={field.value}
                onValueChange={field.onChange}
                items={[
                  { value: "Required", label: t("Required") },
                  { value: "Optional", label: t("Optional") },
                  { value: "Off", label: t("Off") },
                ]}
              />
            )}
          />
        )}
        {added && (
          <>
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              aria-expanded={expanded}
              aria-label={expanded ? t("Close {0}", title) : t("Edit {0}", title)}
              onClick={onToggle}
            >
              <ChevronDownIcon
                className={cn("size-4 transition-transform", expanded && "rotate-180")}
              />
            </Button>
            <Button
              type="button"
              size="icon-sm"
              variant="ghost"
              aria-label={t("Remove {0}", title)}
              onClick={onRemove}
            >
              <Trash01Icon className="size-4" />
            </Button>
          </>
        )}
      </div>
      {added && expanded && (
        <div className="border-border bg-muted/40 border-t px-4 py-3 pl-12">
          <FormGroup cols={2}>
            <FormControl>
              <InputField
                control={control}
                name={`items.${index}.custom.label`}
                label={t("Name")}
                placeholder={t("Lumper receipt checked")}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <SelectField
                control={control}
                name={`items.${index}.custom.check`}
                label={t("Ticked by")}
                options={[
                  { value: "Manual", label: "A person on the case" },
                  ...(kind === "ReadyToBill"
                    ? [{ value: "Document", label: "A document on file" }]
                    : []),
                ]}
              />
            </FormControl>
            {custom?.check === "Document" && (
              <FormControl cols="full">
                <DocumentTypeAutocompleteField
                  control={control}
                  name={`items.${index}.custom.documentTypeId`}
                  label={t("Document type")}
                  description={t("The step is done once an accepted copy is attached to the shipment.")}
                  rules={{ required: true }}
                />
              </FormControl>
            )}
            <FormControl>
              <InputField
                control={control}
                name={`items.${index}.custom.stepLabel`}
                label={t("Button")}
                placeholder={t("Check the lumper receipt")}
                description={t("What the next-step button says. The step's name when empty.")}
              />
            </FormControl>
            <FormControl cols="full">
              <TextareaField
                control={control}
                name={`items.${index}.custom.prompt`}
                label={t("What the button asks the agent")}
                placeholder={t("Find the lumper receipt and check it against the charge.")}
                description={t(
                  "Sent to the case's agent as the person's own message, with the record named.",
                )}
              />
            </FormControl>
          </FormGroup>
        </div>
      )}
    </li>
  );
}
