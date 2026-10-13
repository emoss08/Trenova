import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { stripFragmentText } from "./strip-fragment-text.mjs";

const fragment = (name, typeName, fields) =>
  [
    `export const ${name}FragmentDoc = new TypedDocumentString(\``,
    `    fragment ${name} on ${typeName} {`,
    ...fields.map((field) => `  ${field}`),
    "}",
    `    \`, {"fragmentName":"${name}"}) as unknown as TypedDocumentString<${name}Fragment, unknown>;`,
  ].join("\n");

const operation =
  'export const ShipmentDocument = {"__meta__":{"kind":"query","name":"Shipment","hash":"sha256:ab"}} as unknown as TypedDocumentString<ShipmentQuery, ShipmentQueryVariables>;';

describe("stripFragmentText", () => {
  it("replaces each fragment's GraphQL text with its name, keeping the declared type", () => {
    const source = [fragment("CustomerFields", "Customer", ["id", "name"]), operation].join("\n");

    assert.equal(
      stripFragmentText(source),
      [
        'export const CustomerFieldsFragmentDoc = {"__meta__":{"fragmentName":"CustomerFields"}} as unknown as TypedDocumentString<CustomerFieldsFragment, unknown>;',
        operation,
      ].join("\n"),
    );
  });

  it("rewrites every fragment in the file, not only the first", () => {
    const source = [
      fragment("A", "Customer", ["id"]),
      fragment("B", "Shipment", ["id", "proNumber"]),
      fragment("C", "Location", ["id"]),
    ].join("\n");

    const out = stripFragmentText(source);
    assert.equal(out.match(/"fragmentName"/g)?.length, 3);
    assert.ok(!out.includes("fragment A on"));
    assert.ok(!out.includes("proNumber"));
  });

  it("leaves a file without fragments unchanged", () => {
    assert.equal(stripFragmentText(operation), operation);
  });

  it("fails when a document with GraphQL text is left that it does not recognise", () => {
    const unknownShape =
      'export const XFragmentDoc = new TypedDocumentString(`fragment X on Y { id }`, {"fragmentName":"X","extra":true}) as unknown as TypedDocumentString<XFragment, unknown>;';

    assert.throws(() => stripFragmentText(unknownShape), /still carry GraphQL text/);
  });
});
