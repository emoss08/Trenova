// Sends every operation the accounting setup modal uses (handlers/accounting.mjs) to the mock
// API and holds each response to its selection set in
// client/packages/graphql/src/operations/accounting-sync/operations.graphql: every selected field
// present, nothing unselected, no null on a non-null type, enums and scalars of the right kind.
// Walks setup end to end for each provider, the refusals a write should make, and the catalog.
//
//   node e2e/mock-api/check-accounting.mjs        (mock on :8080, or MOCK_API_URL)
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import {
  buildSchema,
  isEnumType,
  isListType,
  isNonNullType,
  isScalarType,
  Kind,
  parse,
} from "graphql";
import { ACCOUNTING_HANDLERS } from "./handlers/accounting.mjs";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const GRAPHQL_SRC = fileURLToPath(new URL("../../../../packages/graphql/src/", import.meta.url));
const schema = buildSchema(readFileSync(`${GRAPHQL_SRC}schema.graphql`, "utf8"));

const CONTRACT = ["operations/accounting-sync/operations.graphql"];
const documents = CONTRACT.map((file) => ({
  file,
  doc: parse(readFileSync(`${GRAPHQL_SRC}${file}`, "utf8")),
}));

const fragments = new Map();
const operations = new Map();
for (const { file, doc } of documents) {
  for (const def of doc.definitions) {
    if (def.kind === Kind.FRAGMENT_DEFINITION) fragments.set(def.name.value, def);
    if (def.kind === Kind.OPERATION_DEFINITION)
      operations.set(def.name.value, { def, contract: CONTRACT.includes(file) });
  }
}
// The file also holds the sync ledger's operations, which the ledger page sends; the contract
// here is the setup modal's, which handlers/accounting.mjs answers.
const contractOperations = Object.keys(ACCOUNTING_HANDLERS);

const failures = [];
const exercised = new Set();
const fail = (message) => {
  failures.push(message);
  console.log(`  ✗ ${message}`);
};

function directiveAllows(node, variables, defaults) {
  for (const directive of node.directives ?? []) {
    const arg = directive.arguments.find((a) => a.name.value === "if");
    if (!arg) continue;
    const value =
      arg.value.kind === Kind.VARIABLE
        ? (variables[arg.value.name.value] ?? defaults[arg.value.name.value])
        : arg.value.value;
    if (directive.name.value === "include" && !value) return false;
    if (directive.name.value === "skip" && value) return false;
  }
  return true;
}

function collect(selectionSet, typeName, variables, defaults, into = new Map()) {
  for (const selection of selectionSet.selections) {
    if (!directiveAllows(selection, variables, defaults)) continue;
    if (selection.kind === Kind.FIELD) {
      const key = selection.alias?.value ?? selection.name.value;
      if (!into.has(key)) into.set(key, []);
      into.get(key).push(selection);
    } else if (selection.kind === Kind.FRAGMENT_SPREAD) {
      const fragment = fragments.get(selection.name.value);
      if (!fragment) throw new Error(`unknown fragment ${selection.name.value}`);
      if (fragment.typeCondition.name.value === typeName)
        collect(fragment.selectionSet, typeName, variables, defaults, into);
    } else if (selection.kind === Kind.INLINE_FRAGMENT) {
      if (!selection.typeCondition || selection.typeCondition.name.value === typeName)
        collect(selection.selectionSet, typeName, variables, defaults, into);
    }
  }
  return into;
}

function checkScalar(value, type, path) {
  const name = type.name;
  const ok = {
    Int: Number.isInteger(value),
    Timestamp: Number.isInteger(value),
    Float: typeof value === "number" && Number.isFinite(value),
    String: typeof value === "string",
    ID: typeof value === "string" && value.length > 0,
    Decimal: typeof value === "string" && /^-?\d+(\.\d+)?$/.test(value),
    Boolean: typeof value === "boolean",
  }[name];
  if (ok === false) fail(`${path}: ${JSON.stringify(value)} is not a ${name}`);
}

function check(value, type, nodes, path, ctx) {
  if (isNonNullType(type)) {
    if (value === null || value === undefined) {
      fail(`${path}: null on non-null ${type}`);
      return;
    }
    check(value, type.ofType, nodes, path, ctx);
    return;
  }
  if (value === null) return;
  if (value === undefined) {
    fail(`${path}: undefined (a nullable field must still be present, as null)`);
    return;
  }
  if (isListType(type)) {
    if (!Array.isArray(value)) {
      fail(`${path}: expected a list`);
      return;
    }
    value.forEach((item, i) => check(item, type.ofType, nodes, `${path}[${i}]`, ctx));
    return;
  }
  if (isEnumType(type)) {
    if (!type.getValue(value)) fail(`${path}: ${JSON.stringify(value)} is not a ${type.name}`);
    return;
  }
  if (isScalarType(type)) {
    checkScalar(value, type, path);
    return;
  }
  if (typeof value !== "object" || Array.isArray(value)) {
    fail(`${path}: expected an object of ${type.name}`);
    return;
  }
  const selected = new Map();
  for (const node of nodes)
    if (node.selectionSet)
      collect(node.selectionSet, type.name, ctx.variables, ctx.defaults, selected);
  const fields = type.getFields();
  for (const [key, fieldNodes] of selected) {
    const fieldName = fieldNodes[0].name.value;
    if (fieldName === "__typename") continue;
    if (!(key in value)) {
      fail(`${path}.${key}: missing`);
      continue;
    }
    check(value[key], fields[fieldName].type, fieldNodes, `${path}.${key}`, ctx);
  }
  for (const key of Object.keys(value)) {
    if (key !== "__typename" && !selected.has(key))
      fail(`${path}.${key}: not selected by the operation`);
  }
}

async function post(path, body, method = "POST") {
  const init = { method, headers: { "content-type": "application/json" } };
  if (body) init.body = JSON.stringify(body);
  const response = await fetch(`${MOCK}${path}`, init);
  return response.json();
}

async function scenario(overrides) {
  return post("/__mock/scenario", {
    accounting: "connect",
    accountingSystem: "QuickBooksOnline",
    accountingApp: "instance",
    accountingAuth: "signIn",
    ...overrides,
  });
}

/** Sends one operation and checks its response; `expectError` checks the presented domain error instead. */
async function op(name, variables = {}, { expectError = null } = {}) {
  const entry = operations.get(name);
  if (!entry) throw new Error(`no operation ${name}`);
  exercised.add(name);
  const result = await post("/graphql", { operationName: name, variables });
  if (expectError) {
    const error = result.errors?.[0];
    if (!error) fail(`${name}: expected a ${expectError} error, got data`);
    else if (error.extensions?.code !== expectError)
      fail(`${name}: expected ${expectError}, got ${error.extensions?.code} (${error.message})`);
    else console.log(`  ✓ ${name} refused: ${error.message}`);
    return null;
  }
  if (result.errors?.length) {
    fail(`${name}: ${result.errors[0].message.split("\n")[0]}`);
    return null;
  }
  const defaults = Object.fromEntries(
    (entry.def.variableDefinitions ?? [])
      .filter((v) => v.defaultValue)
      .map((v) => [
        v.variable.name.value,
        v.defaultValue.value === "true"
          ? true
          : v.defaultValue.value === "false"
            ? false
            : v.defaultValue.value,
      ]),
  );
  const before = failures.length;
  const rootType =
    entry.def.operation === "mutation" ? schema.getMutationType() : schema.getQueryType();
  check(result.data, rootType, [entry.def], name, { variables, defaults });
  if (failures.length === before) console.log(`  ✓ ${name}`);
  return result.data;
}

function expect(condition, message) {
  if (condition) console.log(`    · ${message}`);
  else fail(message);
}

const status = async (system) =>
  (await op("AccountingSyncStatus", { integrationType: system })).accountingSyncStatus;

console.log("catalog");
const catalog = await (await fetch(`${MOCK}/api/v1/integrations/catalog/`)).json();
const accountingItems = catalog.items.filter((item) => item.category === "Accounting");
expect(
  catalog.items.length === 20,
  `the catalog lists the server's 20 integrations (got ${catalog.items.length})`,
);
expect(
  accountingItems.map((item) => item.type).join() ===
    "QuickBooksOnline,Xero,BusinessCentral,NetSuite",
  "the accounting systems in the server's order",
);
expect(
  catalog.items.every((item, i, all) => i === 0 || all[i - 1].sortOrder <= item.sortOrder),
  "sorted by sortOrder",
);
expect(
  catalog.items.every(
    (item) => item.status.connection === "disconnected" && item.configSpec.length === 0,
  ),
  "nothing installed on a new organization",
);

console.log("QuickBooks Online: sign in and set up");
await scenario({});
const fresh = await status("QuickBooksOnline");
expect(
  fresh.connection === null && fresh.available,
  "nothing connected, and the server's app is available",
);
expect(
  fresh.app.redirectUrl.endsWith("/admin/integrations/quickbooks/callback"),
  "the redirect is the app's callback",
);
const start = (await op("StartAccountingAuthorization", { integrationType: "QuickBooksOnline" }))
  .startAccountingAuthorization;
const callback = new URL(start.authorizeUrl);
expect(
  callback.searchParams.get("realmId") === "9341453052141850",
  "the callback carries the company",
);
const finished = (
  await op("FinishAccountingAuthorization", {
    input: {
      integrationType: "QuickBooksOnline",
      code: callback.searchParams.get("code"),
      state: callback.searchParams.get("state"),
      realmId: callback.searchParams.get("realmId"),
    },
  })
).finishAccountingAuthorization;
expect(
  finished.connection?.setupStep === "Mode",
  "a new connection starts at choosing what is sent",
);
await op("StartAccountingAuthorization", { integrationType: "Xero" });
await op(
  "FinishAccountingAuthorization",
  { input: { integrationType: "Xero", code: "c", state: "s", realmId: "x" } },
  { expectError: "ALREADY_CONNECTED" },
);
await op("ChooseAccountingSyncMode", {
  input: { integrationType: "QuickBooksOnline", mode: "Ledger", granularity: "DailySummary" },
});
const mappingStep = (await status("QuickBooksOnline")).connection;
expect(
  mappingStep.setupStep === "Mappings" && mappingStep.ledgerGranularity === "DailySummary",
  "journal entries, summed by day, then matching",
);
const summary0 = (await op("AccountingMappingSummary", { integrationType: "QuickBooksOnline" }))
  .accountingMappingSummary;
expect(
  summary0.requiredTotal === 4 && summary0.requiredConfirmed === 0,
  "4 required mappings, none confirmed",
);
await op(
  "CompleteAccountingSetup",
  { integrationType: "QuickBooksOnline" },
  { expectError: "MAPPINGS_INCOMPLETE" },
);
const required = (
  await op("AccountingMappings", {
    integrationType: "QuickBooksOnline",
    first: 50,
    filter: { requiredOnly: true },
  })
).accountingMappings.edges.map((edge) => edge.node);
const proposed = (
  await op("AccountingMappings", {
    integrationType: "QuickBooksOnline",
    first: 2,
    filter: { states: ["Proposed"] },
  })
).accountingMappings;
expect(proposed.pageInfo.hasNextPage && proposed.edges.length === 2, "proposals page");
await op("AccountingMappings", {
  integrationType: "QuickBooksOnline",
  first: 2,
  after: proposed.pageInfo.endCursor,
  filter: { states: ["Proposed"] },
});
const accounts = (
  await op("AccountingReferenceObjects", {
    integrationType: "QuickBooksOnline",
    kind: "Account",
    query: "rev",
    limit: 10,
  })
).accountingReferenceObjects;
expect(accounts.length === 2, `searching accounts for "rev" finds 2 (got ${accounts.length})`);
const [first, second, ...rest] = required;
await op("ConfirmAccountingMappings", {
  input: rest
    .filter((row) => row.externalId)
    .map((row) => ({ id: row.id, externalId: row.externalId })),
});
await op("SetAccountingMapping", {
  input: { mappingId: first.id, externalId: "acct-1100", reason: "A/R" },
});
await op("RejectAccountingMapping", { id: second.id });
await op("CreateAccountingReferenceRecord", {
  input: { mappingId: second.id, name: "Freight Income" },
});
const unmatched = required.find((row) => !row.externalId);
if (unmatched)
  await op("SetAccountingMapping", { input: { mappingId: unmatched.id, externalId: "acct-4000" } });
await op("ClearAccountingMapping", {
  id: (
    await op("AccountingMappings", {
      integrationType: "QuickBooksOnline",
      first: 50,
      filter: { states: ["Proposed"] },
    })
  ).accountingMappings.edges[0].node.id,
});
await op("RefreshAccountingReferenceData", { integrationType: "QuickBooksOnline" });
const summary1 = (await op("AccountingMappingSummary", { integrationType: "QuickBooksOnline" }))
  .accountingMappingSummary;
expect(
  summary1.canCompleteSetup,
  `every required mapping confirmed (${summary1.requiredConfirmed}/${summary1.requiredTotal})`,
);
await op("CompleteAccountingSetup", { integrationType: "QuickBooksOnline" });
const enabled = (
  await op("EnableAccountingSync", {
    input: {
      integrationType: "QuickBooksOnline",
      startDate: Math.floor(Date.now() / 1000) - 86400 * 30,
      autoSync: false,
      backfill: true,
      openingBalances: true,
    },
  })
).enableAccountingSync;
expect(
  enabled.setupStep === "Complete" && enabled.ledgerOpeningBalancesSentAt !== null,
  "sending started, opening balances sent",
);
const settings = (
  await op("UpdateAccountingSyncSettings", {
    input: {
      integrationType: "QuickBooksOnline",
      autoSync: true,
      driverSettlements: true,
      inboundPayments: "Apply",
    },
  })
).updateAccountingSyncSettings;
expect(
  settings.syncsDriverSettlements && settings.inboundPaymentPolicy === "Apply",
  "settings saved",
);
await op("CheckAccountingConnection", { integrationType: "QuickBooksOnline" });
const tenant = (
  await op("SaveAccountingApp", {
    input: {
      integrationType: "QuickBooksOnline",
      environment: "Sandbox",
      clientId: "ABclient",
      clientSecret: "secret",
    },
  })
).saveAccountingApp;
expect(
  tenant.app.activeSource === "Tenant" &&
    tenant.app.webhookPath.endsWith("/acctapp_quickbooksonline/"),
  "the organization's own app, with its webhook path",
);
await op(
  "SaveAccountingApp",
  { input: { integrationType: "Xero", environment: "Production", clientId: "XERO" } },
  { expectError: "INVALID" },
);
await op("RemoveAccountingApp", { integrationType: "QuickBooksOnline" });
const gone = (await op("DisconnectAccountingSystem", { integrationType: "QuickBooksOnline" }))
  .disconnectAccountingSystem;
expect(gone.status === "Disconnected", "disconnected");
await op(
  "CheckAccountingConnection",
  { integrationType: "QuickBooksOnline" },
  { expectError: "NOT_CONNECTED" },
);

console.log("Xero: choose the organisation");
const xeroStart = new URL(
  (await op("StartAccountingAuthorization", { integrationType: "Xero" }))
    .startAccountingAuthorization.authorizeUrl,
);
expect(!xeroStart.searchParams.has("realmId"), "Xero's callback carries no company");
const choice = (
  await op("FinishAccountingAuthorization", {
    input: { integrationType: "Xero", code: "c", state: "s" },
  })
).finishAccountingAuthorization;
expect(
  choice.connection === null && choice.companies.length === 2,
  "two organisations to choose from",
);
await op(
  "ChooseAccountingCompany",
  { input: { integrationType: "Xero", choiceToken: "stale", companyId: choice.companies[0].id } },
  { expectError: "CHOICE_EXPIRED" },
);
const xero = (
  await op("ChooseAccountingCompany", {
    input: {
      integrationType: "Xero",
      choiceToken: choice.choiceToken,
      companyId: choice.companies[1].id,
    },
  })
).chooseAccountingCompany;
expect(
  xero.externalCompanyName === "Peak Freight (Demo Company)",
  "the chosen organisation is connected",
);
await op(
  "ChooseAccountingSyncMode",
  { input: { integrationType: "Xero", mode: "Ledger", granularity: "Detailed" } },
  { expectError: "INVALID" },
);
expect(
  (await op("AccountingMappingSummary", { integrationType: "Xero" })).accountingMappingSummary
    .requiredTotal === 2,
  "Xero requires the deposit account and the freight line",
);

console.log("NetSuite: connect with a certificate, then renew it");
await scenario({});
await op(
  "ConnectAccountingWithCertificate",
  { input: { integrationType: "NetSuite", accountId: "1234567", certificateId: "cert" } },
  { expectError: "NO_CERTIFICATE" },
);
await op("PrepareAccountingCertificate", { input: { integrationType: "NetSuite" } });
expect(
  (await status("NetSuite")).pendingCertificate !== null,
  "the certificate waits to be uploaded",
);
const certified = (
  await op("ConnectAccountingWithCertificate", {
    input: { integrationType: "NetSuite", accountId: "1234567", certificateId: "certA" },
  })
).connectAccountingWithCertificate.connection;
expect(
  certified.authMethod === "ClientCredentials" && certified.certificateId === "certA",
  "connected with the certificate",
);
await op("PrepareAccountingCertificate", { input: { integrationType: "NetSuite" } });
const renewed = (
  await op("RenewAccountingCertificate", {
    input: { connectionId: certified.id, certificateId: "certB" },
  })
).renewAccountingCertificate;
expect(renewed.certificateId === "certB", "renewed");

console.log("scenarios");
for (const [overrides, check, message] of [
  [
    { accounting: "mappings" },
    (s) => s.connection.setupStep === "Mappings",
    "mappings: matching, one confirmed",
  ],
  [
    { accounting: "connected", accountingSystem: "BusinessCentral" },
    (s) => s.webhookSubscriptions?.active === 3,
    "Business Central keeps its subscriptions",
  ],
  [
    { accounting: "failing" },
    (s) => s.connection.status === "Failing" && s.connection.consecutiveFailures === 4,
    "failing: four calls in a row",
  ],
  [{ accounting: "revoked" }, (s) => s.connection.status === "Revoked", "revoked"],
  [
    { accounting: "expiring", accountingSystem: "NetSuite" },
    (s) => s.connection.refreshTokenAbsoluteExpiresAt - Date.now() / 1000 < 2 * 86400,
    "a NetSuite sign-in inside its two days",
  ],
  [
    { accounting: "connected", accountingSystem: "NetSuite", accountingAuth: "certificate" },
    (s) => s.connection.authMethod === "ClientCredentials",
    "NetSuite on a certificate",
  ],
  [
    { accountingApp: "none" },
    (s) => !s.available && s.app.activeSource === null,
    "no app: connecting needs the organization's keys",
  ],
  [
    { accountingApp: "tenant" },
    (s) => s.app.tenantApp?.clientId === "ABmockTenantClient7f3e",
    "the organization's own app",
  ],
]) {
  await scenario(overrides);
  expect(check(await status(overrides.accountingSystem ?? "QuickBooksOnline")), message);
}
await scenario({ accounting: "connected" });
const connectedCatalog = await (await fetch(`${MOCK}/api/v1/integrations/catalog/`)).json();
expect(
  connectedCatalog.items.find((item) => item.type === "QuickBooksOnline").status.connection ===
    "connected",
  "the catalog shows the connected system connected",
);

await scenario({});

const missing = contractOperations.filter((name) => !exercised.has(name));
if (missing.length) fail(`operations never sent: ${missing.join(", ")}`);
console.log(
  `\n${contractOperations.length} contract operations, ${exercised.size} exercised, ${failures.length} failure(s)`,
);
if (failures.length) process.exitCode = 1;
