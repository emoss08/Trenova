import { useQueryState } from "nuqs";
import { useCallback } from "react";
import {
  EXTRACTION_DIALOG_PARAM,
  extractionDialogParser,
  type ExtractionDialog,
} from "../../../ai-control-tabs";

/** One of document extraction's dialogs, open while the address names it. */
export function useExtractionDialog(dialog: ExtractionDialog) {
  const [current, setCurrent] = useQueryState(EXTRACTION_DIALOG_PARAM, extractionDialogParser);
  const setOpen = useCallback(
    (open: boolean) => void setCurrent(open ? dialog : null),
    [dialog, setCurrent],
  );
  return [current === dialog, setOpen] as const;
}
