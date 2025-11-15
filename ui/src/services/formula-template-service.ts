/*
 * Formula Template Service
 * API client for formula template operations
 */

import { apiClient } from "./api-client";

export interface VariableInfo {
  name: string;
  type: string;
  description: string;
  category: string;
  example?: string;
}

export interface FunctionInfo {
  name: string;
  description: string;
  minArgs: number;
  maxArgs: number; // -1 means unlimited
  signature: string;
  example: string;
  category: string;
}

export interface VariablesResponse {
  variables: VariableInfo[];
  categories: string[];
  count: number;
}

export interface FunctionsResponse {
  functions: FunctionInfo[];
  categories: string[];
  count: number;
}

export interface ValidateExpressionRequest {
  expression: string;
}

export interface ValidateExpressionResponse {
  valid: boolean;
  error?: string;
  line?: number;
  column?: number;
  message?: string;
}

export interface TestFormulaRequest {
  expression: string;
  variables: Record<string, any>;
}

export interface TestFormulaResponse {
  success: boolean;
  result?: any;
  error?: string;
  usedVariables?: string[];
  steps?: string[];
  resultType?: string;
}

export const formulaTemplateService = {
  /**
   * Get all available variables for autocomplete
   */
  async getVariables(category?: string): Promise<VariablesResponse> {
    const params = category ? { category } : {};
    const response = await apiClient.get<VariablesResponse>(
      "/formula-templates/variables",
      { params }
    );
    return response.data;
  },

  /**
   * Get all available functions for autocomplete
   */
  async getFunctions(category?: string): Promise<FunctionsResponse> {
    const params = category ? { category } : {};
    const response = await apiClient.get<FunctionsResponse>(
      "/formula-templates/functions",
      { params }
    );
    return response.data;
  },

  /**
   * Validate a formula expression syntax
   */
  async validateExpression(
    expression: string
  ): Promise<ValidateExpressionResponse> {
    const response = await apiClient.post<ValidateExpressionResponse>(
      "/formula-templates/validate",
      { expression }
    );
    return response.data;
  },

  /**
   * Test a formula with sample data
   */
  async testFormula(
    expression: string,
    variables: Record<string, any>
  ): Promise<TestFormulaResponse> {
    const response = await apiClient.post<TestFormulaResponse>(
      "/formula-templates/test",
      { expression, variables }
    );
    return response.data;
  },
};
