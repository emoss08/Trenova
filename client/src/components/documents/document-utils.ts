export function getFileCategory(fileType: string, fileName?: string): string {
  const type = fileType.toLowerCase();
  const ext = fileName?.split(".").pop()?.toLowerCase() ?? "";

  if (type === "application/pdf" || ext === "pdf") {
    return "pdf";
  }
  if (type.startsWith("image/")) {
    return "image";
  }
  if (
    type.includes("spreadsheet") ||
    type.includes("excel") ||
    ["xlsx", "xls", "csv"].includes(ext)
  ) {
    return "spreadsheet";
  }
  if (
    type.includes("word") ||
    type.includes("document") ||
    type === "text/plain" ||
    ["doc", "docx", "txt", "rtf"].includes(ext)
  ) {
    return "document";
  }
  return "default";
}
