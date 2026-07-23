import {
  PermissionOperationAutocompleteField,
  PermissionResourceAutocompleteField,
} from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { OperationDefinition, ResourceDefinition } from "@/lib/role-api";
import type { AccessPolicyConditionRow, AccessPolicyFormValues } from "@trenova/shared/types/iam";
import { PlusIcon, XIcon } from "lucide-react";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";
import { policyEffectOptions } from "./constants";

type AccessPolicyFormProps = {
  resources: ResourceDefinition[];
  operations: OperationDefinition[];
  resourcesLoading: boolean;
  operationsLoading: boolean;
};

export function AccessPolicyForm({
  resources,
  operations,
  resourcesLoading,
  operationsLoading,
}: AccessPolicyFormProps) {
  const { control, setValue } = useFormContext<AccessPolicyFormValues>();
  const selectedResourceName = useWatch({ control, name: "resource" });
  const selectedResource = resources.find((resource) => resource.resource === selectedResourceName);
  const availableOperations = selectedResource?.operations.length
    ? selectedResource.operations
    : operations;

  return (
    <>
      <FormSection title="Policy Decision">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Policy Name"
              placeholder="Require managed devices for billing exports"
              description="Administrative name that explains when this policy should match."
              maxLength={120}
            />
          </FormControl>
          <FormControl>
            <PermissionResourceAutocompleteField<AccessPolicyFormValues>
              control={control}
              rules={{ required: true }}
              name="resource"
              resources={resources}
              disabled={resourcesLoading}
              onValueChange={() =>
                setValue("operation", "", { shouldDirty: true, shouldValidate: true })
              }
            />
          </FormControl>
          <FormControl>
            <PermissionOperationAutocompleteField<AccessPolicyFormValues>
              control={control}
              rules={{ required: true }}
              name="operation"
              operations={availableOperations}
              disabled={operationsLoading || availableOperations.length === 0}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="effect"
              label="Effect"
              placeholder="Select effect"
              description="Authorization decision returned when this policy matches."
              options={policyEffectOptions}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true, min: 0 }}
              name="priority"
              label="Priority"
              placeholder="100"
              description="Lower numbers evaluate first. Use gaps to leave room for future rules."
              min={0}
            />
          </FormControl>
          <FormControl cols="full">
            <SwitchField
              control={control}
              name="enabled"
              label="Enabled"
              description="Evaluate this policy during access decisions."
              outlined
            />
          </FormControl>
        </FormGroup>
      </FormSection>
      <PolicyConditionsSection />
    </>
  );
}

function PolicyConditionsSection() {
  const { control } = useFormContext<AccessPolicyFormValues>();
  const { append, fields, remove } = useFieldArray({
    control,
    name: "conditionRows",
    keyName: "fieldId",
  });

  const addCondition = () => {
    append({
      id: `${Date.now()}-${fields.length}`,
      key: "",
      value: "",
    });
  };

  return (
    <FormSection
      title="Conditions"
      titleCount={fields.length}
      description="Optional claim or context key/value checks persisted with the policy."
      className="border-t py-2"
      action={
        <Button type="button" size="sm" variant="outline" onClick={addCondition}>
          <PlusIcon />
          Add condition
        </Button>
      }
    >
      <div className="space-y-3">
        {fields.length > 0 ? (
          <div className="space-y-2">
            {fields.map((field, index) => (
              <ConditionRowFields
                key={field.fieldId}
                index={index}
                condition={field}
                onRemove={() => remove(index)}
              />
            ))}
          </div>
        ) : (
          <div className="rounded-md border border-dashed bg-muted/30 p-3 text-xs text-muted-foreground">
            No conditions. The policy applies whenever the resource and operation match.
          </div>
        )}
      </div>
    </FormSection>
  );
}

function ConditionRowFields({
  index,
  condition,
  onRemove,
}: {
  index: number;
  condition: AccessPolicyConditionRow;
  onRemove: () => void;
}) {
  const { control } = useFormContext<AccessPolicyFormValues>();

  return (
    <div className="flex flex-row items-center justify-between">
      <InputField
        control={control}
        name={`conditionRows.${index}.key`}
        label="Condition Key"
        placeholder="Claim"
        description="Claim, signal, or context key to evaluate."
        defaultValue={condition.key}
      />
      <InputField
        control={control}
        name={`conditionRows.${index}.value`}
        label="Condition Value"
        placeholder="Expected value"
        description="Expected value for the configured condition key."
        defaultValue={condition.value}
      />
      <div className="flex items-end pb-0.5">
        <Button type="button" size="icon-sm" variant="ghost" onClick={onRemove}>
          <XIcon />
        </Button>
      </div>
    </div>
  );
}
