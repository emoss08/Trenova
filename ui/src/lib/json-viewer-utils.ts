/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export function isLineRemoved(line: string, newLines: string[]): boolean {
  // * Quick sanity check - if the exact line exists in newLines, it wasn't removed
  if (newLines.includes(line)) {
    return false;
  }

  const trimmedLine = line.trim();
  // * Skip lines that are just structural (brackets, commas, etc)
  if (trimmedLine === "" || /^[{}[\],]$/.test(trimmedLine)) {
    return false;
  }

  // * For property values, we need to identify if the property exists with a different value
  if (trimmedLine.includes(":")) {
    // * Extract the property key (removing quotes and whitespace)
    const keyMatch = trimmedLine.match(/"([^"]+)"/);
    if (keyMatch) {
      const key = keyMatch[1];

      // * Look for the same key in new lines
      const keyPattern = new RegExp(`"${key}"\\s*:`);
      const newLineWithSameKey = newLines.find((nl) => keyPattern.test(nl));

      // * If the key exists in newLines but with a different value, it was modified (consider as removed)
      if (newLineWithSameKey && newLineWithSameKey !== line) {
        // Normalize lines by removing trailing commas before comparison
        const normalizedLine = line.replace(/,\s*$/, "");
        const normalizedNewLine = newLineWithSameKey.replace(/,\s*$/, "");

        if (normalizedLine !== normalizedNewLine) {
          return true;
        }
        return false;
      }

      // * If the key doesn't exist at all in newLines, it was removed completely
      if (!newLineWithSameKey) {
        return true;
      }
    }
  }

  // *For other significant content (not just structural elements),
  // *if it doesn't exist in newLines, consider it removed
  const isSignificantContent =
    trimmedLine.includes('"') ||
    trimmedLine.includes("true") ||
    trimmedLine.includes("false") ||
    trimmedLine.includes("null") ||
    /\d+/.test(trimmedLine);

  if (
    isSignificantContent &&
    !newLines.some((nl) => {
      // Normalize by removing trailing commas for comparison
      const normalizedLine = trimmedLine.replace(/,\s*$/, "");
      const normalizedNl = nl.trim().replace(/,\s*$/, "");
      return normalizedNl.includes(normalizedLine);
    })
  ) {
    return true;
  }

  return false;
}

export function isLineAdded(line: string, oldLines: string[]): boolean {
  // * Quick sanity check - if the exact line exists in oldLines, it wasn't added
  if (oldLines.includes(line)) {
    return false;
  }

  const trimmedLine = line.trim();
  // * Skip lines that are just structural (brackets, commas, etc)
  if (trimmedLine === "" || /^[{}[\],]$/.test(trimmedLine)) {
    return false;
  }

  // * For property values, we need to identify if the property exists with a different value
  if (trimmedLine.includes(":")) {
    // * Extract the property key (removing quotes and whitespace)
    const keyMatch = trimmedLine.match(/"([^"]+)"/);
    if (keyMatch) {
      const key = keyMatch[1];

      // * Look for the same key in old lines
      const keyPattern = new RegExp(`"${key}"\\s*:`);
      const oldLineWithSameKey = oldLines.find((ol) => keyPattern.test(ol));

      // * If the key exists in oldLines but with a different value, it was modified (consider as added)
      if (oldLineWithSameKey && oldLineWithSameKey !== line) {
        // Normalize lines by removing trailing commas before comparison
        const normalizedLine = line.replace(/,\s*$/, "");
        const normalizedOldLine = oldLineWithSameKey.replace(/,\s*$/, "");

        if (normalizedLine !== normalizedOldLine) {
          return true;
        }
        return false;
      }

      // * If the key doesn't exist at all in oldLines, it was added completely
      if (!oldLineWithSameKey) {
        return true;
      }
    }
  }

  // * For other significant content (not just structural elements),
  // * if it doesn't exist in oldLines, consider it added
  const isSignificantContent =
    trimmedLine.includes('"') ||
    trimmedLine.includes("true") ||
    trimmedLine.includes("false") ||
    trimmedLine.includes("null") ||
    /\d+/.test(trimmedLine);

  if (
    isSignificantContent &&
    !oldLines.some((ol) => {
      // Normalize by removing trailing commas for comparison
      const normalizedLine = trimmedLine.replace(/,\s*$/, "");
      const normalizedOl = ol.trim().replace(/,\s*$/, "");
      return normalizedOl.includes(normalizedLine);
    })
  ) {
    return true;
  }

  return false;
}
