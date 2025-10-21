import { http } from "@/lib/http-client";
import type { VariableSchema } from "@/lib/schemas/variable-schema";

interface ValidateResponse {
  valid: boolean;
  error?: string;
}

type ValidateFormatSQLRequest = {
  formatSQL: string;
};

type TestFormatRequest = {
  formatSQL: string;
  testValue: string;
};

interface TestFormatResponse {
  result: string;
}

interface TestVariableResponse {
  result: string;
}

export class VariableAPI {
  async validateFormatSQL(request: ValidateFormatSQLRequest) {
    const response = await http.post<ValidateResponse>(
      "/variable-formats/validate/",
      request,
    );
    return response.data;
  }

  async testFormat(request: TestFormatRequest) {
    const response = await http.post<TestFormatResponse>(
      "/variable-formats/test/",
      request,
    );
    return response.data;
  }

  async validateVariableQuery(query: string) {
    const response = await http.post<ValidateResponse>("/variables/validate/", {
      query,
    });
    return response.data;
  }

  async testVariable(query: string, testParams: Record<string, any>) {
    const response = await http.post<TestVariableResponse>("/variables/test/", {
      query,
      testParams,
    });
    return response.data;
  }

  async getVariablesByContext(context: string) {
    const response = await http.get<VariableSchema[]>(
      `/variables/context/${context}/`,
    );
    return response.data;
  }
}
