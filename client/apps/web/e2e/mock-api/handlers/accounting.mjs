// Accounting sync and the integrations catalog: the GraphQL operations the accounting modal,
// its sign-in callback pages and the certificate flow send, and the REST routes the
// integrations page reads. Writes are stateful within a scenario, so setup can be clicked
// through from connecting to sending. See README.md, "Accounting integrations".
import {
  ACCOUNTING_STAGES,
  ACCOUNTING_SYSTEMS,
  buildConnection,
  buildMappings,
  COMPANIES,
  COMPANY_CHOICES,
  MOCK_CERTIFICATE_PEM,
  PROFILES,
  REFERENCE_OBJECTS,
  referenceObject,
  webhookPath,
} from "../fixtures/accounting.mjs";
import { CATALOG_DEFINITIONS } from "../fixtures/integration-catalog.mjs";
import { USER } from "../fixtures/session.mjs";
import { TriageError } from "./triage-model.mjs";

const DAY = 86400;
const WEB_ORIGIN = process.env.MOCK_WEB_ORIGIN ?? "http://localhost:5173";
const ME = { id: USER.id, name: USER.name };

/** The accounting state for a scenario: one connected system at the chosen stage. */
export function createAccountingState(scenario, anchor) {
  const system = ACCOUNTING_SYSTEMS.includes(scenario.accountingSystem)
    ? scenario.accountingSystem
    : "QuickBooksOnline";
  const stage = ACCOUNTING_STAGES.has(scenario.accounting) ? scenario.accounting : "connect";
  const appMode = ["instance", "tenant", "none"].includes(scenario.accountingApp)
    ? scenario.accountingApp
    : "instance";
  const certificate = system === "NetSuite" && scenario.accountingAuth === "certificate";

  const apps = Object.fromEntries(ACCOUNTING_SYSTEMS.map((s) => [s, null]));
  if (appMode === "tenant") {
    apps[system] = tenantApp(system, {
      environment: PROFILES[system].profile.environments[0],
      clientId: "ABmockTenantClient7f3e",
      hasWebhookVerifier: PROFILES[system].profile.webhookKeyLabel !== "",
      updatedAt: anchor - 45 * DAY,
    });
  }

  const connections = Object.fromEntries(ACCOUNTING_SYSTEMS.map((s) => [s, null]));
  const mappings = Object.fromEntries(ACCOUNTING_SYSTEMS.map((s) => [s, buildMappings(s, anchor)]));
  if (stage !== "connect") {
    connections[system] = buildConnection(system, stage, anchor, {
      appSource: appMode === "tenant" ? "Tenant" : "Instance",
      certificate,
    });
    const settled = !["mode", "mappings"].includes(stage);
    for (const row of mappings[system]) {
      // Past matching, every proposal was confirmed; while matching, the most certain one was.
      if (row.state !== "Proposed") continue;
      if (settled || row.trenovaKey === mappings[system][0].trenovaKey)
        confirm(row, row.externalId, anchor);
    }
  }

  return {
    system,
    appMode,
    apps,
    connections,
    mappings,
    pendingCertificates: Object.fromEntries(ACCOUNTING_SYSTEMS.map((s) => [s, null])),
    choices: new Map(),
  };
}

function tenantApp(system, { environment, clientId, hasWebhookVerifier, updatedAt, version = 1 }) {
  return {
    id: `acctapp_${system.toLowerCase()}`,
    integrationType: system,
    environment,
    clientId,
    hasWebhookVerifier,
    version,
    updatedAt,
  };
}

function confirm(row, externalId, at, source = row.source ?? "Manual") {
  const ref = Object.values(REFERENCE_OBJECTS)
    .flat()
    .find((candidate) => candidate.externalId === externalId);
  row.externalId = externalId;
  row.externalName = ref?.label ?? row.externalName;
  row.state = "Confirmed";
  row.source = source;
  row.confirmedBy = ME;
  row.confirmedAt = at;
  row.version += 1;
  row.updatedAt = at;
}

const live = (connection) => connection !== null && connection.status !== "Disconnected";

function accounting(state) {
  return state.accounting;
}

function systemOf(v) {
  const system = v.integrationType ?? v.input?.integrationType;
  if (!ACCOUNTING_SYSTEMS.includes(system)) {
    throw new TriageError(`Unknown accounting system ${system}`, { field: "integrationType" });
  }
  return system;
}

function connectionOf(state, system) {
  const connection = accounting(state).connections[system];
  if (!live(connection)) {
    throw new TriageError(`${PROFILES[system].name} is not connected`, {
      code: "NOT_CONNECTED",
      type: "business-rule-violation",
    });
  }
  return connection;
}

function touch(state, connection) {
  connection.version += 1;
  connection.updatedAt = state.now();
  return connection;
}

/** accountingSyncStatus: what the modal reads first. */
function syncStatus(state, system) {
  const acct = accounting(state);
  const meta = PROFILES[system];
  const app = acct.apps[system];
  const instanceAppAvailable = acct.appMode !== "none";
  const connection = acct.connections[system];
  const subscriptions =
    meta.profile.webhookSubscriptions && live(connection)
      ? {
          configured: true,
          active: connection.status === "Connected" ? 3 : 2,
          pending: 0,
          failed: connection.status === "Connected" ? 0 : 1,
          nextExpiryAt: state.now() + 2 * DAY + 5 * 3600,
          lastError:
            connection.status === "Connected"
              ? null
              : "Business Central refused the subscription for salesInvoices.",
        }
      : null;
  return {
    integrationType: system,
    providerName: meta.name,
    profile: meta.profile,
    available: instanceAppAvailable || app !== null,
    app: {
      activeSource: app ? "Tenant" : instanceAppAvailable ? "Instance" : null,
      instanceAppAvailable,
      instanceEnvironment: instanceAppAvailable ? "Production" : null,
      redirectUrl: `${WEB_ORIGIN}${meta.profile.callbackPath}`,
      webhookPath: webhookPath(system, app?.id ?? ""),
      tenantApp: app,
    },
    connection,
    webhookSubscriptions: subscriptions,
    pendingCertificate: acct.pendingCertificates[system],
  };
}

function mappingSummary(state, system) {
  const rows = accounting(state).mappings[system];
  const groups = new Map();
  for (const row of rows) {
    const group = groups.get(row.targetType) ?? {
      targetType: row.targetType,
      unmatched: 0,
      proposed: 0,
      confirmed: 0,
    };
    group[row.state.toLowerCase()] += 1;
    groups.set(row.targetType, group);
  }
  const required = rows.filter((row) => row.required);
  const requiredConfirmed = required.filter((row) => row.state === "Confirmed").length;
  return {
    integrationType: system,
    providerName: PROFILES[system].name,
    connection: accounting(state).connections[system],
    groups: [...groups.values()],
    requiredTotal: required.length,
    requiredConfirmed,
    canCompleteSetup: requiredConfirmed === required.length,
  };
}

function findMapping(state, id) {
  for (const rows of Object.values(accounting(state).mappings)) {
    const row = rows.find((candidate) => candidate.id === id);
    if (row) return row;
  }
  throw new TriageError("Mapping not found", { code: "NOT_FOUND", type: "not-found" });
}

function page(rows, first = 50, after = null) {
  const offset = after ? Number(Buffer.from(after, "base64url").toString().split(":")[1]) : 0;
  const slice = rows.slice(offset, offset + first);
  return {
    edges: slice.map((node) => ({ node })),
    pageInfo: {
      hasNextPage: offset + first < rows.length,
      endCursor: slice.length
        ? Buffer.from(`offset:${offset + slice.length}`).toString("base64url")
        : null,
    },
  };
}

/** A freshly authorized connection, which starts at choosing what is sent. */
function connect(state, system, { certificate = false, companyId = null } = {}) {
  const acct = accounting(state);
  const other = ACCOUNTING_SYSTEMS.find((s) => s !== system && live(acct.connections[s]));
  if (other) {
    throw new TriageError(
      `This organization is connected to ${PROFILES[other].name}. Disconnect it first.`,
      { code: "ALREADY_CONNECTED", type: "business-rule-violation" },
    );
  }
  const now = state.now();
  const connection = buildConnection(system, "mode", now + 41 * DAY, {
    appSource: acct.apps[system] ? "Tenant" : "Instance",
    certificate,
  });
  Object.assign(connection, {
    connectedAt: now,
    lastCheckedAt: now,
    lastSuccessAt: now,
    lastFailureAt: null,
    lastWebhookAt: null,
    referenceRefreshStartedAt: now,
    referenceRefreshedAt: now + 2,
    updatedAt: now,
    version: 1,
  });
  if (companyId) {
    const choice = (COMPANY_CHOICES[system] ?? []).find((company) => company.id === companyId);
    if (choice) connection.externalCompanyName = choice.name;
  }
  acct.connections[system] = connection;
  acct.mappings[system] = buildMappings(system, now);
  return connection;
}

function authorizationResult(connection, companies = [], token = null, expiresAt = null) {
  return { connection, companies, choiceToken: token, choiceExpiresAt: expiresAt };
}

export const ACCOUNTING_HANDLERS = {
  AccountingSyncStatus: (state, v) => ({ accountingSyncStatus: syncStatus(state, systemOf(v)) }),

  SaveAccountingApp: (state, { input }) => {
    const system = systemOf({ input });
    const acct = accounting(state);
    const existing = acct.apps[system];
    if (!input.clientId?.trim()) {
      throw new TriageError("Enter the client ID", { field: "clientId" });
    }
    if (!existing && !input.clientSecret) {
      throw new TriageError("Enter the client secret that goes with this client ID", {
        field: "clientSecret",
      });
    }
    acct.apps[system] = tenantApp(system, {
      environment: input.environment,
      clientId: input.clientId.trim(),
      hasWebhookVerifier: input.clearWebhookVerifierToken
        ? false
        : Boolean(input.webhookVerifierToken) || (existing?.hasWebhookVerifier ?? false),
      updatedAt: state.now(),
      version: (existing?.version ?? 0) + 1,
    });
    return { saveAccountingApp: syncStatus(state, system) };
  },

  RemoveAccountingApp: (state, v) => {
    const system = systemOf(v);
    accounting(state).apps[system] = null;
    return { removeAccountingApp: syncStatus(state, system) };
  },

  AccountingMappingSummary: (state, v) => ({
    accountingMappingSummary: mappingSummary(state, systemOf(v)),
  }),

  AccountingMappings: (state, v) => {
    const filter = v.filter ?? {};
    const search = filter.search?.trim().toLowerCase() ?? "";
    const rows = accounting(state).mappings[systemOf(v)].filter(
      (row) =>
        (!filter.requiredOnly || row.required) &&
        (!filter.states?.length || filter.states.includes(row.state)) &&
        (!filter.targetTypes?.length || filter.targetTypes.includes(row.targetType)) &&
        (!search ||
          row.targetLabel.toLowerCase().includes(search) ||
          row.externalName.toLowerCase().includes(search)),
    );
    return { accountingMappings: page(rows, v.first ?? 50, v.after ?? null) };
  },

  AccountingReferenceObjects: (state, v) => {
    const system = systemOf(v);
    const query = v.query?.trim().toLowerCase() ?? "";
    const rows = (REFERENCE_OBJECTS[v.kind] ?? [])
      .filter(
        (ref) =>
          !query || ref.label.toLowerCase().includes(query) || (ref.number ?? "").startsWith(query),
      )
      .slice(0, v.limit ?? 25)
      .map((ref) => referenceObject(system, ref));
    return { accountingReferenceObjects: rows };
  },

  ConfirmAccountingMappings: (state, { input }) => ({
    confirmAccountingMappings: input.map(({ id, externalId }) => {
      const row = findMapping(state, id);
      confirm(row, externalId, state.now());
      return row;
    }),
  }),

  RejectAccountingMapping: (state, { id }) => {
    const row = findMapping(state, id);
    Object.assign(row, {
      state: "Unmatched",
      externalId: "",
      externalName: "",
      source: null,
      confidence: null,
      reason: "",
      prechecked: false,
      version: row.version + 1,
      updatedAt: state.now(),
    });
    return { rejectAccountingMapping: row };
  },

  SetAccountingMapping: (state, { input }) => {
    const row = findMapping(state, input.mappingId);
    confirm(row, input.externalId, state.now(), "Manual");
    if (input.reason) row.reason = input.reason;
    return { setAccountingMapping: row };
  },

  ClearAccountingMapping: (state, { id }) => {
    const row = findMapping(state, id);
    Object.assign(row, {
      state: "Unmatched",
      externalId: "",
      externalName: "",
      source: null,
      confirmedBy: null,
      confirmedAt: null,
      version: row.version + 1,
      updatedAt: state.now(),
    });
    return { clearAccountingMapping: row };
  },

  CreateAccountingReferenceRecord: (state, { input }) => {
    const row = findMapping(state, input.mappingId);
    const name = input.name?.trim() || row.targetLabel;
    const ref = {
      kind: row.providerKind,
      externalId: `new-${row.id}`,
      label: name,
      companyName: name,
    };
    REFERENCE_OBJECTS[row.providerKind].push(ref);
    confirm(row, ref.externalId, state.now(), "CreatedInProvider");
    return { createAccountingReferenceRecord: row };
  },

  RefreshAccountingReferenceData: (state, v) => {
    const connection = connectionOf(state, systemOf(v));
    connection.referenceRefreshStartedAt = state.now();
    connection.referenceRefreshedAt = state.now() + 1;
    connection.referenceRefreshError = "";
    return { refreshAccountingReferenceData: touch(state, connection) };
  },

  CompleteAccountingSetup: (state, v) => {
    const system = systemOf(v);
    const connection = connectionOf(state, system);
    if (!mappingSummary(state, system).canCompleteSetup) {
      throw new TriageError("Confirm every required mapping before finishing setup", {
        code: "MAPPINGS_INCOMPLETE",
        type: "business-rule-violation",
      });
    }
    connection.setupStep = "StartDate";
    return { completeAccountingSetup: touch(state, connection) };
  },

  StartAccountingAuthorization: (state, v) => {
    const system = systemOf(v);
    const meta = PROFILES[system];
    // Stands in for the provider's sign-in page: it approves at once and sends the person to
    // the app's own callback, with the company when the provider's callback carries one.
    const params = new URLSearchParams({
      code: "mock-authorization-code",
      state: `mock-${system}`,
    });
    if (meta.profile.callbackCarriesCompany) params.set("realmId", COMPANIES[system].id);
    return {
      startAccountingAuthorization: {
        authorizeUrl: `${WEB_ORIGIN}${meta.profile.callbackPath}?${params}`,
        expiresAt: state.now() + 600,
      },
    };
  },

  FinishAccountingAuthorization: (state, { input }) => {
    const system = systemOf({ input });
    const choices = COMPANY_CHOICES[system];
    if (choices && !input.realmId) {
      const token = `choice_${system}_${state.now()}`;
      accounting(state).choices.set(token, system);
      return {
        finishAccountingAuthorization: authorizationResult(null, choices, token, state.now() + 900),
      };
    }
    return { finishAccountingAuthorization: authorizationResult(connect(state, system)) };
  },

  ChooseAccountingCompany: (state, { input }) => {
    const system = systemOf({ input });
    if (accounting(state).choices.get(input.choiceToken) !== system) {
      throw new TriageError("The time to choose ran out. Connect again.", {
        code: "CHOICE_EXPIRED",
        type: "business-rule-violation",
      });
    }
    accounting(state).choices.delete(input.choiceToken);
    return { chooseAccountingCompany: connect(state, system, { companyId: input.companyId }) };
  },

  PrepareAccountingCertificate: (state, { input }) => {
    const system = systemOf({ input });
    const now = state.now();
    const prepared = {
      integrationType: system,
      certificatePem: MOCK_CERTIFICATE_PEM,
      expiresAt: now + 730 * DAY,
      preparedAt: now,
    };
    accounting(state).pendingCertificates[system] = prepared;
    return { prepareAccountingCertificate: prepared };
  },

  ConnectAccountingWithCertificate: (state, { input }) => {
    const system = systemOf({ input });
    const acct = accounting(state);
    if (!acct.pendingCertificates[system]) {
      throw new TriageError("Generate a certificate first", {
        code: "NO_CERTIFICATE",
        type: "business-rule-violation",
      });
    }
    const connection = connect(state, system, { certificate: true });
    connection.certificateId = input.certificateId;
    connection.certificateExpiresAt = acct.pendingCertificates[system].expiresAt;
    connection.refreshTokenAbsoluteExpiresAt = connection.certificateExpiresAt;
    acct.pendingCertificates[system] = null;
    return { connectAccountingWithCertificate: authorizationResult(connection) };
  },

  RenewAccountingCertificate: (state, { input }) => {
    const acct = accounting(state);
    const system = ACCOUNTING_SYSTEMS.find((s) => acct.connections[s]?.id === input.connectionId);
    const connection = connectionOf(state, system);
    const expiresAt = acct.pendingCertificates[system]?.expiresAt ?? state.now() + 730 * DAY;
    Object.assign(connection, {
      certificateId: input.certificateId,
      certificateExpiresAt: expiresAt,
      refreshTokenAbsoluteExpiresAt: expiresAt,
      status: "Connected",
      consecutiveFailures: 0,
      lastErrorCategory: null,
      lastErrorMessage: "",
    });
    acct.pendingCertificates[system] = null;
    return { renewAccountingCertificate: touch(state, connection) };
  },

  DisconnectAccountingSystem: (state, v) => {
    const connection = connectionOf(state, systemOf(v));
    connection.status = "Disconnected";
    connection.disconnectedAt = state.now();
    return { disconnectAccountingSystem: touch(state, connection) };
  },

  CheckAccountingConnection: (state, v) => {
    const connection = connectionOf(state, systemOf(v));
    const now = state.now();
    connection.lastCheckedAt = now;
    if (connection.status === "Connected") connection.lastSuccessAt = now;
    else connection.lastFailureAt = now;
    return { checkAccountingConnection: touch(state, connection) };
  },

  ChooseAccountingSyncMode: (state, { input }) => {
    const system = systemOf({ input });
    const connection = connectionOf(state, system);
    if (input.mode === "Ledger" && !PROFILES[system].profile.ledgerAvailable) {
      throw new TriageError(PROFILES[system].profile.ledgerUnavailableReason, { field: "mode" });
    }
    connection.syncMode = input.mode;
    connection.ledgerGranularity = input.mode === "Ledger" ? input.granularity : null;
    connection.setupStep = "Mappings";
    return { chooseAccountingSyncMode: touch(state, connection) };
  },

  EnableAccountingSync: (state, { input }) => {
    const connection = connectionOf(state, systemOf({ input }));
    const now = state.now();
    Object.assign(connection, {
      setupStep: "Complete",
      syncStartDate: input.startDate,
      syncEnabledAt: now,
      autoSync: input.autoSync,
      syncsDriverSettlements: Boolean(input.driverSettlements),
      driverSettlementsEnabledAt: input.driverSettlements ? now : null,
      ledgerOpeningBalancesSentAt: input.openingBalances ? now : null,
    });
    return { enableAccountingSync: touch(state, connection) };
  },

  UpdateAccountingSyncSettings: (state, { input }) => {
    const connection = connectionOf(state, systemOf({ input }));
    const turningOn = input.driverSettlements && !connection.syncsDriverSettlements;
    Object.assign(connection, {
      autoSync: input.autoSync,
      syncsDriverSettlements: input.driverSettlements,
      driverSettlementsEnabledAt: input.driverSettlements
        ? turningOn
          ? state.now()
          : connection.driverSettlementsEnabledAt
        : null,
      inboundPaymentPolicy: input.inboundPayments ?? connection.inboundPaymentPolicy,
    });
    return { updateAccountingSyncSettings: touch(state, connection) };
  },
};

/** integrationservice.buildCatalogStatus. */
function catalogStatus(enabled, configured) {
  return {
    connection: enabled ? "connected" : "disconnected",
    connectionLabel: enabled ? "Connected" : "Disconnected",
    configuration: configured ? "configured" : "needs_setup",
    configurationLabel: configured ? "Configured" : "Needs Setup",
  };
}

export const INTEGRATION_ROUTES = [
  [
    "GET",
    /^\/api\/v1\/integrations\/catalog\/?$/,
    (state) => {
      const acct = accounting(state);
      const items = CATALOG_DEFINITIONS.map((definition) => {
        // An accounting system counts as installed while it holds a live connection; the
        // other integrations start uninstalled, as on a new organization.
        const installed = ACCOUNTING_SYSTEMS.includes(definition.type)
          ? live(acct.connections[definition.type])
          : definition.type === "GoogleMaps" && state.scenario.maps;
        return {
          ...definition,
          enabled: installed,
          configured: installed,
          status: catalogStatus(installed, installed),
          configSpec: [],
          supportsTestConnect: false,
        };
      });
      items.sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name));
      return { items };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/integrations\/([A-Za-z]+)\/config\/?$/,
    (_state, path) => ({
      type: /integrations\/([A-Za-z]+)\/config/.exec(path)[1],
      enabled: false,
      fields: [],
      spec: [],
      updatedAt: 0,
    }),
  ],
];
