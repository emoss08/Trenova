import { usePresenceViewers } from "@/hooks/use-presence-viewers";
import { recordPresence, viewPresence } from "@/lib/presence-scopes";
import { RecordPresenceProvider, useRecordViewers } from "@/contexts/record-presence-context";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { ViewersStack } from "./viewers-stack";

/** Who else is looking at this table, for its toolbar. */
export function DataTableViewers({ resource }: { resource: string }) {
  const t = useT();
  const { viewers } = usePresenceViewers(viewPresence(resource));
  return <ViewersStack viewers={viewers} label={(names) => t("Also on this table: {0}", names)} />;
}

/**
 * Joins the open record's presence while its panel is open for editing, and tells the
 * panel who else has it open, so two people do not overwrite each other unawares.
 */
export function RecordPresence({
  resource,
  recordId,
  children,
}: {
  resource: string | undefined;
  recordId: string | null | undefined;
  children: ReactNode;
}) {
  const { viewers } = usePresenceViewers(
    resource && recordId ? recordPresence(resource, recordId) : null,
  );
  return <RecordPresenceProvider value={viewers}>{children}</RecordPresenceProvider>;
}

/** Who else has the open record open, for the panel's header. */
export function RecordViewers() {
  const t = useT();
  const viewers = useRecordViewers();
  return <ViewersStack viewers={viewers} label={(names) => t("Also has this open: {0}", names)} />;
}
