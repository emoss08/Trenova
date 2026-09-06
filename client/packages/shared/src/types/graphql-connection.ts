import type { ResultOf } from "@graphql-typed-document-node/core";
import type { TypedGraphQLDocument } from "./graphql";

type FragmentRefsKey = " $fragmentRefs";
type FragmentNameKey = " $fragmentName";
type FragmentMarkerKey = FragmentRefsKey | FragmentNameKey;

type IsAny<T> = 0 extends 1 & T ? true : false;

type Simplify<T> = { [K in keyof T]: T[K] } & {};

type IntersectBoxed<U> = (U extends [unknown] ? (value: U[0]) => void : never) extends (
  value: infer I,
) => void
  ? I
  : never;

type IntersectFragmentRefs<TRefs> = IntersectBoxed<
  { [K in keyof TRefs]: [UnmaskFragments<TRefs[K]>] }[keyof TRefs]
>;

type FragmentRefsOf<T> = FragmentRefsKey extends keyof T ? NonNullable<T[FragmentRefsKey]> : never;

type UnmaskedFragmentRefs<T> = [FragmentRefsOf<T>] extends [never]
  ? unknown
  : IntersectFragmentRefs<FragmentRefsOf<T>>;

type UnmaskObject<T> = Simplify<
  {
    [K in keyof T as K extends FragmentMarkerKey ? never : K]: UnmaskFragments<T[K]>;
  } & UnmaskedFragmentRefs<T>
>;

export type UnmaskFragments<T> =
  IsAny<T> extends true
    ? T
    : T extends ReadonlyArray<infer U>
      ? T extends unknown[]
        ? UnmaskFragments<U>[]
        : readonly UnmaskFragments<U>[]
      : T extends (...args: never[]) => unknown
        ? T
        : T extends object
          ? UnmaskObject<T>
          : T;

type ConnectionShape = {
  edges: ReadonlyArray<{ node: unknown }>;
  pageInfo: unknown;
};

export type ConnectionKeys<TResult> = {
  [K in keyof TResult & string]-?: NonNullable<TResult[K]> extends ConnectionShape ? K : never;
}[keyof TResult & string];

export type ConnectionNode<TResult, K extends ConnectionKeys<TResult>> =
  NonNullable<TResult[K]> extends { edges: ReadonlyArray<{ node: infer TNode }> } ? TNode : never;

export type DataTableRow<
  TDocument extends TypedGraphQLDocument<unknown, never>,
  K extends ConnectionKeys<ResultOf<TDocument>>,
> = UnmaskFragments<ConnectionNode<ResultOf<TDocument>, K>>;
