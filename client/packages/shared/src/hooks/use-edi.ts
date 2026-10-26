import { use, useSyncExternalStore } from "react";
import { getEdiSnapshot, loadEdi, subscribeEdi, type EdiRuntime } from "@trenova/shared/edi/runtime";

// useEdi returns the loaded EDI module, suspending the component (React's use()) until it has
// loaded. Render code and CodeMirror extensions then call it synchronously. A failed load
// rejects to the nearest error boundary, and the next render that asks loads again.
export function useEdi(): EdiRuntime {
  const loaded = useSyncExternalStore(subscribeEdi, getEdiSnapshot, getEdiSnapshot);
  return loaded ?? use(loadEdi());
}
