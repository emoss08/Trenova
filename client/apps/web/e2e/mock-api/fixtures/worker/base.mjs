// The worker record page's sample organization: its clock, ids, the people in the office and the
// catalogs every worker's file draws on (credential types, courses, templates, policies, plans,
// pay profiles, shift templates, custom fields). Design: design-plans/worker-page/DESIGN.md §6.

export const DAY = 86400;
export const HOUR = 3600;
export const MINUTE = 60;

/** The page is drawn for Sunday 22 November 2026, 4:08 PM in the viewer's zone (UTC). */
export const WORKER_NOW = Date.UTC(2026, 10, 22, 16, 8, 0) / 1000;

/** Midnight UTC of a calendar day, the way the server stores dates. */
export function day(year, month, date) {
  return Date.UTC(year, month - 1, date) / 1000;
}

/** A time of day on a calendar day, in UTC. */
export function at(year, month, date, hour, minute = 0) {
  return Date.UTC(year, month - 1, date, hour, minute) / 1000;
}

/** Whole UTC days from `now` to `then`, as `workercredential.DaysUntil` counts them. */
export function daysUntil(then, now = WORKER_NOW) {
  return Math.floor(then / DAY) - Math.floor(now / DAY);
}

/** An opaque id in the product's style: a kind prefix and a 26-character ULID body. */
export function pid(prefix, n) {
  return `${prefix}_01JF8Q2M3V${String(n).padStart(16, "0")}`;
}

export const ORG_ID = "org_mock";
export const BU_ID = "bu_mock";
export const TENANT = { organizationId: ORG_ID, businessUnitId: BU_ID };

/** The office (DESIGN §6). Dana is the safety manager reading the page. */
export const PEOPLE = {
  dana: {
    id: pid("usr", 1),
    name: "Dana Whitfield",
    emailAddress: "dana.whitfield@trenovafreight.example",
  },
  marcus: {
    id: pid("usr", 2),
    name: "Marcus Bell",
    emailAddress: "marcus.bell@trenovafreight.example",
  },
  priya: {
    id: pid("usr", 3),
    name: "Priya Raman",
    emailAddress: "priya.raman@trenovafreight.example",
  },
};

export function userRef(person) {
  return person ? { id: person.id, name: person.name } : null;
}

export const STATES = {
  IL: { id: pid("ust", 14), name: "Illinois", abbreviation: "IL" },
  IN: { id: pid("ust", 15), name: "Indiana", abbreviation: "IN" },
  WI: { id: pid("ust", 50), name: "Wisconsin", abbreviation: "WI" },
  OH: { id: pid("ust", 36), name: "Ohio", abbreviation: "OH" },
  MI: { id: pid("ust", 23), name: "Michigan", abbreviation: "MI" },
  IA: { id: pid("ust", 16), name: "Iowa", abbreviation: "IA" },
  MO: { id: pid("ust", 26), name: "Missouri", abbreviation: "MO" },
};

export const FLEET_CODES = {
  "MW-OTR": {
    id: pid("flc", 1),
    code: "MW-OTR",
    color: "#14b8a6",
    description: "Midwest over the road",
  },
  "CHI-LOCAL": {
    id: pid("flc", 2),
    code: "CHI-LOCAL",
    color: "#6366f1",
    description: "Chicago local",
  },
  "MW-REG": {
    id: pid("flc", 3),
    code: "MW-REG",
    color: "#f59e0b",
    description: "Midwest regional",
  },
  "IND-DED": {
    id: pid("flc", 4),
    code: "IND-DED",
    color: "#ec4899",
    description: "Indianapolis dedicated",
  },
};

export const POSITIONS = {
  companyDriver: { id: pid("jpo", 1), title: "Company driver" },
  leadDriver: { id: pid("jpo", 2), title: "Lead driver" },
  trainer: { id: pid("jpo", 3), title: "Driver trainer" },
  ownerOperator: { id: pid("jpo", 4), title: "Owner-operator" },
};

const stamp = { version: 1, createdAt: day(2022, 1, 10), updatedAt: day(2025, 1, 10) };

// ---- Credential types ---------------------------------------------------------------------

function credentialType(n, spec) {
  return {
    id: pid("wct", n),
    ...TENANT,
    description: null,
    status: "Active",
    requiredForDriverTypes: null,
    validityMonths: null,
    requiresNumber: false,
    requiresDocument: true,
    profileField: null,
    isSystem: true,
    activeCredentialCount: 0,
    ...stamp,
    sortOrder: n,
    ...spec,
  };
}

export const CREDENTIAL_TYPES = {
  cdl: credentialType(1, {
    code: "CDL",
    name: "Commercial driver's license",
    category: "License",
    isRequired: true,
    renewalWindowDays: 60,
    validityMonths: 96,
    requiresNumber: true,
    profileField: "licenseExpiry",
  }),
  medical: credentialType(2, {
    code: "MEC",
    name: "Medical examiner's certificate",
    category: "Medical",
    isRequired: true,
    renewalWindowDays: 90,
    validityMonths: 24,
    requiresNumber: true,
    profileField: "medicalCardExpiry",
  }),
  hazmat: credentialType(3, {
    code: "HAZMAT",
    name: "Hazmat endorsement",
    category: "Endorsement",
    isRequired: true,
    requiredForDriverTypes: ["OTR", "Regional"],
    renewalWindowDays: 60,
    validityMonths: 60,
    profileField: "hazmatExpiry",
  }),
  tanker: credentialType(4, {
    code: "TANKER",
    name: "Tanker endorsement",
    category: "Endorsement",
    isRequired: true,
    requiredForDriverTypes: ["OTR", "Regional"],
    renewalWindowDays: 30,
    requiresDocument: false,
  }),
  twic: credentialType(5, {
    code: "TWIC",
    name: "TWIC card",
    category: "Security",
    isRequired: true,
    requiredForDriverTypes: ["OTR", "Regional"],
    renewalWindowDays: 60,
    validityMonths: 60,
    requiresNumber: true,
    profileField: "twicExpiry",
  }),
  mvr: credentialType(6, {
    code: "MVR",
    name: "Motor vehicle record",
    category: "Background",
    isRequired: true,
    renewalWindowDays: 30,
    validityMonths: 12,
    profileField: "mvrDueDate",
  }),
  roadTest: credentialType(7, {
    code: "ROADTEST",
    name: "Road test certificate",
    category: "Certification",
    isRequired: true,
    renewalWindowDays: 30,
  }),
  background: credentialType(8, {
    code: "BACKGROUND",
    name: "Background check",
    category: "Background",
    isRequired: true,
    renewalWindowDays: 30,
  }),
  passport: credentialType(9, {
    code: "PASSPORT",
    name: "Passport",
    category: "Other",
    isRequired: false,
    renewalWindowDays: 90,
    validityMonths: 120,
    requiresNumber: true,
    isSystem: false,
  }),
};

export const CREDENTIAL_TYPE_LIST = Object.values(CREDENTIAL_TYPES);

/** The credential types a driver type must hold, in the catalog's order. */
export function requiredCredentialTypes(driverType) {
  return CREDENTIAL_TYPE_LIST.filter(
    (type) =>
      type.isRequired &&
      (!type.requiredForDriverTypes || type.requiredForDriverTypes.includes(driverType)),
  );
}

// ---- Training courses ---------------------------------------------------------------------

function course(n, spec) {
  return {
    id: pid("trc", n),
    ...TENANT,
    description: null,
    status: "Active",
    contentUrl: null,
    passingScore: "80",
    validityMonths: null,
    renewalWindowDays: 60,
    isRequired: true,
    requiredForDriverTypes: [],
    dueDaysAfterAssignment: 45,
    requiresAcknowledgement: false,
    openRecordCount: 0,
    ...stamp,
    sortOrder: n,
    ...spec,
  };
}

export const COURSES = {
  hazmat: course(1, {
    code: "HM-REFRESH",
    name: "Hazmat awareness refresher",
    category: "Compliance",
    delivery: "Online",
    durationMinutes: 45,
    validityMonths: 36,
    requiredForDriverTypes: ["OTR", "Regional"],
    description: "49 CFR 172.704 general awareness, safety and security training.",
  }),
  defensive: course(2, {
    code: "DEF-DRIVE",
    name: "Defensive driving",
    category: "Safety",
    delivery: "Classroom",
    durationMinutes: 240,
    validityMonths: 24,
  }),
  hos: course(3, {
    code: "HOS-ELD",
    name: "Hours of service and ELD basics",
    category: "Compliance",
    delivery: "Online",
    durationMinutes: 60,
    validityMonths: 24,
  }),
  cargo: course(4, {
    code: "CARGO-SEC",
    name: "Cargo securement",
    category: "Safety",
    delivery: "OnTheJob",
    durationMinutes: 90,
    validityMonths: 24,
  }),
  winter: course(5, {
    code: "WINTER",
    name: "Winter driving",
    category: "Safety",
    delivery: "Online",
    durationMinutes: 30,
    validityMonths: 12,
    renewalWindowDays: 30,
  }),
  eldt: course(6, {
    code: "ELDT",
    name: "Entry-level driver training",
    category: "Orientation",
    delivery: "Classroom",
    durationMinutes: 480,
    passingScore: null,
  }),
  dock: course(7, {
    code: "DOCK-CS",
    name: "Customer service on the dock",
    category: "Other",
    delivery: "Online",
    durationMinutes: 30,
    isRequired: false,
    passingScore: null,
  }),
};

export const COURSE_LIST = Object.values(COURSES);

export function requiredCourses(driverType) {
  return COURSE_LIST.filter(
    (entry) =>
      entry.isRequired &&
      (entry.requiredForDriverTypes.length === 0 ||
        entry.requiredForDriverTypes.includes(driverType)),
  );
}

// ---- Review templates ---------------------------------------------------------------------

const REVIEW_ITEMS = [
  { key: "safety", label: "Safety and compliance", description: null, weight: 1 },
  { key: "ontime", label: "On-time performance", description: null, weight: 1 },
  { key: "equipment", label: "Equipment care", description: null, weight: 1 },
  { key: "customer", label: "Customer service", description: null, weight: 1 },
  { key: "paperwork", label: "Paperwork and logs", description: null, weight: 1 },
];

export const REVIEW_TEMPLATES = {
  annual: {
    id: pid("prt", 1),
    ...TENANT,
    code: "DRV-ANNUAL",
    name: "Driver annual",
    description: "The yearly review every driver gets.",
    status: "Active",
    isDefault: true,
    cadenceMonths: 12,
    items: REVIEW_ITEMS,
    openReviewCount: 3,
    ...stamp,
  },
  ninety: {
    id: pid("prt", 2),
    ...TENANT,
    code: "DRV-90",
    name: "90-day review",
    description: "The end of probation.",
    status: "Active",
    isDefault: false,
    cadenceMonths: null,
    items: REVIEW_ITEMS,
    openReviewCount: 0,
    ...stamp,
  },
};

// ---- Checklist templates ------------------------------------------------------------------

function templateItem(templateN, n, spec) {
  return {
    id: pid("wcti", templateN * 100 + n),
    templateId: pid("wctm", templateN),
    description: null,
    required: true,
    dueOffsetDays: 14,
    credentialTypeId: null,
    documentTypeId: null,
    documentTypeName: null,
    credentialType: null,
    sortOrder: n,
    ...spec,
  };
}

function credentialItemSpec(type) {
  return {
    kind: "Credential",
    credentialTypeId: type.id,
    credentialType: { id: type.id, code: type.code, name: type.name },
  };
}

export const CHECKLIST_TEMPLATES = {
  onboarding: {
    id: pid("wctm", 1),
    ...TENANT,
    code: "ONBOARD",
    name: "Onboarding",
    description: "Everything a new driver needs before the first load.",
    kind: "Onboarding",
    trigger: "Hired",
    status: "Active",
    isDefault: true,
    openChecklistCount: 1,
    ...stamp,
    items: [
      templateItem(1, 1, {
        label: "Employment application",
        kind: "Document",
        owner: "HR",
        dueOffsetDays: 0,
        documentTypeName: "Driver application",
      }),
      templateItem(1, 2, {
        label: "Commercial driver's license on file",
        owner: "Safety",
        dueOffsetDays: 0,
        ...credentialItemSpec(CREDENTIAL_TYPES.cdl),
      }),
      templateItem(1, 3, {
        label: "Medical card on file",
        owner: "Safety",
        dueOffsetDays: 0,
        ...credentialItemSpec(CREDENTIAL_TYPES.medical),
      }),
      templateItem(1, 4, {
        label: "Motor vehicle record at hire",
        owner: "Safety",
        dueOffsetDays: 3,
        ...credentialItemSpec(CREDENTIAL_TYPES.mvr),
      }),
      templateItem(1, 5, {
        label: "Pre-employment drug test",
        kind: "Task",
        owner: "Safety",
        dueOffsetDays: 3,
      }),
      templateItem(1, 6, {
        label: "Clearinghouse pre-employment query",
        kind: "Task",
        owner: "Safety",
        dueOffsetDays: 3,
      }),
      templateItem(1, 7, {
        label: "Road test",
        owner: "Safety",
        dueOffsetDays: 7,
        ...credentialItemSpec(CREDENTIAL_TYPES.roadTest),
      }),
      templateItem(1, 8, {
        label: "Request safety history from previous employers",
        kind: "Task",
        owner: "Safety",
        dueOffsetDays: 3,
      }),
      templateItem(1, 9, {
        label: "W-4 and I-9",
        kind: "Document",
        owner: "Payroll",
        dueOffsetDays: 3,
        documentTypeName: "Tax and eligibility forms",
      }),
      templateItem(1, 10, {
        label: "Direct deposit form",
        kind: "Document",
        owner: "Payroll",
        dueOffsetDays: 7,
        documentTypeName: "Direct deposit form",
      }),
      templateItem(1, 11, {
        label: "Assign a pay profile",
        kind: "Task",
        owner: "Payroll",
        dueOffsetDays: 7,
      }),
      templateItem(1, 12, {
        label: "Issue fuel card",
        kind: "Equipment",
        owner: "Fleet",
        dueOffsetDays: 7,
      }),
      templateItem(1, 13, {
        label: "Invite to Dash",
        kind: "PortalAccess",
        owner: "IT",
        dueOffsetDays: 7,
      }),
      templateItem(1, 14, {
        label: "Ride-along with a trainer",
        kind: "Task",
        owner: "Dispatch",
        dueOffsetDays: 14,
      }),
    ],
  },
  annualReview: {
    id: pid("wctm", 2),
    ...TENANT,
    code: "ANNUAL-FILE",
    name: "Annual file review",
    description: "The yearly pass over every driver's qualification file.",
    kind: "Custom",
    trigger: "Manual",
    status: "Active",
    isDefault: false,
    openChecklistCount: 6,
    ...stamp,
    items: [
      templateItem(2, 1, {
        label: "Medical card on file",
        owner: "Safety",
        dueOffsetDays: 7,
        ...credentialItemSpec(CREDENTIAL_TYPES.medical),
      }),
      templateItem(2, 2, {
        label: "Annual driving record review",
        owner: "Safety",
        dueOffsetDays: 7,
        ...credentialItemSpec(CREDENTIAL_TYPES.mvr),
      }),
      templateItem(2, 3, {
        label: "Clearinghouse annual query",
        kind: "Task",
        owner: "Safety",
        dueOffsetDays: 37,
      }),
      templateItem(2, 4, {
        label: "Home address confirmed",
        kind: "Task",
        owner: "HR",
        dueOffsetDays: 14,
      }),
      templateItem(2, 5, {
        label: "Emergency contact confirmed",
        kind: "Task",
        owner: "HR",
        dueOffsetDays: 14,
      }),
      templateItem(2, 6, {
        label: "Driver handbook acknowledged",
        kind: "Document",
        owner: "HR",
        dueOffsetDays: 21,
        documentTypeName: "Driver handbook",
      }),
      templateItem(2, 7, {
        label: "I-9 reverification",
        kind: "Document",
        owner: "HR",
        dueOffsetDays: 21,
        documentTypeName: "Tax and eligibility forms",
      }),
      templateItem(2, 8, {
        label: "Observed ride-along",
        kind: "Task",
        owner: "Dispatch",
        required: false,
        dueOffsetDays: 28,
      }),
      templateItem(2, 9, {
        label: "W-4 reviewed for next year",
        kind: "Document",
        owner: "Payroll",
        dueOffsetDays: 28,
        documentTypeName: "Tax and eligibility forms",
      }),
    ],
  },
  offboarding: {
    id: pid("wctm", 3),
    ...TENANT,
    code: "OFFBOARD",
    name: "Offboarding",
    description: "Closing out a driver who leaves.",
    kind: "Offboarding",
    trigger: "Terminated",
    status: "Active",
    isDefault: true,
    openChecklistCount: 0,
    ...stamp,
    items: [
      templateItem(3, 1, {
        label: "Collect fuel card",
        kind: "Equipment",
        owner: "Fleet",
        dueOffsetDays: 1,
      }),
      templateItem(3, 2, {
        label: "Final settlement",
        kind: "Task",
        owner: "Payroll",
        dueOffsetDays: 7,
      }),
      templateItem(3, 3, {
        label: "Revoke Dash access",
        kind: "PortalAccess",
        owner: "IT",
        dueOffsetDays: 1,
      }),
    ],
  },
};

// ---- Time off policies, shifts, benefit plans, pay profiles -------------------------------

export const PTO_POLICIES = {
  otr: {
    id: pid("ptp", 1),
    name: "OTR standard",
    code: "PTO-OTR",
    status: "Active",
    countWeekends: false,
    requiresApproval: true,
    enforceBalance: true,
  },
  local: {
    id: pid("ptp", 2),
    name: "Local hourly",
    code: "PTO-LOCAL",
    status: "Active",
    countWeekends: false,
    requiresApproval: true,
    enforceBalance: true,
  },
};

export const SHIFT_TEMPLATES = {
  early: {
    id: pid("sht", 1),
    code: "EARLY-5",
    name: "Early local, Mon to Fri",
    color: "#0ea5e9",
    daysOfWeek: "1,2,3,4,5",
    startMinute: 5 * 60,
    durationMinutes: 10 * 60,
    cycleWeeks: null,
  },
  late: {
    id: pid("sht", 2),
    code: "LATE-4",
    name: "Late local, four tens",
    color: "#a855f7",
    daysOfWeek: "1,2,3,4",
    startMinute: 14 * 60,
    durationMinutes: 10 * 60,
    cycleWeeks: null,
  },
};

export const BENEFIT_PLANS = {
  medical: {
    id: pid("bnp", 1),
    code: "BCBS-PPO",
    name: "Blue Cross PPO",
    planType: "Medical",
    carrier: "Blue Cross Blue Shield of Illinois",
  },
  dental: {
    id: pid("bnp", 2),
    code: "DELTA",
    name: "Delta Dental",
    planType: "Dental",
    carrier: "Delta Dental of Illinois",
  },
  retirement: {
    id: pid("bnp", 3),
    code: "401K",
    name: "401(k)",
    planType: "Retirement",
    carrier: "Fidelity",
  },
  vision: {
    id: pid("bnp", 4),
    code: "VSP",
    name: "VSP Vision",
    planType: "Vision",
    carrier: "VSP",
  },
};

function component(profileN, n, spec) {
  return {
    id: pid("ppc", profileN * 100 + n),
    revenueBasis: null,
    bands: null,
    isActive: true,
    ...spec,
  };
}

export const PAY_PROFILES = {
  otrMileage: {
    id: pid("ppr", 1),
    name: "OTR mileage, company driver",
    classification: "CompanyDriver",
    currencyCode: "USD",
    guaranteedPeriodMinimumMinor: 120000,
    components: [
      component(1, 1, {
        kind: "Linehaul",
        method: "PerLoadedMile",
        description: "Loaded miles",
        rate: "0.60",
      }),
      component(1, 2, {
        kind: "Linehaul",
        method: "PerEmptyMile",
        description: "Empty miles",
        rate: "0.45",
      }),
      component(1, 3, {
        kind: "StopPay",
        method: "PerStop",
        description: "Stop pay",
        rate: "25.00",
      }),
      component(1, 4, {
        kind: "Detention",
        method: "PerHour",
        description: "Detention",
        rate: "20.00",
      }),
    ],
  },
  localHourly: {
    id: pid("ppr", 2),
    name: "Local hourly",
    classification: "CompanyDriver",
    currencyCode: "USD",
    guaranteedPeriodMinimumMinor: 0,
    components: [
      component(2, 1, {
        kind: "Linehaul",
        method: "PerHour",
        description: "Hours worked",
        rate: "27.50",
      }),
      component(2, 2, {
        kind: "Detention",
        method: "PerHour",
        description: "Detention",
        rate: "18.00",
      }),
    ],
  },
  ownerOperator: {
    id: pid("ppr", 3),
    name: "Owner-operator percentage",
    classification: "OwnerOperator",
    currencyCode: "USD",
    guaranteedPeriodMinimumMinor: 0,
    components: [
      component(3, 1, {
        kind: "Linehaul",
        method: "PercentOfRevenue",
        description: "Share of linehaul",
        rate: "72",
        revenueBasis: "LinehaulPlusFuelSurcharge",
      }),
    ],
  },
};

// ---- Custom fields ------------------------------------------------------------------------

export const WORKER_CUSTOM_FIELDS = [
  {
    id: pid("cfd", 1),
    name: "truck_stop_loyalty_number",
    label: "Truck stop loyalty number",
    description: "Pilot Flying J card on file.",
    fieldType: "text",
    isRequired: false,
    displayOrder: 1,
    color: null,
    options: [],
    validationRules: { minLength: null, maxLength: 20, min: null, max: null, pattern: null },
    defaultValue: null,
    uiAttributes: { placeholder: "PF-0000-0000", helpText: "", width: "half" },
  },
  {
    id: pid("cfd", 2),
    name: "uniform_size",
    label: "Uniform size",
    description: null,
    fieldType: "select",
    isRequired: false,
    displayOrder: 2,
    color: null,
    options: ["XS", "S", "M", "L", "XL", "XXL"].map((size) => ({
      value: size,
      label: size,
      color: "",
      description: "",
    })),
    validationRules: null,
    defaultValue: null,
    uiAttributes: { placeholder: "Choose a size", helpText: "", width: "half" },
  },
];
