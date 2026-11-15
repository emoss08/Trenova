/**
 * How a value a page keeps in the address is written there and read back. A codec reads one
 * query value (a string, already decoded by the router) and writes one; a value it cannot read
 * (a link cut short, an address edited by hand) reads as nothing, so the page falls back to its
 * default rather than failing.
 */
export type SearchHistory = "push" | "replace";

export interface SearchCodec<T> {
  /** The value a query string holds, or null when it holds none this codec reads. */
  parse(value: string): T | null;
  /** The query string for a value. An empty string leaves the key out of the address. */
  serialize(value: T): string;
  /** Whether two values are the same, so a value equal to the default is left out. */
  eq(a: T, b: T): boolean;
  /** Whether a change of this key is a new history entry; replaced when unset. */
  readonly history?: SearchHistory;
  /** Reads a missing or unreadable value as `value`, and leaves `value` out of the address. */
  withDefault(value: NonNullable<T>): SearchCodecWithDefault<NonNullable<T>>;
  withOptions(options: SearchCodecOptions): SearchCodec<T>;
}

export interface SearchCodecWithDefault<T> extends Omit<
  SearchCodec<T>,
  "withDefault" | "withOptions"
> {
  readonly defaultValue: T;
  withOptions(options: SearchCodecOptions): SearchCodecWithDefault<T>;
}

export interface SearchCodecOptions {
  history?: SearchHistory;
}

/**
 * Any codec, as a map of them is held. Its methods take their value bivariantly, so a codec of
 * any type is one; SearchValues recovers each key's own type from the map.
 */
export type AnySearchCodec = SearchCodec<unknown> | SearchCodecWithDefault<unknown>;

/** The codecs a page reads its address with, by key. */
export type SearchCodecMap = Readonly<Record<string, AnySearchCodec>>;

/** The value one codec reads: its default's type when it has one, else the value or null. */
export type SearchValue<C> =
  C extends SearchCodecWithDefault<infer T> ? T : C extends SearchCodec<infer T> ? T | null : never;

/** The values a codec map reads from the address. */
export type SearchValues<M extends SearchCodecMap> = { [K in keyof M]: SearchValue<M[K]> };

interface CodecParts<T> {
  parse: (value: string) => T | null;
  serialize: (value: T) => string;
  eq?: (a: T, b: T) => boolean;
}

function withDefaultOf<T>(
  parts: Required<CodecParts<T>> & { history?: SearchHistory },
  defaultValue: T,
): SearchCodecWithDefault<T> {
  return {
    ...parts,
    defaultValue,
    withOptions: (options) => withDefaultOf({ ...parts, ...options }, defaultValue),
  };
}

function build<T>(parts: Required<CodecParts<T>> & { history?: SearchHistory }): SearchCodec<T> {
  return {
    ...parts,
    withDefault: (value) =>
      withDefaultOf(parts as unknown as Required<CodecParts<NonNullable<T>>>, value),
    withOptions: (options) => build({ ...parts, ...options }),
  };
}

const same = <T>(a: T, b: T) => a === b;

/** A codec of the page's own: how its value is read from and written to one query value. */
export function searchCodec<T>({ parse, serialize, eq = same }: CodecParts<T>): SearchCodec<T> {
  return build({ parse, serialize, eq });
}

/** The query value as it is. */
export const searchString: SearchCodec<string> = searchCodec<string>({
  parse: (value) => value,
  serialize: (value) => value,
});

/** A whole number; anything that does not start with one reads as nothing. */
export const searchInteger: SearchCodec<number> = searchCodec<number>({
  parse: (value) => {
    const parsed = Number.parseInt(value, 10);
    return Number.isNaN(parsed) ? null : parsed;
  },
  serialize: (value) => Math.round(value).toFixed(),
});

/** `true` or `false`, in any case; anything else is false. */
export const searchBoolean: SearchCodec<boolean> = searchCodec<boolean>({
  parse: (value) => value.toLowerCase() === "true",
  serialize: (value) => String(value),
});

/** One of `values`; anything else reads as nothing. */
export function searchLiteral<const T extends string>(values: readonly T[]): SearchCodec<T> {
  return searchCodec<T>({
    parse: (value) => ((values as readonly string[]).includes(value) ? (value as T) : null),
    serialize: (value) => value,
  });
}

/** One of `values`, a number written as it reads; anything else reads as nothing. */
export function searchNumberLiteral<const T extends number>(values: readonly T[]): SearchCodec<T> {
  return searchCodec<T>({
    parse: (value) => {
      const parsed = Number.parseFloat(value);
      return (values as readonly number[]).includes(parsed) ? (parsed as T) : null;
    },
    serialize: (value) => String(value),
  });
}

/**
 * A list, each item written with `item` and joined by `separator` (an item holding the separator
 * keeps it percent-encoded). An item `item` cannot read is dropped; an empty value is no items.
 */
export function searchList<T>(item: SearchCodec<T>, separator = ","): SearchCodec<T[]> {
  const encoded = encodeURIComponent(separator);
  return searchCodec<T[]>({
    parse: (value) => {
      if (value === "") return [];
      const items: T[] = [];
      for (const part of value.split(separator)) {
        const parsed = item.parse(part.replaceAll(encoded, separator));
        if (parsed !== null) items.push(parsed);
      }
      return items;
    },
    serialize: (values) =>
      values.map((value) => item.serialize(value).replaceAll(separator, encoded)).join(separator),
    eq: (a, b) => a.length === b.length && a.every((value, index) => item.eq(value, b[index] as T)),
  });
}

/**
 * A value written as JSON, read back through `validate`, which returns the value or null when the
 * JSON is not one (a schema's `safeParse`, a hand-written check).
 */
export function searchJson<T>(validate: (value: unknown) => T | null): SearchCodec<T> {
  return searchCodec<T>({
    parse: (value) => {
      try {
        return validate(JSON.parse(value));
      } catch {
        return null;
      }
    },
    serialize: (value) => JSON.stringify(value),
    eq: (a, b) => a === b || JSON.stringify(a) === JSON.stringify(b),
  });
}
