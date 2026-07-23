import { useMutation } from "@tanstack/react-query";
import { testExpression } from "@/lib/formula-template-api";
import type {
  TestExpressionRequest,
  TestExpressionResponse,
} from "@/types/formula-template";

export function useExpressionTest() {
  return useMutation<TestExpressionResponse, Error, TestExpressionRequest>({
    mutationFn: testExpression,
  });
}
