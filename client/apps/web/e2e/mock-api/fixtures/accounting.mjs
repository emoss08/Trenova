// Accounting sync: the four providers' profiles, the organization's connection at each
// setup stage, the mappings Trenova proposes and the provider's reference records.
//
// Profiles are accountingprovider.providerProfiles
// (services/tms/internal/modules/accounting/domain/accountingprovider/profile.go); mapping keys
// and labels are accountingsync.MappingTargetType.Keys and accountingmappingtargets' label maps.

const DAY = 86400;

export const ACCOUNTING_SYSTEMS = ["QuickBooksOnline", "Xero", "BusinessCentral", "NetSuite"];

export const ACCOUNTING_STAGES = new Set([
  "connect",
  "mode",
  "mappings",
  "startDate",
  "connected",
  "degraded",
  "failing",
  "revoked",
  "expiring",
]);

export const PROFILES = {
  QuickBooksOnline: {
    name: "QuickBooks Online",
    webhookSlug: "quickbooks",
    requiredAccountRoles: ["ARAccount", "RevenueAccount", "DepositAccount"],
    accountRoles: [
      "ARAccount",
      "RevenueAccount",
      "DepositAccount",
      "WriteOffAccount",
      "APAccount",
      "PurchasedTransportationAccount",
    ],
    refreshTokenAbsoluteLifetime: 5 * 365 * DAY,
    profile: {
      appName: "Intuit app",
      webhookKeyLabel: "Webhook verifier token",
      environments: ["Sandbox", "Production"],
      ledgerAvailable: true,
      ledgerUnavailableReason: null,
      callbackCarriesCompany: true,
      callbackPath: "/admin/integrations/quickbooks/callback",
      lineKind: "Item",
      referenceKinds: ["Account", "Item", "Customer", "Vendor", "Term", "PaymentMethod"],
      inboundPaymentsAvailable: true,
      inboundUnavailableReason: null,
      webhookSubscriptions: false,
      revokesTokens: true,
      pkce: false,
      clientCredentials: false,
      reconnectWarningSeconds: 14 * DAY,
    },
  },
  Xero: {
    name: "Xero",
    webhookSlug: "xero",
    requiredAccountRoles: ["DepositAccount"],
    accountRoles: ["DepositAccount", "PurchasedTransportationAccount"],
    refreshTokenAbsoluteLifetime: 0,
    profile: {
      appName: "Xero app",
      webhookKeyLabel: "Webhook key",
      environments: ["Production"],
      ledgerAvailable: false,
      ledgerUnavailableReason:
        "Xero manual journals cannot post to accounts receivable, accounts payable or bank accounts, or name a customer or supplier, so Trenova's journals cannot be sent to Xero as they are. Send documents instead.",
      callbackCarriesCompany: false,
      callbackPath: "/admin/integrations/xero/callback",
      lineKind: "Account",
      referenceKinds: ["Account", "Item", "Customer", "Vendor"],
      inboundPaymentsAvailable: true,
      inboundUnavailableReason: null,
      webhookSubscriptions: false,
      revokesTokens: true,
      pkce: false,
      clientCredentials: false,
      reconnectWarningSeconds: 14 * DAY,
    },
  },
  BusinessCentral: {
    name: "Business Central",
    webhookSlug: "businesscentral",
    requiredAccountRoles: ["DepositAccount"],
    accountRoles: ["DepositAccount", "PurchasedTransportationAccount"],
    refreshTokenAbsoluteLifetime: 0,
    profile: {
      appName: "Microsoft Entra app",
      webhookKeyLabel: "",
      environments: ["Production"],
      ledgerAvailable: false,
      ledgerUnavailableReason:
        "Business Central journal lines cannot post to a customer or vendor or apply to an invoice, so Trenova's journals cannot be sent to Business Central as they are. Send documents instead.",
      callbackCarriesCompany: false,
      callbackPath: "/admin/integrations/business-central/callback",
      lineKind: "Item",
      referenceKinds: ["Account", "Item", "Customer", "Vendor", "Term"],
      inboundPaymentsAvailable: false,
      inboundUnavailableReason:
        "Business Central's API does not let Trenova read posted payments or which invoices they paid, so payments recorded in Business Central are not brought in. An invoice paid there shows as a balance difference on the drift page.",
      webhookSubscriptions: true,
      revokesTokens: false,
      pkce: false,
      clientCredentials: false,
      reconnectWarningSeconds: 14 * DAY,
    },
  },
  NetSuite: {
    name: "NetSuite",
    webhookSlug: "",
    requiredAccountRoles: ["DepositAccount"],
    accountRoles: ["DepositAccount", "PurchasedTransportationAccount"],
    refreshTokenAbsoluteLifetime: 7 * DAY,
    profile: {
      appName: "NetSuite integration record",
      webhookKeyLabel: "",
      environments: ["Production"],
      ledgerAvailable: false,
      ledgerUnavailableReason:
        "Ledger mode for NetSuite is not available yet: journal entries through REST web services are unconfirmed. Send documents instead.",
      callbackCarriesCompany: true,
      callbackPath: "/admin/integrations/netsuite/callback",
      lineKind: "Item",
      referenceKinds: ["Account", "Item", "Customer", "Vendor", "Term"],
      inboundPaymentsAvailable: true,
      inboundUnavailableReason: null,
      webhookSubscriptions: false,
      revokesTokens: true,
      pkce: true,
      clientCredentials: true,
      reconnectWarningSeconds: 2 * DAY,
    },
  },
};

/** accountingprovider.ProviderProfile.WebhookPath / AppWebhookPath. */
export function webhookPath(system, appId = "") {
  const slug = PROFILES[system].webhookSlug;
  if (!slug) return "";
  return appId ? `/webhooks/accounting/${slug}/${appId}/` : `/webhooks/accounting/${slug}/`;
}

/** The company each provider connects, as its own API names it. */
export const COMPANIES = {
  QuickBooksOnline: {
    id: "9341453052141850",
    name: "Peak Freight",
    legalName: "Peak Freight LLC",
    country: "US",
    currency: "USD",
    multiCurrency: false,
  },
  Xero: {
    id: "b0f9c3a2-7e41-4d0a-9a3b-3c1f6f0e2d11",
    name: "Peak Freight",
    legalName: "Peak Freight Limited",
    country: "US",
    currency: "USD",
    multiCurrency: true,
  },
  BusinessCentral: {
    id: "f2d8e5b1-0c3a-4f6e-8b9d-1a2c3e4f5a6b",
    name: "CRONUS Peak Freight",
    legalName: "Peak Freight Inc.",
    country: "US",
    currency: "USD",
    multiCurrency: false,
  },
  NetSuite: {
    id: "7012345",
    name: "Peak Freight (US)",
    legalName: "Peak Freight Holdings, Inc.",
    country: "US",
    currency: "USD",
    multiCurrency: true,
  },
};

/** Organisations a provider offers after sign-in when the callback does not carry one. */
export const COMPANY_CHOICES = {
  Xero: [
    { id: COMPANIES.Xero.id, name: "Peak Freight" },
    { id: "5e7a1c90-22b4-4c8e-b1f3-9d0a6e2c4b17", name: "Peak Freight (Demo Company)" },
  ],
  BusinessCentral: [
    { id: COMPANIES.BusinessCentral.id, name: "CRONUS Peak Freight" },
    { id: "0a1b2c3d-4e5f-6a7b-8c9d-0e1f2a3b4c5d", name: "CRONUS USA, Inc." },
  ],
};

const ROLE_LABELS = {
  ARAccount: "Accounts receivable",
  RevenueAccount: "Revenue",
  DepositAccount: "Deposit account",
  WriteOffAccount: "Short pay and write-off",
  APAccount: "Accounts payable",
  PurchasedTransportationAccount: "Purchased transportation",
};

const LINE_LABELS = { Freight: "Freight charges", Memo: "Memo lines" };

/** The provider's records Trenova's mappings point at, by reference kind. */
export const REFERENCE_OBJECTS = {
  Account: [
    ["1100", "Accounts Receivable (A/R)", "Accounts Receivable", "AccountsReceivable"],
    ["1010", "Operating Checking", "Bank", "Checking"],
    ["1020", "Undeposited Funds", "Other Current Asset", "UndepositedFunds"],
    ["2000", "Accounts Payable (A/P)", "Accounts Payable", "AccountsPayable"],
    ["4000", "Freight Revenue", "Income", "ServiceFeeIncome"],
    ["4100", "Accessorial Revenue", "Income", "ServiceFeeIncome"],
    ["5000", "Purchased Transportation", "Cost of Goods Sold", "OtherCostsOfServiceCos"],
    ["6900", "Bad Debt and Short Pays", "Expense", "BadDebts"],
  ].map(([number, label, accountType, accountSubType]) => ({
    kind: "Account",
    externalId: `acct-${number}`,
    label,
    number,
    accountType,
    accountSubType,
  })),
  Item: [
    ["Freight", "Service"],
    ["Fuel Surcharge", "Service"],
    ["Detention", "Service"],
    ["Lumper Fee", "Service"],
    ["Short-pay Write-off", "NonInventory"],
  ].map(([label, itemType], index) => ({
    kind: "Item",
    externalId: `item-${index + 1}`,
    label,
    itemType,
  })),
  Customer: [
    ["Acme Retail Group", "Bentonville", "AR", "72712"],
    ["Northwind Grocers", "Columbus", "OH", "43215"],
    ["Crestline Building Supply", "Denver", "CO", "80202"],
    ["Harborline Imports", "Long Beach", "CA", "90802"],
  ].map(([companyName, city, state, postalCode], index) => ({
    kind: "Customer",
    externalId: `cust-${index + 1}`,
    label: companyName,
    companyName,
    city,
    state,
    postalCode,
  })),
  Vendor: [
    ["Blue Ridge Transport", "Roanoke", "VA", "24011"],
    ["Lone Star Carriers", "Fort Worth", "TX", "76102"],
    ["Great Lakes Haulers", "Toledo", "OH", "43604"],
  ].map(([companyName, city, state, postalCode], index) => ({
    kind: "Vendor",
    externalId: `vend-${index + 1}`,
    label: companyName,
    companyName,
    city,
    state,
    postalCode,
  })),
  Term: [
    ["Net 15", 15],
    ["Net 30", 30],
    ["Due on receipt", 0],
  ].map(([label, dueDays], index) => ({
    kind: "Term",
    externalId: `term-${index + 1}`,
    label,
    dueDays,
  })),
  PaymentMethod: ["ACH", "Check", "Wire"].map((label, index) => ({
    kind: "PaymentMethod",
    externalId: `pm-${index + 1}`,
    label,
  })),
};

export function referenceObject(system, ref) {
  return {
    id: `acctref_${system}_${ref.externalId}`,
    kind: ref.kind,
    externalId: ref.externalId,
    label: ref.label,
    // The schema's strings are non-null: a field a kind does not have is empty.
    number: ref.number ?? "",
    accountType: ref.accountType ?? "",
    accountSubType: ref.accountSubType ?? "",
    itemType: ref.itemType ?? "",
    companyName: ref.companyName ?? "",
    city: ref.city ?? "",
    state: ref.state ?? "",
    postalCode: ref.postalCode ?? "",
    dueDays: ref.dueDays ?? null,
    active: true,
    usable: true,
  };
}

function findRef(kind, label) {
  return REFERENCE_OBJECTS[kind].find((ref) => ref.label === label) ?? null;
}

/**
 * The mappings Trenova builds for a provider: every account role it offers, the freight and
 * memo lines, and the customers, carriers and charges Trenova has, each with the record Trenova
 * proposes where it is sure enough.
 */
export function buildMappings(system, anchor) {
  const meta = PROFILES[system];
  const lineKind = meta.profile.lineKind;
  const rows = [];
  let n = 0;
  const add = (row) => {
    n += 1;
    const proposal = row.proposal ? findRef(row.providerKind, row.proposal) : null;
    rows.push({
      id: `acctm_${system}_${n}`,
      targetType: row.targetType,
      trenovaObjectId: row.trenovaObjectId ?? null,
      trenovaKey: row.trenovaKey,
      targetLabel: row.targetLabel,
      providerKind: row.providerKind,
      externalId: proposal?.externalId ?? "",
      externalName: proposal?.label ?? "",
      state: proposal ? "Proposed" : "Unmatched",
      source: proposal ? "Suggested" : null,
      confidence: proposal ? row.confidence : null,
      reason: proposal ? row.reason : "",
      required: row.required ?? false,
      prechecked: proposal ? (row.confidence ?? 0) >= 0.9 : false,
      candidates: proposal
        ? [
            {
              externalId: proposal.externalId,
              name: proposal.label,
              score: row.confidence,
              reason: row.reason,
            },
            ...(row.alternative
              ? [
                  {
                    externalId: findRef(row.providerKind, row.alternative).externalId,
                    name: row.alternative,
                    score: Math.round((row.confidence - 0.31) * 100) / 100,
                    reason: "A similar name",
                  },
                ]
              : []),
          ]
        : [],
      confirmedBy: null,
      confirmedAt: null,
      version: 1,
      updatedAt: anchor - 2 * DAY,
    });
  };

  const roleProposals = {
    ARAccount: ["Accounts Receivable (A/R)", 0.98, "The provider's accounts receivable account"],
    RevenueAccount: ["Freight Revenue", 0.86, "Named for freight income", "Accessorial Revenue"],
    DepositAccount: [
      "Undeposited Funds",
      0.93,
      "The account payments wait in before deposit",
      "Operating Checking",
    ],
    WriteOffAccount: ["Bad Debt and Short Pays", 0.74, "An expense account for bad debt"],
    APAccount: ["Accounts Payable (A/P)", 0.97, "The provider's accounts payable account"],
    PurchasedTransportationAccount: ["Purchased Transportation", 0.95, "Same name"],
  };
  for (const role of meta.accountRoles) {
    const [proposal, confidence, reason, alternative] = roleProposals[role];
    add({
      targetType: "AccountRole",
      trenovaKey: role,
      targetLabel: ROLE_LABELS[role],
      providerKind: "Account",
      required: meta.requiredAccountRoles.includes(role),
      proposal: role === "WriteOffAccount" ? null : proposal,
      confidence,
      reason,
      alternative,
    });
  }
  add({
    targetType: "LineType",
    trenovaKey: "Freight",
    targetLabel: LINE_LABELS.Freight,
    providerKind: lineKind,
    required: true,
    proposal: lineKind === "Item" ? "Freight" : "Freight Revenue",
    confidence: 0.96,
    reason: "Same name",
  });
  add({
    targetType: "LineType",
    trenovaKey: "Memo",
    targetLabel: LINE_LABELS.Memo,
    providerKind: lineKind,
    proposal: null,
  });
  for (const [key, proposal, confidence] of [
    ["FSC", "Fuel Surcharge", 0.91],
    ["DET", "Detention", 0.88],
    ["LUMP", "Lumper Fee", 0.82],
  ]) {
    add({
      targetType: "AccessorialCharge",
      trenovaObjectId: `acc_${key.toLowerCase()}`,
      trenovaKey: key,
      targetLabel: key,
      providerKind: lineKind,
      proposal: lineKind === "Item" ? proposal : "Accessorial Revenue",
      confidence,
      reason: "Same name",
    });
  }
  for (const [name, confidence] of [
    ["Acme Retail Group", 0.99],
    ["Northwind Grocers", 0.94],
    ["Crestline Building Supply", 0.77],
  ]) {
    add({
      targetType: "Customer",
      trenovaObjectId: `cus_${name.split(" ")[0].toLowerCase()}`,
      trenovaKey: name,
      targetLabel: name,
      providerKind: "Customer",
      proposal: name,
      confidence,
      reason: confidence > 0.9 ? "Same name and billing address" : "Same name",
    });
  }
  add({
    targetType: "Carrier",
    trenovaObjectId: "car_blueridge",
    trenovaKey: "Blue Ridge Transport",
    targetLabel: "Blue Ridge Transport",
    providerKind: "Vendor",
    proposal: "Blue Ridge Transport",
    confidence: 0.97,
    reason: "Same name and tax ID",
  });
  return rows;
}

/** The connection at a setup stage, as AccountingConnectionFields selects it. */
export function buildConnection(
  system,
  stage,
  anchor,
  { appSource = "Instance", certificate = false } = {},
) {
  const meta = PROFILES[system];
  const company = COMPANIES[system];
  const complete = !["mode", "mappings", "startDate"].includes(stage);
  const setupStep =
    { mode: "Mode", mappings: "Mappings", startDate: "StartDate" }[stage] ?? "Complete";
  const connectedAt = anchor - 41 * DAY;
  const status =
    { degraded: "Degraded", failing: "Failing", revoked: "Revoked" }[stage] ?? "Connected";
  const absolute = certificate
    ? anchor + (stage === "expiring" ? 9 * DAY : 690 * DAY)
    : meta.refreshTokenAbsoluteLifetime === 0
      ? 0
      : stage === "expiring"
        ? anchor + Math.round(meta.profile.reconnectWarningSeconds / 2)
        : connectedAt + meta.refreshTokenAbsoluteLifetime;
  const failing = status === "Degraded" || status === "Failing";
  return {
    id: `acctc_${system}`,
    integrationType: system,
    status,
    appSource,
    appEnvironment: appSource === "Tenant" ? "Sandbox" : "Production",
    externalCompanyName: company.name,
    externalLegalName: company.legalName,
    externalCountry: company.country,
    externalHomeCurrency: company.currency,
    externalMultiCurrencyEnabled: company.multiCurrency,
    externalBooksClosedThrough: anchor - 123 * DAY,
    lastCheckedAt: anchor - 6 * 60,
    lastSuccessAt: failing
      ? anchor - 47 * 60
      : stage === "revoked"
        ? anchor - 3 * DAY
        : anchor - 6 * 60,
    lastFailureAt: failing
      ? anchor - 6 * 60
      : stage === "revoked"
        ? anchor - 3 * DAY + 900
        : anchor - 9 * DAY,
    consecutiveFailures: status === "Failing" ? 4 : status === "Degraded" ? 1 : 0,
    lastErrorCategory: failing ? "Transient" : stage === "revoked" ? "Revoked" : null,
    lastErrorMessage: failing
      ? `${meta.name} did not answer in time.`
      : stage === "revoked"
        ? `${meta.name} no longer accepts this authorization.`
        : "",
    lastWebhookAt: meta.webhookSlug ? anchor - 22 * 60 : null,
    refreshTokenAbsoluteExpiresAt: absolute,
    authMethod: certificate ? "ClientCredentials" : "Authorization",
    certificateId: certificate ? "kZ3qv1nP0sXb7c2dA9eF4gH6jK8lM0nO" : null,
    certificateExpiresAt: certificate ? absolute : null,
    connectedAt,
    disconnectedAt: null,
    setupStep,
    syncMode: "Document",
    ledgerGranularity: null,
    ledgerOpeningBalancesSentAt: null,
    externalFiscalYearStartMonth: 1,
    syncStartDate: complete || stage === "startDate" ? anchor - 40 * DAY : null,
    syncEnabledAt: complete ? anchor - 40 * DAY : null,
    autoSync: true,
    driverSettlementsEnabledAt: null,
    syncsDriverSettlements: false,
    inboundPaymentPolicy: meta.profile.inboundPaymentsAvailable ? "Propose" : "Off",
    changesReadAt: complete ? anchor - 4 * 60 : null,
    changesErrorMessage: "",
    pausedAt: null,
    pausedBy: null,
    pausedReason: "",
    referenceRefreshStartedAt: anchor - 41 * DAY,
    referenceRefreshedAt: anchor - 41 * DAY + 95,
    referenceRefreshError: "",
    version: 4,
    updatedAt: anchor - 6 * 60,
  };
}

export const MOCK_CERTIFICATE_PEM = `-----BEGIN CERTIFICATE-----
MIIBszCCAVmgAwIBAgIUQm9ja1RyZW5vdmFOZXRTdWl0ZUNlcnQwCgYIKoZIzj0E
AwIwLzEtMCsGA1UEAwwkVHJlbm92YSBOZXRTdWl0ZSBjbGllbnQgY3JlZGVudGlh
bHMwHhcNMjYxMDMxMDAwMDAwWhcNMjgxMDMwMDAwMDAwWjAvMS0wKwYDVQQDDCRU
cmVub3ZhIE5ldFN1aXRlIGNsaWVudCBjcmVkZW50aWFsczBZMBMGByqGSM49AgEG
CCqGSM49AwEHA0IABE1vY2sgY2VydGlmaWNhdGUgZm9yIHRoZSBtb2NrIEFQSS4=
-----END CERTIFICATE-----
`;
