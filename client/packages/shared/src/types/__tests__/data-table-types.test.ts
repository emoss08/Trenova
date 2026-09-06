import {
  AccessorialChargeTableDocument,
  TractorTableDocument,
  WorkerPtoTableDocument,
  type AccessorialChargeTableQuery,
  type AccessorialMethod,
  type EntityStatus,
  type TractorTableQuery,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type {
  ConnectionKeys,
  ConnectionNode,
  DataTableConfigRow,
  DataTableGraphQLConfig,
  DataTableGraphQLSource,
  DataTableRow,
  UnmaskFragments,
} from "@trenova/shared/types/data-table";
import { describe, expect, expectTypeOf, it } from "vitest";

type AddressFragment = { street: string; city: string } & {
  " $fragmentName"?: "AddressFragment";
};

type ContactFragment = {
  email: string;
  address: { " $fragmentRefs"?: { AddressFragment: AddressFragment } } | null;
} & { " $fragmentName"?: "ContactFragment" };

type TagFragment = { label: string } & { " $fragmentName"?: "TagFragment" };

type AuditFragment = { version: number; updatedAt: number } & {
  " $fragmentName"?: "AuditFragment";
};

type RowFragment = {
  id: string;
  name: string;
  contact: { " $fragmentRefs"?: { ContactFragment: ContactFragment } } | null;
  tags: Array<{ " $fragmentRefs"?: { TagFragment: TagFragment } }>;
  metadata: unknown;
  optional?: string;
} & { " $fragmentName"?: "RowFragment" };

type MaskedNode = {
  " $fragmentRefs"?: { RowFragment: RowFragment; AuditFragment: AuditFragment };
};

type SyntheticResult = {
  rows: {
    totalCount?: number | null;
    edges: Array<{ node: MaskedNode }>;
    pageInfo: { hasNextPage: boolean; endCursor: string | null };
  };
  scalar: string;
  detail: { id: string } | null;
  edgesWithoutPageInfo: { edges: Array<{ node: { id: string } }> };
};

describe("UnmaskFragments", () => {
  it("replaces fragment refs with the intersection of the spread fragments", () => {
    type Row = UnmaskFragments<MaskedNode>;

    expectTypeOf<Row["id"]>().toEqualTypeOf<string>();
    expectTypeOf<Row["name"]>().toEqualTypeOf<string>();
    expectTypeOf<Row["version"]>().toEqualTypeOf<number>();
    expectTypeOf<Row["updatedAt"]>().toEqualTypeOf<number>();
    expectTypeOf<Row["metadata"]>().toEqualTypeOf<unknown>();
    expectTypeOf<Row["optional"]>().toEqualTypeOf<string | undefined>();
    expectTypeOf<Row>().not.toHaveProperty(" $fragmentRefs");
    expectTypeOf<Row>().not.toHaveProperty(" $fragmentName");
  });

  it("unmasks nested refs through nullable unions and arrays", () => {
    type Row = UnmaskFragments<MaskedNode>;

    expectTypeOf<Row["contact"]>().toEqualTypeOf<{
      email: string;
      address: { street: string; city: string } | null;
    } | null>();
    expectTypeOf<Row["tags"]>().toEqualTypeOf<Array<{ label: string }>>();
  });

  it("keeps optional property modifiers when there are no fragment markers", () => {
    type Plain = UnmaskFragments<{ id: string; note?: string | null }>;

    expectTypeOf<Plain>().toEqualTypeOf<{ id: string; note?: string | null }>();
  });

  it("passes primitives, any, and readonly arrays through unchanged", () => {
    expectTypeOf<UnmaskFragments<string>>().toEqualTypeOf<string>();
    expectTypeOf<UnmaskFragments<number | null>>().toEqualTypeOf<number | null>();
    expectTypeOf<UnmaskFragments<any>>().toBeAny();
    expectTypeOf<UnmaskFragments<ReadonlyArray<TagFragment>>>().toEqualTypeOf<
      ReadonlyArray<{ label: string }>
    >();
  });
});

describe("ConnectionKeys and ConnectionNode", () => {
  it("lists only the result keys that carry a cursor connection", () => {
    expectTypeOf<ConnectionKeys<SyntheticResult>>().toEqualTypeOf<"rows">();
    expectTypeOf<
      ConnectionKeys<AccessorialChargeTableQuery>
    >().toEqualTypeOf<"accessorialCharges">();
    expectTypeOf<ConnectionKeys<TractorTableQuery>>().toEqualTypeOf<"tractors">();
  });

  it("exposes the raw, still-masked node under a connection key", () => {
    expectTypeOf<ConnectionNode<SyntheticResult, "rows">>().toEqualTypeOf<MaskedNode>();
  });
});

describe("defineDataTableGraphQLConfig", () => {
  const accessorialCharges = defineDataTableGraphQLConfig({
    document: AccessorialChargeTableDocument,
    operationName: "AccessorialChargeTable",
    connectionKey: "accessorialCharges",
  });

  it("infers the connection key literal and the unmasked row type", () => {
    expectTypeOf(accessorialCharges.connectionKey).toEqualTypeOf<"accessorialCharges">();

    type Row = DataTableConfigRow<typeof accessorialCharges>;
    expectTypeOf<Row>().toEqualTypeOf<
      DataTableRow<typeof AccessorialChargeTableDocument, "accessorialCharges">
    >();
    expectTypeOf<Row["id"]>().toEqualTypeOf<string>();
    expectTypeOf<Row["code"]>().toEqualTypeOf<string>();
    expectTypeOf<Row["status"]>().toEqualTypeOf<EntityStatus>();
    expectTypeOf<Row["method"]>().toEqualTypeOf<AccessorialMethod>();
    expectTypeOf<Row["amount"]>().toEqualTypeOf<number>();
    expectTypeOf<Row>().not.toHaveProperty(" $fragmentRefs");
  });

  it("unmasks fragments nested inside the row fragment", () => {
    const tractors = defineDataTableGraphQLConfig({
      document: TractorTableDocument,
      operationName: "TractorTable",
      connectionKey: "tractors",
      extraVariables: { includeEquipmentDetails: true },
    });

    type Row = DataTableConfigRow<typeof tractors>;
    expectTypeOf<NonNullable<Row["equipmentType"]>>().toEqualTypeOf<{
      id: string;
      code: string;
      color: string;
    }>();
  });

  it("rejects a connection key the document does not return", () => {
    defineDataTableGraphQLConfig({
      document: AccessorialChargeTableDocument,
      operationName: "AccessorialChargeTable",
      // @ts-expect-error "accessorialCharge" is not a connection in this result
      connectionKey: "accessorialCharge",
    });
  });

  it("accepts an explicit row type only when the fragment structurally satisfies it", () => {
    type Narrow = { id: string; code: string };
    type ClaimsExtraField = { id: string; code: string; nickname: string };

    const narrow = defineDataTableGraphQLConfig<
      typeof AccessorialChargeTableDocument,
      "accessorialCharges",
      Narrow
    >({
      document: AccessorialChargeTableDocument,
      operationName: "AccessorialChargeTable",
      connectionKey: "accessorialCharges",
    });
    expectTypeOf<DataTableConfigRow<typeof narrow>>().toEqualTypeOf<Narrow>();

    const withoutMapNode = {
      document: AccessorialChargeTableDocument,
      operationName: "AccessorialChargeTable",
      connectionKey: "accessorialCharges",
    } as const;
    defineDataTableGraphQLConfig<
      typeof AccessorialChargeTableDocument,
      "accessorialCharges",
      ClaimsExtraField
    >(
      // @ts-expect-error the fragment does not fetch `nickname`, so mapNode is required
      withoutMapNode,
    );

    const reshaped = defineDataTableGraphQLConfig({
      document: AccessorialChargeTableDocument,
      operationName: "AccessorialChargeTable",
      connectionKey: "accessorialCharges",
      mapNode: (node) => ({ id: node.id, code: node.code, nickname: node.description }),
    });
    expectTypeOf<DataTableConfigRow<typeof reshaped>>().toEqualTypeOf<ClaimsExtraField>();
  });

  it("types extra variables from the document variables", () => {
    defineDataTableGraphQLConfig({
      document: TractorTableDocument,
      operationName: "TractorTable",
      connectionKey: "tractors",
      // @ts-expect-error `includeTrailerDetails` is not a TractorTable variable
      extraVariables: { includeTrailerDetails: true },
    });

    defineDataTableGraphQLConfig({
      document: TractorTableDocument,
      operationName: "TractorTable",
      connectionKey: "tractors",
      extraVariables: ({ pageSize }) => ({ includeWorkerDetails: pageSize > 25 }),
    });
  });

  it("only allows input extra variables the connection input actually declares", () => {
    const pto = defineDataTableGraphQLConfig({
      document: WorkerPtoTableDocument,
      operationName: "WorkerPtoTable",
      connectionKey: "workerPTOEntries",
      inputExtraVariables: { includeWorker: true },
    });
    expectTypeOf(pto.inputExtraVariables).not.toBeNever();

    defineDataTableGraphQLConfig({
      document: WorkerPtoTableDocument,
      operationName: "WorkerPtoTable",
      connectionKey: "workerPTOEntries",
      // @ts-expect-error the paging keys are owned by the table and cannot be overridden
      inputExtraVariables: { first: 500 },
    });

    defineDataTableGraphQLConfig({
      document: AccessorialChargeTableDocument,
      operationName: "AccessorialChargeTable",
      connectionKey: "accessorialCharges",
      // @ts-expect-error DataTableConnectionInput has no keys beyond the reserved ones
      inputExtraVariables: { includeWorker: true },
    });
  });

  it("is consumable as a DataTableGraphQLSource of its row or any supertype", () => {
    type Row = DataTableConfigRow<typeof accessorialCharges>;

    expectTypeOf(accessorialCharges).toExtend<DataTableGraphQLSource<Row>>();
    expectTypeOf(accessorialCharges).toExtend<DataTableGraphQLSource<{ id: string }>>();
    expectTypeOf(accessorialCharges).not.toExtend<
      DataTableGraphQLSource<{ id: string; nickname: string }>
    >();
    expectTypeOf<
      DataTableGraphQLConfig<typeof AccessorialChargeTableDocument, "accessorialCharges">
    >().toExtend<DataTableGraphQLSource<Row>>();
  });

  it("returns the config it was given", () => {
    expect(accessorialCharges).toEqual({
      document: AccessorialChargeTableDocument,
      operationName: "AccessorialChargeTable",
      connectionKey: "accessorialCharges",
    });
  });
});
