//! `lading.starter`: the project a new partner starts from, for an organisation that hauls
//! tendered loads (`carrier`) or tenders loads to carriers (`broker`). Every starter checks
//! clean and its tests pass; the onboarding agent and the studio build on it.

use serde::{Deserialize, Serialize};

use super::project::SourceFile;

/// What the starter is for.
#[derive(Debug, Clone, Deserialize)]
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
}

/// A starter project.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct Starter {
    /// Its files.
    pub files: Vec<StarterFile>,
}

/// One file.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct StarterFile {
    /// Its path.
    pub path: String,
    /// Its text.
    pub source: String,
}

const PARTNER: &str = r#"partner {{name}} "{{label}}"

interchange {
  them {{themQualifier}} "{{themId}}"
  us {{usQualifier}} "{{usId}}"
  version 004010
  separators "*" "~" ">" "^"
  usage test
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
  expect x12 contains "B1*"
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
  expect x12 contains "AT7*"
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
  expect x12 contains "L3*****100000~"
}
"#;

const CARRIER_FLOW: &str = r#"flow tender with {{name}}

receive 204 as tender
  via tenderIn, rules tenderChecks
  key tender.key
  acknowledge with 997
  on Original -> create
  on Change   -> update stops, commodities, refs keep customer, total
    hold when tender.total != current.total "{{label}} changed the rate"
  on Cancel   -> cancel unless dispatched else hold "{{label}} cancelled after dispatch"

respond 990 via responseOut
  accept automatically when partner.autoAccept and not held and not declined
  decline automatically when declined and partner.autoDecline
  otherwise planner decides within 2 h else escalate "{{label}} tender {tender.key} needs an answer"

send 214 via statusOut on StopArrived, StopDeparted, Delivered, EtaChanged batch within 5 min until Delivered

send 210 via invoiceOut on InvoicePosted

expect 997 for every send within 30 min else resend 1 then escalate "{{label}} has not acknowledged {message.set} {message.control}"
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

const BROKER_FLOW: &str = r#"flow carrierTender with {{name}}

send 204 via tenderOut on TenderOffered, TenderCancelled key tender.key

receive 990 as response
  via responseIn
  key response.key
  acknowledge with 997
  -> apply

receive 214 as status
  via statusIn
  key status.key
  acknowledge with 997
  -> apply

receive 210 as invoice
  via invoiceIn
  key invoice.key ?? invoice.number
  acknowledge with 997
  -> apply

expect 997 for every send within 30 min else resend 1 then escalate "{{label}} has not acknowledged {message.set} {message.control}"
"#;

/// The starter project for a request.
#[must_use]
pub fn starter(request: &StarterRequest) -> Starter {
    let name = sanitize(&request.name);
    let label = if request.label.trim().is_empty() {
        name.clone()
    } else {
        request.label.replace('"', "'")
    };
    let (them_qualifier, them_id) = request
        .them
        .clone()
        .unwrap_or_else(|| ("ZZ".into(), name.to_uppercase()));
    let (us_qualifier, us_id) = request
        .us
        .clone()
        .unwrap_or_else(|| ("ZZ".into(), "TRENOVA".into()));
    let zone = request
        .zone
        .clone()
        .filter(|zone| super::tz::Zone::named(zone).is_some())
        .unwrap_or_else(|| "America/Chicago".into());
    let fill = |template: &str| {
        template
            .replace("{{name}}", &name)
            .replace("{{label}}", &label)
            .replace("{{themQualifier}}", &them_qualifier)
            .replace("{{themId}}", &them_id.replace('"', ""))
            .replace("{{usQualifier}}", &us_qualifier)
            .replace("{{usId}}", &us_id.replace('"', ""))
            .replace("{{zone}}", &zone)
    };
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
    let role_files: &[(&str, &str)] = if request.role == "broker" {
        &[
            ("maps/tender-out.lading", TENDER_OUT),
            ("maps/response-in.lading", RESPONSE_IN),
            ("maps/status-in.lading", STATUS_IN),
            ("maps/invoice-in.lading", INVOICE_IN),
            ("flows/carrier-tender.lading", BROKER_FLOW),
        ]
    } else {
        &[
            ("maps/tender-in.lading", TENDER_IN),
            ("rules/tender-checks.lading", TENDER_CHECKS),
            ("maps/response-out.lading", RESPONSE_OUT),
            ("maps/status-out.lading", STATUS_OUT),
            ("maps/invoice-out.lading", INVOICE_OUT),
            ("flows/tender.lading", CARRIER_FLOW),
        ]
    };
    for (path, template) in role_files {
        files.push(StarterFile {
            path: (*path).to_owned(),
            source: fill(template),
        });
    }
    Starter { files }
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
    use crate::lading::project::{Context, Org, Project};
    use crate::lading::testing::run_tests;

    fn checked(role: &str) {
        let request = StarterRequest {
            name: "Swift Logistics".into(),
            label: "Swift".into(),
            role: role.into(),
            them: None,
            us: None,
            zone: None,
        };
        let project = Project {
            files: starter(&request)
                .files
                .into_iter()
                .map(Into::into)
                .collect(),
            samples: Vec::new(),
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
        assert!(failing.is_empty(), "{}", failing.join("\n"));
    }

    #[test]
    fn the_carrier_starter_checks_clean() {
        checked("carrier");
    }

    #[test]
    fn the_broker_starter_checks_clean() {
        checked("broker");
    }

    #[test]
    fn names_become_lading_names() {
        assert_eq!(sanitize("Swift Logistics"), "swiftLogistics");
        assert_eq!(sanitize("3M Co"), "mCo");
        assert_eq!(sanitize("for"), "partnerfor");
    }
}
