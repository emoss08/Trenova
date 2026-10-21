import {
  REFERENCE_VIEWS,
  type ReferenceKind,
} from "@trenova/shared/types/generated/reference-views";
import { z } from "zod";

/**
 * Part of every kind's reference view: a record's id, its tenant and its revision say nothing
 * about its content. A kind with no view of its own shows only these.
 */
const REFERENCE_BOOKKEEPING = [
  "id",
  "organizationId",
  "businessUnitId",
  "version",
  "createdAt",
  "updatedAt",
] as const;

type Bookkeeping = (typeof REFERENCE_BOOKKEEPING)[number];

/**
 * A field of a kind's reference view: what any record linking to it may always show. A kind
 * with no view of its own shows only its bookkeeping fields.
 */
export type ReferenceField<K extends string> = K extends ReferenceKind
  ? (typeof REFERENCE_VIEWS)[K][number]
  : Bookkeeping;

type LinkedShape<K extends string, S extends z.core.$ZodShape> = {
  -readonly [F in keyof S]: F extends ReferenceField<K> ? S[F] : z.ZodOptional<z.ZodNullable<S[F]>>;
};

function referenceView(kind: string): readonly string[] {
  return Object.hasOwn(REFERENCE_VIEWS, kind)
    ? REFERENCE_VIEWS[kind as ReferenceKind]
    : REFERENCE_BOOKKEEPING;
}

/**
 * A record of another kind as a REST response carries it inside the record the route returned:
 * a shipment's customer, a billing queue item's biller. The server shows every reader the kind's
 * reference view; any other field is `null` for a reader without the kind's read, and the
 * response names the record in its `Withheld-By-Access` header (see
 * docs/engineering/reference-views.md). Every field outside the view is therefore nullable here,
 * and a screen that draws one draws `WithheldValue` where it is null.
 */
export function linkedRecordSchema<K extends string, S extends z.core.$ZodShape>(
  kind: K,
  schema: z.ZodObject<S>,
): z.ZodObject<LinkedShape<K, S>> {
  const view = new Set<string>(referenceView(kind));
  const shape: Record<string, z.ZodType> = {};
  for (const [field, fieldSchema] of Object.entries(schema.shape)) {
    const type = fieldSchema as z.ZodType;
    shape[field] = view.has(field) ? type : type.nullable().optional();
  }

  return z.object(shape) as unknown as z.ZodObject<LinkedShape<K, S>>;
}

/** Whether a field of a kind is one every reader of a linking record is shown. */
export function isReferenceField(kind: string, field: string): boolean {
  return referenceView(kind).includes(field);
}
