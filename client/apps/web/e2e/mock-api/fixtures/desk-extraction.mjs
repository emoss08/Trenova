// A rate confirmation read by document intelligence, as the Desk's extraction
// artifact shows it. Each page is a list of text lines placed as fractions of
// the page, the way `documentcontent.LayoutLine` stores them; the page images
// and the PDF are drawn from the same lines, so a box built from a line sits
// on that line's text. The payload is built the way
// `assistantservice/extraction.go` builds it from the draft
// `get_shipment_draft` returns.

export const EXTRACTION_THREAD_ID = "athr_desk_extract";
export const EXTRACTION_AGENT_ID = "agd_dispatch";
export const EXTRACTION_PROPOSAL_ID = "aprop_extract_ratecon";
export const EXTRACTION_SHIPMENT_ID = "shp_01JRNL482913";

/** Letter paper drawn at 150 dpi, as the page image endpoint draws it. */
const PORTRAIT = { width: 1275, height: 1650 };
const LANDSCAPE = { width: 1650, height: 1275 };

const line = (t, x, y, w, h = 0.014) => ({ t, x, y, w, h });

/** Page 1: a broker's rate confirmation, born digital. */
const RATECON_PAGE_1 = {
  size: PORTRAIT,
  scan: null,
  lines: [
    line("NORTHLINE LOGISTICS", 0.08, 0.05, 0.3, 0.02),
    line("Rate Confirmation", 0.62, 0.05, 0.28, 0.02),
    line("Load # NL-482913", 0.62, 0.085, 0.2),
    line("Date: 10/15/2026", 0.62, 0.105, 0.17),
    line("Carrier: Redwood Freight LLC MC 784512", 0.08, 0.15, 0.42),
    line("Bill to: Northline Logistics, 400 Commerce St, Dallas TX 75202", 0.08, 0.17, 0.66),
    line("PICKUP 1", 0.08, 0.23, 0.1),
    line("Ridgefield Paper Co.", 0.08, 0.25, 0.22),
    line("1450 Industrial Pkwy, Memphis, TN 38118", 0.08, 0.27, 0.42),
    line("10/20/2026 08:00-12:00 Appt PU# 77812", 0.08, 0.29, 0.42),
    line("DELIVERY 1", 0.52, 0.23, 0.12),
    line("Anyco Clothes DC #425", 0.52, 0.25, 0.24),
    line("9000 Distribution Way, Dallas, TX 75241", 0.52, 0.27, 0.4),
    line("10/21/2026 FCFS 06:00-14:00", 0.52, 0.29, 0.3),
    line("Equipment: 53' Dry Van", 0.08, 0.35, 0.25),
    line("Commodity: Paper products, 24 pallets", 0.08, 0.37, 0.4),
    line("Weight: 38,500 lbs", 0.08, 0.39, 0.2),
    line("PO 4500123998 BOL 88213", 0.52, 0.35, 0.28),
    line("Line haul $2,150.00", 0.52, 0.45, 0.22),
    line("Fuel surcharge $315.00", 0.52, 0.47, 0.24),
    line("Total $2,465.00", 0.52, 0.5, 0.18, 0.018),
    line("Carrier must call for appointment 24 hours ahead of delivery.", 0.08, 0.58, 0.62),
    line("Detention $50/hr after 2 hours free. Lumper paid by receiver.", 0.08, 0.6, 0.62),
    line("Payment terms: Net 30 from receipt of signed POD and invoice.", 0.08, 0.62, 0.62),
  ],
};

/**
 * Page 2: the signed second leg, scanned sideways on a flatbed with a grey margin around the
 * paper. The paper sits inset in the image, so a box placed against the image (not the paper)
 * is the one that lines up.
 */
const RATECON_PAGE_2 = {
  size: LANDSCAPE,
  scan: { x: 0.05, y: 0.07, w: 0.9, h: 0.86 },
  lines: [
    line("DELIVERY 2", 0.1, 0.14, 0.12, 0.018),
    line("Anyco Clothes #612", 0.1, 0.18, 0.2, 0.018),
    line("220 Market St, Fort Worth, TX 76102", 0.1, 0.22, 0.34, 0.018),
    line("10/22/2026 Appt 09:00", 0.1, 0.26, 0.22, 0.018),
    line("Receiver notes: dock 14, call 30 min out", 0.1, 0.32, 0.4, 0.018),
    line("Accepted by carrier: J. Ortega", 0.56, 0.72, 0.3, 0.018),
    line("Signature ____________________", 0.56, 0.77, 0.32, 0.018),
  ],
};

/** A bill of lading photographed on a phone: one image, one page. */
const BOL_IMAGE = {
  size: { width: 1200, height: 1600 },
  scan: { x: 0.03, y: 0.02, w: 0.94, h: 0.96 },
  lines: [
    line("STRAIGHT BILL OF LADING", 0.24, 0.06, 0.52, 0.022),
    line("BOL No. 88213", 0.08, 0.12, 0.2),
    line("Ship from: Ridgefield Paper Co.", 0.08, 0.16, 0.36),
    line("1450 Industrial Pkwy, Memphis, TN 38118", 0.08, 0.18, 0.44),
    line("Ship to: Anyco Clothes DC #425", 0.08, 0.23, 0.36),
    line("9000 Distribution Way, Dallas, TX 75241", 0.08, 0.25, 0.44),
    line("24 PLT Paper products 38,500 lbs Class 55", 0.08, 0.33, 0.5),
    line("Shipper signature ____ Date 10/20/2026", 0.08, 0.84, 0.46),
  ],
};

export const EXTRACTION_DOCUMENTS = {
  ratecon: {
    documentId: "doc_01JRNLRATECON",
    fileName: "Northline_RateCon_NL-482913.pdf",
    contentType: "application/pdf",
    fileSize: 184_220,
    pages: [RATECON_PAGE_1, RATECON_PAGE_2],
  },
  image: {
    documentId: "doc_01JRBOLPHOTO",
    fileName: "IMG_4471.jpg",
    contentType: "image/jpeg",
    fileSize: 2_412_004,
    pages: [BOL_IMAGE],
  },
};

const field = (label, value, confidence, pageNumber, extra = {}) => ({
  label,
  value,
  confidence,
  pageNumber,
  ...extra,
});

/** The draft `get_shipment_draft` returns for the rate confirmation. */
function rateconDraft(documentId, attachedShipmentId) {
  return {
    documentId,
    status: "Ready",
    reviewStatus: "NeedsReview",
    documentKind: "RateConfirmation",
    confidence: 0.91,
    kindConfidence: 0.94,
    attachedShipmentId,
    missingFields: ["Pro Number", "Piece Count"],
    fields: {
      loadNumber: field("Load Number", "NL-482913", 0.97, 1),
      carrierName: field("Carrier Name", "Redwood Freight LLC", 0.95, 1),
      billTo: field("Bill To", "Northline Logistics", 0.71, 1, { reviewRequired: true }),
      shipper: field("Shipper", "Ridgefield Paper Co.", 0.95, 1),
      consignee: field("Consignee", "Anyco Clothes DC #425", 0.93, 1),
      pickupWindow: field("Pickup Window", "10/20/2026 08:00-12:00", 0.9, 1),
      pickupNumber: field("Pickup Number", "77812", 0.83, 1),
      deliveryWindow: field("Delivery Window", "10/21/2026", 0.62, 0, { conflict: true }),
      poNumber: field("PO Number", "4500123998", 0.94, 1),
      bol: field("BOL", "88213", 0.86, 1),
      equipmentType: field("Equipment Type", "53' Dry Van", 0.88, 1),
      commodity: field("Commodity", "Paper products", 0.78, 1),
      weight: field("Weight", 38500, 0.91, 1),
      rate: field("Rate", "2465.00", 0.96, 1),
      fuelSurcharge: field("Fuel Surcharge", "315.00", 0.9, 1),
      paymentTerms: field("Payment Terms", "Net 30", 0.84, 1),
    },
    stops: [
      {
        sequence: 1,
        role: "pickup",
        name: "Ridgefield Paper Co.",
        addressLine1: "1450 Industrial Pkwy",
        city: "Memphis",
        state: "TN",
        postalCode: "38118",
        date: "10/20/2026",
        timeWindow: "08:00-12:00",
        appointmentRequired: true,
        confidence: 0.93,
        pageNumber: 1,
      },
      {
        sequence: 2,
        role: "delivery",
        name: "Anyco Clothes DC #425",
        addressLine1: "9000 Distribution Way",
        city: "Dallas",
        state: "TX",
        postalCode: "75241",
        date: "10/21/2026",
        timeWindow: "06:00-14:00",
        confidence: 0.9,
        pageNumber: 1,
      },
      {
        sequence: 3,
        role: "delivery",
        name: "Anyco Clothes #612",
        addressLine1: "220 Market St",
        city: "Fort Worth",
        state: "TX",
        postalCode: "76102",
        date: "10/22/2026",
        timeWindow: "09:00",
        appointmentRequired: true,
        confidence: 0.74,
        reviewRequired: true,
        pageNumber: 2,
      },
    ],
    conflicts: [
      {
        key: "deliveryWindow",
        label: "Delivery Window",
        values: ["10/21/2026", "10/22/2026"],
      },
    ],
  };
}

/** The draft for the photographed bill of lading. */
function bolDraft(documentId) {
  return {
    documentId,
    status: "Ready",
    reviewStatus: "Ready",
    documentKind: "BillOfLading",
    confidence: 0.88,
    kindConfidence: 0.9,
    missingFields: [],
    fields: {
      bol: field("BOL", "88213", 0.92, 1),
      shipper: field("Shipper", "Ridgefield Paper Co.", 0.9, 1),
      consignee: field("Consignee", "Anyco Clothes DC #425", 0.89, 1),
      commodity: field("Commodity", "Paper products", 0.85, 1),
      weight: field("Weight", 38500, 0.87, 1),
      pickupDate: field("Pickup Date", "10/20/2026", 0.81, 1),
    },
    stops: [],
    conflicts: [],
  };
}

/** A rate confirmation with many fields and long values, none of them placed on the page. */
function longDraft(documentId) {
  const draft = rateconDraft(documentId, "");
  draft.fields = {
    ...draft.fields,
    instructions: field(
      "Instructions",
      "Carrier must call for appointment 24 hours ahead of delivery. Driver must check in at the guard shack with the PO and load number, wear a high-visibility vest and steel-toed boots, and stay with the truck while it is unloaded. No overnight parking on site; the nearest truck stop is 4 miles south on I-35E.",
      0.69,
      1,
      { reviewRequired: true },
    ),
    carrierContact: field(
      "Carrier Contact",
      "Maria Ortega-Villanueva · dispatch@redwoodfreight.example · (901) 555-0142 ext. 3",
      0.81,
      1,
    ),
    referenceNumber: field("Reference Number", "NLX-2026-10-0048213-REV-B", 0.77, 1),
    serviceType: field("Service Type", "Truckload", 0.9, 1),
    scac: field("SCAC", "RWFL", 0.88, 1),
    trailerNumber: field("Trailer Number", "RW53-11872", 0.8, 1),
  };
  return draft;
}

function comparableIndex(text) {
  let hay = "";
  const index = [];
  [...text].forEach((char, position) => {
    if (/[\p{L}\p{N}]/u.test(char)) {
      const lower = char.toLowerCase();
      hay += lower;
      for (let i = 0; i < lower.length; i += 1) index.push(position);
    }
  });
  return { hay, index };
}

const comparable = (text) => comparableIndex(text).hay;

const round4 = (value) => Math.round(value * 10000) / 10000;

/**
 * Where a value sits on the pages, as `extraction.go`'s locate finds it: the line the reading
 * quoted, then a line carrying the field's label before the value, then any line holding it,
 * on the named page first; the value measured on the line's own characters.
 */
function locate(reading, pages) {
  const words = reading.value.split(/\s+/).filter(Boolean);
  for (let n = words.length; n > 0; n -= 1) {
    const value = comparable(words.slice(0, n).join(" "));
    if (n < words.length && [...value].length < 6) break;
    const found = locateValue(reading, value, pages);
    if (found) return found;
  }
  return null;
}

function locateValue(reading, value, pages) {
  if (value === "") return null;
  const needle = [...value].length > 24 ? [...value].slice(0, 24).join("") : value;
  const evidence = comparable(reading.evidence ?? "");
  const label = comparable(reading.label ?? "");
  let best = null;
  for (const pass of [0, 1]) {
    pages.forEach((page, index) => {
      const number = index + 1;
      const named = reading.page > 0 && number === reading.page;
      if (reading.page > 0 && (pass === 0) !== named) return;
      for (const line of page.lines) {
        const { hay } = comparableIndex(line.t);
        const at = hay.indexOf(needle);
        if (at < 0 || hay === "") continue;
        let span = needle.length;
        while (span < value.length && at + span < hay.length && hay[at + span] === value[span])
          span += 1;
        let score = 1;
        let weight = 0;
        if (
          evidence !== "" &&
          (evidence.includes(hay) || (evidence.length > value.length && hay.includes(evidence)))
        ) {
          score += 2;
          weight = hay.length;
        }
        if (label !== "" && hay.slice(0, at).includes(label)) score += 1;
        if (!best || score > best.score || (score === best.score && weight > best.weight)) {
          best = { page: number, line, at, span, score, weight };
        }
      }
    });
    if (best || reading.page <= 0) break;
  }
  if (!best) return null;
  const { index } = comparableIndex(best.line.t);
  const length = [...best.line.t].length;
  const first = index[best.at];
  const last = index[best.at + best.span - 1] + 1;
  const x = best.line.x + (best.line.w * first) / length;
  const right = best.line.x + (best.line.w * last) / length;
  return {
    page: best.page,
    box: {
      x: round4(x),
      y: round4(best.line.y),
      w: round4(Math.max(right - x, 0.02)),
      h: round4(Math.max(best.line.h, 0.01)),
    },
  };
}

function displayValue(value) {
  if (value === null || value === undefined) return "";
  if (typeof value === "number") return Number.isInteger(value) ? String(value) : value.toFixed(2);
  if (typeof value === "boolean") return value ? "Yes" : "No";
  return String(value).trim();
}

const roundConfidence = (value) => Math.round(Math.max(0, Math.min(1, value)) * 100) / 100;

function extractionField(reading, pages) {
  const confidence = roundConfidence(reading.confidence);
  const reason = reading.conflict
    ? "conflict"
    : reading.review
      ? "review"
      : confidence < 0.8
        ? "low"
        : "";
  const out = {
    key: reading.key,
    label: reading.label,
    value: reading.value,
    confidence,
    needsLook: reason !== "",
  };
  if (reason) out.reason = reason;
  if (reading.readings?.length > 1) out.readings = reading.readings;
  if (reading.stop) Object.assign(out, reading.stop);
  const found = pages ? locate(reading, pages) : null;
  if (found) {
    out.page = found.page;
    out.box = found.box;
  } else if (reading.page > 0) {
    out.page = reading.page;
  }
  return out;
}

function stopAddress(stop) {
  let place = stop.city ?? "";
  if (stop.state) place = place ? `${place}, ${stop.state}` : stop.state;
  if (stop.postalCode) place = `${place} ${stop.postalCode}`.trim();
  return [stop.addressLine1, stop.addressLine2, place].filter(Boolean).join(", ");
}

/** The readings `draftReadings` builds: each field, then each stop as its name, address and window. */
function draftReadings(draft) {
  const conflicts = Object.fromEntries(
    (draft.conflicts ?? []).map((conflict) => [
      conflict.key,
      [...new Set(conflict.values.map(displayValue))],
    ]),
  );
  const out = [];
  for (const [key, entry] of Object.entries(draft.fields)) {
    const value = displayValue(entry.value);
    if (value === "") continue;
    out.push({
      key,
      label: entry.label,
      value,
      evidence: entry.evidenceExcerpt ?? "",
      confidence: entry.confidence,
      page: entry.pageNumber ?? 0,
      review: Boolean(entry.reviewRequired),
      conflict: Boolean(entry.conflict) || (conflicts[key]?.length ?? 0) > 1,
      readings: conflicts[key] ?? [],
    });
  }
  draft.stops.forEach((stop, index) => {
    const sequence = stop.sequence || index + 1;
    const parts = [
      ["name", "Name", stop.name ?? ""],
      ["address", "Address", stopAddress(stop)],
      ["window", "Window", [stop.date, stop.timeWindow].filter(Boolean).join(" ")],
    ];
    for (const [part, label, value] of parts) {
      if (value === "") continue;
      out.push({
        key: `stop${sequence}.${part}`,
        label,
        value,
        evidence: stop.evidenceExcerpt ?? "",
        confidence: stop.confidence,
        page: stop.pageNumber ?? 0,
        review: Boolean(stop.reviewRequired),
        stop: {
          stop: sequence,
          role: (stop.role ?? "").toLowerCase(),
          part,
          ...(stop.appointmentRequired && part === "window" ? { appointment: true } : {}),
        },
      });
    }
  });
  return out;
}

/** The fields as `extractionFields` lists them: on the page in reading order, then by key. */
function extractionFields(draft, pages) {
  const out = draftReadings(draft).map((reading) => extractionField(reading, pages));
  const pageOf = (entry) => entry.page ?? 2 ** 31 - 1;
  const topOf = (entry) => entry.box?.y ?? 2;
  return out.sort(
    (a, b) =>
      pageOf(a) - pageOf(b) || topOf(a) - topOf(b) || (a.key < b.key ? -1 : a.key > b.key ? 1 : 0),
  );
}

const KIND_LABELS = {
  RateConfirmation: "Rate confirmation",
  BillOfLading: "Bill of lading",
};

const MISSING_KEYS = { "Pro Number": "proNumber", "Piece Count": "pieceCount" };

/** The artifact payload `extractionArtifact` stores for a draft. */
function payloadOf(draft, document, pages) {
  const fields = extractionFields(draft, pages);
  const pageCount = Math.max(pages?.length ?? 0, ...fields.map((entry) => entry.page ?? 0), 1);
  const payload = {
    documentId: draft.documentId,
    fileName: document.fileName,
    contentType: document.contentType,
    documentKind: draft.documentKind,
    kindLabel: KIND_LABELS[draft.documentKind] ?? "Document",
    kindConfidence: draft.kindConfidence,
    confidence: draft.confidence,
    reviewStatus: draft.reviewStatus,
    pageCount,
    pages: (pages ?? []).map((page, index) => ({
      number: index + 1,
      width: page.size.width,
      height: page.size.height,
    })),
    fields,
    needsLook: fields.filter((entry) => entry.needsLook).length,
    missing: draft.missingFields,
    missingFields: draft.missingFields.map((label) => ({
      key: MISSING_KEYS[label] ?? label,
      label,
    })),
    tool: "get_shipment_draft",
  };
  if (draft.attachedShipmentId) payload.attachedShipmentId = draft.attachedShipmentId;
  return payload;
}

/** The payload an artifact recorded before reasons, stop parts, pages and keyed missing fields. */
function legacyPayload(draft, document) {
  const fields = [];
  for (const [key, entry] of Object.entries(draft.fields)) {
    const value = displayValue(entry.value);
    if (value === "") continue;
    const confidence = roundConfidence(entry.confidence);
    fields.push({
      key,
      label: entry.label,
      value,
      confidence,
      needsLook: Boolean(entry.reviewRequired || entry.conflict) || confidence < 0.8,
      page: entry.pageNumber || undefined,
    });
  }
  draft.stops.forEach((stop, index) => {
    const place = [stop.name, [stop.city, stop.state].filter(Boolean).join(", "), stop.date]
      .filter(Boolean)
      .join(" · ");
    fields.push({
      key: `stop${index + 1}`,
      label: stop.role.charAt(0).toUpperCase() + stop.role.slice(1),
      value: place,
      confidence: roundConfidence(stop.confidence),
      needsLook: Boolean(stop.reviewRequired) || stop.confidence < 0.8,
      page: stop.pageNumber,
    });
  });
  return {
    documentId: draft.documentId,
    fileName: document.fileName,
    documentKind: draft.documentKind,
    kindLabel: KIND_LABELS[draft.documentKind] ?? "Document",
    confidence: draft.confidence,
    reviewStatus: draft.reviewStatus,
    pageCount: 2,
    fields,
    needsLook: fields.filter((entry) => entry.needsLook).length,
    missing: draft.missingFields,
    tool: "get_shipment_draft",
  };
}

export const EXTRACTION_VARIANTS = ["ratecon", "image", "legacy", "nobox", "broken", "long"];

/** The document a variant shows. */
export function extractionDocument(variant) {
  return variant === "image" ? EXTRACTION_DOCUMENTS.image : EXTRACTION_DOCUMENTS.ratecon;
}

/** The artifact payload for a variant, with the shipment attached when it already exists. */
export function extractionPayload(variant, attached = false) {
  const document = extractionDocument(variant);
  const shipment = attached ? EXTRACTION_SHIPMENT_ID : "";
  switch (variant) {
    case "image":
      return payloadOf(bolDraft(document.documentId), document, document.pages);
    case "legacy":
      return legacyPayload(rateconDraft(document.documentId, shipment), document);
    case "nobox":
      return payloadOf(rateconDraft(document.documentId, shipment), document, null);
    case "long":
      return payloadOf(longDraft(document.documentId), document, null);
    default:
      return payloadOf(rateconDraft(document.documentId, shipment), document, document.pages);
  }
}

function escapeXml(text) {
  return text.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

/**
 * A page drawn as the page image endpoint draws it: the paper (inset on a grey scan bed when
 * the page was scanned) with each line's text set inside its box. Each word is set where its
 * share of the line's characters puts it, as a reader that reports word positions places it,
 * so a box built from the layout covers exactly its words.
 */
export function pageSvg(document, pageNumber) {
  const page = document.pages[pageNumber - 1];
  if (!page) return null;
  const { width, height } = page.size;
  const paper = page.scan ?? { x: 0, y: 0, w: 1, h: 1 };
  const text = page.lines
    .map((entry) => {
      const h = entry.h * height;
      const y = entry.y * height + h * 0.82;
      const bold = /^[A-Z0-9 #]+$/.test(entry.t) || entry.t.startsWith("Total");
      const length = [...entry.t].length;
      let at = 0;
      return entry.t
        .split(" ")
        .map((word) => {
          const start = at;
          at += [...word].length + 1;
          if (word === "") return "";
          const x = (entry.x + (entry.w * start) / length) * width;
          const span = ((entry.w * [...word].length) / length) * width;
          return `<text x="${x.toFixed(1)}" y="${y.toFixed(1)}" font-size="${(h * 0.92).toFixed(1)}" textLength="${span.toFixed(1)}" lengthAdjust="spacingAndGlyphs"${bold ? ' font-weight="700"' : ""}>${escapeXml(word)}</text>`;
        })
        .join("");
    })
    .join("");
  const rules = page.scan
    ? ""
    : `<rect x="${0.08 * width}" y="${0.205 * height}" width="${0.84 * width}" height="2" fill="#d4d4d4"/><rect x="${0.08 * width}" y="${0.425 * height}" width="${0.84 * width}" height="2" fill="#d4d4d4"/>`;
  const bed = page.scan ? `<rect width="${width}" height="${height}" fill="#8f8f8a"/>` : "";
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">${bed}<rect x="${paper.x * width}" y="${paper.y * height}" width="${paper.w * width}" height="${paper.h * height}" fill="${page.scan ? "#f4f1ea" : "#ffffff"}"/>${rules}<g font-family="Helvetica, Arial, sans-serif" fill="#1a1a1a"${page.scan ? ` transform="rotate(0.4 ${width / 2} ${height / 2})"` : ""}>${text}</g></svg>`;
}

/**
 * The document as a real PDF, so the regular viewer has a file to open: each page at its size
 * in points with its lines set in Helvetica where the layout puts them.
 */
export function documentPdf(document) {
  const objects = [];
  const add = (body) => {
    objects.push(body);
    return objects.length;
  };
  const catalog = add("");
  const pagesRef = add("");
  const font = add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>");
  const kids = [];
  for (const page of document.pages) {
    const w = (page.size.width / 150) * 72;
    const h = (page.size.height / 150) * 72;
    const ops = page.lines
      .map((entry) => {
        const size = entry.h * h * 0.9;
        const x = entry.x * w;
        const y = h - (entry.y + entry.h * 0.82) * h;
        const safe = entry.t.replace(/[\\()]/g, (c) => `\\${c}`);
        return `BT /F1 ${size.toFixed(2)} Tf ${x.toFixed(2)} ${y.toFixed(2)} Td (${safe}) Tj ET`;
      })
      .join("\n");
    const stream = add(`<< /Length ${Buffer.byteLength(ops)} >>\nstream\n${ops}\nendstream`);
    kids.push(
      add(
        `<< /Type /Page /Parent ${pagesRef} 0 R /MediaBox [0 0 ${w.toFixed(2)} ${h.toFixed(2)}] /Resources << /Font << /F1 ${font} 0 R >> >> /Contents ${stream} 0 R >>`,
      ),
    );
  }
  objects[catalog - 1] = `<< /Type /Catalog /Pages ${pagesRef} 0 R >>`;
  objects[pagesRef - 1] =
    `<< /Type /Pages /Kids [${kids.map((kid) => `${kid} 0 R`).join(" ")}] /Count ${kids.length} >>`;
  let out = "%PDF-1.4\n";
  const offsets = [];
  for (const [index, body] of objects.entries()) {
    offsets.push(Buffer.byteLength(out));
    out += `${index + 1} 0 obj\n${body}\nendobj\n`;
  }
  const xref = Buffer.byteLength(out);
  out += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
  for (const offset of offsets) out += `${String(offset).padStart(10, "0")} 00000 n \n`;
  out += `trailer\n<< /Size ${objects.length + 1} /Root ${catalog} 0 R >>\nstartxref\n${xref}\n%%EOF\n`;
  return Buffer.from(out, "latin1");
}
