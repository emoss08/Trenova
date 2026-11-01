import {
  CreateCustomFieldDefinitionDocument,
  DeleteCustomFieldDefinitionDocument,
  PatchCustomFieldDefinitionDocument,
  UpdateCustomFieldDefinitionDocument,
  type CustomFieldDefinitionInput,
  type CustomFieldDefinitionPatchInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { CustomFieldDefinition } from "@/types/custom-field";

export function toCustomFieldDefinitionInput(
  data: CustomFieldDefinition,
): CustomFieldDefinitionInput {
  return {
    resourceType: data.resourceType,
    name: data.name,
    label: data.label,
    description: data.description || null,
    fieldType: data.fieldType,
    isRequired: data.isRequired,
    isActive: data.isActive,
    displayOrder: data.displayOrder,
    color: data.color || null,
    options: data.options.map((option) => ({
      value: option.value,
      label: option.label,
      color: option.color || null,
      description: option.description || null,
    })),
    validationRules: data.validationRules
      ? {
          minLength: data.validationRules.minLength ?? null,
          maxLength: data.validationRules.maxLength ?? null,
          min: data.validationRules.min ?? null,
          max: data.validationRules.max ?? null,
          pattern: data.validationRules.pattern || null,
        }
      : null,
    defaultValue: data.defaultValue ?? null,
    uiAttributes: data.uiAttributes
      ? {
          placeholder: data.uiAttributes.placeholder || null,
          helpText: data.uiAttributes.helpText || null,
          width: data.uiAttributes.width || null,
        }
      : null,
    version: data.version,
  };
}

export class CustomFieldService {
  public async create(data: CustomFieldDefinition) {
    const response = await requestGraphQL({
      document: CreateCustomFieldDefinitionDocument,
      operationName: "CreateCustomFieldDefinition",
      variables: { input: toCustomFieldDefinitionInput(data) },
    });

    return response.createCustomFieldDefinition;
  }

  public async update(id: string, data: CustomFieldDefinition) {
    const response = await requestGraphQL({
      document: UpdateCustomFieldDefinitionDocument,
      operationName: "UpdateCustomFieldDefinition",
      variables: { id, input: toCustomFieldDefinitionInput(data) },
    });

    return response.updateCustomFieldDefinition;
  }

  public async patch(id: string, input: CustomFieldDefinitionPatchInput) {
    const response = await requestGraphQL({
      document: PatchCustomFieldDefinitionDocument,
      operationName: "PatchCustomFieldDefinition",
      variables: { id, input },
    });

    return response.patchCustomFieldDefinition;
  }

  public async delete(id: string): Promise<void> {
    await requestGraphQL({
      document: DeleteCustomFieldDefinitionDocument,
      operationName: "DeleteCustomFieldDefinition",
      variables: { id },
    });
  }
}
