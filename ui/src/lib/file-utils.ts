/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

export const generateFilename = (baseName: string, extension: string) => {
  const date = new Date();
  const timestamp = `_${date.getFullYear()}${(date.getMonth() + 1)
    .toString()
    .padStart(2, "0")}${date.getDate().toString().padStart(2, "0")}_${date
    .getHours()
    .toString()
    .padStart(2, "0")}${date.getMinutes().toString().padStart(2, "0")}${date
    .getSeconds()
    .toString()
    .padStart(2, "0")}`;
  return `${baseName}${timestamp}.${extension}`;
};

// Helper function to convert data to CSV
export const convertToCsv = (data: {
  columns: string[];
  rows: any[][];
}): string => {
  if (!data || !data.columns || !data.rows) return "";
  const header = data.columns.join(",") + "\n";
  const rows = data.rows
    .map((row) =>
      row
        .map((cell) => {
          const cellStr = String(
            cell === null || cell === undefined ? "" : cell,
          );
          // Escape quotes and commas
          return `"${cellStr.replace(/"/g, '""')}"`;
        })
        .join(","),
    )
    .join("\n");
  return header + rows;
};

// Helper function to trigger file download
export const downloadFile = (
  filename: string,
  content: string,
  mimeType: string,
) => {
  const blob = new Blob([content], { type: mimeType });
  const link = document.createElement("a");
  link.href = URL.createObjectURL(blob);
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(link.href);
};
