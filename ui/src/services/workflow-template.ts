import { http } from "@/lib/http-client";
import type {
  CreateTemplateRequestSchema,
  UseTemplateRequestSchema,
  WorkflowSchema,
  WorkflowTemplateSchema,
} from "@/lib/schemas/workflow-schema";
import type { ListResult } from "@/types/server";

export type ListTemplatesParams = {
  category?: string;
  search?: string;
  isSystem?: boolean;
  isPublic?: boolean;
  page?: number;
  limit?: number;
};

export class WorkflowTemplateAPI {
  /**
   * List workflow templates with optional filters
   */
  async list(params?: ListTemplatesParams) {
    const response = await http.get<ListResult<WorkflowTemplateSchema>>(
      "/workflow-templates/",
      { params },
    );
    return response.data;
  }

  /**
   * Get a single template by ID
   */
  async getById(id: string) {
    const response = await http.get<WorkflowTemplateSchema>(
      `/workflow-templates/${id}/`,
    );
    return response.data;
  }

  /**
   * Create a new template
   */
  async create(data: CreateTemplateRequestSchema) {
    const response = await http.post<WorkflowTemplateSchema>(
      "/workflow-templates/",
      data,
    );
    return response.data;
  }

  /**
   * Update an existing template
   */
  async update(id: string, data: Partial<CreateTemplateRequestSchema>) {
    const response = await http.put<WorkflowTemplateSchema>(
      `/workflow-templates/${id}/`,
      data,
    );
    return response.data;
  }

  /**
   * Delete a template
   */
  async delete(id: string) {
    await http.delete(`/workflow-templates/${id}/`);
  }

  /**
   * Create a workflow from a template
   */
  async use(templateId: string, data: UseTemplateRequestSchema) {
    const response = await http.post<WorkflowSchema>(
      `/workflow-templates/${templateId}/use/`,
      data,
    );
    return response.data;
  }
}

export const workflowTemplateAPI = new WorkflowTemplateAPI();
