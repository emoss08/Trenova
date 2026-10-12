import { isRecordEntityType, RECORD_LINKS } from "@/config/record-links";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { humanizeToolName } from "@/components/assistant/proposal-state";

/** A notice in place of a body, for an artifact that holds nothing to show. */
export function Notice({ children }: { children: React.ReactNode }) {
  return <div className="dk-ax-pad dk-ax-oldnote">{children}</div>;
}

/**
 * entityName names a record type or a report dataset in the reader's language: a record
 * type by its record-link label, a dataset ("shipments") by its humanized name, which the
 * catalog holds for every dataset the app has a page for.
 */
export function entityName(entity: string, t: TranslateFn): string {
  return isRecordEntityType(entity) ? t(RECORD_LINKS[entity].label) : t(humanizeToolName(entity));
}
