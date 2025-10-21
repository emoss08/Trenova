import { InvalidParam } from "@/types/errors";
import { useCallback, useState } from "react";

export function useValidationWarnings() {
  const [warnings, setWarnings] = useState<Record<string, string>>({});

  const addWarnings = useCallback((fieldErrors: InvalidParam[]) => {
    const newWarnings: Record<string, string> = {};
    fieldErrors.forEach((error) => {
      if (error.priority === "LOW") {
        newWarnings[error.name] = error.reason;
      }
    });
    setWarnings(newWarnings);
  }, []);

  const clearWarning = useCallback((field: string) => {
    setWarnings((prev) => {
      const newWarnings = { ...prev };
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete newWarnings[field];
      return newWarnings;
    });
  }, []);

  const clearAllWarnings = useCallback(() => {
    setWarnings({});
  }, []);

  const getWarning = useCallback(
    (field: string): string | undefined => {
      return warnings[field];
    },
    [warnings],
  );

  const hasWarnings = useCallback(() => {
    return Object.keys(warnings).length > 0;
  }, [warnings]);

  return {
    warnings,
    addWarnings,
    clearWarning,
    clearAllWarnings,
    getWarning,
    hasWarnings,
  };
}
