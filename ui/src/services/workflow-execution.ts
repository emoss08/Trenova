import { http } from "@/lib/http-client";
import type {
  ExecutionStatusType,
  TriggerWorkflowRequestSchema,
  WorkflowExecutionSchema,
  WorkflowExecutionStepSchema,
} from "@/lib/schemas/workflow-schema";
import type { ListResult } from "@/types/server";

export type ListExecutionsParams = {
  workflowId?: string;
  status?: ExecutionStatusType;
  page?: number;
  limit?: number;
};

export class WorkflowExecutionAPI {
  /**
   * List workflow executions with optional filters
   */
  async list(params?: ListExecutionsParams) {
    const response = await http.get<ListResult<WorkflowExecutionSchema>>(
      "/workflow-executions/",
      { params },
    );
    return response.data;
  }

  /**
   * Get a single execution by ID
   */
  async getById(id: string) {
    const response = await http.get<WorkflowExecutionSchema>(
      `/workflow-executions/${id}/`,
    );
    return response.data;
  }

  /**
   * Get execution steps for a specific execution
   */
  async getSteps(executionId: string) {
    const response = await http.get<WorkflowExecutionStepSchema[]>(
      `/workflow-executions/${executionId}/steps/`,
    );
    return response.data;
  }

  /**
   * Trigger a workflow execution manually
   */
  async trigger(workflowId: string, data?: TriggerWorkflowRequestSchema) {
    const response = await http.post<WorkflowExecutionSchema>(
      `/workflow-executions/trigger/${workflowId}/`,
      data || {},
    );
    return response.data;
  }

  /**
   * Cancel a running execution
   */
  async cancel(executionId: string) {
    const response = await http.post<WorkflowExecutionSchema>(
      `/workflow-executions/${executionId}/cancel/`,
    );
    return response.data;
  }

  /**
   * Retry a failed execution
   */
  async retry(executionId: string) {
    const response = await http.post<WorkflowExecutionSchema>(
      `/workflow-executions/${executionId}/retry/`,
    );
    return response.data;
  }
}

export const workflowExecutionAPI = new WorkflowExecutionAPI();
