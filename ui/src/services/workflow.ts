import { http } from "@/lib/http-client";
import type {
  CreateVersionRequestSchema,
  CreateWorkflowRequestSchema,
  SaveDefinitionRequestSchema,
  UpdateWorkflowRequestSchema,
  WorkflowSchema,
  WorkflowStatusType,
  WorkflowVersionSchema,
  TriggerTypeType,
} from "@/lib/schemas/workflow-schema";
import type { ListResult } from "@/types/server";

export type ListWorkflowsParams = {
  status?: WorkflowStatusType;
  triggerType?: TriggerTypeType;
  search?: string;
  page?: number;
  limit?: number;
};

export class WorkflowAPI {
  /**
   * List workflows with optional filters
   */
  async list(params?: ListWorkflowsParams) {
    const response = await http.get<ListResult<WorkflowSchema>>(
      "/workflows/",
      { params },
    );
    return response.data;
  }

  /**
   * Get a single workflow by ID
   */
  async getById(id: string) {
    const response = await http.get<WorkflowSchema>(`/workflows/${id}/`);
    return response.data;
  }

  /**
   * Create a new workflow
   */
  async create(data: CreateWorkflowRequestSchema) {
    const response = await http.post<WorkflowSchema>("/workflows/", data);
    return response.data;
  }

  /**
   * Update an existing workflow
   */
  async update(id: string, data: UpdateWorkflowRequestSchema) {
    const response = await http.put<WorkflowSchema>(`/workflows/${id}/`, data);
    return response.data;
  }

  /**
   * Delete a workflow
   */
  async delete(id: string) {
    await http.delete(`/workflows/${id}/`);
  }

  /**
   * List versions for a workflow
   */
  async listVersions(workflowId: string) {
    const response = await http.get<WorkflowVersionSchema[]>(
      `/workflows/${workflowId}/versions/`,
    );
    return response.data;
  }

  /**
   * Create a new version of a workflow
   */
  async createVersion(
    workflowId: string,
    data: CreateVersionRequestSchema,
  ) {
    const response = await http.post<WorkflowVersionSchema>(
      `/workflows/${workflowId}/versions/`,
      data,
    );
    return response.data;
  }

  /**
   * Get a specific version
   */
  async getVersion(workflowId: string, versionId: string) {
    const response = await http.get<WorkflowVersionSchema>(
      `/workflows/${workflowId}/versions/${versionId}/`,
    );
    return response.data;
  }

  /**
   * Publish a workflow version
   */
  async publishVersion(workflowId: string, versionId: string) {
    const response = await http.post<WorkflowSchema>(
      `/workflows/${workflowId}/versions/${versionId}/publish/`,
    );
    return response.data;
  }

  /**
   * Unpublish a workflow version
   */
  async unpublishVersion(workflowId: string, versionId: string) {
    const response = await http.post<WorkflowSchema>(
      `/workflows/${workflowId}/versions/${versionId}/unpublish/`,
    );
    return response.data;
  }

  /**
   * Save workflow definition (nodes and edges)
   */
  async saveDefinition(
    workflowId: string,
    versionId: string,
    data: SaveDefinitionRequestSchema,
  ) {
    const response = await http.put<WorkflowVersionSchema>(
      `/workflows/${workflowId}/versions/${versionId}/definition/`,
      data,
    );
    return response.data;
  }

  /**
   * Activate a workflow (will setup triggers)
   */
  async activate(workflowId: string) {
    const response = await http.post<WorkflowSchema>(
      `/workflows/${workflowId}/activate/`,
    );
    return response.data;
  }

  /**
   * Deactivate a workflow (will pause triggers)
   */
  async deactivate(workflowId: string) {
    const response = await http.post<WorkflowSchema>(
      `/workflows/${workflowId}/deactivate/`,
    );
    return response.data;
  }

  /**
   * Archive a workflow (will remove triggers)
   */
  async archive(workflowId: string) {
    const response = await http.post<WorkflowSchema>(
      `/workflows/${workflowId}/archive/`,
    );
    return response.data;
  }
}

export const workflowAPI = new WorkflowAPI();
