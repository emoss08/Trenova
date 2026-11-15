import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  BASE,
  camelToSnake,
  loadRepository,
  mixedOperations,
  readOperations,
  readSchema,
} from "./plan-mixed-operations.mjs";

// Operations that select an add-on's root beside another plan's root on purpose, each with
// why a plan without the add-on never runs it. Anything else is split or put behind @include.
const ALLOWED = new Map([
  [
    "CarrierIntelligence",
    "Runs only where the person reads carrier intelligence (the carrier's intelligence tab, " +
      "a tender's override), a Carrier & Brokerage resource: without the add-on the tab shows " +
      "the add-on explanation and the override is never offered, so the query is never sent.",
  ],
]);

// A small catalog in the shape readGoCatalog returns: customers are base, factoring
// companies are Accounting & Cash's, and the leave resolver checks a Safety resource.
const catalog = {
  resources: new Set(["customer", "factoring_company", "worker_leave"]),
  featureOfResource: new Map([
    ["customer", "tms.billing"],
    ["factoring_company", "tms.factoring"],
    ["worker_leave", "workforce.time_off"],
  ]),
  addOnOfFeature: new Map([
    ["tms.billing", BASE],
    ["tms.factoring", "ACC"],
    ["workforce.time_off", "HRS"],
  ]),
  resolverChecks: new Map([["query.leavebalance", new Set(["worker_leave"])]]),
};

const schema = readSchema(`
  directive @kind(resource: String!) on OBJECT
  type Customer { id: ID! }
  type FactoringCompany { id: ID! }
  type FactoringCompanyEdge { node: FactoringCompany! }
  type FactoringCompanyConnection { edges: [FactoringCompanyEdge!]! }
  type FactoringCard @kind(resource: "factoring_company") { id: ID! }
  type LeaveBalance { days: Int! }
  type Query {
    customer(id: ID!): Customer
    factoringCompanies: FactoringCompanyConnection!
    factoringCard: FactoringCard
    leaveBalance: LeaveBalance!
  }
  type Mutation { updateCustomer: Customer! }
`);

function mixedNames(source) {
  return mixedOperations(
    readOperations([{ path: "fixture.graphql", source }]),
    schema,
    catalog,
  ).map((operation) => operation.name);
}

describe("mixedOperations", () => {
  it("finds a base root beside an add-on root reached through a connection", () => {
    assert.deepEqual(
      mixedNames("query A { customer(id: 1) { id } factoringCompanies { edges { node { id } } } }"),
      ["A"],
    );
  });

  it("reads a root's kind from @kind and its resolver's checks", () => {
    assert.deepEqual(mixedNames("query B { customer(id: 1) { id } factoringCard { id } }"), ["B"]);
    assert.deepEqual(mixedNames("query C { customer(id: 1) { id } leaveBalance { days } }"), ["C"]);
  });

  it("finds two different add-ons in one operation", () => {
    assert.deepEqual(mixedNames("query D { factoringCard { id } leaveBalance { days } }"), ["D"]);
  });

  it("accepts an add-on root behind @include or @skip, also through a fragment", () => {
    assert.deepEqual(
      mixedNames(
        "query E($f: Boolean!) { customer(id: 1) { id } factoringCard @include(if: $f) { id } }",
      ),
      [],
    );
    assert.deepEqual(
      mixedNames(
        "query F($f: Boolean!) { customer(id: 1) { id } ... on Query @skip(if: $f) { leaveBalance { days } } }",
      ),
      [],
    );
  });

  it("accepts operations within one plan", () => {
    assert.deepEqual(mixedNames("query G { customer(id: 1) { id } __typename }"), []);
    assert.deepEqual(
      mixedNames("query H { factoringCard { id } factoringCompanies { edges { node { id } } } }"),
      [],
    );
  });

  it("names a type's resource the way the server does", () => {
    assert.equal(camelToSnake("FactoringCompany"), "factoring_company");
    assert.equal(camelToSnake("APIKey"), "api_key");
    assert.equal(camelToSnake("EDI"), "edi");
  });
});

describe("the client's operations", () => {
  const { operations, schema: repoSchema, catalog: repoCatalog } = loadRepository();
  const mixed = mixedOperations(operations, repoSchema, repoCatalog);

  it("reads the catalog it checks against", () => {
    assert.ok(repoCatalog.featureOfResource.size > 150, "resource features were not read");
    assert.ok(repoCatalog.addOnOfFeature.size > 30, "feature add-ons were not read");
    assert.ok(repoCatalog.resolverChecks.size > 500, "resolver checks were not read");
    assert.ok(operations.length > 500, "operations were not read");
  });

  it("never selects an add-on's root beside another plan's root unconditionally", () => {
    const unexplained = mixed
      .filter((operation) => !ALLOWED.has(operation.name))
      .map(
        (operation) =>
          `${operation.path} ${operation.name}: ` +
          operation.roots
            .map(
              (root) =>
                `${root.field}${root.conditional ? " (conditional)" : ""} [${root.addOns.join(", ")}]`,
            )
            .join("; "),
      );
    assert.deepEqual(
      unexplained,
      [],
      "Split the add-on root into its own operation loaded only when usePlanFeature(feature).available, " +
        "or put it behind @include(if:) driven by it; or add the operation to ALLOWED with why it never runs without the add-on.",
    );
  });

  it("allows only operations that are still mixed", () => {
    const stillMixed = new Set(mixed.map((operation) => operation.name));
    assert.deepEqual(
      [...ALLOWED.keys()].filter((name) => !stillMixed.has(name)),
      [],
      "Remove these from ALLOWED: they no longer mix plans.",
    );
  });
});
