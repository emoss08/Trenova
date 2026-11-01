import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  CaseChecklistTemplateFieldsFragmentDoc,
  CaseChecklistTemplatesDocument,
  DeleteCaseChecklistTemplateDocument,
  SaveCaseChecklistTemplateDocument,
  type CaseChecklistItemMode,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  checklistTemplateSchema,
  checklistTemplatesSchema,
  type ChecklistKind,
  type SaveChecklistTemplate,
} from "@/types/case-checklist";

/** How Desk case checklists are laid out, for the organization and its customers. */
export class CaseChecklistService {
  public async list(kind: ChecklistKind, { signal }: { signal?: AbortSignal } = {}) {
    const response = await requestGraphQL({
      document: CaseChecklistTemplatesDocument,
      operationName: "CaseChecklistTemplates",
      variables: { kind },
      signal,
    });
    const templates = response.caseChecklistTemplates;

    return safeParse(
      checklistTemplatesSchema,
      {
        kind: templates.kind,
        organization: getFragmentData(
          CaseChecklistTemplateFieldsFragmentDoc,
          templates.organization,
        ),
        customers: templates.customers.map((customer) =>
          getFragmentData(CaseChecklistTemplateFieldsFragmentDoc, customer),
        ),
        locked: templates.locked,
      },
      "Case Checklist",
    );
  }

  public async save(template: SaveChecklistTemplate) {
    const response = await requestGraphQL({
      document: SaveCaseChecklistTemplateDocument,
      operationName: "SaveCaseChecklistTemplate",
      variables: {
        input: {
          id: template.id || null,
          version: template.version,
          kind: template.kind,
          customerId: template.customerId || null,
          items: template.items.map((item) => ({
            key: item.key,
            mode: item.mode as CaseChecklistItemMode,
            custom: item.custom
              ? {
                  label: item.custom.label,
                  check: item.custom.check,
                  documentTypeId: item.custom.documentTypeId || null,
                  stepLabel: item.custom.stepLabel ?? null,
                  prompt: item.custom.prompt ?? null,
                }
              : null,
          })),
        },
      },
    });

    return safeParse(
      checklistTemplateSchema,
      getFragmentData(CaseChecklistTemplateFieldsFragmentDoc, response.saveCaseChecklistTemplate),
      "Case Checklist",
    );
  }

  /** A customer's template goes back to the organization's; the organization's to the default. */
  public async remove(id: string) {
    await requestGraphQL({
      document: DeleteCaseChecklistTemplateDocument,
      operationName: "DeleteCaseChecklistTemplate",
      variables: { id },
    });
  }
}
