// tariff-service.ts is what every editor feature asks the rules module through: the loaded
// runtime, the file's kind and the organisation's context, read through getters so the
// CodeMirror extensions are built once and still see the latest of each. It also holds the one
// check per text: the linter, the folding service and the Studio's own analysis all ask for the
// same source with the same context, so the module checks it once.

import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import { tariffCheck } from "@trenova/shared/rules/tariff";
import {
  EMPTY_TARIFF_CONTEXT,
  type TariffCheckResult,
  type TariffContext,
  type TariffFileKind,
  type TariffRange,
  type TariffTraceEvent,
} from "@trenova/shared/types/tariff";

/** Where a name is declared outside this file: an imported library and the range in its source. */
/**
 * A declaration in an imported library: the module it is in, its range in that library's
 * source, and the name that was followed (a standard library's source is not shipped to the
 * client, so its declaration is found by name in the catalog).
 */
export type TariffDefinitionTarget = { module: string; range: TariffRange; name: string };

export type TariffLanguageService = {
  rules: () => RulesRuntime;
  kind: () => TariffFileKind;
  context: () => TariffContext;
  /** The last preview's trace, for the values hover cards and inlay hints show. */
  trace: () => readonly TariffTraceEvent[];
  /** Opens a declaration that lives in an imported library. */
  openDefinition: (target: TariffDefinitionTarget) => void;
};

type CacheEntry = {
  rules: RulesRuntime;
  source: string;
  kind: TariffFileKind;
  context: TariffContext;
  result: TariffCheckResult;
};

const CACHE_SIZE = 4;
const cache: CacheEntry[] = [];

/**
 * The module's check of a source, remembered for the last few texts so every consumer of the
 * same text and context shares one check. The context is compared by identity: it comes from
 * one query, so a new object means new knowledge.
 */
export function checkTariffSource(
  rules: RulesRuntime,
  source: string,
  kind: TariffFileKind,
  context: TariffContext | null,
): TariffCheckResult {
  const knowledge = context ?? EMPTY_TARIFF_CONTEXT;
  const hit = cache.find(
    (entry) =>
      entry.rules === rules &&
      entry.source === source &&
      entry.kind === kind &&
      entry.context === knowledge,
  );
  if (hit) return hit.result;
  const result = tariffCheck(rules, { source, kind, context: knowledge });
  cache.unshift({ rules, source, kind, context: knowledge, result });
  cache.length = Math.min(cache.length, CACHE_SIZE);
  return result;
}

/** The check of what a service's editor holds now. */
export function checkService(service: TariffLanguageService, source: string): TariffCheckResult {
  return checkTariffSource(service.rules(), source, service.kind(), service.context());
}
