import type { DocumentTypeDecoration } from "@graphql-typed-document-node/core";

export type GraphQLDocumentMeta = {
  hash?: string;
};

export type GraphQLExecutableDocument =
  | string
  | {
      __meta__?: GraphQLDocumentMeta;
      toString(): string;
    };

export type TypedGraphQLDocument<TResult = unknown, TVariables = never> =
  DocumentTypeDecoration<TResult, TVariables> & {
    __meta__?: GraphQLDocumentMeta;
    toString(): string;
  };

export type GraphQLDocument<TResult = unknown, TVariables = never> =
  | string
  | TypedGraphQLDocument<TResult, TVariables>;
