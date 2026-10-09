/* eslint-disable react-refresh/only-export-components */
import type { PresenceViewer } from "@/hooks/use-presence-viewers";
import { createContext, useContext } from "react";

const NO_VIEWERS: readonly PresenceViewer[] = [];

const RecordPresenceContext = createContext<readonly PresenceViewer[]>(NO_VIEWERS);

export const RecordPresenceProvider = RecordPresenceContext.Provider;

/** Everyone else with the record in the open panel open too. */
export function useRecordViewers(): readonly PresenceViewer[] {
  return useContext(RecordPresenceContext);
}
