import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type { FieldValues, Resolver } from "react-hook-form";

/**
 * A form resolver whose schema checks something with the rules module, such as whether a
 * charge's allocation rows add up (`withShipmentAllocations`).
 *
 * The module may not have loaded when a form first validates: validation runs on a change, on
 * submit, or when a form subscribed to `isValid` mounts, none of which waits for it. A
 * resolver is asynchronous, so this one waits for the module (loading it if nothing has yet)
 * and then validates with the schema `build` makes from it, built once per loaded module. The
 * loader is imported only when a form first validates, so nothing of the module reaches a
 * bundle that merely imports the form. A failed load rejects the validation, and the next one
 * loads again.
 */
export function rulesResolver<TValues extends FieldValues>(
  build: (rules: RulesRuntime) => Resolver<TValues>,
): Resolver<TValues> {
  let built: { rules: RulesRuntime; resolver: Resolver<TValues> } | null = null;

  return async (values, context, options) => {
    const { loadRules } = await import("@trenova/shared/rules/load");
    const rules = await loadRules();
    if (built?.rules !== rules) built = { rules, resolver: build(rules) };
    return built.resolver(values, context, options);
  };
}
