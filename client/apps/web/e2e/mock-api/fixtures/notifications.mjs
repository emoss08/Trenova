// The notification panel's subjects: what is the person's own, what went to their team, what
// they set aside and what is finished. Times are relative to the real clock, because the
// panel sorts a row into its part by comparing `surfacesAt` with the browser's own time.

const MINUTE = 60;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

const BASE = {
  organizationId: "org_mock",
  businessUnitId: "bu_mock",
  targetUserId: "usr_mock",
  channel: "user",
  relatedEntities: null,
  dismissedAt: null,
  resolvedAt: null,
  resolution: "",
  occurrenceCount: 1,
};

function row(now, id, ago, fields) {
  const at = now - ago;
  return {
    ...BASE,
    id,
    readAt: null,
    createdAt: at,
    surfacesAt: at,
    data: null,
    ...fields,
  };
}

function exception(now, id, ago, eventType, title, message, extra = {}) {
  return row(now, id, ago, {
    eventType,
    priority: "high",
    source: "watchtower",
    reason: "direct",
    title,
    message,
    ...extra,
    data: { itemId: `wti_${id}`, link: "/desk/needs", ...extra.data },
  });
}

function rollup(now, id, ago, title, count, message, recent, extra = {}) {
  return row(now, id, ago, {
    eventType: "watchtower.critical",
    priority: "high",
    source: "watchtower",
    reason: "team",
    title,
    message,
    occurrenceCount: count,
    ...extra,
    data: {
      link: "/desk/needs",
      recent: recent.map(([ref, label, minutes]) => ({
        ref,
        label,
        link: "/desk/needs",
        at: now - minutes * MINUTE,
      })),
      ...extra.data,
    },
  });
}

function proposals(now, id, ago, title, message, extra = {}) {
  return row(now, id, ago, {
    eventType: "agent.proposals_pending",
    priority: "medium",
    source: "agent",
    reason: "team",
    title,
    message,
    ...extra,
    data: { link: "/desk/needs?lens=decisions", ...extra.data },
  });
}

export function buildNotifications(now) {
  const forYou = [
    exception(
      now,
      "y1",
      14 * MINUTE,
      "watchtower.escalated",
      "Missed pickup",
      "Continental Brands Reno Plant, Reno, NV. The window closed at 9:00 AM and no arrival is recorded.",
      { priority: "critical", data: { severity: "Critical" } },
    ),
    exception(
      now,
      "y2",
      38 * MINUTE,
      "watchtower.assigned",
      "Projected late",
      "Reaches Kowalski Home Goods in Kansas City, MO about 2 h 10 min after the delivery window closes.",
    ),
    row(now, "y3", HOUR, {
      eventType: "shipment_comment_mention",
      priority: "medium",
      source: "shipment_comment",
      reason: "direct",
      title: "Dana Whitfield mentioned you",
      message: "@Maya can you confirm the reefer setpoint before this leaves the yard?",
      data: { authorId: "usr_dana", authorName: "Dana Whitfield" },
      relatedEntities: { shipmentId: "shp_mock_mention" },
    }),
    row(now, "y4", 2 * HOUR, {
      eventType: "dash.pto_requested",
      priority: "medium",
      source: "dash",
      reason: "direct",
      title: "Willie Ford asked for time off",
      message: "Nov 24 to Nov 26, three days. The peak-season blackout applies to these dates.",
      data: { ptoId: "pto_mock_1" },
    }),
    exception(
      now,
      "y5",
      3 * HOUR,
      "watchtower.assigned",
      "POD missing",
      "Delivered for Premier Glass yesterday, and no proof of delivery is on file.",
    ),
    row(now, "y6", 4 * HOUR, {
      eventType: "assistant_reply_ready",
      priority: "low",
      source: "assistant",
      reason: "direct",
      title: "Your reply is ready",
      message: "Why did detention on S26110504383871 run past free time?",
      data: { kind: "assistant_reply_ready", threadId: "athr_desk_live", status: "Completed" },
    }),
    row(now, "y7", DAY + 2 * HOUR, {
      eventType: "record_shared",
      priority: "low",
      source: "sharing",
      reason: "direct",
      readAt: now - DAY,
      title: "Marcus Lee shared an invoice with you",
      message: "INV-004512 · Vanguard Frozen Foods · $4,820.00",
      data: {
        sharedById: "usr_marcus",
        sharedByName: "Marcus Lee",
        link: "/billing/invoices",
      },
    }),
    row(now, "y8", DAY + 3 * HOUR, {
      eventType: "report_run_completed",
      priority: "low",
      source: "report",
      reason: "direct",
      readAt: now - DAY,
      title: "Customer scorecard is ready",
      message: "The report you ran finished in 12 seconds.",
      data: {
        runId: "rrun_mock_1",
        reportName: "Customer scorecard",
        format: "xlsx",
        byteSize: 48_213,
        rowCount: 48,
        artifactExpiresAt: now + 6 * DAY,
      },
    }),
    row(now, "y9", DAY + 5 * HOUR, {
      eventType: "tender_accepted",
      priority: "low",
      source: "tender",
      reason: "direct",
      readAt: now - DAY,
      title: "Tender accepted",
      message: "Blue Ridge Freight accepted S26110504383871 at $2,140.00.",
      data: { link: "/dispatch/console" },
    }),
  ];

  const team = [
    rollup(
      now,
      "t1",
      22 * MINUTE,
      "Missed delivery",
      14,
      "Mills & Lee LLC Store #1234 in Savannah, GA closed with no arrival recorded.",
      [
        ["S26100809512328", "Mills & Lee LLC, Savannah, GA", 22],
        ["S26110235603574", "Jackson & Jenkins, Memphis, TN", 64],
        ["S26110451030811", "Northstar Enterprises, Tampa, FL", 121],
      ],
    ),
    rollup(
      now,
      "t2",
      41 * MINUTE,
      "Missed pickup",
      11,
      "Continental Brands Warehouse in Phoenix, AZ closed with no arrival recorded.",
      [
        ["S26110451030896", "Continental Brands, Reno, NV", 41],
        ["S26110451030902", "Continental Brands, Ontario, CA", 70],
        ["S26110451030871", "Continental Brands, Phoenix, AZ", 75],
      ],
      { priority: "critical", data: { severity: "Critical" } },
    ),
    rollup(
      now,
      "t3",
      HOUR,
      "Projected late",
      11,
      "S26110223794696 reaches Kansas City, MO after its window.",
      [
        ["S26110223794696", "Kowalski Home Goods, Kansas City, MO", 60],
        ["S26110504383871", "Vanguard Frozen Foods, Detroit, MI", 185],
        ["S26110505758381", "Jackson & Jenkins, Memphis, TN", 242],
      ],
    ),
    proposals(
      now,
      "t4",
      HOUR + 5 * MINUTE,
      "Billing exception agent has changes waiting for a decision",
      "247 changes wait. Nothing runs until someone decides.",
      { occurrenceCount: 247, data: { waitingHours: 74 } },
    ),
    proposals(
      now,
      "t5",
      3 * HOUR,
      "Detention desk has changes waiting for a decision",
      "It proposes escalating detention on 12 loads.",
      { occurrenceCount: 12 },
    ),
    rollup(
      now,
      "t6",
      5 * HOUR,
      "POD missing",
      6,
      "Delivered for Premier Glass, and no proof of delivery is on file.",
      [
        ["S26110308789302", "Premier Glass", 300],
        ["S26110235603574", "Jackson & Jenkins Partners", 362],
        ["S26110308789317", "Premier Glass", 371],
      ],
      { readAt: now - 4 * HOUR },
    ),
    row(now, "t7", DAY + HOUR, {
      eventType: "permission.role_changed",
      priority: "medium",
      source: "permission",
      reason: "team",
      readAt: now - DAY,
      title: "A role changed",
      message: "Billing clerk gained Approve invoices, changed by Dana Whitfield.",
      data: { link: "/admin/roles" },
    }),
    row(now, "t8", DAY + 2 * HOUR, {
      eventType: "billing_transfer_completed",
      priority: "low",
      source: "billing",
      reason: "team",
      readAt: now - DAY,
      title: "Transfer to billing finished",
      message: "42 shipments moved to the billing queue. None were sent back.",
      data: { link: "/billing-queue" },
    }),
    proposals(
      now,
      "t9",
      DAY + 4 * HOUR,
      "Credential desk has changes waiting for a decision",
      "It proposes renewal reminders for 9 drivers.",
      { occurrenceCount: 9, readAt: now - DAY },
    ),
  ];

  const later = [
    {
      ...proposals(
        now,
        "l1",
        2 * HOUR,
        "Load monitor has changes waiting for a decision",
        "It proposes check calls on 6 loads.",
        { occurrenceCount: 6 },
      ),
      surfacesAt: now + 17 * HOUR,
    },
    {
      ...row(now, "l2", 6 * HOUR, {
        eventType: "carrier_insurance_expiring",
        priority: "medium",
        source: "carrier",
        reason: "team",
        title: "A carrier’s cargo insurance expires in 9 days",
        message: "Tenders to this carrier stop once the policy lapses.",
        data: { link: "/dispatch/carrier-monitoring" },
      }),
      surfacesAt: now + 3 * DAY,
    },
  ];

  const finished = (entry, ago, resolution = "") => ({
    ...entry,
    readAt: now - ago,
    dismissedAt: now - ago,
    resolvedAt: resolution ? now - ago : null,
    resolution,
  });
  const done = [
    finished(
      exception(
        now,
        "d1",
        3 * HOUR,
        "watchtower.assigned",
        "Missed pickup",
        "Continental Brands, Phoenix, AZ. The window closed and no arrival was recorded.",
      ),
      HOUR,
      "resolved",
    ),
    finished(
      proposals(
        now,
        "d2",
        6 * HOUR,
        "Dispatch desk has changes waiting for a decision",
        "It proposes re-tendering 6 loads.",
        { occurrenceCount: 6 },
      ),
      2 * HOUR,
      "decided",
    ),
    finished(
      row(now, "d3", 7 * HOUR, {
        eventType: "tender_accepted",
        priority: "low",
        source: "tender",
        reason: "direct",
        title: "Tender accepted",
        message: "Summit Carriers accepted S26110504383871 at $1,980.00.",
        data: { link: "/dispatch/console" },
      }),
      5 * HOUR,
    ),
  ];

  return [...forYou, ...team, ...later, ...done];
}
