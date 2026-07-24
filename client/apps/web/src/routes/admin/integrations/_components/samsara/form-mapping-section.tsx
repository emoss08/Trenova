import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import type { TelematicsFormMapping } from "@/lib/graphql/telematics";
import {
  deleteTelematicsFormMappingGraphQL,
  saveTelematicsFormMappingGraphQL,
} from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Switch } from "@trenova/shared/components/ui/switch";
import type { SelectOption } from "@trenova/shared/types/fields";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PlusIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import { useFieldArray, useForm, useWatch, type Control } from "react-hook-form";
import { toast } from "sonner";

const TARGET_KIND = {
  shipmentField: "ShipmentField",
  shipmentCustomField: "ShipmentCustomField",
  stopField: "StopField",
} as const;

const TARGET_KIND_OPTIONS: SelectOption[] = [
  { value: TARGET_KIND.shipmentField, label: "Shipment Field" },
  { value: TARGET_KIND.shipmentCustomField, label: "Shipment Custom Field" },
  { value: TARGET_KIND.stopField, label: "Stop Field" },
];

const SHIPMENT_FIELD_OPTIONS: SelectOption[] = [
  { value: "bol", label: "BOL" },
  { value: "temperatureMin", label: "Temp Min" },
  { value: "temperatureMax", label: "Temp Max" },
  { value: "pieces", label: "Pieces" },
  { value: "weight", label: "Weight" },
];

const STOP_FIELD_OPTIONS: SelectOption[] = [
  { value: "pieces", label: "Pieces" },
  { value: "weight", label: "Weight" },
];

type MappingItemValues = {
  sourceFieldLabel: string;
  targetKind: string;
  targetField: string;
  targetCustomFieldKey: string;
};

type MappingFormValues = {
  id: string | null;
  name: string;
  templateId: string;
  templateName: string;
  description: string;
  enabled: boolean;
  version: number | null;
  items: MappingItemValues[];
};

function emptyItem(): MappingItemValues {
  return {
    sourceFieldLabel: "",
    targetKind: TARGET_KIND.shipmentField,
    targetField: "",
    targetCustomFieldKey: "",
  };
}

function blankForm(): MappingFormValues {
  return {
    id: null,
    name: "",
    templateId: "",
    templateName: "",
    description: "",
    enabled: true,
    version: null,
    items: [emptyItem()],
  };
}

function toFormValues(mapping: TelematicsFormMapping): MappingFormValues {
  return {
    id: mapping.id,
    name: mapping.name,
    templateId: mapping.templateId,
    templateName: mapping.templateName,
    description: mapping.description,
    enabled: mapping.enabled,
    version: mapping.version,
    items:
      mapping.items.length > 0
        ? mapping.items.map((item) => ({
            sourceFieldLabel: item.sourceFieldLabel,
            targetKind: item.targetKind,
            targetField: item.targetField,
            targetCustomFieldKey: item.targetCustomFieldKey,
          }))
        : [emptyItem()],
  };
}

function isItemComplete(item: MappingItemValues): boolean {
  if (item.sourceFieldLabel.trim().length === 0) {
    return false;
  }
  if (item.targetKind === TARGET_KIND.shipmentCustomField) {
    return item.targetCustomFieldKey.trim().length > 0;
  }
  return item.targetField.trim().length > 0;
}

function toSaveInput(values: MappingFormValues) {
  const completeItems = values.items.filter(isItemComplete);
  return {
    id: values.id ?? undefined,
    templateId: values.templateId.trim(),
    templateName: values.templateName.trim() || undefined,
    name: values.name.trim(),
    description: values.description.trim() || undefined,
    enabled: values.enabled,
    version: values.version ?? undefined,
    items: completeItems.map((item) => {
      const isCustom = item.targetKind === TARGET_KIND.shipmentCustomField;
      return {
        sourceFieldLabel: item.sourceFieldLabel.trim(),
        targetKind: item.targetKind,
        targetField: isCustom ? undefined : item.targetField,
        targetCustomFieldKey: isCustom ? item.targetCustomFieldKey.trim() : undefined,
      };
    }),
  };
}

function MappingItemRow({
  control,
  index,
  onRemove,
  canRemove,
}: {
  control: Control<MappingFormValues>;
  index: number;
  onRemove: () => void;
  canRemove: boolean;
}) {
  const targetKind = useWatch({ control, name: `items.${index}.targetKind` });
  const isCustom = targetKind === TARGET_KIND.shipmentCustomField;
  const isStop = targetKind === TARGET_KIND.stopField;

  return (
    <div className="flex flex-col gap-3 rounded-md border border-border bg-muted/30 p-3">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-muted-foreground">Field {index + 1}</span>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="size-6 p-0 text-muted-foreground hover:text-destructive"
          onClick={onRemove}
          disabled={!canRemove}
          aria-label={`Remove field ${index + 1}`}
        >
          <Trash2Icon className="size-3.5" />
        </Button>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <InputField
          name={`items.${index}.sourceFieldLabel`}
          control={control}
          label="Source Field Label"
          placeholder="Trailer Temperature (°F)"
          description="The exact form field label as it appears in the driver form."
        />
        <SelectField
          name={`items.${index}.targetKind`}
          control={control}
          label="Target Kind"
          options={TARGET_KIND_OPTIONS}
          placeholder="Select target"
        />
        {isCustom ? (
          <InputField
            name={`items.${index}.targetCustomFieldKey`}
            control={control}
            label="Target Custom Field Key"
            placeholder="e.g. reeferSetpoint"
            description="The custom field key on the shipment to populate."
          />
        ) : (
          <SelectField
            name={`items.${index}.targetField`}
            control={control}
            label="Target Field"
            options={isStop ? STOP_FIELD_OPTIONS : SHIPMENT_FIELD_OPTIONS}
            placeholder="Select field"
          />
        )}
      </div>
    </div>
  );
}

function MappingEditor({
  initial,
  onCancel,
  onSaved,
}: {
  initial: MappingFormValues;
  onCancel: () => void;
  onSaved: () => void;
}) {
  const queryClient = useQueryClient();
  const { control, handleSubmit } = useForm<MappingFormValues>({ defaultValues: initial });
  const itemsArray = useFieldArray({ control, name: "items" });

  const name = useWatch({ control, name: "name" });
  const templateId = useWatch({ control, name: "templateId" });
  const items = useWatch({ control, name: "items" });
  const hasCompleteItem = (items ?? []).some(isItemComplete);
  const canSave = name?.trim().length > 0 && templateId?.trim().length > 0 && hasCompleteItem;

  const saveMutation = useMutation({
    mutationFn: (values: MappingFormValues) =>
      saveTelematicsFormMappingGraphQL(toSaveInput(values)),
    onSuccess: async () => {
      toast.success(initial.id ? "Form mapping updated" : "Form mapping created");
      await queryClient.invalidateQueries({
        queryKey: queries.telematics.formMappings().queryKey,
      });
      onSaved();
    },
    onError: (error) => {
      toast.error("Failed to save form mapping", {
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const onSubmit = (values: MappingFormValues) => {
    saveMutation.mutate(values);
  };

  return (
    <div className="rounded-md border border-border p-3">
      <Form onSubmit={handleSubmit(onSubmit)} className="space-y-3">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              name="name"
              control={control}
              label="Name"
              placeholder="Reefer temperature capture"
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <InputField
              name="templateId"
              control={control}
              label="Template ID"
              placeholder="Samsara form template id"
              rules={{ required: true }}
              description="Find this in Samsara under the driver form's settings, or from a form submission."
            />
          </FormControl>
          <FormControl>
            <InputField
              name="templateName"
              control={control}
              label="Template Name (optional)"
              placeholder="Reefer Pre-Trip"
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              name="enabled"
              control={control}
              label="Enabled"
              description="Apply this mapping to incoming form submissions."
              outlined
            />
          </FormControl>
        </FormGroup>

        <div className="flex items-center justify-between gap-2 border-t border-border pt-3">
          <span className="text-sm font-medium">Field mappings</span>
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={() => itemsArray.append(emptyItem())}
          >
            <PlusIcon className="size-3" />
            Add field
          </Button>
        </div>

        <div className="flex flex-col gap-2">
          {itemsArray.fields.map((field, index) => (
            <MappingItemRow
              key={field.id}
              control={control}
              index={index}
              canRemove={itemsArray.fields.length > 1}
              onRemove={() => itemsArray.remove(index)}
            />
          ))}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-border pt-3">
          <Button type="button" variant="outline" size="sm" onClick={onCancel}>
            Cancel
          </Button>
          <Button
            type="submit"
            size="sm"
            disabled={!canSave || saveMutation.isPending}
            isLoading={saveMutation.isPending}
            loadingText="Saving..."
          >
            {initial.id ? "Save changes" : "Create mapping"}
          </Button>
        </div>
      </Form>
    </div>
  );
}

function MappingRow({
  mapping,
  onEdit,
  onDelete,
  onToggle,
  toggling,
}: {
  mapping: TelematicsFormMapping;
  onEdit: () => void;
  onDelete: () => void;
  onToggle: (next: boolean) => void;
  toggling: boolean;
}) {
  return (
    <div className="flex items-center gap-3 px-3 py-2.5">
      <div className="min-w-0 flex-1">
        <div className="flex min-w-0 items-center gap-2">
          <p className="truncate text-sm font-medium">{mapping.name}</p>
          <Badge variant="secondary" className="shrink-0">
            {mapping.items.length} {mapping.items.length === 1 ? "field" : "fields"}
          </Badge>
        </div>
        <p className="truncate text-xs text-muted-foreground">
          {mapping.templateName || mapping.templateId}
        </p>
      </div>
      <Switch
        checked={mapping.enabled}
        disabled={toggling}
        onCheckedChange={onToggle}
        aria-label={`Toggle ${mapping.name}`}
      />
      <Button type="button" variant="outline" size="xs" onClick={onEdit}>
        Edit
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="xs"
        className="size-7 p-0 text-muted-foreground hover:text-destructive"
        onClick={onDelete}
        aria-label={`Delete ${mapping.name}`}
      >
        <Trash2Icon className="size-3.5" />
      </Button>
    </div>
  );
}

export function SamsaraFormMappingSection({ open }: { open: boolean }) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<MappingFormValues | null>(null);
  const [pendingDelete, setPendingDelete] = useState<TelematicsFormMapping | null>(null);

  const mappingsQuery = useQuery({
    ...queries.telematics.formMappings(),
    enabled: open,
  });

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: queries.telematics.formMappings().queryKey });

  const toggleMutation = useMutation({
    mutationFn: (mapping: TelematicsFormMapping) =>
      saveTelematicsFormMappingGraphQL(
        toSaveInput({ ...toFormValues(mapping), enabled: !mapping.enabled }),
      ),
    onSuccess: async (_data, mapping) => {
      toast.success(mapping.enabled ? "Mapping disabled" : "Mapping enabled");
      await invalidate();
    },
    onError: (error) => {
      toast.error("Failed to update mapping", {
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteTelematicsFormMappingGraphQL(id),
    onSuccess: async () => {
      toast.success("Form mapping deleted");
      setPendingDelete(null);
      await invalidate();
    },
    onError: (error) => {
      toast.error("Failed to delete form mapping", {
        description: error instanceof Error ? error.message : undefined,
      });
    },
  });

  const mappings = mappingsQuery.data ?? [];

  let body: React.ReactNode;
  if (mappingsQuery.isLoading) {
    body = (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-14 w-full rounded-md" />
        <Skeleton className="h-14 w-full rounded-md" />
      </div>
    );
  } else if (mappingsQuery.isError) {
    body = (
      <div className="rounded-md border border-dashed p-6 text-center">
        <p className="text-sm font-medium">Form mappings could not be loaded</p>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="mt-3"
          onClick={() => void mappingsQuery.refetch()}
        >
          Try again
        </Button>
      </div>
    );
  } else if (mappings.length === 0) {
    body = (
      <div className="rounded-md border border-dashed p-6 text-center">
        <p className="text-sm font-medium">No form mappings yet</p>
        <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">
          Map a driver form&apos;s fields onto shipment data.
        </p>
      </div>
    );
  } else {
    body = (
      <div className="divide-y divide-border rounded-md border border-border">
        {mappings.map((mapping) => (
          <MappingRow
            key={mapping.id}
            mapping={mapping}
            toggling={toggleMutation.isPending && toggleMutation.variables?.id === mapping.id}
            onToggle={() => toggleMutation.mutate(mapping)}
            onEdit={() => setEditing(toFormValues(mapping))}
            onDelete={() => setPendingDelete(mapping)}
          />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 border-t border-border pt-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <p className="text-sm font-semibold">Form field mapping</p>
          <p className="text-xs text-muted-foreground">
            Map driver form template fields onto shipment and stop data.
          </p>
        </div>
        {editing === null ? (
          <Button type="button" variant="outline" size="sm" onClick={() => setEditing(blankForm())}>
            <PlusIcon className="size-3" />
            Add mapping
          </Button>
        ) : null}
      </div>

      {editing !== null ? (
        <MappingEditor
          key={editing.id ?? "new"}
          initial={editing}
          onCancel={() => setEditing(null)}
          onSaved={() => setEditing(null)}
        />
      ) : (
        body
      )}

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(next) => {
          if (!next) {
            setPendingDelete(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete form mapping?</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDelete
                ? `"${pendingDelete.name}" will no longer apply to incoming form submissions. This cannot be undone.`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleteMutation.isPending}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleteMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                if (pendingDelete) {
                  deleteMutation.mutate(pendingDelete.id);
                }
              }}
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
