import type { PaletteRecord } from "../../palette-model";

/** What every record preview body takes: the record, and a way to open another record it names. */
export interface RecordPreviewProps {
  record: PaletteRecord;
  onNavigate: (href: string) => void;
}
