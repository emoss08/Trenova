import { RecordKindIDPrefix, type RecordKind } from "@trenova/shared/types/generated/record-kinds";

const KIND_BY_PREFIX: ReadonlyMap<string, RecordKind> = new Map(
  (Object.entries(RecordKindIDPrefix) as [RecordKind, string][]).map(([kind, prefix]) => [
    prefix,
    kind,
  ]),
);

/** The kind of record an id is, read from its prefix (`wcred_…` is a worker credential); null when no kind mints it. */
export function recordKindOf(id: string): RecordKind | null {
  const cut = id.indexOf("_");
  if (cut <= 0) return null;
  return KIND_BY_PREFIX.get(id.slice(0, cut + 1)) ?? null;
}
