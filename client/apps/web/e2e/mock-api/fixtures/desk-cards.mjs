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
  "tractor-t-101": ["Where is T-101 right now, and is it due for anything?", "T-101 is rolling south on I-35E near Waxahachie at 61 mph with Rosa Diaz driving; its registration runs through May and its last pre-trip was clean. [Tractor T-101](artifact:art_tractor-t-101) has the rest."],
  "tractor-t-207": ["Why is T-207 sitting?", "T-207 is at maintenance at Rush Truck Center in Oklahoma City: its last pre-trip found two defects still open, and its registration expires in nine days. [Tractor T-207](artifact:art_tractor-t-207) lays it out."],
  "tractor-t-999": ["Show me T-999.", "I looked up T-999 a moment ago, but it has since been removed. [Tractor T-999](artifact:art_tractor-t-999) is what I found then."],
  "tractor-t-318": ["Pull up T-318.", "T-318 joined the fleet two days ago and has nothing recorded against it yet: no GPS report, no inspections, no loads. [Tractor T-318](artifact:art_tractor-t-318) shows what is on file."],
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
      { ...base, id: "msg_q", sequence: 1, role: "User", content: question, createdAt: anchor - 6 * 60 },
      { ...base, id: "msg_desk_cards", sequence: 2, role: "Assistant", content: reply, createdAt: anchor - 5 * 60 },
    ],
  };
}
