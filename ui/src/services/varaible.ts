import { http } from "@/lib/http-client";
import {
  VariableFormatSchema,
  VariableSchema,
} from "@/lib/schemas/variable-schema";

export async function validateFormatSQL(formatSQL: string) {
  const response = await http.post("/variable-formats/validate/", {
    formatSQL,
  });
  return response.data;
}

export async function testFormat(
  format: VariableFormatSchema,
  testValue: string,
) {
  const response = await http.post("/variable-formats/test/", {
    format,
    testValue,
  });
  return response.data;
}

export async function validateVariableQuery(query: string) {
  const response = await http.post("/variables/validate/", { query });
  return response.data;
}

export async function testVariable(
  variable: VariableSchema,
  testParams: Record<string, any>,
) {
  const response = await http.post("/variables/test/", {
    variable,
    testParams,
  });
  return response.data;
}

export async function getVariablesByContext(context: string) {
  const response = await http.get(`/variables/context/${context}/`);
  return response.data;
}
