// Kept apart from capture-cover-sheet.ts, which brings pdf-lib: the dialog that prints the
// sheets loads the builder only when someone asks for them, and opens the result from here.

/**
 * Opens the finished sheets in a new tab, where the browser's own viewer
 * prints them. The tab is opened without "noopener", because with it
 * window.open returns null whether or not a tab opened; the opener is cut by
 * hand instead. A download is offered only when the tab was really blocked.
 */
export function openCoverSheetsForPrinting(
  bytes: Uint8Array,
  fileName: string,
): "opened" | "downloaded" {
  const url = URL.createObjectURL(new Blob([new Uint8Array(bytes)], { type: "application/pdf" }));
  window.setTimeout(() => URL.revokeObjectURL(url), 60_000);

  const tab = window.open(url, "_blank");
  if (tab !== null) {
    tab.opener = null;
    return "opened";
  }

  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  link.click();
  return "downloaded";
}
