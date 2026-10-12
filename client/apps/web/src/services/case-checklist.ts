import { api } from "@trenova/shared/lib/api";
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
    const response = await api.get(`/case-checklists/?kind=${encodeURIComponent(kind)}`, {
      signal,
    });

    return safeParse(checklistTemplatesSchema, response, "Case Checklist");
  }

  public async save(template: SaveChecklistTemplate) {
    const response = await api.put("/case-checklists/", template);

    return safeParse(checklistTemplateSchema, response, "Case Checklist");
  }

  /** A customer's template goes back to the organization's; the organization's to the default. */
  public async remove(id: string) {
    await api.delete(`/case-checklists/${id}/`);
  }
}
