// The records the Desk's resource cards are drawn for, each as the artifact the
// agent's get_ lookup would have left in a conversation, and the data behind
// each card. Every time is relative to the scenario's anchor.

export const DESK_THREAD_ID = "thr_desk_cards";

const DAY = 86_400;
const HOUR = 3_600;

/** The artifacts the Desk can open in the cards thread, keyed by their slug. */
export function deskCardArtifacts(anchor) {
  const artifact = (slug, entity, recordId, title) => ({
    id: `art_${slug}`,
    threadId: DESK_THREAD_ID,
    messageId: "msg_desk_cards",
    runId: "",
    proposalId: "",
    planId: "",
    kind: "entity_card",
    status: "Ready",
    title,
    payload: {
      display: 1,
      entity,
      recordId,
      tool: `get_${entity}`,
      fields: [{ key: "code", label: "Code", type: "text", value: title }],
    },
    sourceToolCallId: `call_${slug}`,
    pinned: false,
    lineageId: "",
    lineageSeq: 1,
    slug,
    turn: "Show me the tractor",
    createdAt: anchor - 5 * 60,
    updatedAt: anchor - 5 * 60,
  });
  return {
    "tractor-t-101": artifact("tractor-t-101", "tractor", "tr_101", "Tractor T-101"),
    "tractor-t-207": artifact("tractor-t-207", "tractor", "tr_207", "Tractor T-207"),
    "tractor-t-318": artifact("tractor-t-318", "tractor", "tr_318", "Tractor T-318"),
    "tractor-t-999": artifact("tractor-t-999", "tractor", "tr_999", "Tractor T-999"),
    "trailer-tl-501": artifact("trailer-tl-501", "trailer", "trl_501", "Trailer TL-501"),
    "trailer-tl-622": artifact("trailer-tl-622", "trailer", "trl_622", "Trailer TL-622"),
    "trailer-tl-700": artifact("trailer-tl-700", "trailer", "trl_700", "Trailer TL-700"),
    "trailer-tl-999": artifact("trailer-tl-999", "trailer", "trl_999", "Trailer TL-999"),
    "worker-rosa": artifact("worker-rosa", "worker", "wrk_rosa", "Rosa Diaz"),
    "worker-ana": artifact("worker-ana", "worker", "wrk_ana", "Ana Ruiz-Montenegro"),
    "worker-jordan": artifact("worker-jordan", "worker", "wrk_new", "Jordan Lee"),
    "worker-gone": artifact("worker-gone", "worker", "wrk_999", "Casey Morgan"),
    "location-dallas": artifact("location-dallas", "location", "loc_dal", "Dallas yard"),
    "location-tyson": artifact("location-tyson", "location", "loc_tyson", "Tyson Foods DC"),
    "location-new": artifact("location-new", "location", "loc_new", "Gate 7 overflow lot"),
    "location-gone": artifact("location-gone", "location", "loc_999", "Old Fort Worth yard"),
    "carrier-redriver": artifact(
      "carrier-redriver",
      "carrier",
      "car_redriver",
      "Red River Freight",
    ),
    "carrier-lonestar": artifact(
      "carrier-lonestar",
      "carrier",
      "car_lonestar",
      "Lone Star Haulers",
    ),
    "carrier-new": artifact("carrier-new", "carrier", "car_new", "Northgate Transport"),
    "carrier-gone": artifact("carrier-gone", "carrier", "car_999", "Prairie Line Carriers"),
    "order-acme": artifact("order-acme", "order", "ord_acme", "Order ORD-2610-0118"),
    "order-tyson": artifact("order-tyson", "order", "ord_tyson", "Order ORD-2609-0977"),
    "order-draft": artifact("order-draft", "order", "ord_draft", "Order ORD-2610-0131"),
    "order-gone": artifact("order-gone", "order", "ord_999", "Order ORD-2608-0001"),
  };
}

const worker = (id, firstName, lastName) => ({ id, firstName, lastName });

/**
 * Three tractors that between them draw every state the card has:
 * T-101 is on the road with a co-driver and a clean record; T-207 is a leased
 * unit held at maintenance with an expiring registration, open defects and a
 * stale GPS report; T-318 is new to the fleet with nothing recorded against it.
 */
export function deskTractors(anchor) {
  const rosa = worker("wrk_rosa", "Rosa", "Diaz");
  const sam = worker("wrk_sam", "Samuel", "Okafor");
  const dee = worker("wrk_dee", "Deandra", "Whitfield-Castellanos");
  return {
    tr_101: {
      tractor: {
        id: "tr_101",
        code: "T-101",
        status: "Available",
        make: "Freightliner",
        model: "Cascadia 126",
        year: 2023,
        vin: "3AKJHHDR5PSNA4471",
        licensePlateNumber: "TX-48812",
        registrationNumber: "REG-9981244",
        registrationExpiry: anchor + 214 * DAY,
        externalId: "SAM-4471",
        fuelType: "Diesel",
        iftaQualified: true,
        ownershipType: "CompanyOwned",
        lessorName: "",
        leaseReference: "",
        leaseEndDate: null,
        createdAt: anchor - 2 * 365 * DAY,
        state: { abbreviation: "TX" },
        equipmentType: { id: "et_slp", code: "SLP", description: "Sleeper tractor" },
        equipmentManufacturer: { id: "em_fl", name: "Freightliner" },
        fleetCode: { id: "fc_dal", code: "DAL" },
        primaryWorker: rosa,
        secondaryWorker: sam,
        ownerWorker: null,
        lastKnownLocation: { id: "loc_dal", name: "Dallas yard" },
      },
      position: {
        latitude: 32.3932,
        longitude: -96.8461,
        headingDegrees: 172,
        speedMph: 61.4,
        engineState: "On",
        fuelPercent: 64,
        odometerMeters: 293_118_400,
        formattedLocation: "I-35E southbound near Waxahachie, TX",
        recordedAt: anchor - 95,
      },
      inspections: Array.from({ length: 9 }, (_, index) => ({
        id: `vinsp_101_${String(9 - index).padStart(2, "0")}`,
        inspectionType: index % 2 === 0 ? "preTrip" : "postTrip",
        safetyStatus: "safe",
        workerName: index % 3 === 2 ? "Samuel Okafor" : "Rosa Diaz",
        startedAt: anchor - 6 * HOUR - index * 14 * HOUR,
        location: ["Dallas, TX", "Oklahoma City, OK", "Wichita, KS", "Amarillo, TX"][index % 4],
        odometerMeters: 293_000_000 - index * 820_000,
        defectCount: index === 3 ? 1 : 0,
        unresolvedDefectCount: 0,
      })),
      documents: [
        ["Registration 2026.pdf", "application/pdf", 182_400],
        ["Annual inspection.pdf", "application/pdf", 344_120],
        ["IFTA decal.jpg", "image/jpeg", 96_200],
      ],
      shipments: 7,
    },
    tr_207: {
      tractor: {
        id: "tr_207",
        code: "T-207",
        status: "AtMaintenance",
        make: "Kenworth",
        model: "T680",
        year: 2020,
        vin: "1XKYD49X0LJ392207",
        licensePlateNumber: "OK-2207P",
        registrationNumber: "REG-2207761",
        registrationExpiry: anchor + 9 * DAY,
        externalId: "",
        fuelType: "Diesel",
        iftaQualified: true,
        ownershipType: "Leased",
        lessorName: "Penske Truck Leasing",
        leaseReference: "PTL-55-20914",
        leaseEndDate: anchor + 41 * DAY,
        createdAt: anchor - 3 * 365 * DAY,
        state: { abbreviation: "OK" },
        equipmentType: { id: "et_day", code: "DAY", description: "Day cab" },
        equipmentManufacturer: { id: "em_kw", name: "Kenworth" },
        fleetCode: { id: "fc_okc", code: "OKC" },
        primaryWorker: dee,
        secondaryWorker: null,
        ownerWorker: null,
        lastKnownLocation: { id: "loc_okc", name: "Rush Truck Center, Oklahoma City" },
      },
      position: {
        latitude: 35.4676,
        longitude: -97.5164,
        headingDegrees: 270,
        speedMph: 0,
        engineState: "Off",
        fuelPercent: 22,
        odometerMeters: 811_402_900,
        formattedLocation: "Rush Truck Center, 8700 W Reno Ave, Oklahoma City, OK 73127",
        recordedAt: anchor - 2 * DAY - 3 * HOUR,
      },
      inspections: [
        {
          id: "vinsp_207_02",
          inspectionType: "preTrip",
          safetyStatus: "unsafe",
          workerName: "Deandra Whitfield-Castellanos",
          startedAt: anchor - 2 * DAY - 5 * HOUR,
          location: "Oklahoma City, OK",
          odometerMeters: 811_400_000,
          defectCount: 3,
          unresolvedDefectCount: 2,
        },
        {
          id: "vinsp_207_01",
          inspectionType: "postTrip",
          safetyStatus: "safe",
          workerName: null,
          startedAt: anchor - 4 * DAY,
          location: null,
          odometerMeters: null,
          defectCount: 0,
          unresolvedDefectCount: 0,
        },
      ],
      documents: [["Lease agreement PTL-55-20914.pdf", "application/pdf", 512_000]],
      shipments: 3,
    },
    tr_318: {
      tractor: {
        id: "tr_318",
        code: "T-318",
        status: "Available",
        make: "",
        model: "",
        year: null,
        vin: "",
        licensePlateNumber: "",
        registrationNumber: "",
        registrationExpiry: null,
        externalId: "",
        fuelType: "Diesel",
        iftaQualified: false,
        ownershipType: "CompanyOwned",
        lessorName: "",
        leaseReference: "",
        leaseEndDate: null,
        createdAt: anchor - 2 * DAY,
        state: null,
        equipmentType: { id: "et_slp", code: "SLP", description: "Sleeper tractor" },
        equipmentManufacturer: null,
        fleetCode: null,
        primaryWorker: worker("wrk_new", "Jordan", "Lee"),
        secondaryWorker: null,
        ownerWorker: null,
        lastKnownLocation: null,
      },
      position: null,
      inspections: [],
      documents: [],
      shipments: 0,
    },
  };
}

/**
 * Three trailers between them drawing every state the card has: TL-501 sits at
 * the Dallas yard, inspected and registered; TL-622 is a leased reefer out of
 * service at a customer's dock with its annual inspection overdue and an
 * owner-operator's name on it; TL-700 is new with nothing on file.
 */
export function deskTrailers(anchor) {
  const base = {
    externalId: "",
    lessorName: "",
    leaseReference: "",
    leaseEndDate: null,
    ownerWorker: null,
  };
  return {
    trl_501: {
      trailer: {
        ...base,
        id: "trl_501",
        code: "TL-501",
        status: "Available",
        make: "Wabash",
        model: "DuraPlate 53'",
        year: 2022,
        vin: "1JJV532D8NL501501",
        licensePlateNumber: "TX-T5501",
        registrationNumber: "REG-T5501",
        registrationExpiry: anchor + 188 * DAY,
        lastInspectionDate: anchor - 97 * DAY,
        maxLoadWeight: 45000,
        externalId: "SKY-501",
        ownershipType: "CompanyOwned",
        createdAt: anchor - 3 * 365 * DAY,
        registrationState: { abbreviation: "TX" },
        equipmentType: { id: "et_dv", code: "DV53", description: "53' dry van" },
        equipmentManufacturer: { id: "em_wb", name: "Wabash" },
        fleetCode: { id: "fc_dal", code: "DAL" },
        lastKnownLocation: {
          id: "loc_dal",
          name: "Dallas yard",
          latitude: 32.7801,
          longitude: -96.8005,
        },
      },
      documents: [
        ["Annual inspection 2026.pdf", "application/pdf", 288_400],
        ["Registration TL-501.pdf", "application/pdf", 141_900],
      ],
      shipments: 6,
    },
    trl_622: {
      trailer: {
        ...base,
        id: "trl_622",
        code: "TL-622",
        status: "OutOfService",
        make: "Utility",
        model: "3000R Reefer",
        year: 2017,
        vin: "1UYVS2538HU622622",
        licensePlateNumber: "AR-R6220",
        registrationNumber: "REG-R6220",
        registrationExpiry: anchor - 6 * DAY,
        lastInspectionDate: anchor - 402 * DAY,
        maxLoadWeight: 43500,
        ownershipType: "OwnerOperator",
        ownerWorker: { id: "wrk_ana", firstName: "Ana", lastName: "Ruiz-Montenegro" },
        createdAt: anchor - 5 * 365 * DAY,
        registrationState: { abbreviation: "AR" },
        equipmentType: { id: "et_rf", code: "RF53", description: "53' reefer" },
        equipmentManufacturer: { id: "em_ut", name: "Utility Trailer Manufacturing" },
        fleetCode: { id: "fc_lit", code: "LIT" },
        lastKnownLocation: {
          id: "loc_tyson",
          name: "Tyson Foods Distribution Center, Dock 14, Springdale",
          latitude: 36.1867,
          longitude: -94.1288,
        },
      },
      documents: [["Reefer unit service record.pdf", "application/pdf", 402_000]],
      shipments: 2,
    },
    trl_700: {
      trailer: {
        ...base,
        id: "trl_700",
        code: "TL-700",
        status: "Available",
        make: "",
        model: "",
        year: null,
        vin: "",
        licensePlateNumber: "",
        registrationNumber: "",
        registrationExpiry: null,
        lastInspectionDate: null,
        maxLoadWeight: null,
        ownershipType: "CompanyOwned",
        createdAt: anchor - DAY,
        registrationState: null,
        equipmentType: { id: "et_fb", code: "FB48", description: "48' flatbed" },
        equipmentManufacturer: null,
        fleetCode: null,
        lastKnownLocation: null,
      },
      documents: [],
      shipments: 0,
    },
  };
}

const HOUR_MS = HOUR * 1000;

/**
 * Three workers between them drawing every state the card has: Rosa Diaz is
 * driving T-101 with most of her day left and a clean record; Ana
 * Ruiz-Montenegro, an owner-operator, is held off the board with an expired
 * medical card, a hazmat endorsement about to lapse, a missing port pass and a
 * driving violation; Jordan Lee started this week with nothing on file.
 */
export function deskWorkers(anchor) {
  const person = {
    externalId: "",
    assignmentBlocked: "",
    canBeAssigned: true,
    availableForDispatch: true,
  };
  const clocks = (drive, shift, cycle, rest) => ({
    driveRemainingMs: drive,
    driveLimitMs: 11 * HOUR_MS,
    shiftRemainingMs: shift,
    shiftLimitMs: 14 * HOUR_MS,
    cycleRemainingMs: cycle,
    cycleLimitMs: 70 * HOUR_MS,
    breakRemainingMs: rest,
    breakLimitMs: 8 * HOUR_MS,
  });
  const credential = (id, name, health, expiresAt, days, required = true) => ({
    health,
    daysUntilExpiry: days,
    required,
    credentialType: { id: `ct_${id}`, name, profileField: "" },
    credential: expiresAt === undefined ? null : { id: `cr_${id}`, expiresAt },
  });
  return {
    wrk_rosa: {
      worker: {
        ...person,
        id: "wrk_rosa",
        status: "Active",
        type: "Employee",
        driverType: "OTR",
        firstName: "Rosa",
        lastName: "Diaz",
        phoneNumber: "(214) 555-0148",
        email: "rosa.diaz@skyline-freight.example",
        city: "Mesquite",
        externalId: "EMP-1048",
        createdAt: anchor - 4 * 365 * DAY,
        state: { abbreviation: "TX" },
        fleetCode: { id: "fc_dal", code: "DAL" },
        manager: { id: "usr_mia", name: "Mia Chen" },
        profile: {
          licenseNumber: "TX40291187",
          cdlClass: "A",
          cdlRestrictions: "",
          endorsement: "N",
          licenseExpiry: anchor + 2 * 365 * DAY,
          hazmatExpiry: null,
          medicalCardExpiry: anchor + 214 * DAY,
          twicExpiry: anchor + 3 * 365 * DAY,
          physicalDueDate: anchor + 214 * DAY,
          mvrDueDate: anchor + 120 * DAY,
          hireDate: anchor - 4 * 365 * DAY,
          isQualified: true,
          disqualificationReason: "",
          licenseState: { abbreviation: "TX" },
        },
      },
      hos: {
        state: {
          dutyStatus: "driving",
          ...clocks(6.75 * HOUR_MS, 8.5 * HOUR_MS, 41.25 * HOUR_MS, 2.5 * HOUR_MS),
          currentTractorId: "tr_101",
          rulesetCycle: "USA 70 hour / 8 day",
          recordedAt: anchor - 3 * 60,
        },
        violations: [],
      },
      standing: { standing: "Good", concerns: [] },
      credentials: [
        credential("forklift", "Forklift operator", "Valid", anchor + 400 * DAY, 400),
        credential("defensive", "Defensive driving course", "Valid", anchor + 160 * DAY, 160),
      ],
      documents: [
        ["CDL Rosa Diaz.pdf", "application/pdf", 212_400],
        ["Medical examiner certificate.pdf", "application/pdf", 98_300],
      ],
      shipments: 7,
    },
    wrk_ana: {
      worker: {
        ...person,
        id: "wrk_ana",
        status: "Active",
        type: "Contractor",
        driverType: "Regional",
        firstName: "Ana",
        lastName: "Ruiz-Montenegro",
        phoneNumber: "(479) 555-0122",
        email: "ana@ruizmontenegro-trucking.example",
        city: "Springdale",
        assignmentBlocked: "Medical card expired; held until a new certificate is on file",
        canBeAssigned: false,
        availableForDispatch: false,
        createdAt: anchor - 6 * 365 * DAY,
        state: { abbreviation: "AR" },
        fleetCode: { id: "fc_lit", code: "LIT" },
        manager: { id: "usr_omar", name: "Omar Haddad" },
        profile: {
          licenseNumber: "AR99120455",
          cdlClass: "A",
          cdlRestrictions: "L",
          endorsement: "X",
          licenseExpiry: anchor + 41 * DAY,
          hazmatExpiry: anchor + 12 * DAY,
          medicalCardExpiry: anchor - 4 * DAY,
          twicExpiry: null,
          physicalDueDate: anchor - 4 * DAY,
          mvrDueDate: anchor + 18 * DAY,
          hireDate: anchor - 6 * 365 * DAY,
          isQualified: false,
          disqualificationReason: "Medical examiner's certificate expired",
          licenseState: { abbreviation: "AR" },
        },
      },
      hos: {
        state: {
          dutyStatus: "offDuty",
          ...clocks(0, 0.75 * HOUR_MS, 6.5 * HOUR_MS, 0),
          currentTractorId: null,
          rulesetCycle: "USA 70 hour / 8 day",
          recordedAt: anchor - 9 * HOUR,
        },
        violations: [
          {
            violationType: "SHIFT_DRIVING",
            description: "Drove 11 h 40 m in the shift, 40 m past the limit",
            durationMs: 40 * 60_000,
            violationStartAt: anchor - DAY - 4 * HOUR,
          },
          {
            violationType: "BREAK",
            description: "No 30-minute break before 8 hours of driving",
            durationMs: 22 * 60_000,
            violationStartAt: anchor - 5 * DAY,
          },
        ],
      },
      standing: {
        standing: "Blocked",
        concerns: [
          {
            severity: "Critical",
            code: "not_assignable",
            headline: "Cannot be assigned",
            detail: "The worker is off the dispatch board until this is lifted.",
          },
          {
            severity: "Critical",
            code: "credentials_expired",
            headline: "1 credential has expired",
            detail: "The worker is not qualified to drive until these are renewed.",
          },
          {
            severity: "Critical",
            code: "open_safety_events",
            headline: "2 safety events are open",
            detail: "Review them with the driver and close them out.",
          },
          {
            severity: "Warning",
            code: "training_due",
            headline: "Hazmat refresher due in 12 days",
            detail: "Book the course before the endorsement lapses.",
          },
        ],
      },
      credentials: [
        credential("port", "Port of Houston access", "Missing", undefined, null),
        credential("reefer", "Reefer unit operation", "ExpiringSoon", anchor + 9 * DAY, 9),
        credential("forklift", "Forklift operator", "Valid", anchor + 300 * DAY, 300),
      ],
      documents: [["Lease agreement 2026.pdf", "application/pdf", 520_100]],
      shipments: 3,
    },
    wrk_new: {
      worker: {
        ...person,
        id: "wrk_new",
        status: "Active",
        type: "Employee",
        driverType: "Local",
        firstName: "Jordan",
        lastName: "Lee",
        phoneNumber: "",
        email: "",
        city: "",
        createdAt: anchor - 2 * DAY,
        state: null,
        fleetCode: null,
        manager: null,
        profile: null,
      },
      hos: { state: null, violations: [] },
      standing: { standing: "Good", concerns: [] },
      credentials: [],
      documents: [],
      shipments: 0,
    },
  };
}

/**
 * Three locations between them drawing every state the card has: the Dallas
 * yard is the company's own terminal with a round fence, short waits and its
 * own detention terms; the Tyson distribution center in Springdale takes
 * trucks by appointment behind a drawn fence, keeps them long and has a run of
 * late deliveries; the Gate 7 lot was added this week with no address, no
 * coordinates and nothing recorded at it.
 */
export function deskLocations(anchor) {
  const category = (name, type, facilityType, flags) => ({
    id: `lc_${type.toLowerCase()}`,
    name,
    type,
    facilityType,
    hasSecureParking: false,
    requiresAppointment: false,
    allowsOvernight: false,
    hasRestroom: false,
    ...flags,
  });
  const policy = (id, name, overrides) => ({
    id,
    name,
    code: id.toUpperCase(),
    status: "Active",
    priority: 10,
    customerId: null,
    stopTypes: [],
    appointmentStopsOnly: false,
    billingFreeMinutes: 120,
    pickupFreeMinutes: null,
    deliveryFreeMinutes: null,
    effectiveStartDate: null,
    effectiveEndDate: null,
    ...overrides,
  });
  const failure = (id, number, type, status, lateMinutes, daysAgo, proNumber) => ({
    id,
    number,
    type,
    status,
    stopType: type.includes("Pickup") ? "Pickup" : "Delivery",
    lateMinutes,
    actualArrival: anchor - daysAgo * DAY,
    shipment: { id: `shp_${id}`, proNumber },
  });
  return {
    loc_dal: {
      location: {
        id: "loc_dal",
        status: "Active",
        code: "DAL-YD",
        name: "Dallas yard",
        description: "Drop yard and fuel island. Gate code changes monthly; ask dispatch.",
        addressLine1: "1801 Commerce St",
        addressLine2: "",
        city: "Dallas",
        postalCode: "75201",
        timezone: "America/Chicago",
        isGeocoded: true,
        latitude: 32.7801,
        longitude: -96.8005,
        geofenceType: "circle",
        geofenceRadiusMeters: 250,
        geofenceVertices: [],
        createdAt: anchor - 4 * 365 * DAY,
        state: { abbreviation: "TX" },
        locationCategory: category("Company terminal", "Terminal", null, {
          hasSecureParking: true,
          allowsOvernight: true,
          hasRestroom: true,
        }),
      },
      dwell: {
        locationId: "loc_dal",
        stopCount: 142,
        breachCount: 9,
        avgDwellMinutes: 48,
        medianDwellMinutes: 41,
        p90DwellMinutes: 96,
        billedAmount: "675.0000",
        waivedAmount: "0.0000",
        disputeCount: 0,
      },
      policies: [
        policy("dp_dal", "Dallas yard standard", {
          pickupFreeMinutes: 60,
          deliveryFreeMinutes: 90,
        }),
      ],
      failures: [
        failure("sf_dal1", "SF-2610-0031", "LatePickup", "Resolved", 25, 12, "S2610-1251"),
      ],
      shipments: 8,
    },
    loc_tyson: {
      location: {
        id: "loc_tyson",
        status: "Active",
        code: "TYS-SPD",
        name: "Tyson Foods Distribution Center, Springdale",
        description: "Receiving closes at 14:00. Lumper fees paid at the window.",
        addressLine1: "2200 W Don Tyson Pkwy",
        addressLine2: "Dock 14",
        city: "Springdale",
        postalCode: "72762",
        timezone: "America/Chicago",
        isGeocoded: true,
        latitude: 36.1867,
        longitude: -94.1288,
        geofenceType: "draw",
        geofenceRadiusMeters: null,
        geofenceVertices: [
          { latitude: 36.1884, longitude: -94.1315 },
          { latitude: 36.1886, longitude: -94.1262 },
          { latitude: 36.1851, longitude: -94.1258 },
          { latitude: 36.1849, longitude: -94.1291 },
          { latitude: 36.1862, longitude: -94.1318 },
        ],
        createdAt: anchor - 6 * 365 * DAY,
        state: { abbreviation: "AR" },
        locationCategory: category("Customer DC", "DistributionCenter", "ColdStorage", {
          requiresAppointment: true,
          hasRestroom: true,
        }),
      },
      dwell: {
        locationId: "loc_tyson",
        stopCount: 64,
        breachCount: 27,
        avgDwellMinutes: 171,
        medianDwellMinutes: 148,
        p90DwellMinutes: 312,
        billedAmount: "4860.0000",
        waivedAmount: "520.0000",
        disputeCount: 3,
      },
      policies: [
        policy("dp_tys_app", "Tyson appointment loads", {
          customerId: "cus_tyson",
          stopTypes: ["Delivery"],
          appointmentStopsOnly: true,
          billingFreeMinutes: 90,
          priority: 30,
        }),
        policy("dp_tys", "Tyson Springdale", { billingFreeMinutes: 120, priority: 20 }),
        policy("dp_tys_old", "Tyson Springdale (2025)", {
          status: "Inactive",
          priority: 5,
          effectiveEndDate: anchor - 120 * DAY,
        }),
      ],
      policyTotal: 4,
      failures: [
        failure("sf_tys1", "SF-2610-0058", "LateDelivery", "Open", 95, 1, "S2610-1366"),
        failure("sf_tys2", "SF-2610-0052", "AppointmentMissed", "Reviewed", 140, 4, "S2610-1342"),
        failure("sf_tys3", "SF-2610-0047", "LateDelivery", "Open", 55, 9, "S2610-1307"),
        failure("sf_tys4", "SF-2609-0391", "LateDelivery", "Resolved", 35, 22, "S2609-1180"),
        failure("sf_tys5", "SF-2609-0377", "MissedDelivery", "Voided", 0, 30, "S2609-1132"),
        failure("sf_tys6", "SF-2609-0350", "LateDelivery", "Resolved", 70, 41, "S2609-1077"),
      ],
      shipments: 5,
    },
    loc_new: {
      location: {
        id: "loc_new",
        status: "Active",
        code: "G7-LOT",
        name: "Gate 7 overflow lot",
        description: "",
        addressLine1: "",
        addressLine2: "",
        city: "",
        postalCode: "",
        timezone: "",
        isGeocoded: false,
        latitude: null,
        longitude: null,
        geofenceType: "auto",
        geofenceRadiusMeters: null,
        geofenceVertices: [],
        createdAt: anchor - 2 * DAY,
        state: null,
        locationCategory: null,
      },
      dwell: null,
      policies: [],
      failures: [],
      shipments: 0,
    },
  };
}

/**
 * Three carriers between them drawing every state the card has: Red River
 * Freight is qualified, vetted and clean with its insurance current and a run
 * of paid settlements; Lone Star Haulers is disqualified, its cargo policy
 * lapsed, its broker authority revoked and two BASICs past their thresholds;
 * Northgate Transport was added this week, not yet vetted, with nothing on
 * file.
 */
export function deskCarriers(anchor) {
  const base = {
    dbaName: null,
    carrierType: "Common",
    safetyRating: "Satisfactory",
    disqualifiedReason: null,
    w9OnFile: true,
    paymentMethod: "ACHManual",
    paymentTermDays: 30,
    remitToName: null,
    addressLine2: null,
    externalId: null,
    notes: null,
    intelReviewRequired: false,
  };
  const policy = (id, policyType, providerName, coverageAmount, daysLeft, isVerified = true) => ({
    id,
    policyType,
    policyNumber: `${policyType.slice(0, 2).toUpperCase()}-${id.slice(-4)}`,
    providerName,
    coverageAmount,
    effectiveDate: anchor + (daysLeft - 365) * DAY,
    expirationDate: anchor + daysLeft * DAY,
    isVerified,
  });
  const grant = (status, flags = {}) => ({
    status,
    pending: false,
    revocationPending: false,
    grantedAt: anchor - 6 * 365 * DAY,
    ...flags,
  });
  const basic = (name, percentile, threshold, alert = false) => ({
    basic: name,
    measure: percentile === null ? null : percentile / 18,
    percentile,
    threshold,
    alert,
  });
  const settlement = (id, number, status, endDaysAgo, netPayableMinor, shipmentCount) => ({
    id,
    settlementNumber: number,
    status,
    periodStart: anchor - (endDaysAgo + 14) * DAY,
    periodEnd: anchor - endDaysAgo * DAY,
    payDate: anchor - (endDaysAgo - 10) * DAY,
    netPayableMinor,
    shipmentCount,
    currencyCode: "USD",
  });
  return {
    car_redriver: {
      carrier: {
        ...base,
        id: "car_redriver",
        status: "Active",
        code: "RRF",
        name: "Red River Freight",
        dbaName: "Red River Logistics",
        dotNumber: "3141592",
        mcNumber: "MC-271828",
        scac: "RRFT",
        complianceStatus: "Qualified",
        qualifiedAt: anchor - 210 * DAY,
        addressLine1: "400 Industrial Blvd",
        city: "Shreveport",
        postalCode: "71101",
        phone: "(318) 555-0190",
        email: "dispatch@redriver.example",
        intelRiskLevel: "Low",
        createdAt: anchor - 3 * 365 * DAY,
        state: { abbreviation: "LA" },
        contacts: [
          {
            id: "cc_rr1",
            name: "Marcus Bell",
            title: "Dispatch manager",
            email: "marcus@redriver.example",
            phone: "(318) 555-0191",
            isPrimary: true,
            receivesRateConfirmations: true,
          },
          {
            id: "cc_rr2",
            name: "Tanya Okoro",
            title: "Billing",
            email: "billing@redriver.example",
            phone: null,
            isPrimary: false,
            receivesRateConfirmations: false,
          },
        ],
        insurancePolicies: [
          policy("pol_rr_auto", "AutoLiability", "Great West Casualty", "1000000", 241),
          policy("pol_rr_cargo", "CargoLiability", "Great West Casualty", "100000", 241),
          policy("pol_rr_gl", "GeneralLiability", "Travelers", "2000000", 118),
        ],
      },
      eligibility: { blockers: [], warnings: [], advisories: [], findings: [] },
      intel: {
        id: "snap_rr",
        provider: "CarrierOK",
        notFound: false,
        riskLevel: "Low",
        fetchedAt: anchor - 5 * HOUR,
        effectiveAsOf: anchor - 5 * HOUR,
        profile: {
          authority: {
            common: grant("Active"),
            contract: grant("Active"),
            broker: grant("None"),
            totalRevocations: 0,
            lastRevocationAt: null,
          },
          safety: {
            rating: "Satisfactory",
            ratingDate: anchor - 800 * DAY,
            outOfServiceOrder: false,
            outOfServiceAt: null,
          },
          basics: [
            basic("UnsafeDriving", 22, 65),
            basic("HOSCompliance", 38, 65),
            basic("DriverFitness", null, 80),
            basic("VehicleMaintenance", 47, 80),
          ],
        },
      },
      settlements: [
        settlement("cs_rr1", "CS-2610-0042", "Approved", 1, 1_245_050, 4),
        settlement("cs_rr2", "CS-2609-0031", "Paid", 15, 1_908_200, 6),
        settlement("cs_rr3", "CS-2609-0017", "Paid", 29, 864_000, 3),
      ],
      shipments: 6,
    },
    car_lonestar: {
      carrier: {
        ...base,
        id: "car_lonestar",
        status: "Active",
        code: "LSH",
        name: "Lone Star Haulers",
        dotNumber: "2718281",
        mcNumber: "MC-161803",
        scac: "LSHA",
        complianceStatus: "Disqualified",
        disqualifiedReason: "Cargo insurance lapsed",
        safetyRating: "Conditional",
        qualifiedAt: anchor - 500 * DAY,
        addressLine1: "9100 Airline Dr",
        city: "Houston",
        postalCode: "77037",
        phone: "(713) 555-0144",
        email: null,
        intelRiskLevel: "High",
        intelReviewRequired: true,
        createdAt: anchor - 2 * 365 * DAY,
        state: { abbreviation: "TX" },
        contacts: [
          {
            id: "cc_ls1",
            name: "Ray Gutierrez",
            title: "Owner",
            email: null,
            phone: "(713) 555-0145",
            isPrimary: true,
            receivesRateConfirmations: true,
          },
        ],
        insurancePolicies: [
          policy("pol_ls_cargo", "CargoLiability", "Prime Insurance", "100000", -3),
          policy("pol_ls_auto", "AutoLiability", "Prime Insurance", "750000", 12),
          policy("pol_ls_gl", "GeneralLiability", "Hiscox", "1000000", 190, false),
        ],
      },
      eligibility: {
        blockers: ["Carrier compliance status is Disqualified", "CargoLiability insurance expired"],
        warnings: ["AutoLiability insurance expires within 30 days"],
        advisories: ["Broker authority was revoked"],
        findings: [
          {
            code: "record.compliance_status",
            source: "Record",
            severity: "Blocker",
            message: "Carrier compliance status is Disqualified",
            requiresOverride: false,
          },
          {
            code: "record.insurance_expired",
            source: "Record",
            severity: "Blocker",
            message: "Cargo liability insurance expired 3 days ago",
            requiresOverride: false,
          },
          {
            code: "record.insurance_expiring",
            source: "Record",
            severity: "Warning",
            message: "Auto liability insurance expires in 12 days",
            requiresOverride: true,
          },
          {
            code: "intel.basic_alert",
            source: "Intelligence",
            severity: "Warning",
            message: "Hours-of-Service and Vehicle Maintenance BASICs are in alert",
            requiresOverride: true,
          },
          {
            code: "intel.authority_revoked",
            source: "Intelligence",
            severity: "Advisory",
            message: "Broker authority was revoked last year",
            requiresOverride: false,
          },
        ],
      },
      intel: {
        id: "snap_ls",
        provider: "CarrierOK",
        notFound: false,
        riskLevel: "High",
        fetchedAt: anchor - 2 * DAY,
        effectiveAsOf: anchor - 2 * DAY,
        profile: {
          authority: {
            common: grant("Active", { revocationPending: true }),
            contract: grant("Inactive"),
            broker: grant("Revoked"),
            totalRevocations: 2,
            lastRevocationAt: anchor - 300 * DAY,
          },
          safety: {
            rating: "Conditional",
            ratingDate: anchor - 120 * DAY,
            outOfServiceOrder: false,
            outOfServiceAt: null,
          },
          basics: [
            basic("UnsafeDriving", 58, 65),
            basic("HOSCompliance", 83, 65, true),
            basic("VehicleMaintenance", 91, 80, true),
            basic("CrashIndicator", 44, 65),
          ],
        },
      },
      settlements: [settlement("cs_ls1", "CS-2608-0102", "Voided", 55, 412_500, 2)],
      shipments: 2,
    },
    car_new: {
      carrier: {
        ...base,
        id: "car_new",
        status: "Active",
        code: "NGT",
        name: "Northgate Transport",
        dotNumber: "4012345",
        mcNumber: null,
        scac: null,
        complianceStatus: "Pending",
        safetyRating: "NotRated",
        qualifiedAt: null,
        w9OnFile: false,
        addressLine1: null,
        city: null,
        postalCode: null,
        phone: null,
        email: null,
        intelRiskLevel: null,
        createdAt: anchor - DAY,
        state: null,
        contacts: [],
        insurancePolicies: [],
      },
      eligibility: {
        blockers: ["Carrier compliance status is Pending"],
        warnings: [],
        advisories: [],
        findings: [
          {
            code: "record.compliance_status",
            source: "Record",
            severity: "Blocker",
            message: "Carrier compliance status is Pending",
            requiresOverride: false,
          },
          {
            code: "record.insurance_missing",
            source: "Record",
            severity: "Blocker",
            message: "No auto liability or cargo liability policy is on file",
            requiresOverride: false,
          },
        ],
      },
      intel: null,
      settlements: [],
      shipments: 0,
    },
  };
}

/**
 * Three orders between them drawing every state the card has: Acme's order is
 * on the road with one leg delivered and a lumper charge waiting to be
 * invoiced; Tyson's is completed with a detention charge split between two
 * payers and nothing invoiced yet, so billing is due; the third is a draft
 * taken this morning, not yet priced, with nothing on it.
 */
export function deskOrders(anchor) {
  const charge = (id, description, amount, invoicedDaysAgo, allocations = 0) => ({
    id,
    description,
    amount,
    invoiceId: invoicedDaysAgo === null ? null : `inv_${id}`,
    invoicedAt: invoicedDaysAgo === null ? null : anchor - invoicedDaysAgo * DAY,
    allocations: Array.from({ length: allocations }, (_, index) => ({ id: `${id}_a${index}` })),
  });
  const entry = (id, operation, hoursAgo, name, changes = null, comment = null) => ({
    id,
    operation,
    timestamp: anchor - hoursAgo * HOUR,
    changes,
    comment,
    principalType: name ? "session_user" : "system",
    user: name ? { id: `usr_${id}`, name } : null,
  });
  return {
    ord_acme: {
      order: {
        id: "ord_acme",
        status: "InProgress",
        orderNumber: "ORD-2610-0118",
        poNumber: "PO-88213",
        bol: "BOL-55120",
        currencyCode: "USD",
        quotedAmount: "4000.0000",
        baseAmount: "3800.0000",
        totalAmount: "4285.0000",
        createdAt: anchor - 3 * DAY,
        updatedAt: anchor - 2 * HOUR,
        customer: { id: "cus_acme", code: "ACME", name: "Acme Foods" },
        charges: [
          charge("och_a1", "Fuel surcharge", "400.0000", 1),
          charge("och_a2", "Lumper at Dallas DC", "85.0000", null),
        ],
      },
      activity: [
        entry("ae_a1", "update", 2, "Mia Chen", {
          status: { from: "Confirmed", to: "InProgress" },
        }),
        entry("ae_a2", "update", 20, "Mia Chen", {
          poNumber: { from: null, to: "PO-88213" },
          bol: { from: null, to: "BOL-55120" },
        }),
        entry("ae_a3", "update", 30, null, { totalAmount: { from: "4200", to: "4285" } }),
        entry("ae_a4", "approve", 50, "Omar Haddad", null, "Rate confirmed with Acme"),
        entry("ae_a5", "create", 72, "Mia Chen"),
        entry("ae_a6", "update", 72, "Mia Chen", { quotedAmount: { from: null, to: "4000" } }),
      ],
      shipments: 2,
    },
    ord_tyson: {
      order: {
        id: "ord_tyson",
        status: "Completed",
        orderNumber: "ORD-2609-0977",
        poNumber: "TY-4400912",
        bol: null,
        currencyCode: "USD",
        quotedAmount: "6100.0000",
        baseAmount: "6100.0000",
        totalAmount: "7365.0000",
        createdAt: anchor - 21 * DAY,
        updatedAt: anchor - DAY,
        customer: { id: "cus_tyson", code: "TYSON", name: "Tyson Foods" },
        charges: [
          charge("och_t1", "Detention at Springdale, 4 h 15 m", "1020.0000", null, 2),
          charge("och_t2", "Reefer fuel", "245.0000", null),
        ],
      },
      activity: [
        entry("ae_t1", "update", 24, "Ray Gutierrez", {
          status: { from: "InProgress", to: "Completed" },
        }),
        entry("ae_t2", "update", 30, "Ray Gutierrez", {
          charges: { from: null, to: "added" },
          totalAmount: { from: "6345", to: "7365" },
          baseAmount: { from: "6100", to: "6100" },
          currencyCode: { from: "USD", to: "USD" },
        }),
      ],
      shipments: 3,
    },
    ord_draft: {
      order: {
        id: "ord_draft",
        status: "Draft",
        orderNumber: "ORD-2610-0131",
        poNumber: null,
        bol: null,
        currencyCode: "USD",
        quotedAmount: null,
        baseAmount: null,
        totalAmount: null,
        createdAt: anchor - 2 * HOUR,
        updatedAt: anchor - 2 * HOUR,
        customer: { id: "cus_globex", code: "GLOBEX", name: "Globex Corporation" },
        charges: [],
      },
      activity: [],
      shipments: 0,
    },
  };
}

/** A document filed on a tractor, as the documents API returns it. */
export function deskDocument(resourceType, resourceId, [name, fileType, fileSize], index, anchor) {
  return {
    id: `doc_${resourceId}_${index}`,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    lineageId: `doc_${resourceId}_${index}`,
    versionNumber: 1,
    isCurrentVersion: true,
    fileName: name,
    originalName: name,
    fileSize,
    fileType,
    storagePath: `mock/${resourceId}/${index}`,
    status: "Active",
    description: null,
    resourceId,
    resourceType,
    processingProfile: "none",
    isPublic: false,
    uploadedById: "usr_mock",
    approvedAt: index === 0 ? anchor - 30 * DAY : null,
    previewStatus: "Unsupported",
    contentStatus: "Indexed",
    hasExtractedText: false,
    shipmentDraftStatus: "Unavailable",
    documentType: null,
    version: 1,
    createdAt: anchor - (index + 1) * 20 * DAY,
    updatedAt: anchor - (index + 1) * 20 * DAY,
  };
}

/** Who asked about which tractor, and what the agent answered, for each card's conversation. */
const TURNS = {
  "tractor-t-101": [
    "Where is T-101 right now, and is it due for anything?",
    "T-101 is rolling south on I-35E near Waxahachie at 61 mph with Rosa Diaz driving; its registration runs through May and its last pre-trip was clean. [Tractor T-101](artifact:art_tractor-t-101) has the rest.",
  ],
  "tractor-t-207": [
    "Why is T-207 sitting?",
    "T-207 is at maintenance at Rush Truck Center in Oklahoma City: its last pre-trip found two defects still open, and its registration expires in nine days. [Tractor T-207](artifact:art_tractor-t-207) lays it out.",
  ],
  "tractor-t-999": [
    "Show me T-999.",
    "I looked up T-999 a moment ago, but it has since been removed. [Tractor T-999](artifact:art_tractor-t-999) is what I found then.",
  ],
  "trailer-tl-501": [
    "Where is TL-501 and is it due for inspection?",
    "TL-501 is at the Dallas yard, empty and available; its annual inspection is good until January and its registration runs through April. [Trailer TL-501](artifact:art_trailer-tl-501) has the details.",
  ],
  "trailer-tl-622": [
    "Can we use TL-622 tomorrow?",
    "Not as it stands: TL-622 is out of service at the Tyson dock in Springdale, its annual inspection is overdue and its registration expired six days ago. [Trailer TL-622](artifact:art_trailer-tl-622) lists what needs clearing.",
  ],
  "trailer-tl-700": [
    "Show me TL-700.",
    "TL-700 was added yesterday and nothing is on file for it yet. [Trailer TL-700](artifact:art_trailer-tl-700) shows what is there.",
  ],
  "trailer-tl-999": [
    "Show me TL-999.",
    "TL-999 has been removed since I looked it up. [Trailer TL-999](artifact:art_trailer-tl-999) is what I found then.",
  ],
  "worker-rosa": [
    "Can Rosa take another load today?",
    "Yes: Rosa Diaz is driving T-101 with 6 h 45 m of drive time left and 41 h on her cycle, and nothing on her record needs attention. [Rosa Diaz](artifact:art_worker-rosa) has her clocks and papers.",
  ],
  "worker-ana": [
    "Why is Ana off the board?",
    "Ana Ruiz-Montenegro's medical card expired four days ago, so she is held until a new certificate is filed; her hazmat endorsement also lapses in 12 days and she has two open hours-of-service violations. [Ana Ruiz-Montenegro](artifact:art_worker-ana) lists all of it.",
  ],
  "worker-jordan": [
    "Pull up Jordan Lee.",
    "Jordan Lee started this week and has no driver profile, hours or loads on file yet. [Jordan Lee](artifact:art_worker-jordan) shows what is there.",
  ],
  "worker-gone": [
    "Show me Casey Morgan.",
    "Casey Morgan's record has been removed since I looked it up. [Casey Morgan](artifact:art_worker-gone) is what I found then.",
  ],
  "location-dallas": [
    "How long do trucks wait at the Dallas yard?",
    "Not long: over the last 90 days trucks spent 48 minutes on average at the Dallas yard, and only 9 of 142 stops ran past free time. [Dallas yard](artifact:art_location-dallas) has the figures and its detention terms.",
  ],
  "location-tyson": [
    "Is the Tyson DC in Springdale a problem?",
    "It is the slowest dock we serve: trucks wait 2 h 51 m on average and 27 of 64 stops ran past free time, with three late deliveries open. [Tyson Foods Distribution Center, Springdale](artifact:art_location-tyson) lays out the waits, the terms and the failures.",
  ],
  "location-new": [
    "Pull up the Gate 7 overflow lot.",
    "The Gate 7 overflow lot was added two days ago and has no address or coordinates yet, and no load has stopped there. [Gate 7 overflow lot](artifact:art_location-new) shows what is on file.",
  ],
  "location-gone": [
    "Show me the old Fort Worth yard.",
    "The old Fort Worth yard has been removed since I looked it up. [Old Fort Worth yard](artifact:art_location-gone) is what I found then.",
  ],
  "carrier-redriver": [
    "Can we tender the Shreveport load to Red River?",
    "Yes: Red River Freight is qualified, its intelligence is clean and every policy runs past January. [Red River Freight](artifact:art_carrier-redriver) has their authority, BASICs and recent settlements.",
  ],
  "carrier-lonestar": [
    "Why was Lone Star Haulers rejected?",
    "Lone Star Haulers is disqualified: its cargo policy lapsed three days ago, its auto liability runs out in 12 days and two BASICs are in alert. [Lone Star Haulers](artifact:art_carrier-lonestar) lists everything standing in the way.",
  ],
  "carrier-new": [
    "Pull up Northgate Transport.",
    "Northgate Transport was added yesterday and is still pending qualification, with no insurance or contacts on file and no vetting yet. [Northgate Transport](artifact:art_carrier-new) shows what is there.",
  ],
  "carrier-gone": [
    "Show me Prairie Line Carriers.",
    "Prairie Line Carriers has been removed since I looked it up. [Prairie Line Carriers](artifact:art_carrier-gone) is what I found then.",
  ],
  "order-acme": [
    "Where is Acme's order PO-88213?",
    "Acme's order ORD-2610-0118 is in progress: one of its two legs has delivered and the other is on the road. The lumper charge from Dallas still has to be invoiced. [Order ORD-2610-0118](artifact:art_order-acme) has the legs, charges and history.",
  ],
  "order-tyson": [
    "Is the Tyson order ready to bill?",
    "Yes: ORD-2609-0977 completed yesterday and none of its $1,265.00 in charges has been invoiced yet; the detention is split between Tyson and the consignee. [Order ORD-2609-0977](artifact:art_order-tyson) lays it out.",
  ],
  "order-draft": [
    "Pull up the Globex order from this morning.",
    "ORD-2610-0131 for Globex is still a draft, not priced and with no legs yet. [Order ORD-2610-0131](artifact:art_order-draft) shows what is there.",
  ],
  "order-gone": [
    "Show me ORD-2608-0001.",
    "That order has been removed since I looked it up. [Order ORD-2608-0001](artifact:art_order-gone) is what I found then.",
  ],
  "tractor-t-318": [
    "Pull up T-318.",
    "T-318 joined the fleet two days ago and has nothing recorded against it yet: no GPS report, no inspections, no loads. [Tractor T-318](artifact:art_tractor-t-318) shows what is on file.",
  ],
};

/** The conversation a card's artifact sits in: the question and the reply that names it. */
export function deskThread(anchor, slug) {
  const [question, reply] = TURNS[slug] ?? TURNS["tractor-t-101"];
  const base = { threadId: DESK_THREAD_ID, kind: "Message", toolFailed: false, refused: false };
  return {
    thread: {
      id: DESK_THREAD_ID,
      businessUnitId: "bu_mock",
      organizationId: "org_mock",
      userId: "usr_mock",
      agentDefinitionId: "agd_dispatch",
      title: question,
      status: "Active",
      lastMessageAt: anchor - 4 * 60,
      lastReadAt: anchor,
      origin: "Desk",
      pinned: false,
      canContinue: true,
      version: 1,
      createdAt: anchor - 6 * 60,
      updatedAt: anchor - 4 * 60,
    },
    messages: [
      {
        ...base,
        id: "msg_q",
        sequence: 1,
        role: "User",
        content: question,
        createdAt: anchor - 6 * 60,
      },
      {
        ...base,
        id: "msg_desk_cards",
        sequence: 2,
        role: "Assistant",
        content: reply,
        createdAt: anchor - 5 * 60,
      },
    ],
  };
}
