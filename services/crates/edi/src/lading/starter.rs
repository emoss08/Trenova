//! `lading.starter`: the project a new partner starts from, for an organisation that hauls
//! tendered loads (`carrier`) or tenders loads to carriers (`broker`). Every starter checks
//! clean and its tests pass; the onboarding agent and the studio build on it.

use serde::{Deserialize, Serialize};

use super::project::SourceFile;

/// What the starter is for.
#[derive(Debug, Clone, Default, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct StarterRequest {
    /// The partner's file-level name (`acme`).
    pub name: String,
    /// What people call it.
    #[serde(default)]
    pub label: String,
    /// `carrier` (the partner tenders loads to the organisation) or `broker` (the
    /// organisation tenders loads to the partner).
    pub role: String,
    /// The partner's ISA qualifier and ID.
    #[serde(default)]
    pub them: Option<(String, String)>,
    /// The organisation's ISA qualifier and ID.
    #[serde(default)]
    pub us: Option<(String, String)>,
    /// The partner's time zone.
    #[serde(default)]
    pub zone: Option<String>,
    /// The GS application codes: the partner's, then the organisation's.
    #[serde(default)]
    pub group: Option<(String, String)>,
    /// The X12 version (`004010`).
    #[serde(default)]
    pub version: Option<String>,
    /// The element, segment, component and repetition separators Trenova writes.
    #[serde(default)]
    pub separators: Option<Vec<String>>,
    /// `test` or `production`.
    #[serde(default)]
    pub usage: Option<String>,
    /// The functional acknowledgment: `997`, `999` or `none`.
    #[serde(default)]
    pub acknowledge: Option<String>,
    /// How many minutes the partner has to acknowledge what Trenova sends.
    #[serde(default)]
    pub ack_within: Option<u32>,
    /// The sets the partner exchanges (`204`, `990`, `214`, `210`); every set of the role when
    /// empty.
    #[serde(default)]
    pub sets: Vec<String>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Starter {
    /// Its files.
    pub files: Vec<StarterFile>,
    /// Its samples: one message of each set it reads, which its flow tests receive.
    pub samples: Vec<StarterSample>,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StarterFile {
    /// Its path.
    pub path: String,
    /// Its text.
    pub source: String,
}

#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StarterSample {
    /// Its path.
    pub path: String,
    /// Its text.
    pub content: String,
}

const PARTNER: &str = r#"partner {{name}} "{{label}}"

interchange {
  them {{themQualifier}} "{{themId}}"
  us {{usQualifier}} "{{usId}}"
{{group}}  version {{version}}
  separators {{separators}}
  usage {{usage}}
}

zone {{zone}}
currency "USD"

/// Accept tenders automatically, without a planner.
setting autoAccept: Bool = false
/// Send a decline automatically when a rule declines a tender.
setting autoDecline: Bool = false
/// The heaviest load accepted automatically.
setting maxWeight: Weight = 45_000 lb in 1 lb..80_000 lb
"#;

const COMMON: &str = r#"library common

crosswalk stopTypes: Code<S502> <-> StopType {
  "LD", "CL", "PL" <-> Pickup
  "UL", "CU", "PU" <-> Delivery
}

crosswalk purposes: Code<B2A01> <-> TenderPurpose {
  "00" <-> Original
  "01" <-> Cancel
  "04" <-> Change
  "05" -> Change
  else -> Original
}

crosswalk payments: Code<B206> <-> Payment {
  "PP" <-> Prepaid
  "CC" <-> Collect
  "TP" <-> ThirdParty
}

crosswalk decisions: Code<B104> <-> Decision {
  "A" <-> Accept
  "D" <-> Decline
}

crosswalk statusCodes: Code<AT701> <-> EventKind {
  "AF" <-> DepartedPickup
  "X3" <-> ArrivedPickup
  "X1" <-> ArrivedDelivery
  "D1" <-> Delivered
  "CD" <-> DepartedDelivery
  "X6" <-> EnRoute
  "AG" <-> EtaChanged
  "SD" <-> Delayed
  "AA" <-> AppointmentSet
  "A9" <-> Exception
  "CP" <-> Loaded
  "P1" <-> Dispatched
}

/// The appointment window from a stop's dates.
fn window(dates: [Segment<G62>], zone: Zone) -> Window? {
  let opens = dates.find(d => d.01 in ["37", "53", "69", "70", "10", "68"])
  let closes = dates.find(d => d.01 in ["38", "54"])
  Window {
    start <- at(opens.02, opens.04, zone)
    end   <- at(closes.02, closes.04, zone)
  }
}

test "a window opens at its first date" {
  expect window([], partner.zone) == none
}
"#;

const TENDER_IN: &str = r#"map tenderIn
  from x12.004010.204
  to   loadTender
  use  common

key      <- B204 else reject "The tender has no shipment ID"
purpose  <- B2A01 via purposes
payment  <- B206 via payments
bol      <- L11[02 == "BM"].01
poNumber <- L11[02 == "PO"].01
refs     <- refs(L11)
customer <- lookup Customer(0100[N101 == "BT"].N104)
weight   <- weight(L301, L312 ?? "L")
total    <- money(L305, partner.currency)

stops <- each 0300 as stop {
  let party = stop.0310
  sequence <- stop.S501
  type     <- stop.S502 via stopTypes else reject "Stop {stop.S501} has no type"
  location <- lookup Location(party.N104) learn address(party)
  name     <- party.N102.titleCase()
  address  <- address(party)
  window   <- window(stop.G62, location.zone ?? partner.zone)
    else reject "Stop {stop.S501} has no appointment"
}
"#;

const TENDER_CHECKS: &str = r#"rules tenderChecks for tenderIn

rule "pickup comes before delivery" {
  require stops.first.window.start <= stops.last.window.start
    else decline "Delivery is scheduled before pickup"
}

rule "within what we haul automatically" {
  require (weight ?? 0 lb) <= partner.maxWeight
    else hold "Over {partner.maxWeight}: a planner decides"
}
"#;

const RESPONSE_OUT: &str = r#"map responseOut
  from tenderResponse
  to   x12.004010.990
  use  common

B1 {
  01 <- scac ?? org.scac
  02 <- key
  03 <- respondedAt
  04 <- decision via decisions
}
N9 when exists reason { 01 <- "ZZ", 02 <- reason.truncate(30) }

property "every decision writes a reservation code" {
  vary decision in Decision
  given tenderResponse { key: "T-1", decision: decision, respondedAt: @2026-01-10T14:30Z }
  expect x12 contains "B1{{e}}"
}
"#;

const STATUS_OUT: &str = r#"map statusOut
  from shipmentStatus
  to   x12.004010.214
  use  common

B10 { 01 <- proNumber, 02 <- key, 03 <- org.scac }
L11 when exists bol { 01 <- bol, 02 <- "BM" }

0205 <- each events as event {
  LX { 01 <- index + 1 }
  0210 {
    AT7 {
      01 <- event.kind via statusCodes
      02 <- event.reason ?? "NS"
      05 <- event.at
      06 <- event.at
      07 <- event.at.zoneCode()
    }
    MS1 when exists event.city { 01 <- event.city, 02 <- event.state, 03 <- event.country }
  }
}

property "every event writes a status" {
  vary kind in EventKind
  given shipmentStatus { key: "T-1", events: [{ kind: kind, at: @2026-01-10T14:30Z }] }
  expect x12 contains "AT7{{e}}"
}
"#;

const INVOICE_OUT: &str = r#"map invoiceOut
  from freightInvoice
  to   x12.004010.210

B3 {
  02 <- number
  03 <- key
  04 <- "PP"
  06 <- date
  07 <- total
  09 <- deliveryDate
  10 <- if exists deliveryDate then "035" else none
  11 <- org.scac
}

each refs as ref {
  N9 { 01 <- ref.key, 02 <- ref.value }
}

0400 <- each lines as line {
  LX { 01 <- line.sequence }
  L1 { 01 <- line.sequence, 04 <- line.amount }
}

L3 { 05 <- total }

test "an invoice writes its total" {
  given freightInvoice { number: "INV-1", date: @2026-01-12, total: $1_000.00 }
  expect x12 contains "L3{{e}}{{e}}{{e}}{{e}}{{e}}100000{{s}}"
}
"#;

const CARRIER_RECEIVE: &str = r#"
receive 204 as tender
  via tenderIn, rules tenderChecks
  key tender.key
  acknowledge with {{ack}}
  on Original -> create
  on Change   -> update stops, commodities, refs keep customer, total
    hold when tender.total != current.total "{{label}} changed the rate"
  on Cancel   -> cancel unless dispatched else hold "{{label}} cancelled after dispatch"
"#;

const CARRIER_RESPOND: &str = r#"
respond 990 via responseOut
  accept automatically when partner.autoAccept and not held and not declined
  decline automatically when declined and partner.autoDecline
  otherwise planner decides within 2 h else escalate "{{label}} tender {tender.key} needs an answer"
"#;

const CARRIER_STATUS: &str = r"
send 214 via statusOut on StopArrived, StopDeparted, Delivered, EtaChanged batch within 5 min until Delivered
";

const CARRIER_INVOICE: &str = r"
send 210 via invoiceOut on InvoicePosted
";

const EXPECT: &str = r#"
expect {{ack}} for every send within {{ackWithin}} min else resend 1 then escalate "{{label}} has not acknowledged {message.set} {message.control}"
"#;

const CARRIER_TENDER_TESTS: &str = r#"
test "a new tender becomes a load" {
  receive file "samples/tender-original.x12"
  expect created
}

test "a change to the rate is held for a planner" {
  receive file "samples/tender-original.x12"
  receive file "samples/tender-change.x12"
  expect held "{{label}} changed the rate"
}

test "a cancellation cancels the load" {
  receive file "samples/tender-original.x12"
  receive file "samples/tender-cancel.x12"
  expect cancelled
}
"#;

const CARRIER_ACK_TEST: &str = r#"
test "a tender is acknowledged" {
  receive file "samples/tender-original.x12"
  expect sent {{ack}}
}
"#;

const CARRIER_RESPOND_TESTS: &str = r#"
test "an accepted tender is answered" {
  receive file "samples/tender-original.x12"
  respond accept
  expect sent 990 accept
}

test "a declined tender is answered" {
  receive file "samples/tender-original.x12"
  respond decline "No capacity"
  expect sent 990 decline
}

test "an unanswered tender escalates" {
  receive file "samples/tender-original.x12"
  advance 2 h
  expect escalated "{{label}} tender L-1001 needs an answer"
}
"#;

const CARRIER_STATUS_TEST: &str = r#"
test "an arrival is sent as a status" {
  event StopArrived {
    key: "L-1001"
    events: [{ kind: ArrivedDelivery, at: @2026-01-10T14:30Z, stop: 2, city: "Chicago", state: "IL" }]
  }
  advance 5 min
  expect sent 214
}
"#;

const CARRIER_INVOICE_TEST: &str = r#"
test "a posted invoice is sent" {
  event InvoicePosted { number: "INV-1", date: @2026-01-12, key: "L-1001", total: $1_000.00 }
  expect sent 210
}
"#;

const TENDER_OUT: &str = r#"map tenderOut
  from loadTender
  to   x12.004010.204
  use  common

B2 { 02 <- org.scac, 04 <- key, 06 <- payment via payments ?? "PP" }
B2A { 01 <- purpose via purposes ?? "00" }
L11 when exists bol { 01 <- bol, 02 <- "BM" }
L11 when exists poNumber { 01 <- poNumber, 02 <- "PO" }

0300 <- each stops as stop {
  S5 { 01 <- stop.sequence, 02 <- stop.type via stopTypes }
  G62 {
    01 <- if stop.type == Pickup then "37" else "53"
    02 <- stop.window.start
    03 <- "U"
    04 <- stop.window.start
    05 <- stop.window.start.zoneCode()
  }
  0310 {
    N1 { 01 <- if stop.type == Pickup then "SH" else "CN", 02 <- stop.name ?? stop.location.label }
    N3 when exists stop.address { 01 <- stop.address.line1, 02 <- stop.address.line2 }
    N4 when exists stop.address {
      01 <- stop.address.city
      02 <- stop.address.state
      03 <- stop.address.postal
      04 <- stop.address.country
    }
  }
}

L3 { 01 <- weight.value(lb), 02 <- if exists weight then "G" else none, 05 <- total, 12 <- if exists weight then "L" else none }
"#;

const RESPONSE_IN: &str = r#"map responseIn
  from x12.004010.990
  to   tenderResponse
  use  common

key         <- B102
decision    <- B104 via decisions else reject "The response has no decision"
scac        <- B101
respondedAt <- at(B103, none, partner.zone) ?? now()
reason      <- N902
"#;

const STATUS_IN: &str = r#"map statusIn
  from x12.004010.214
  to   shipmentStatus
  use  common

key       <- B1002 else reject "The status has no shipment ID"
proNumber <- B1001
refs      <- refs(L11)

events <- each 0205 as item {
  let status = item.0210.first
  kind    <- status.AT701 via statusCodes else Exception
  reason  <- status.AT702
  at      <- at(status.AT705, status.AT706, zoneFromCode(status.AT707 ?? "") ?? partner.zone)
    else reject "A status has no date"
  local   <- not exists zoneFromCode(status.AT707 ?? "")
  stop    <- int(item.L11[02 == "QN"].01)
  city    <- status.MS101
  state   <- status.MS102
  country <- status.MS103
}
"#;

const INVOICE_IN: &str = r#"map invoiceIn
  from x12.004010.210
  to   freightInvoice

number       <- B302
date         <- B306
key          <- B303
deliveryDate <- B309
total        <- money(B307, partner.currency)
refs         <- refs(N9)

lines <- each 0400 as line {
  sequence    <- line.LX01
  code        <- line.L1.first.08
  amount      <- money(line.L1.first.04, partner.currency) else reject "Line {line.LX01} has no charge"
  description <- line.L5.first.02
}
"#;

const BROKER_TENDER: &str = r"
send 204 via tenderOut on TenderOffered, TenderCancelled key tender.key
";

const BROKER_RESPONSE: &str = r"
receive 990 as response
  via responseIn
  key response.key
  acknowledge with {{ack}}
  -> apply
";

const BROKER_STATUS: &str = r"
receive 214 as status
  via statusIn
  key status.key
  acknowledge with {{ack}}
  -> apply
";

const BROKER_INVOICE: &str = r"
receive 210 as invoice
  via invoiceIn
  key invoice.key ?? invoice.number
  acknowledge with {{ack}}
  -> apply
";

const BROKER_TENDER_TEST: &str = r#"
test "an offered load is tendered" {
  event TenderOffered {
    key: "L-1001"
    stops: [
      { sequence: 1, type: Pickup, window: { start: @2026-01-10T08:00[America/Chicago] } },
      { sequence: 2, type: Delivery, window: { start: @2026-01-11T10:00[America/Chicago] } }
    ]
  }
  expect sent 204
}
"#;

const BROKER_RESPONSE_TEST: &str = r#"
test "a carrier's answer is applied" {
  receive file "samples/response.x12"
  expect applied
}
"#;

const BROKER_STATUS_TEST: &str = r#"
test "a carrier's status is applied" {
  receive file "samples/status.x12"
  expect applied
}
"#;

const BROKER_INVOICE_TEST: &str = r#"
test "a carrier's invoice is applied" {
  receive file "samples/invoice.x12"
  expect applied
}
"#;

/// The ISA ID a partner gets when the request names none: its name's letters and digits,
/// uppercased, cut to the 15 characters ISA06 holds.
fn default_interchange_id(name: &str) -> String {
    name.chars()
        .filter(char::is_ascii_alphanumeric)
        .take(15)
        .collect::<String>()
        .to_ascii_uppercase()
}

/// The envelope a starter writes, from the request with Trenova's defaults.
struct Envelope {
    them: (String, String),
    us: (String, String),
    group: Option<(String, String)>,
    version: String,
    separators: [char; 4],
    usage: &'static str,
    ack: &'static str,
    ack_within: u32,
}

impl Envelope {
    fn of(request: &StarterRequest, name: &str) -> Self {
        let clean = |text: &str| text.replace('"', "");
        let them = request.them.clone().map_or_else(
            || ("ZZ".to_owned(), default_interchange_id(name)),
            |(qualifier, id)| (qualifier, clean(&id)),
        );
        let us = request.us.clone().map_or_else(
            || ("ZZ".to_owned(), "TRENOVA".to_owned()),
            |(qualifier, id)| (qualifier, clean(&id)),
        );
        let version = request
            .version
            .clone()
            .filter(|version| version.len() == 6 && version.chars().all(|c| c.is_ascii_digit()))
            .unwrap_or_else(|| "004010".to_owned());
        let separators = request
            .separators
            .as_ref()
            .and_then(|given| {
                let chars: Vec<char> = given
                    .iter()
                    .filter_map(|text| {
                        let mut chars = text.chars();
                        match (chars.next(), chars.next()) {
                            (Some(c), None)
                                if !c.is_alphanumeric() && !matches!(c, '"' | '{' | '}' | ' ') =>
                            {
                                Some(c)
                            }
                            _ => None,
                        }
                    })
                    .collect();
                let distinct = chars
                    .iter()
                    .collect::<std::collections::BTreeSet<_>>()
                    .len();
                (chars.len() == 4 && distinct == 4)
                    .then(|| [chars[0], chars[1], chars[2], chars[3]])
            })
            .unwrap_or(['*', '~', '>', '^']);
        let usage = match request.usage.as_deref() {
            Some("production") => "production",
            _ => "test",
        };
        let ack = match request.acknowledge.as_deref() {
            Some("999") => "999",
            Some("none") => "none",
            _ => "997",
        };
        Self {
            them,
            us,
            group: request
                .group
                .clone()
                .map(|(theirs, ours)| (clean(&theirs), clean(&ours))),
            version,
            separators,
            usage,
            ack,
            ack_within: request
                .ack_within
                .filter(|minutes| (1..=10_080).contains(minutes))
                .unwrap_or(30),
        }
    }

    /// An interchange from the partner holding one transaction set.
    fn sample(&self, set: &str, group: &str, control: u32, body: &[&str]) -> String {
        let [element, segment, component, repetition] = self.separators;
        let pad = |text: &str| format!("{text:<15}");
        let (theirs, ours) = self
            .group
            .clone()
            .unwrap_or_else(|| (self.them.1.clone(), self.us.1.clone()));
        let header = [
            "ISA".to_owned(),
            "00".to_owned(),
            " ".repeat(10),
            "00".to_owned(),
            " ".repeat(10),
            self.them.0.clone(),
            pad(&self.them.1),
            self.us.0.clone(),
            pad(&self.us.1),
            "260109".to_owned(),
            "1200".to_owned(),
            repetition.to_string(),
            "00401".to_owned(),
            format!("{control:09}"),
            "0".to_owned(),
            if self.usage == "production" { "P" } else { "T" }.to_owned(),
            component.to_string(),
        ];
        let mut lines = vec![header.join(&element.to_string())];
        lines.push(
            [
                "GS",
                group,
                &theirs,
                &ours,
                "20260109",
                "1200",
                &control.to_string(),
                "X",
                &self.version,
            ]
            .join(&element.to_string()),
        );
        lines.push(format!("ST{element}{set}{element}0001"));
        lines.extend(
            body.iter()
                .map(|line| line.replace('*', &element.to_string())),
        );
        lines.push(format!("SE{element}{}{element}0001", body.len() + 2));
        lines.push(format!("GE{element}1{element}{control}"));
        lines.push(format!("IEA{element}1{element}{control:09}"));
        lines
            .into_iter()
            .map(|line| format!("{line}{segment}"))
            .collect::<Vec<_>>()
            .join("\n")
            + "\n"
    }
}

const TENDER_BODY: &[&str] = &[
    "B2**TRNV**L-1001**PP",
    "B2A*00*LT",
    "L11*BOL-55*BM",
    "N1*BT*SHIPPER*93*BT-1",
    "S5*1*LD",
    "G62*69*20260110*U*0800",
    "N1*SH*ORIGIN DC*93*DC-1",
    "N3*100 MAIN ST",
    "N4*DALLAS*TX*75201*US",
    "S5*2*UL",
    "G62*70*20260111*X*1000",
    "N1*CN*DESTINATION*93*ST-7",
    "N3*5 ELM ST",
    "N4*CHICAGO*IL*60601*US",
    "L3*42000*G***185000*******L",
];

const RESPONSE_BODY: &[&str] = &["B1*SWFT*L-1001*20260110*A"];

const STATUS_BODY: &[&str] = &[
    "B10*PRO-9*L-1001*SWFT",
    "LX*1",
    "AT7*X1*NS***20260110*1430*CT",
    "MS1*CHICAGO*IL*US",
];

const INVOICE_BODY: &[&str] = &[
    "B3**INV-1*L-1001*PP**20260112*100000**20260111*035*SWFT",
    "LX*1",
    "L1*1***100000",
    "L3*****100000",
];

const LOAD_TENDER_DOCUMENT: &str = r#"{
  "key": "L-1001",
  "purpose": "Original",
  "bol": "BOL-55",
  "weight": { "value": 42000, "unit": "lb" },
  "total": { "amount": "1850.00", "currency": "USD" },
  "refs": { "BM": "BOL-55" },
  "stops": [
    { "sequence": 1, "type": "Pickup", "name": "ORIGIN DC", "window": { "start": "2026-01-10T14:00:00Z" } },
    { "sequence": 2, "type": "Delivery", "name": "DESTINATION", "window": { "start": "2026-01-11T16:00:00Z" } }
  ]
}
"#;

const TENDER_RESPONSE_DOCUMENT: &str = r#"{
  "key": "L-1001",
  "decision": "Accept",
  "scac": "TRNV",
  "respondedAt": "2026-01-09T18:00:00Z"
}
"#;

const SHIPMENT_STATUS_DOCUMENT: &str = r#"{
  "key": "L-1001",
  "proNumber": "PRO-9",
  "events": [
    { "kind": "ArrivedDelivery", "at": "2026-01-11T15:30:00Z", "stop": 2, "city": "Chicago", "state": "IL", "country": "US" }
  ]
}
"#;

const FREIGHT_INVOICE_DOCUMENT: &str = r#"{
  "number": "INV-1001",
  "date": "2026-01-12",
  "key": "L-1001",
  "deliveryDate": "2026-01-11",
  "total": { "amount": "2531.10", "currency": "USD" },
  "billTo": { "name": "Acme Manufacturing Co" },
  "lines": [
    { "sequence": 1, "description": "Linehaul", "rate": "2.45", "amount": { "amount": "2145.00", "currency": "USD" } },
    { "sequence": 2, "code": "FUE", "description": "Fuel surcharge", "amount": { "amount": "386.10", "currency": "USD" } }
  ],
  "refs": { "BM": "BOL-77120" }
}
"#;

/// A message the partner might send of one transaction set, in the envelope the request
/// describes: the starter's own message of the set, or none for a set it has none of.
#[must_use]
pub fn set_sample(request: &StarterRequest, set: &str) -> Option<String> {
    let (group, control, body) = match set {
        "204" => ("SM", 101, TENDER_BODY),
        "990" => ("GF", 201, RESPONSE_BODY),
        "214" => ("QM", 202, STATUS_BODY),
        "210" => ("IM", 203, INVOICE_BODY),
        _ => return None,
    };
    let envelope = Envelope::of(request, &sanitize(&request.name));
    Some(envelope.sample(set, group, control, body))
}

/// An example Trenova document of a record, as an outbound map's sample holds it.
#[must_use]
pub fn document_sample(document: &str) -> Option<&'static str> {
    match document {
        "loadTender" => Some(LOAD_TENDER_DOCUMENT),
        "tenderResponse" => Some(TENDER_RESPONSE_DOCUMENT),
        "shipmentStatus" => Some(SHIPMENT_STATUS_DOCUMENT),
        "freightInvoice" => Some(FREIGHT_INVOICE_DOCUMENT),
        _ => None,
    }
}

/// The starter project for a request.
#[must_use]
#[allow(clippy::too_many_lines)]
pub fn starter(request: &StarterRequest) -> Starter {
    let name = sanitize(&request.name);
    let label = if request.label.trim().is_empty() {
        name.clone()
    } else {
        request.label.replace('"', "'").replace(['{', '}'], "")
    };
    let envelope = Envelope::of(request, &name);
    let zone = request
        .zone
        .clone()
        .filter(|zone| super::tz::Zone::named(zone).is_some())
        .unwrap_or_else(|| "America/Chicago".into());
    let separators = envelope
        .separators
        .iter()
        .map(|c| format!("\"{}\"", c.escape_default()))
        .collect::<Vec<_>>()
        .join(" ");
    let group = envelope
        .group
        .as_ref()
        .map(|(theirs, ours)| format!("  group \"{theirs}\" <-> \"{ours}\"\n"))
        .unwrap_or_default();
    let fill = |template: &str| {
        template
            .replace("{{name}}", &name)
            .replace("{{label}}", &label)
            .replace("{{themQualifier}}", &envelope.them.0)
            .replace("{{themId}}", &envelope.them.1)
            .replace("{{usQualifier}}", &envelope.us.0)
            .replace("{{usId}}", &envelope.us.1)
            .replace("{{group}}", &group)
            .replace("{{version}}", &envelope.version)
            .replace("{{separators}}", &separators)
            .replace("{{usage}}", envelope.usage)
            .replace("{{zone}}", &zone)
            .replace("{{ack}}", envelope.ack)
            .replace("{{ackWithin}}", &envelope.ack_within.to_string())
            .replace(
                "{{e}}",
                &envelope.separators[0].escape_default().to_string(),
            )
            .replace(
                "{{s}}",
                &envelope.separators[1].escape_default().to_string(),
            )
    };
    let wants =
        |set: &str| request.sets.is_empty() || request.sets.iter().any(|wanted| wanted == set);
    let mut files = vec![
        StarterFile {
            path: "partner.lading".into(),
            source: fill(PARTNER),
        },
        StarterFile {
            path: "lib/common.lading".into(),
            source: fill(COMMON),
        },
    ];
    let mut samples = Vec::new();
    let mut flow;
    let mut tests = String::new();
    let mut sends = false;
    let broker = request.role == "broker";
    let source_file = |path: &str, template: &str| StarterFile {
        path: path.to_owned(),
        source: fill(template),
    };
    if broker {
        flow = format!("flow carrierTender with {name}\n");
        if wants("204") {
            files.push(source_file("maps/tender-out.lading", TENDER_OUT));
            flow.push_str(BROKER_TENDER);
            tests.push_str(BROKER_TENDER_TEST);
            sends = true;
        }
        if wants("990") {
            files.push(source_file("maps/response-in.lading", RESPONSE_IN));
            flow.push_str(BROKER_RESPONSE);
            tests.push_str(BROKER_RESPONSE_TEST);
            samples.push(StarterSample {
                path: "samples/response.x12".into(),
                content: envelope.sample("990", "GF", 201, RESPONSE_BODY),
            });
        }
        if wants("214") {
            files.push(source_file("maps/status-in.lading", STATUS_IN));
            flow.push_str(BROKER_STATUS);
            tests.push_str(BROKER_STATUS_TEST);
            samples.push(StarterSample {
                path: "samples/status.x12".into(),
                content: envelope.sample("214", "QM", 202, STATUS_BODY),
            });
        }
        if wants("210") {
            files.push(source_file("maps/invoice-in.lading", INVOICE_IN));
            flow.push_str(BROKER_INVOICE);
            tests.push_str(BROKER_INVOICE_TEST);
            samples.push(StarterSample {
                path: "samples/invoice.x12".into(),
                content: envelope.sample("210", "IM", 203, INVOICE_BODY),
            });
        }
    } else {
        flow = format!("flow tender with {name}\n");
        if wants("204") {
            files.push(source_file("maps/tender-in.lading", TENDER_IN));
            files.push(source_file("rules/tender-checks.lading", TENDER_CHECKS));
            flow.push_str(CARRIER_RECEIVE);
            tests.push_str(CARRIER_TENDER_TESTS);
            if envelope.ack != "none" {
                tests.push_str(CARRIER_ACK_TEST);
            }
            let changed: Vec<&str> = TENDER_BODY
                .iter()
                .map(|line| match *line {
                    "B2A*00*LT" => "B2A*04*LT",
                    "L3*42000*G***185000*******L" => "L3*42000*G***195000*******L",
                    other => other,
                })
                .collect();
            let cancelled: Vec<&str> = TENDER_BODY
                .iter()
                .map(|line| {
                    if *line == "B2A*00*LT" {
                        "B2A*01*LT"
                    } else {
                        line
                    }
                })
                .collect();
            samples.push(StarterSample {
                path: "samples/tender-original.x12".into(),
                content: envelope.sample("204", "SM", 101, TENDER_BODY),
            });
            samples.push(StarterSample {
                path: "samples/tender-change.x12".into(),
                content: envelope.sample("204", "SM", 102, &changed),
            });
            samples.push(StarterSample {
                path: "samples/tender-cancel.x12".into(),
                content: envelope.sample("204", "SM", 103, &cancelled),
            });
            if wants("990") {
                files.push(source_file("maps/response-out.lading", RESPONSE_OUT));
                flow.push_str(CARRIER_RESPOND);
                tests.push_str(CARRIER_RESPOND_TESTS);
                sends = true;
            }
        }
        if wants("214") {
            files.push(source_file("maps/status-out.lading", STATUS_OUT));
            flow.push_str(CARRIER_STATUS);
            tests.push_str(CARRIER_STATUS_TEST);
            sends = true;
        }
        if wants("210") {
            files.push(source_file("maps/invoice-out.lading", INVOICE_OUT));
            flow.push_str(CARRIER_INVOICE);
            tests.push_str(CARRIER_INVOICE_TEST);
            sends = true;
        }
    }
    if sends && envelope.ack != "none" {
        flow.push_str(EXPECT);
    }
    flow.push_str(&tests);
    files.push(StarterFile {
        path: if broker {
            "flows/carrier-tender.lading".into()
        } else {
            "flows/tender.lading".into()
        },
        source: fill(&flow),
    });
    Starter { files, samples }
}

/// A partner name as a Lading name: camelCase letters and digits, starting with a letter.
#[must_use]
pub fn sanitize(name: &str) -> String {
    let mut out = String::new();
    let mut upper = false;
    for c in name.chars() {
        if c.is_ascii_alphanumeric() {
            if out.is_empty() {
                if c.is_ascii_alphabetic() {
                    out.push(c.to_ascii_lowercase());
                }
            } else if upper {
                out.push(c.to_ascii_uppercase());
            } else {
                out.push(c);
            }
            upper = false;
        } else {
            upper = !out.is_empty();
        }
    }
    if out.is_empty() || super::syntax::token::is_reserved(&out) {
        format!("partner{out}")
    } else {
        out
    }
}

impl From<StarterFile> for SourceFile {
    fn from(file: StarterFile) -> Self {
        Self {
            path: file.path,
            source: file.source,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{StarterRequest, sanitize, starter};
    use crate::lading::check::check;
    use crate::lading::project::{Context, Org, Project, Sample};
    use crate::lading::testing::run_tests;

    fn checked(request: &StarterRequest) {
        let role = request.role.as_str();
        let built = starter(request);
        let project = Project {
            files: built.files.into_iter().map(Into::into).collect(),
            samples: built
                .samples
                .into_iter()
                .map(|sample| Sample {
                    path: sample.path,
                    content: sample.content,
                })
                .collect(),
        };
        let context = Context {
            org: Org {
                name: "Trenova".into(),
                scac: "TRNV".into(),
                zone: "America/Chicago".into(),
            },
            ..Context::default()
        };
        let program = check(&project, &context);
        let found: Vec<String> = program
            .units
            .iter()
            .flat_map(|unit| {
                unit.diagnostics.iter().map(move |diagnostic| {
                    let line = unit.source[..diagnostic.span.start as usize]
                        .matches('\n')
                        .count()
                        + 1;
                    format!(
                        "{}:{line}: {} {}",
                        unit.path,
                        diagnostic.code,
                        diagnostic.message.english()
                    )
                })
            })
            .collect();
        assert!(found.is_empty(), "{role}:\n{}", found.join("\n"));
        let report = run_tests(&program);
        let failing: Vec<String> = report
            .tests
            .iter()
            .filter(|test| !test.passed)
            .map(|test| {
                format!(
                    "{}: {:?}",
                    test.name,
                    test.failures
                        .iter()
                        .map(|failure| failure.message.english())
                        .collect::<Vec<_>>()
                )
            })
            .collect();
        assert!(failing.is_empty(), "{role}:\n{}", failing.join("\n"));
        let uncovered: Vec<String> = report
            .coverage
            .iter()
            .filter(|file| program.units[file.unit].header().kind() == "flow")
            .flat_map(|file| {
                let source = &program.units[file.unit].source;
                file.uncovered.iter().map(move |span| {
                    source[span.start as usize..span.end as usize]
                        .lines()
                        .next()
                        .unwrap_or_default()
                        .to_owned()
                })
            })
            .collect();
        assert!(
            uncovered.is_empty(),
            "{role}: no test reaches {uncovered:?}"
        );
    }

    fn request(role: &str) -> StarterRequest {
        StarterRequest {
            name: "Swift Logistics".into(),
            label: "Swift".into(),
            role: role.into(),
            ..StarterRequest::default()
        }
    }

    #[test]
    fn the_carrier_starter_is_ready() {
        checked(&request("carrier"));
    }

    #[test]
    fn the_broker_starter_is_ready() {
        checked(&request("broker"));
    }

    #[test]
    fn a_starter_takes_the_partners_envelope_and_sets() {
        let mut carrier = request("carrier");
        carrier.them = Some(("01".into(), "123456789".into()));
        carrier.us = Some(("ZZ".into(), "TRNV-SWIFT".into()));
        carrier.group = Some(("SWIFT".into(), "TRENOVA".into()));
        carrier.separators = Some(vec!["|".into(), "\n".into(), ":".into(), "^".into()]);
        carrier.usage = Some("production".into());
        carrier.acknowledge = Some("999".into());
        carrier.ack_within = Some(60);
        carrier.sets = vec!["204".into(), "990".into()];
        checked(&carrier);
        let built = starter(&carrier);
        let partner = &built.files[0].source;
        assert!(
            partner.contains("group \"SWIFT\" <-> \"TRENOVA\""),
            "{partner}"
        );
        assert!(partner.contains("usage production"), "{partner}");
        let flow = &built.files.last().expect("flow").source;
        assert!(flow.contains("acknowledge with 999"), "{flow}");
        assert!(flow.contains("within 60 min"), "{flow}");
        assert!(!flow.contains("send 214"), "{flow}");
        assert!(
            built
                .files
                .iter()
                .all(|file| file.path != "maps/status-out.lading")
        );

        let mut quiet = request("broker");
        quiet.acknowledge = Some("none".into());
        checked(&quiet);
        let built = starter(&quiet);
        let flow = &built.files.last().expect("flow").source;
        assert!(!flow.contains("\nexpect "), "{flow}");
    }

    #[test]
    fn a_project_runs_without_the_samples_its_tests_read() {
        let built = starter(&request("carrier"));
        let sample = built.samples[0].content.clone();
        let project = Project {
            files: built.files.into_iter().map(Into::into).collect(),
            samples: Vec::new(),
        };
        let program = check(&project, &Context::default());
        assert!(program.has_errors());
        assert!(!program.has_code_errors());
        let result = crate::lading::eval::run::run(
            &program,
            &crate::lading::eval::run::RunRequest {
                map: "tenderIn".into(),
                input: crate::lading::eval::run::RunInput::X12(sample),
                settings: std::collections::BTreeMap::new(),
                lookups: Vec::new(),
                now: None,
                control: crate::lading::eval::run::Control::default(),
                rules: true,
            },
        );
        assert!(result.error.is_none(), "{:?}", result.error);
        assert_eq!(result.transactions.len(), 1);
    }

    #[test]
    fn a_long_partner_name_still_makes_an_interchange_id_that_fits() {
        let mut long = request("carrier");
        long.name = "Acme Manufacturing".into();
        long.label = "Acme Manufacturing".into();
        checked(&long);
        let built = starter(&long);
        assert!(
            built.files[0]
                .source
                .contains("them ZZ \"ACMEMANUFACTURI\""),
            "{}",
            built.files[0].source
        );
    }

    #[test]
    fn names_become_lading_names() {
        assert_eq!(sanitize("Swift Logistics"), "swiftLogistics");
        assert_eq!(sanitize("3M Co"), "mCo");
        assert_eq!(sanitize("for"), "partnerfor");
    }
}
