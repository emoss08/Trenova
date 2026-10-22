import { use, useSyncExternalStore } from "react";
import { loadRules } from "@trenova/shared/rules/load";
import { getRulesSnapshot, subscribeRules, type RulesRuntime } from "@trenova/shared/rules/runtime";

// useRules returns the loaded rules module, suspending the component (React's use()) until it
// has loaded. Render code can then call it synchronously: `useRules().call("tariff.check", …)`.
// Once loaded it never suspends again. A failed load rejects to the nearest error boundary,
// and the next render that asks loads again.
export function useRules(): RulesRuntime {
  const loaded = useSyncExternalStore(subscribeRules, getRulesSnapshot, getRulesSnapshot);
  return loaded ?? use(loadRules());
}
