// The detention charges the Desk cards thread names, as the DeskDetentionCard
// query answers for them, each timed against the mock's now: a Kroger delivery
// waiting on approval over its approval line; a pickup still on the clock with
// its notice overdue; a short stop that left inside its free time; a charge the
// customer disputes; and one waived for goodwill before anyone approved it.
// The trace's words are the engine's own (detentionservice.calculator).

const MINUTE = 60;
const HOUR = 3_600;
const DAY = 86_400;

const user = (id, name) => ({ id, name });

function snapshot(overrides) {
  return {
    policyId: "dtp_kroger",
    policyCode: "KRG-DC",
    policyName: "Kroger DC standard",
    policyVersion: 3,
    specificityScore: 6,
    priority: 20,
    clockStartBasis: "LaterOfArrivalOrAppointment",
    lateArrivalRule: "ClockFromAppointment",
    lateArrivalGraceMinutes: 0,
    freeMinutes: 120,
    payFreeMinutes: 180,
    minimumBillableMinutes: 15,
    billingIncrementMinutes: 15,
    roundingMode: "Up",
    rateSource: "Tiers",
    accessorialChargeId: "",
    accessorialCode: "DET",
    flatRate: "0",
    flatRateUnit: "Hour",
    tiers: [{ fromMinute: 0, toMinute: null, rate: "75", rateUnit: "Hour", label: "Tier 1" }],
    maxBillableMinutesPerStop: null,
    maxChargePerStop: null,
    maxChargePerDay: null,
    maxChargePerShipment: null,
    dayBoundaryMode: "PerStop",
    convertToLayoverAtMinutes: null,
    layoverAccessorialChargeId: null,
    notificationRequirement: "Required",
    notificationLeadMinutes: 30,
    notificationDeadlineMinutes: 0,
    unnotifiedBehavior: "Flag",
    requireApprovalOverAmount: "250",
    autoApproveUnderAmount: null,
    currency: "USD",
    capturedAt: 0,
    ...overrides,
  };
}

function minutesText(minutes) {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return `${rest}m`;
  return rest === 0 ? `${hours}h` : `${hours}h ${rest}m`;
}

/** The trace the engine writes for a stop that ran past its free time and was billed by the hour. */
function billedTrace({
  policy,
  code,
  clockStart,
  clockStop,
  dwell,
  free,
  billable,
  rounded,
  amount,
  notice,
}) {
  const steps = [
    {
      kind: "PolicyResolved",
      label: "Policy applied",
      detail: `${policy} (${code})`,
      term: "specificity 6, priority 20",
      isReduction: false,
    },
    {
      kind: "ClockStart",
      label: "Clock started",
      detail: "Later of arrival or appointment start",
      term: "Clock start basis: LaterOfArrivalOrAppointment",
      timestamp: clockStart,
      isReduction: false,
    },
    {
      kind: "ClockStop",
      label: "Clock stopped",
      detail: "Departure recorded",
      term: "Clock stop basis: Departure",
      timestamp: clockStop,
      isReduction: false,
    },
    {
      kind: "RawDwell",
      label: "Total time on site",
      detail: minutesText(dwell),
      minutes: dwell,
      isReduction: false,
    },
    {
      kind: "FreeTime",
      label: "Free time deducted",
      detail: minutesText(free),
      term: `${free} min contractual free time`,
      minutes: free,
      isReduction: true,
    },
  ];
  if (rounded > 0 && rounded !== billable) {
    steps.push({
      kind: "Rounding",
      label: `Rounded ${minutesText(billable)} to ${minutesText(rounded)}`,
      detail: "Rounded up to the next 15 minutes",
      term: "Up to 15 min increments",
      minutes: rounded,
      isReduction: false,
    });
  }
  if (rounded > 0) {
    steps.push({
      kind: "TierApplied",
      label: "Tier 1",
      detail: `${minutesText(rounded)} at $75.00/hr`,
      term: "From minute 0",
      amount,
      isReduction: false,
    });
  }
  if (notice) {
    steps.push({
      kind: "NotificationGate",
      label: "Notice requirement satisfied",
      detail: "Customer was notified within the contractual window",
      term: "Notification: Required",
      isReduction: false,
    });
  }
  if (Number(amount) > 250) {
    steps.push({
      kind: "NotificationGate",
      label: "Approval required",
      detail: `${Number(amount).toFixed(2)} exceeds the 250.00 approval threshold`,
      term: "Require approval over 250.00",
      isReduction: false,
    });
  }
  steps.push({
    kind: "Final",
    label: "Billable detention",
    detail: `${minutesText(rounded)} billable of ${minutesText(dwell)} on site`,
    amount,
    isReduction: false,
  });
  return { engineVersion: "1.0.0", computedAt: clockStop + MINUTE, deterministic: true, steps };
}

function occurrence(overrides) {
  return {
    stopId: "stp_mock",
    stopType: "Delivery",
    scheduleType: "Appointment",
    appointmentEnd: null,
    noticeDeadlineAt: null,
    isOpen: false,
    arrivedLate: false,
    lateByMinutes: 0,
    capApplied: "None",
    convertedToLayover: false,
    currency: "USD",
    suppressedByGate: false,
    requiresApproval: false,
    approvedAt: null,
    waiverReason: null,
    waiverNote: "",
    waivedAt: null,
    waivedAmount: "0",
    disputeNote: "",
    disputedAt: null,
    approvedBy: null,
    waivedBy: null,
    ...overrides,
  };
}

function factor(key, label, earned, possible, remedy = "") {
  return { key, label, earned, possible, detail: "", remedy: earned < possible ? remedy : "" };
}

function notice(id, at, recipients) {
  return {
    id,
    kind: "Warning",
    channel: "Email",
    deliveryStatus: "Delivered",
    recipients,
    sentAt: at,
    scheduledFor: at,
    failureReason: "",
    satisfiesRequirement: true,
    wasAutomatic: true,
  };
}

/** The detention charges, keyed by ID, each as the card's query answers it. */
export function deskDetentions(anchor) {
  const kroger = anchor - 8 * HOUR;
  const krogerStop = kroger + 7 * HOUR;
  const lineage = anchor - 158 * MINUTE;
  const hd = Math.floor(anchor / DAY) * DAY - 5 * DAY + 13 * HOUR;
  const quick = anchor - 6 * HOUR;
  const disputed = Math.floor(anchor / DAY) * DAY - 9 * DAY + 7 * HOUR;

  const krogerLocation = {
    id: "loc_kroger_atl",
    city: "Forest Park",
    state: { id: "st_ga", abbreviation: "GA" },
  };

  return {
    dto_kroger: {
      occurrence: occurrence({
        id: "dto_kroger",
        customerId: "cus_kroger",
        locationId: "loc_kroger_atl",
        policySnapshot: snapshot({ capturedAt: krogerStop + MINUTE }),
        calculationTrace: billedTrace({
          policy: "Kroger DC standard",
          code: "KRG-DC",
          clockStart: kroger + 42 * MINUTE,
          clockStop: krogerStop,
          dwell: 378,
          free: 120,
          billable: 258,
          rounded: 270,
          amount: "337.5",
          notice: true,
        }),
        appointmentStart: kroger,
        appointmentEnd: kroger + HOUR,
        arrivedAt: kroger + 42 * MINUTE,
        departedAt: krogerStop,
        clockStartAt: kroger + 42 * MINUTE,
        clockStopAt: krogerStop,
        freeTimeExpiresAt: kroger + 162 * MINUTE,
        noticeDueAt: kroger + 132 * MINUTE,
        noticeDeadlineAt: kroger + 162 * MINUTE,
        freeMinutesGranted: 120,
        rawDwellMinutes: 378,
        billableMinutes: 258,
        roundedMinutes: 270,
        billableAmount: "337.5",
        driverPayMinutes: 195,
        driverPayAmount: "130",
        netMargin: "207.5",
        status: "Pending",
        notificationStatus: "Sent",
        noticeSentAt: kroger + 129 * MINUTE,
        requiresApproval: true,
        locationName: "Kroger Atlanta DC",
        customerName: "The Kroger Co.",
        createdAt: kroger + 42 * MINUTE,
        updatedAt: krogerStop + MINUTE,
        location: krogerLocation,
      }),
      evidence: [
        {
          id: "dte_k1",
          kind: "Arrival",
          source: "Telematics",
          summary: "Truck 2207 inside the Kroger Atlanta geofence",
          observedAt: kroger + 42 * MINUTE,
        },
        {
          id: "dte_k2",
          kind: "Appointment",
          source: "EDI",
          summary: "One-hour appointment window from the tender",
          observedAt: kroger + 42 * MINUTE,
        },
        {
          id: "dte_k3",
          kind: "Notice",
          source: "System",
          summary: "Warning notice delivered to 2 recipients",
          observedAt: kroger + 129 * MINUTE,
        },
        {
          id: "dte_k4",
          kind: "Departure",
          source: "Telematics",
          summary: "Truck 2207 left the geofence",
          observedAt: krogerStop,
        },
      ],
      notices: [notice("dtn_k1", kroger + 129 * MINUTE, ["dock-atl@kroger.com", "ap@kroger.com"])],
      collectability: {
        score: 86,
        band: "Strong",
        chainValid: true,
        summary: "Strong claim",
        factors: [
          factor("arrival", "Arrival timestamp", 20, 20),
          factor("departure", "Departure timestamp", 15, 15),
          factor("notice", "Customer notice", 25, 25),
          factor("appointment", "Scheduled appointment", 15, 15),
          factor(
            "document",
            "Supporting document",
            11,
            25,
            "Attach the POD with the receiver's in and out times.",
          ),
        ],
      },
      currentlyBillable: null,
    },
    dto_lineage: {
      occurrence: occurrence({
        id: "dto_lineage",
        customerId: "cus_acme",
        locationId: "loc_lineage_dal",
        policySnapshot: snapshot({
          policyId: "dtp_cold",
          policyCode: "COLD-PU",
          policyName: "Cold storage pickups",
          policyVersion: 2,
          unnotifiedBehavior: "Suppress",
          notificationDeadlineMinutes: 45,
          requireApprovalOverAmount: null,
          capturedAt: lineage,
        }),
        calculationTrace: null,
        stopType: "Pickup",
        appointmentStart: lineage + 9 * MINUTE,
        arrivedAt: lineage,
        departedAt: null,
        clockStartAt: lineage + 9 * MINUTE,
        clockStopAt: null,
        freeTimeExpiresAt: lineage + 129 * MINUTE,
        noticeDueAt: lineage + 99 * MINUTE,
        noticeDeadlineAt: lineage + 165 * MINUTE,
        isOpen: true,
        freeMinutesGranted: 120,
        rawDwellMinutes: 0,
        billableMinutes: 0,
        roundedMinutes: 0,
        billableAmount: "0",
        driverPayMinutes: 0,
        driverPayAmount: "0",
        netMargin: "0",
        status: "Accruing",
        notificationStatus: "Pending",
        noticeSentAt: null,
        locationName: "Lineage Logistics Dallas",
        customerName: "Acme Foods",
        createdAt: lineage,
        updatedAt: lineage,
        location: {
          id: "loc_lineage_dal",
          city: "Dallas",
          state: { id: "st_tx", abbreviation: "TX" },
        },
      }),
      evidence: [
        {
          id: "dte_l1",
          kind: "Arrival",
          source: "Geofence",
          summary: "Truck 1904 inside the Lineage Dallas geofence",
          observedAt: lineage,
        },
        {
          id: "dte_l2",
          kind: "Appointment",
          source: "EDI",
          summary: "Window opens at the appointment from the tender",
          observedAt: lineage,
        },
        {
          id: "dte_l3",
          kind: "StatusChange",
          source: "System",
          summary: "Free time ran out, billing started",
          observedAt: lineage + 129 * MINUTE,
        },
      ],
      notices: [],
      collectability: {
        score: 48,
        band: "Weak",
        chainValid: true,
        summary: "Weak claim",
        factors: [
          factor("arrival", "Arrival timestamp", 20, 20),
          factor(
            "notice",
            "Customer notice",
            0,
            25,
            "Send the notice now; a late notice still earns part of this.",
          ),
          factor("appointment", "Scheduled appointment", 15, 15),
          factor("departure", "Departure timestamp", 0, 15, "Comes in when the truck leaves."),
          factor("document", "Supporting document", 13, 25, "Ask the worker for the gate-in slip."),
        ],
      },
      currentlyBillable: { billableMinutes: 30, billableAmount: "37.5", asOf: anchor },
    },
    dto_quick: {
      occurrence: occurrence({
        id: "dto_quick",
        customerId: "cus_riverside",
        locationId: "loc_riverside",
        policySnapshot: snapshot({
          policyId: "dtp_std",
          policyCode: "STD",
          policyName: "Standard shipper terms",
          policyVersion: 1,
          notificationRequirement: "None",
          notificationLeadMinutes: 0,
          requireApprovalOverAmount: null,
          capturedAt: quick + 95 * MINUTE,
        }),
        calculationTrace: billedTrace({
          policy: "Standard shipper terms",
          code: "STD",
          clockStart: quick,
          clockStop: quick + 94 * MINUTE,
          dwell: 94,
          free: 120,
          billable: 0,
          rounded: 0,
          amount: "0",
          notice: false,
        }),
        stopType: "Pickup",
        scheduleType: "Open",
        appointmentStart: null,
        arrivedAt: quick,
        departedAt: quick + 94 * MINUTE,
        clockStartAt: quick,
        clockStopAt: quick + 94 * MINUTE,
        freeTimeExpiresAt: quick + 120 * MINUTE,
        noticeDueAt: null,
        freeMinutesGranted: 120,
        rawDwellMinutes: 94,
        billableMinutes: 0,
        roundedMinutes: 0,
        billableAmount: "0",
        driverPayMinutes: 0,
        driverPayAmount: "0",
        netMargin: "0",
        status: "NotBillable",
        notificationStatus: "NotRequired",
        noticeSentAt: null,
        locationName: "Riverside Paper Mill",
        customerName: "Riverside Paper",
        createdAt: quick,
        updatedAt: quick + 95 * MINUTE,
        location: {
          id: "loc_riverside",
          city: "Augusta",
          state: { id: "st_ga", abbreviation: "GA" },
        },
      }),
      evidence: [
        {
          id: "dte_q1",
          kind: "Arrival",
          source: "Telematics",
          summary: "Truck 1188 inside the mill's geofence",
          observedAt: quick,
        },
        {
          id: "dte_q2",
          kind: "Departure",
          source: "Telematics",
          summary: "Truck 1188 left the geofence",
          observedAt: quick + 94 * MINUTE,
        },
      ],
      notices: [],
      collectability: {
        score: 70,
        band: "Adequate",
        chainValid: true,
        summary: "Adequate claim",
        factors: [
          factor("arrival", "Arrival timestamp", 20, 20),
          factor("departure", "Departure timestamp", 15, 15),
          factor("document", "Supporting document", 0, 25, "Attach the bill of lading."),
        ],
      },
      currentlyBillable: null,
    },
    dto_disputed: {
      occurrence: occurrence({
        id: "dto_disputed",
        customerId: "cus_kroger",
        locationId: "loc_kroger_atl",
        policySnapshot: snapshot({ capturedAt: disputed + 6 * HOUR }),
        calculationTrace: billedTrace({
          policy: "Kroger DC standard",
          code: "KRG-DC",
          clockStart: disputed,
          clockStop: disputed + 5 * HOUR,
          dwell: 300,
          free: 120,
          billable: 180,
          rounded: 180,
          amount: "225",
          notice: true,
        }),
        appointmentStart: disputed,
        appointmentEnd: disputed + HOUR,
        arrivedAt: disputed - 12 * MINUTE,
        departedAt: disputed + 5 * HOUR,
        clockStartAt: disputed,
        clockStopAt: disputed + 5 * HOUR,
        freeTimeExpiresAt: disputed + 2 * HOUR,
        noticeDueAt: disputed + 90 * MINUTE,
        noticeDeadlineAt: disputed + 2 * HOUR,
        freeMinutesGranted: 120,
        rawDwellMinutes: 300,
        billableMinutes: 180,
        roundedMinutes: 180,
        billableAmount: "225",
        driverPayMinutes: 120,
        driverPayAmount: "80",
        netMargin: "145",
        status: "Disputed",
        notificationStatus: "Sent",
        noticeSentAt: disputed + 84 * MINUTE,
        disputeNote:
          "Kroger says the truck checked in at 9:40, not 8:48; their gate log is attached to the dispute.",
        disputedAt: disputed + 3 * DAY,
        locationName: "Kroger Atlanta DC",
        customerName: "The Kroger Co.",
        createdAt: disputed,
        updatedAt: disputed + 3 * DAY,
        location: krogerLocation,
      }),
      evidence: [
        {
          id: "dte_d1",
          kind: "Arrival",
          source: "DriverApp",
          summary: "Worker checked in at the gate",
          observedAt: disputed - 12 * MINUTE,
        },
        {
          id: "dte_d2",
          kind: "Notice",
          source: "System",
          summary: "Warning notice delivered to 2 recipients",
          observedAt: disputed + 84 * MINUTE,
        },
        {
          id: "dte_d3",
          kind: "Departure",
          source: "DriverApp",
          summary: "Worker checked out",
          observedAt: disputed + 5 * HOUR,
        },
        {
          id: "dte_d4",
          kind: "Dispute",
          source: "Manual",
          summary: "Customer disputes the arrival time",
          observedAt: disputed + 3 * DAY,
        },
      ],
      notices: [notice("dtn_d1", disputed + 84 * MINUTE, ["dock-atl@kroger.com", "ap@kroger.com"])],
      collectability: {
        score: 61,
        band: "Weak",
        chainValid: true,
        summary: "Weak claim",
        factors: [
          factor(
            "arrival",
            "Arrival timestamp",
            10,
            20,
            "A telematics or geofence arrival outweighs a check-in from the app.",
          ),
          factor("notice", "Customer notice", 25, 25),
          factor("appointment", "Scheduled appointment", 15, 15),
          factor("document", "Supporting document", 11, 25, "Attach the signed POD."),
        ],
      },
      currentlyBillable: null,
    },
    dto_hd: {
      occurrence: occurrence({
        id: "dto_hd",
        customerId: "cus_hd",
        locationId: "loc_hd_atl",
        policySnapshot: snapshot({
          policyId: "dtp_hd",
          policyCode: "HD-RDC",
          policyName: "Home Depot RDC",
          policyVersion: 1,
          requireApprovalOverAmount: null,
          capturedAt: hd + 5 * HOUR,
        }),
        calculationTrace: billedTrace({
          policy: "Home Depot RDC",
          code: "HD-RDC",
          clockStart: hd,
          clockStop: hd + 270 * MINUTE,
          dwell: 270,
          free: 120,
          billable: 150,
          rounded: 150,
          amount: "187.5",
          notice: true,
        }),
        appointmentStart: hd,
        arrivedAt: hd,
        departedAt: hd + 270 * MINUTE,
        clockStartAt: hd,
        clockStopAt: hd + 270 * MINUTE,
        freeTimeExpiresAt: hd + 2 * HOUR,
        noticeDueAt: hd + 90 * MINUTE,
        noticeDeadlineAt: hd + 2 * HOUR,
        freeMinutesGranted: 120,
        rawDwellMinutes: 270,
        billableMinutes: 150,
        roundedMinutes: 150,
        billableAmount: "187.5",
        driverPayMinutes: 90,
        driverPayAmount: "60",
        netMargin: "127.5",
        status: "Waived",
        notificationStatus: "Sent",
        noticeSentAt: hd + 80 * MINUTE,
        waiverReason: "CustomerGoodwill",
        waiverNote:
          "Waived for the Q4 volume commitment — they're moving 40 more loads a week to us from November.",
        waivedAt: hd + DAY - 4 * HOUR + 12 * MINUTE,
        waivedAmount: "187.5",
        waivedBy: user("usr_dana", "Dana Ruiz"),
        locationName: "Home Depot RDC Atlanta",
        customerName: "The Home Depot",
        createdAt: hd,
        updatedAt: hd + DAY,
        location: {
          id: "loc_hd_atl",
          city: "McDonough",
          state: { id: "st_ga", abbreviation: "GA" },
        },
      }),
      evidence: [
        {
          id: "dte_h1",
          kind: "Arrival",
          source: "Telematics",
          summary: "Truck 2310 inside the RDC geofence",
          observedAt: hd,
        },
        {
          id: "dte_h2",
          kind: "Departure",
          source: "Telematics",
          summary: "Truck 2310 left the geofence",
          observedAt: hd + 270 * MINUTE,
        },
        {
          id: "dte_h3",
          kind: "Waiver",
          source: "Manual",
          summary: "Waived for customer goodwill",
          observedAt: hd + DAY - 4 * HOUR + 12 * MINUTE,
        },
      ],
      notices: [notice("dtn_h1", hd + 80 * MINUTE, ["rdc-atl@homedepot.com"])],
      collectability: {
        score: 80,
        band: "Adequate",
        chainValid: true,
        summary: "Adequate claim",
        factors: [
          factor("arrival", "Arrival timestamp", 20, 20),
          factor("departure", "Departure timestamp", 15, 15),
          factor("notice", "Customer notice", 25, 25),
          factor("document", "Supporting document", 5, 25, "Attach the signed POD."),
        ],
      },
      currentlyBillable: null,
    },
  };
}

/** The cards thread's detention conversations. */
export const DESK_DETENTION_CONVERSATIONS = {
  "detention-kroger": {
    recordId: "dto_kroger",
    title: "Detention at Kroger Atlanta DC",
    question: "Can I approve the detention at Kroger Atlanta?",
    reply:
      "It comes to $337.50 for 4h 30m past free time, over the $250.00 line so it needs you. The notice went out in its window and the score is strong; the one thing missing is the POD's in and out times. [Kroger Atlanta DC](artifact:art_detention-kroger) has the clock and the receipt.",
  },
  "detention-lineage": {
    recordId: "dto_lineage",
    title: "Detention at Lineage Logistics Dallas",
    question: "Is the truck at Lineage still waiting?",
    reply:
      "Yes, and it is billing now: $37.50 so far. The notice was due at the half hour and has not gone out, and this policy won't bill without it. [Lineage Logistics Dallas](artifact:art_detention-lineage) has the clock.",
  },
  "detention-quick": {
    recordId: "dto_quick",
    title: "Detention at Riverside Paper Mill",
    question: "Did we earn detention at the Riverside mill?",
    reply:
      "No: the truck left after 1h 34m, inside the two hours of free time. [Riverside Paper Mill](artifact:art_detention-quick) shows the clock.",
  },
  "detention-disputed": {
    recordId: "dto_disputed",
    title: "Detention at Kroger Atlanta DC",
    question: "Why hasn't the Kroger detention from last week been billed?",
    reply:
      "Kroger disputes it: their gate log puts the check-in 52 minutes later than ours. Our arrival came from the worker app, which scores lower than telematics. [Kroger Atlanta DC](artifact:art_detention-disputed) has what they dispute and the evidence.",
  },
  "detention-hd": {
    recordId: "dto_hd",
    title: "Detention at Home Depot RDC Atlanta",
    question: "What happened to the Home Depot detention?",
    reply:
      "Dana waived it for goodwill before it was approved. [Home Depot RDC Atlanta](artifact:art_detention-hd) keeps why.",
  },
  "detention-gone": {
    recordId: "dto_999",
    title: "Detention at Sysco Houston",
    question: "Pull up the Sysco Houston detention.",
    reply:
      "That detention charge has been removed since I looked it up. [Sysco Houston](artifact:art_detention-gone) is what I found then.",
  },
};
