/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// Enhanced sensitive data detection patterns
export const SENSITIVE_PATTERNS = {
  // Exact matches for redacted/omitted data
  redacted: /^\[REDACTED\]$/,
  omitted: /^null$/,

  // Masked data patterns
  masked: /^[*•]{3,}$/, // 3 or more asterisks or bullets
  partiallyMasked: /^.{1,3}[*•]{3,}.{0,3}$/, // Pattern like "a***b" or "abc***xyz"

  // ID patterns with masking
  maskedId: /^[a-zA-Z]+_[*]+$/, // Pattern like "usr_****"

  // Email patterns with masking
  maskedEmail: /^.{0,3}[*]+@.+$/, // Pattern like "a***@domain.com"

  // API key patterns
  maskedApiKey: /^[A-Za-z0-9]{1,5}[*]{20,}[A-Za-z0-9]{0,5}$/, // Pattern like "AIza***KQU"
};

export type SensitiveDataType = "redacted" | "omitted" | "masked" | "partial" | null;

export interface SensitiveDataInfo {
  isSensitive: boolean;
  type: SensitiveDataType;
}

// Function to detect sensitive data type
export function detectSensitiveDataType(value: any): SensitiveDataInfo {
  if (value === null || value === undefined) {
    return { isSensitive: false, type: null };
  }

  const strValue = String(value);

  if (SENSITIVE_PATTERNS.redacted.test(strValue)) {
    return { isSensitive: true, type: "redacted" };
  }

  if (
    SENSITIVE_PATTERNS.masked.test(strValue) ||
    SENSITIVE_PATTERNS.maskedId.test(strValue)
  ) {
    return { isSensitive: true, type: "masked" };
  }

  if (
    SENSITIVE_PATTERNS.partiallyMasked.test(strValue) ||
    SENSITIVE_PATTERNS.maskedEmail.test(strValue) ||
    SENSITIVE_PATTERNS.maskedApiKey.test(strValue)
  ) {
    return { isSensitive: true, type: "partial" };
  }

  // Check if the value contains asterisks in a meaningful way
  if (typeof value === "string" && value.includes("*") && value.length > 3) {
    // Check if it's likely a masked value (has enough asterisks)
    const asteriskCount = (value.match(/[*•]/g) || []).length;
    if (asteriskCount >= 3) {
      return { isSensitive: true, type: "partial" };
    }
  }

  return { isSensitive: false, type: null };
}

// Helper to format JSON with spaces after colons
export function formatJsonWithSpaces(data: any): string {
  if (!data) return "";
  try {
    // First stringify with proper indentation
    const jsonString = JSON.stringify(data, null, 2);
    // Add space after colons for better readability
    return jsonString.replace(/("(?:[^"\\]|\\.)*"):/g, '$1: ');
  } catch (error) {
    console.error("Error formatting data:", error);
    return "";
  }
}

// Check if a string value contains sensitive patterns for line-based detection
export function containsSensitivePattern(line: string): boolean {
  // Quick checks for common patterns
  if (line.includes(': "****"')) return true;
  if (line.includes(': "[REDACTED]"')) return true;
  
  // Check for masked values in JSON format
  const valueMatch = line.match(/:\s*"([^"]+)"/);
  if (valueMatch) {
    const value = valueMatch[1];
    const info = detectSensitiveDataType(value);
    return info.isSensitive;
  }
  
  return false;
}