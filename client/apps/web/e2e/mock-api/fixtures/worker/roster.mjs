// The rest of the roster: thirteen lighter workers with varied status, type, compliance and
// standing, so the workers list has something to sort and the roster stepper has neighbours.
// Two of them are the page's variants: Darnell Brooks, whose medical card lapsed (not cleared,
// standing Blocked), and Kenji Tanaka, hired four days ago and mid-onboarding (every list empty).
import {
  CHECKLIST_TEMPLATES,
  COURSES,
  CREDENTIAL_TYPES,
  DAY,
  PAY_PROFILES,
  PEOPLE,
  PTO_POLICIES,
  SHIFT_TEMPLATES,
  at,
  day,
  pid,
  requiredCourses,
  requiredCredentialTypes,
  userRef,
} from "./base.mjs";
import { workerDocument } from "./maria.mjs";
import {
  credential,
  disciplinaryAction,
  employmentEvent,
  ptoRequest,
  safetyEvent,
  trainingRecord,
  worker,
} from "./records.mjs";

export const NOT_CLEARED_ID = pid("wrk", 2);
export const NEW_HIRE_ID = pid("wrk", 3);

const CREDENTIAL_KEYS = Object.fromEntries(
  Object.entries(CREDENTIAL_TYPES).map(([key, type]) => [type.id, key]),
);
const COURSE_KEYS = Object.fromEntries(Object.entries(COURSES).map(([key, c]) => [c.id, key]));

/** Default issue and expiry for a credential type, so a worker holds every one in date. */
function defaultCredential(key, hireDate) {
  switch (key) {
    case "cdl":
      return { issuedAt: day(2021, 5, 10), expiresAt: day(2029, 5, 10), number: "D400-1182-7731" };
    case "medical":
      return { issuedAt: day(2025, 9, 1), expiresAt: day(2027, 9, 1), number: "MEC-50112" };
    case "hazmat":
      return { issuedAt: day(2023, 3, 15), expiresAt: day(2028, 3, 15) };
    case "twic":
      return { issuedAt: day(2023, 1, 20), expiresAt: day(2028, 1, 20), number: "5521-0912-3318" };
    case "mvr":
      return { issuedAt: day(2026, 5, 4), expiresAt: day(2027, 5, 4) };
    default:
      return { issuedAt: hireDate - 5 * DAY };
  }
}

function lightFile(n, spec) {
  const id = pid("wrk", n);
  const hireDate = spec.hireDate;
  const base = worker({
    id,
    licenseExpiry: day(2029, 5, 10),
    medicalCardExpiry: day(2027, 9, 1),
    licenseNumber: `D${400 + n}-${1100 + n * 7}-${7700 + n * 3}`,
    addressLine1: `${100 + n * 37} ${spec.street ?? "Oak St"}`,
    postalCode: spec.postalCode ?? "60601",
    phoneNumber: `(312) 555-01${String(10 + n).padStart(2, "0")}`,
    email: `${spec.firstName.toLowerCase()}.${spec.lastName.toLowerCase()}@example.com`,
    emergencyContactName: spec.emergencyContactName ?? null,
    lastHomeAt: spec.driverType === "Local" ? day(2026, 11, 21) : day(2026, 11, 15),
    manager: "dana",
    position: spec.type === "Contractor" ? "ownerOperator" : "companyDriver",
    ...spec,
  });

  const credentials = [];
  if (!spec.noCredentials) {
    let c = n * 100;
    for (const type of requiredCredentialTypes(base.driverType)) {
      const key = CREDENTIAL_KEYS[type.id];
      const override = spec.credentials?.[key];
      if (override === null) continue;
      credentials.push(
        credential(id, (c += 1), key, {
          issuingAuthority: key === "medical" ? "Concentra" : `${base.state.abbreviation} SOS`,
          verifiedBy: "dana",
          ...defaultCredential(key, hireDate),
          ...override,
        }),
      );
    }
  }

  const training = [];
  const trainingSlots = {};
  if (!spec.noTraining) {
    let t = n * 100;
    for (const course of requiredCourses(base.driverType)) {
      const key = COURSE_KEYS[course.id];
      const override = spec.training?.[key];
      const record = trainingRecord(id, (t += 1), key, {
        assignedAt: day(2026, 9, 14),
        completedAt: day(2026, 10, 1),
        expiresAt: course.validityMonths
          ? day(2026 + Math.ceil(course.validityMonths / 12), 10, 1)
          : null,
        score: course.passingScore ? "90" : null,
        recordedBy: "dana",
        ...override,
      });
      training.push(record);
      trainingSlots[course.id] = record.id;
    }
  }

  const events = [
    employmentEvent(id, n * 100 + 1, "Hired", hireDate, {
      reason: "New hire",
      to: { status: "Active", ...(spec.fleet ? { fleetCode: spec.fleet } : {}) },
    }),
  ];
  if (spec.extraEvents) events.push(...spec.extraEvents(id, n));

  const balances = spec.noPto
    ? {}
    : {
        Vacation: { maxBalanceDays: "15", carriedDays: "2", nextAccrual: null },
        Sick: { maxBalanceDays: "5", carriedDays: "0", nextAccrual: null },
      };
  const ledger = spec.noPto
    ? []
    : [
        {
          ptoType: "Vacation",
          entryType: "Carryover",
          amount: 2,
          effectiveAt: day(2026, 1, 1),
          note: "Carried from 2025",
        },
        {
          ptoType: "Vacation",
          entryType: "Accrual",
          amount: spec.vacationAccrued ?? 8,
          effectiveAt: day(2026, 10, 1),
          note: null,
        },
        {
          ptoType: "Sick",
          entryType: "Accrual",
          amount: 5,
          effectiveAt: day(2026, 1, 1),
          note: null,
        },
      ];
  const payProfile =
    spec.payProfile === undefined
      ? base.driverType === "Local"
        ? PAY_PROFILES.localHourly
        : PAY_PROFILES.otrMileage
      : spec.payProfile;

  return {
    worker: base,
    credentials,
    dqf: {
      hireDate,
      safetyHistoryDueAt: hireDate + 30 * DAY,
      items: spec.dqfItems ?? [
        {
          section: "Credentials",
          code: "CDL",
          name: "Commercial driver's license",
          status: credentials.length ? "Satisfied" : "Missing",
          detail: `Class ${base.profile.cdlClass}`,
          regulation: "391.11",
          expiresAt: null,
        },
        {
          section: "Documents",
          code: "APPLICATION",
          name: "Employment application",
          status: "Satisfied",
          detail: "Signed at hire",
          regulation: "391.21",
          expiresAt: null,
        },
        {
          section: "DrugAlcohol",
          code: "PRE_TEST",
          name: "Pre-employment test",
          status: spec.noDrugTest ? "Missing" : "Satisfied",
          detail: "Negative",
          regulation: "382.301",
          expiresAt: null,
        },
      ],
      verifications: spec.verifications ?? [],
    },
    drugAlcohol: {
      tests: spec.noDrugTest
        ? []
        : [
            {
              id: pid("wdt", n * 10 + 1),
              workerId: id,
              testType: "PreEmployment",
              substance: "Drug",
              status: "Completed",
              result: "Negative",
              isDot: true,
              reason: null,
              scheduledAt: null,
              collectedAt: hireDate - 6 * DAY,
              resultAt: hireDate - 4 * DAY,
              collectionSite: "Concentra",
              collectorName: null,
              specimenId: null,
              labName: "Quest Diagnostics",
              mroName: "Dr. R. Kim",
              mroVerifiedAt: hireDate - 4 * DAY,
              alcoholConcentration: null,
              safetyEventId: null,
              drawEntryId: null,
              documentId: null,
              notes: null,
              version: 2,
              createdAt: hireDate - 6 * DAY,
              updatedAt: hireDate - 4 * DAY,
            },
          ],
      violations: [],
      queries: spec.noDrugTest
        ? []
        : [
            {
              id: pid("wcq", n * 10 + 1),
              workerId: id,
              queryType: "AnnualLimited",
              result: "NoViolations",
              consentObtainedAt: hireDate - 7 * DAY,
              consentExpiresAt: null,
              requestedAt: spec.lastQueryAt ?? day(2026, 3, 2),
              completedAt: spec.lastQueryAt ?? day(2026, 3, 2),
              violationCount: 0,
              reference: null,
              documentId: null,
              notes: null,
              version: 1,
              createdAt: day(2026, 3, 2),
              updatedAt: day(2026, 3, 2),
            },
          ],
      randomSelections: [],
    },
    training,
    trainingSlots,
    reviews: [],
    safety: {
      scorecard: {
        score: spec.safetyScore ?? 96,
        rating: spec.safetyRating ?? "Excellent",
        pointsWatchThreshold: 6,
        pointsAtRiskThreshold: 10,
        accidents: 0,
        preventableAccidents: 0,
        incidents: spec.safetyEvents ? 1 : 0,
        nearMisses: 0,
        citations: 0,
        inspections: spec.noTraining ? 0 : 3,
        inspectionsPassed: spec.noTraining ? 0 : 3,
        inspectionsFailed: 0,
        outOfServiceOrders: 0,
        cleanInspectionRate: spec.noTraining ? null : 1,
      },
      events: spec.safetyEvents ? spec.safetyEvents(id, n) : [],
      actions: spec.actions ? spec.actions(id, n) : [],
      recognitions: [],
      injuries: [],
    },
    employmentEvents: events,
    checklists: spec.checklists ? spec.checklists(id, n) : [],
    leave: {
      entitlement: {
        method: "RollingBackward",
        totalHours: "480",
        usedHours: spec.leaveUsedHours ?? "0",
        remainingHours: String(480 - Number(spec.leaveUsedHours ?? 0)),
        totalWeeks: "12",
        usedWeeks: String(Number(spec.leaveUsedHours ?? 0) / 40),
        remainingWeeks: String(12 - Number(spec.leaveUsedHours ?? 0) / 40),
        exhausted: false,
        eligibleOnTenure: hireDate < day(2025, 11, 22),
        monthsEmployed: Math.max(0, Math.floor((day(2026, 11, 22) - hireDate) / (30.4 * DAY))),
        militaryCaregiver: false,
        window: { from: day(2025, 11, 22), through: day(2026, 11, 22) },
      },
      cases: spec.leaveCases ? spec.leaveCases(id) : [],
    },
    pto: {
      policyAssignments: spec.noPto
        ? []
        : [
            {
              id: pid("wpa", n * 10 + 1),
              workerId: id,
              ptoPolicyId:
                base.driverType === "Local" ? PTO_POLICIES.local.id : PTO_POLICIES.otr.id,
              effectiveFrom: hireDate,
              effectiveTo: null,
              assignedById: PEOPLE.priya.id,
              note: null,
              version: 1,
              createdAt: hireDate,
              updatedAt: hireDate,
              ptoPolicy: {
                ...(base.driverType === "Local" ? PTO_POLICIES.local : PTO_POLICIES.otr),
              },
            },
          ],
      balances,
      requests: spec.ptoRequests ? spec.ptoRequests(base, n) : [],
      ledger,
    },
    schedule: {
      assignments: spec.shift
        ? [
            {
              id: pid("wsa", n),
              workerId: id,
              shiftTemplateId: spec.shift.id,
              effectiveFrom: hireDate,
              effectiveTo: null,
              cycleOffsetWeeks: 0,
              notes: null,
              version: 1,
              shiftTemplate: { ...spec.shift },
            },
          ]
        : [],
      availability: [],
      swaps: [],
    },
    hos: {
      link: spec.hosLink ?? "Reporting",
      provider: "Samsara",
      providerDriverId: String(53000 + n),
      tractorId: pid("trk", 4400 + n),
      tractorCode: String(4400 + n),
      vehicleId: String(281474977140000 + n),
      clocks: {
        dutyStatus: spec.dutyStatus ?? "offDuty",
        dutyStatusSince: at(2026, 11, 22, 6, 0),
        driveRemainingMs: 11 * 3_600_000,
        driveLimitMs: 11 * 3_600_000,
        shiftRemainingMs: 14 * 3_600_000,
        shiftLimitMs: 14 * 3_600_000,
        cycleRemainingMs: 52 * 3_600_000,
        cycleLimitMs: 70 * 3_600_000,
        cycleTomorrowMs: 58 * 3_600_000,
        breakRemainingMs: 8 * 3_600_000,
        breakLimitMs: 8 * 3_600_000,
        cycleStartedAt: day(2026, 11, 16),
      },
      forms: [],
    },
    pay: {
      assignments: payProfile
        ? [
            {
              id: pid("wpy", n * 10 + 1),
              workerId: id,
              payProfileId: payProfile.id,
              effectiveFrom: hireDate,
              effectiveTo: null,
              splitPercent: "100",
              rateOverrides: [],
              notes: "",
              version: 1,
              createdAt: hireDate,
            },
          ]
        : [],
      earnings: {
        accruedEventCount: payProfile ? 3 : 0,
        accruedGrossMinor: payProfile ? 96_420 : 0,
        outstandingAdvances: 0,
        escrowBalanceMinor: 0,
      },
    },
    benefits: {
      enrollments: [],
      totals: {
        grossPayMinor: spec.noPto ? 0 : 5_210_000,
        employerBenefitMinor: 0,
        employeeBenefitMinor: 0,
        accruedTimeOffDays: "0",
      },
    },
    portal: spec.portal ?? {
      linked: false,
      portalUser: null,
      pendingInvitation: null,
      invitations: [],
      acknowledgements: [],
      profileChanges: [],
    },
    documents: spec.noCredentials
      ? []
      : [
          workerDocument(
            id,
            n * 100 + 1,
            "driver-application.pdf",
            "application/pdf",
            380_112,
            hireDate - 10 * DAY,
            "marcus",
          ),
        ],
    memories: [],
    activity: [],
  };
}

function onboardingChecklist(id, n, startedAt) {
  const template = CHECKLIST_TEMPLATES.onboarding;
  const done = new Set([1, 5, 6, 9]);
  return [
    {
      id: pid("wck", n * 10 + 1),
      workerId: id,
      templateId: template.id,
      name: "Onboarding",
      kind: "Onboarding",
      status: "Open",
      startedAt,
      dueAt: startedAt + 14 * DAY,
      completedAt: null,
      cancelledAt: null,
      cancelReason: null,
      sourceEventId: pid("wee", n * 100 + 1),
      startedById: null,
      version: 4,
      createdAt: startedAt,
      updatedAt: startedAt + 2 * DAY,
      startedBy: null,
      items: template.items.map((item) => ({
        id: pid("wci", n * 1000 + item.sortOrder),
        checklistId: pid("wck", n * 10 + 1),
        label: item.label,
        description: item.description,
        kind: item.kind,
        required: item.required,
        owner: item.owner,
        dueAt: startedAt + item.dueOffsetDays * DAY,
        credentialTypeId: item.credentialTypeId,
        documentTypeId: null,
        status: done.has(item.sortOrder) ? "Done" : "Pending",
        completedById: done.has(item.sortOrder) ? PEOPLE.marcus.id : null,
        completedAt: done.has(item.sortOrder) ? startedAt + DAY : null,
        autoCompleted: false,
        note: null,
        evidenceDocumentId: null,
        evidenceCredentialId: null,
        sortOrder: item.sortOrder,
        version: 1,
        completedBy: done.has(item.sortOrder) ? userRef(PEOPLE.marcus) : null,
        evidenceDocument: null,
        evidenceCredential: null,
        credentialType: item.credentialType,
      })),
    },
  ];
}

export function buildRoster() {
  return [
    lightFile(2, {
      firstName: "Darnell",
      lastName: "Brooks",
      gender: "Male",
      dob: day(1979, 7, 22),
      driverType: "OTR",
      fleet: "MW-OTR",
      city: "Aurora",
      street: "Galena Blvd",
      postalCode: "60506",
      hireDate: day(2019, 4, 8),
      endorsement: "N",
      medicalCardExpiry: day(2026, 11, 2),
      emergencyContactName: "Keisha Brooks",
      credentials: {
        medical: { issuedAt: day(2024, 11, 2), expiresAt: day(2026, 11, 2), number: "MEC-40921" },
      },
      safetyScore: 88,
      safetyRating: "Good",
      hosLink: "Reporting",
      portal: {
        linked: true,
        portalUser: {
          id: pid("usr", 102),
          name: "Darnell Brooks",
          emailAddress: "darnell.brooks@example.com",
          lastLoginAt: day(2026, 11, 19),
        },
        pendingInvitation: null,
        invitations: [],
        acknowledgements: [],
        profileChanges: [],
      },
    }),
    lightFile(3, {
      firstName: "Kenji",
      lastName: "Tanaka",
      gender: "Male",
      dob: day(1994, 2, 11),
      driverType: "Regional",
      fleet: "MW-REG",
      city: "Naperville",
      street: "Ogden Ave",
      postalCode: "60540",
      hireDate: day(2026, 11, 18),
      noCredentials: true,
      noTraining: true,
      noDrugTest: true,
      noPto: true,
      payProfile: null,
      hosLink: "NotLinked",
      isQualified: false,
      canBeAssigned: false,
      availableForDispatch: false,
      checklists: (id, n) => onboardingChecklist(id, n, day(2026, 11, 18)),
      portal: {
        linked: false,
        portalUser: null,
        pendingInvitation: {
          id: pid("pin", 31),
          email: "kenji.tanaka@example.com",
          status: "Pending",
          expiresAt: day(2026, 11, 25),
          createdAt: day(2026, 11, 18),
        },
        invitations: [
          {
            id: pid("pin", 31),
            email: "kenji.tanaka@example.com",
            status: "Pending",
            expiresAt: day(2026, 11, 25),
            acceptedAt: null,
            createdAt: day(2026, 11, 18),
            invitedBy: userRef(PEOPLE.marcus),
          },
        ],
        acknowledgements: [],
        profileChanges: [],
      },
    }),
    lightFile(4, {
      firstName: "Rosa",
      lastName: "Delgado",
      gender: "Female",
      dob: day(1988, 10, 3),
      driverType: "Local",
      fleet: "CHI-LOCAL",
      city: "Cicero",
      postalCode: "60804",
      hireDate: day(2021, 6, 14),
      shift: SHIFT_TEMPLATES.early,
      dutyStatus: "onDuty",
    }),
    lightFile(5, {
      firstName: "Tyrone",
      lastName: "Jackson",
      gender: "Male",
      dob: day(1983, 1, 17),
      driverType: "OTR",
      fleet: "MW-OTR",
      city: "Gary",
      state: "IN",
      postalCode: "46402",
      hireDate: day(2020, 2, 3),
      endorsement: "X",
      safetyScore: 80,
      safetyRating: "Watch",
      dutyStatus: "driving",
      safetyEvents: (id, n) => [
        safetyEvent(id, n * 10 + 1, {
          kind: "Incident",
          severity: "Minor",
          status: "Closed",
          occurredAt: day(2026, 7, 9),
          description: "Clipped a mirror in a tight yard.",
          preventable: true,
          points: 6,
          pointsExpireAt: day(2027, 7, 9),
          closedBy: "dana",
          closedAt: day(2026, 7, 12),
        }),
      ],
    }),
    lightFile(6, {
      firstName: "Hannah",
      lastName: "Kowalski",
      gender: "Female",
      dob: day(1991, 5, 29),
      driverType: "Regional",
      fleet: "MW-REG",
      city: "Kenosha",
      state: "WI",
      postalCode: "53140",
      hireDate: day(2023, 8, 21),
      training: {
        defensive: {
          status: "Assigned",
          completedAt: null,
          expiresAt: null,
          score: null,
          assignedAt: day(2026, 9, 15),
          dueAt: day(2026, 10, 30),
        },
      },
    }),
    lightFile(7, {
      firstName: "Samuel",
      lastName: "Okafor",
      gender: "Male",
      dob: day(1975, 12, 2),
      type: "Contractor",
      driverType: "OTR",
      fleet: "MW-OTR",
      city: "Joliet",
      postalCode: "60435",
      hireDate: day(2018, 3, 12),
      payProfile: PAY_PROFILES.ownerOperator,
      noPto: true,
      dutyStatus: "sleeperBed",
    }),
    lightFile(8, {
      firstName: "Lena",
      lastName: "Fischer",
      gender: "Female",
      dob: day(1990, 8, 8),
      driverType: "Local",
      fleet: "CHI-LOCAL",
      city: "Evanston",
      postalCode: "60201",
      hireDate: day(2022, 1, 10),
      licenseExpiry: day(2027, 1, 4),
      credentials: {
        cdl: { issuedAt: day(2019, 1, 4), expiresAt: day(2027, 1, 4), number: "F562-0019-4471" },
      },
      shift: SHIFT_TEMPLATES.late,
    }),
    lightFile(9, {
      firstName: "Marco",
      lastName: "Bianchi",
      gender: "Male",
      dob: day(1986, 4, 19),
      driverType: "Team",
      fleet: "MW-OTR",
      city: "Elgin",
      postalCode: "60120",
      hireDate: day(2024, 5, 6),
      dutyStatus: "driving",
    }),
    lightFile(10, {
      firstName: "Priscilla",
      lastName: "Nguyen",
      gender: "Female",
      dob: day(1992, 9, 30),
      driverType: "Regional",
      fleet: "MW-REG",
      city: "Schaumburg",
      postalCode: "60173",
      hireDate: day(2021, 10, 4),
      canBeAssigned: false,
      availableForDispatch: false,
      leaveUsedHours: "96",
      extraEvents: (id, n) => [
        employmentEvent(id, n * 100 + 2, "LeaveStarted", day(2026, 11, 9), {
          reason: "Parental leave",
          to: { leaveType: "Parental", canBeAssigned: "false" },
          from: { canBeAssigned: "true" },
        }),
      ],
    }),
    lightFile(11, {
      firstName: "Andre",
      lastName: "Williams",
      gender: "Male",
      dob: day(1980, 3, 3),
      driverType: "OTR",
      fleet: "MW-OTR",
      status: "Inactive",
      city: "Hammond",
      state: "IN",
      postalCode: "46320",
      hireDate: day(2017, 7, 24),
      terminationDate: day(2026, 8, 28),
      canBeAssigned: false,
      availableForDispatch: false,
      payProfile: null,
      noPto: true,
      hosLink: "NotLinked",
      extraEvents: (id, n) => [
        employmentEvent(id, n * 100 + 2, "Terminated", day(2026, 8, 28), {
          reason: "Resigned to drive for a family business",
          from: { status: "Active" },
          to: { status: "Inactive", terminationDate: String(day(2026, 8, 28)) },
        }),
      ],
    }),
    lightFile(12, {
      firstName: "Grace",
      lastName: "Thompson",
      gender: "Female",
      dob: day(1978, 11, 11),
      driverType: "OTR",
      fleet: "MW-OTR",
      city: "Bolingbrook",
      postalCode: "60440",
      hireDate: day(2015, 9, 1),
      position: "trainer",
      endorsement: "X",
      dutyStatus: "onDuty",
    }),
    lightFile(13, {
      firstName: "Victor",
      lastName: "Ramirez",
      gender: "Male",
      dob: day(1996, 6, 6),
      driverType: "Local",
      fleet: "IND-DED",
      city: "Indianapolis",
      state: "IN",
      postalCode: "46204",
      hireDate: day(2025, 2, 17),
      shift: SHIFT_TEMPLATES.early,
    }),
    lightFile(14, {
      firstName: "Elena",
      lastName: "Petrova",
      gender: "Female",
      dob: day(1989, 2, 25),
      driverType: "Regional",
      fleet: "MW-REG",
      city: "Rockford",
      postalCode: "61101",
      hireDate: day(2022, 12, 5),
      credentials: { mvr: { issuedAt: day(2025, 12, 10), expiresAt: day(2026, 12, 10) } },
      actions: (id, n) => [
        disciplinaryAction(id, n * 10 + 1, {
          level: "Coaching",
          reason: "Late paperwork",
          issuedAt: day(2026, 10, 2),
          expiresAt: day(2027, 4, 2),
        }),
      ],
      ptoRequests: (base, n) => [
        ptoRequest(base, n * 10 + 1, {
          status: "Requested",
          type: "Vacation",
          startDate: day(2026, 12, 28),
          endDate: day(2026, 12, 31),
          reason: "New Year",
          days: "4",
          requestedAt: day(2026, 11, 16),
        }),
      ],
    }),
  ];
}
