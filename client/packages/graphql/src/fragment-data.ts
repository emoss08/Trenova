/**
 * A narrowed `getFragmentData`.
 *
 * The generated `fragment-masking.ts` declares a `Partial<TType>` overload first,
 * to cover fragments made partial by `@include`/`@skip`. TypeScript picks the first
 * overload whose parameter type matches, and a non-nullable fragment ref matches the
 * partial one structurally — so every unmasked fragment widened to `Partial<T>` and
 * every field became optional.
 *
 * Trenova does not use conditional directives on masked fragments, so that overload is
 * unreachable in practice while poisoning inference everywhere else. This re-declares
 * the remaining overloads and delegates to the generated implementation, which keeps
 * `fragment-masking.ts` byte-identical to codegen output.
 *
 * Drop this module if the codegen preset stops emitting the partial overload first, or
 * if masked fragments start using conditional directives — at which point the partial
 * return type becomes correct and call sites need real null handling.
 */
import type { DocumentTypeDecoration } from "@graphql-typed-document-node/core";

import {
  type FragmentType,
  getFragmentData as getGeneratedFragmentData,
} from "./generated/fragment-masking";

export type { FragmentType };

export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: FragmentType<DocumentTypeDecoration<TType, any>>,
): TType;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: FragmentType<DocumentTypeDecoration<TType, any>> | undefined,
): TType | undefined;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: FragmentType<DocumentTypeDecoration<TType, any>> | null,
): TType | null;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: FragmentType<DocumentTypeDecoration<TType, any>> | null | undefined,
): TType | null | undefined;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: Array<FragmentType<DocumentTypeDecoration<TType, any>>>,
): Array<TType>;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: Array<FragmentType<DocumentTypeDecoration<TType, any>>> | null | undefined,
): Array<TType> | null | undefined;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: ReadonlyArray<FragmentType<DocumentTypeDecoration<TType, any>>>,
): ReadonlyArray<TType>;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType:
    | ReadonlyArray<FragmentType<DocumentTypeDecoration<TType, any>>>
    | null
    | undefined,
): ReadonlyArray<TType> | null | undefined;
export function getFragmentData<TType>(
  documentNode: DocumentTypeDecoration<TType, any>,
  fragmentType: unknown,
): unknown {
  return (
    getGeneratedFragmentData as (
      node: DocumentTypeDecoration<TType, any>,
      fragment: unknown,
    ) => unknown
  )(documentNode, fragmentType);
}
