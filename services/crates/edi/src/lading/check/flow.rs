//! Checks flows: that each clause names maps, rules, events and purposes that exist and fit
//! together, and that each action suits the document it acts on.

use std::collections::{BTreeMap, BTreeSet};

use trenova_module_kit::diagnostic::{Diagnostic, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::env::Env;
use super::{Checker, MapInfo, Program, Source};
use crate::lading::facts::Target;
use crate::lading::schema;
use crate::lading::syntax::ast::{
    Action, ExpectAck, FlowDecl, Header, Ident, Item, Otherwise, Receive, Respond, Send,
};
use crate::lading::types::Type;

/// The name a flow gives a document it sends or answers with.
#[must_use]
pub fn document_binder(document: &str) -> &'static str {
    match document {
        "loadTender" => "tender",
        "tenderResponse" => "response",
        "shipmentStatus" => "status",
        _ => "invoice",
    }
}

/// The sets a flow may receive, respond with or send.
const SETS: &[&str] = &["204", "210", "214", "990"];

/// The flow that first took each received set, response and event, by name.
#[derive(Default)]
struct Routes {
    received: BTreeMap<String, String>,
    responded: BTreeMap<String, String>,
    events: BTreeMap<String, String>,
}

impl Checker<'_> {
    /// Checks a flow file.
    pub(crate) fn check_flow(&mut self, unit: usize, partner: &Ident) {
        let program: &Program = self.program;
        match &program.partner {
            Some(found) if found.name == partner.name => {
                self.occurrence(
                    unit,
                    partner.span,
                    Target::File(partner.name.clone()),
                    false,
                    false,
                );
            }
            Some(found) => {
                let name = found.name.clone();
                self.report(
                    unit,
                    Diagnostic::error(
                        "unknown-partner",
                        partner.span,
                        Message::with(
                            message!("This project's partner is {0}"),
                            [Arg::from(name.as_str())],
                        ),
                    )
                    .with_fix(super::replace_fix(partner.span, &name)),
                );
            }
            None => {}
        }
        let mut env = Env::new(unit);
        let clauses: Vec<FlowDecl> = program.units[unit]
            .items()
            .iter()
            .filter_map(|item| match item {
                Item::Flow(clause) => Some(clause.clone()),
                _ => None,
            })
            .collect();
        for clause in &clauses {
            if let FlowDecl::Receive(receive) = clause {
                let ty = receive
                    .map
                    .as_ref()
                    .and_then(|map| program.maps.get(&map.name))
                    .and_then(|info| info.document)
                    .map_or(Type::Any, Type::Record);
                if !receive.binder.name.is_empty() {
                    env.bind(&receive.binder.name, ty, receive.binder.span);
                    self.occurrence(
                        unit,
                        receive.binder.span,
                        Target::Local {
                            unit,
                            span: receive.binder.span,
                        },
                        true,
                        false,
                    );
                }
            }
        }
        let tender_received = clauses.iter().any(
            |clause| matches!(clause, FlowDecl::Receive(receive) if receive.set.name == "204"),
        );
        let mut received = BTreeSet::new();
        for clause in &clauses {
            match clause {
                FlowDecl::Receive(receive) => {
                    if !received.insert(receive.set.name.clone()) {
                        self.report(
                            unit,
                            Diagnostic::error(
                                "duplicate-receive",
                                receive.set.span,
                                Message::with(
                                    message!("The flow already receives {0}"),
                                    [Arg::from(receive.set.name.as_str())],
                                ),
                            ),
                        );
                    }
                    self.receive(&mut env, receive);
                }
                FlowDecl::Respond(respond) => self.respond(&mut env, respond, tender_received),
                FlowDecl::Send(send) => self.send(&mut env, send),
                FlowDecl::Expect(expect) => self.expect_ack(&mut env, expect),
            }
        }
    }

    /// Reports a set or event two clauses would both take. The runtime hands each message and
    /// each Trenova event to the first flow that takes it, so a second flow receiving the same
    /// set, responding with it or sending on the same event would never run.
    pub(crate) fn check_flow_routing(&mut self) {
        let program: &Program = self.program;
        let mut routes = Routes::default();
        for unit in 0..program.units.len() {
            let Header::Flow { name, .. } = program.units[unit].header() else {
                continue;
            };
            let flow = name.name.as_str();
            let mut pairs = BTreeSet::new();
            for item in program.units[unit].items() {
                match item {
                    Item::Flow(FlowDecl::Receive(receive)) => {
                        self.route_receive(&mut routes, unit, flow, &receive.set);
                    }
                    Item::Flow(FlowDecl::Respond(respond)) => {
                        self.route_respond(&mut routes, unit, flow, &respond.set);
                    }
                    Item::Flow(FlowDecl::Send(clause)) => {
                        for event in &clause.events {
                            if pairs.insert((clause.set.name.clone(), event.name.clone())) {
                                self.route_event(&mut routes, unit, flow, event);
                            } else {
                                self.report(
                                    unit,
                                    Diagnostic::error(
                                        "duplicate-send",
                                        event.span,
                                        Message::with(
                                            message!("The flow already sends {0} on {1}"),
                                            [
                                                Arg::from(clause.set.name.as_str()),
                                                Arg::from(event.name.as_str()),
                                            ],
                                        ),
                                    ),
                                );
                            }
                        }
                    }
                    _ => {}
                }
            }
        }
    }

    fn route_receive(&mut self, routes: &mut Routes, unit: usize, flow: &str, set: &Ident) {
        match routes.received.get(&set.name) {
            Some(other) if other != flow => self.report(
                unit,
                Diagnostic::error(
                    "flow-receive-taken",
                    set.span,
                    Message::with(
                        message!(
                            "Flow {0} already receives {1}, and each {1} goes to the first flow that receives it: receive it in one flow"
                        ),
                        [Arg::from(other.as_str()), Arg::from(set.name.as_str())],
                    ),
                ),
            ),
            Some(_) => {}
            None => {
                routes.received.insert(set.name.clone(), flow.to_owned());
            }
        }
    }

    fn route_respond(&mut self, routes: &mut Routes, unit: usize, flow: &str, set: &Ident) {
        let Some(other) = routes.responded.get(&set.name) else {
            routes.responded.insert(set.name.clone(), flow.to_owned());
            return;
        };
        let message = if other == flow {
            Message::with(
                message!("The flow already responds with {0}"),
                [Arg::from(set.name.as_str())],
            )
        } else {
            Message::with(
                message!("Flow {0} already responds with {1}: respond in one flow"),
                [Arg::from(other.as_str()), Arg::from(set.name.as_str())],
            )
        };
        self.report(
            unit,
            Diagnostic::error("flow-respond-taken", set.span, message),
        );
    }

    fn route_event(&mut self, routes: &mut Routes, unit: usize, flow: &str, event: &Ident) {
        match routes.events.get(&event.name) {
            Some(other) if other != flow => self.report(
                unit,
                Diagnostic::error(
                    "flow-event-taken",
                    event.span,
                    Message::with(
                        message!(
                            "Flow {0} already sends on {1}, and each event goes to the first flow that sends on it: send on it from one flow"
                        ),
                        [Arg::from(other.as_str()), Arg::from(event.name.as_str())],
                    ),
                ),
            ),
            Some(_) => {}
            None => {
                routes.events.insert(event.name.clone(), flow.to_owned());
            }
        }
    }

    fn set(&mut self, env: &Env, set: &Ident) -> bool {
        if SETS.contains(&set.name.as_str()) {
            return true;
        }
        self.report(
            env.unit,
            Diagnostic::error(
                "unknown-set",
                set.span,
                Message::with(
                    message!("A flow exchanges 204, 210, 214 or 990, not {0}"),
                    [Arg::from(set.name.as_str())],
                ),
            ),
        );
        false
    }

    /// Reports a map no file declares, offering the nearest of `names`. A map that is declared
    /// but whose header failed is reported in its own file, not at each use.
    pub(crate) fn unknown_map(&mut self, unit: usize, map: &Ident, names: Vec<&str>) {
        if self.program.declares_map(&map.name) {
            return;
        }
        let mut diagnostic = Diagnostic::error(
            "unknown-map",
            map.span,
            Message::with(
                message!("There is no map named {0}"),
                [Arg::from(map.name.as_str())],
            ),
        );
        if let Some(close) = nearest(&map.name, names) {
            diagnostic = diagnostic.with_fix(super::replace_fix(map.span, close));
        }
        self.report(unit, diagnostic);
    }

    /// The map a clause names, checked for its direction and set.
    fn flow_map(
        &mut self,
        env: &Env,
        map: Option<&Ident>,
        set: &Ident,
        inbound: bool,
        at: Span,
    ) -> Option<MapInfo> {
        let program: &Program = self.program;
        let Some(map) = map else {
            self.report(
                env.unit,
                Diagnostic::error(
                    "flow-map-missing",
                    at,
                    Message::new(message!("Name the map: via mapName")),
                ),
            );
            return None;
        };
        let Some(info) = program.maps.get(&map.name).cloned() else {
            let names = program
                .maps
                .iter()
                .filter(|(_, info)| info.inbound() == inbound)
                .map(|(name, _)| name.as_str())
                .collect();
            self.unknown_map(env.unit, map, names);
            return None;
        };
        self.occurrence(
            env.unit,
            map.span,
            Target::File(map.name.clone()),
            false,
            false,
        );
        if info.inbound() != inbound {
            self.report(
                env.unit,
                Diagnostic::error(
                    "flow-map-direction",
                    map.span,
                    if inbound {
                        Message::with(
                            message!("{0} writes X12; receiving needs a map that reads it"),
                            [Arg::from(map.name.as_str())],
                        )
                    } else {
                        Message::with(
                            message!("{0} reads X12; sending needs a map that writes it"),
                            [Arg::from(map.name.as_str())],
                        )
                    },
                ),
            );
            return None;
        }
        let map_set = info.guide().map(|guide| {
            program.guides[guide]
                .compiled
                .guide()
                .transaction_set
                .clone()
        });
        if let Some(map_set) = map_set
            && map_set != set.name
        {
            self.report(
                env.unit,
                Diagnostic::error(
                    "flow-map-set",
                    map.span,
                    Message::with(
                        message!("{0} is for a {1}, and this clause is about a {2}"),
                        [
                            Arg::from(map.name.as_str()),
                            Arg::from(map_set),
                            Arg::from(set.name.as_str()),
                        ],
                    ),
                ),
            );
        }
        if inbound && !matches!(info.source, Source::X12(_)) {
            self.report(
                env.unit,
                Diagnostic::error(
                    "flow-map-source",
                    map.span,
                    Message::new(message!(
                        "A flow receives X12; a map from CSV or JSON imports files"
                    )),
                ),
            );
        }
        Some(info)
    }

    #[allow(clippy::too_many_lines)]
    fn receive(&mut self, env: &mut Env, receive: &Receive) {
        let program: &Program = self.program;
        self.set(env, &receive.set);
        let info = self.flow_map(env, receive.map.as_ref(), &receive.set, true, receive.span);
        let document = info
            .as_ref()
            .and_then(|info| info.document)
            .and_then(schema::document);
        if let Some(rules) = &receive.rules {
            if let Some((_, for_map)) = program.rules.get(&rules.name) {
                self.occurrence(
                    env.unit,
                    rules.span,
                    Target::File(rules.name.clone()),
                    false,
                    false,
                );
                if receive.map.as_ref().is_some_and(|map| map.name != *for_map) {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "flow-rules-map",
                            rules.span,
                            Message::with(
                                message!("{0} checks {1}, not the map this clause reads with"),
                                [Arg::from(rules.name.as_str()), Arg::from(for_map.as_str())],
                            ),
                        ),
                    );
                }
            } else {
                let names: Vec<&str> = program.rules.keys().map(String::as_str).collect();
                let mut diagnostic = Diagnostic::error(
                    "unknown-rules",
                    rules.span,
                    Message::with(
                        message!("There are no rules named {0}"),
                        [Arg::from(rules.name.as_str())],
                    ),
                );
                if let Some(close) = nearest(&rules.name, names) {
                    diagnostic = diagnostic.with_fix(super::replace_fix(rules.span, close));
                }
                self.report(env.unit, diagnostic);
            }
        }
        match &receive.key {
            Some(key) => {
                self.expect_type(env, key, &Type::Text);
            }
            None => self.report(
                env.unit,
                Diagnostic::error(
                    "flow-key-missing",
                    receive.set.span,
                    Message::with(
                        message!("Say which conversation a message belongs to: key {0}.key"),
                        [Arg::from(receive.binder.name.as_str())],
                    ),
                ),
            ),
        }
        if let Some(ack) = &receive.acknowledge
            && !matches!(ack.name.as_str(), "997" | "999" | "none")
        {
            self.report(
                env.unit,
                Diagnostic::error(
                    "unknown-ack",
                    ack.span,
                    Message::new(message!("Acknowledge with 997, 999 or none")),
                ),
            );
        }
        let Some(document) = document else {
            return;
        };
        let has_purpose = document.field("purpose").is_some();
        let mut seen = BTreeSet::new();
        let mut bare = false;
        if receive.arms.is_empty() {
            self.report(
                env.unit,
                Diagnostic::error(
                    "flow-arms-missing",
                    receive.set.span,
                    if has_purpose {
                        Message::new(message!(
                            "Say what each purpose does: on Original -> create"
                        ))
                    } else {
                        Message::new(message!("Say what a message does: -> apply"))
                    },
                ),
            );
        }
        for arm in &receive.arms {
            match &arm.purpose {
                Some(purpose) if has_purpose => {
                    let known = schema::enumeration("TenderPurpose").and_then(|item| {
                        item.variants
                            .iter()
                            .find(|(variant, _)| *variant == purpose.name)
                    });
                    if let Some((variant, _)) = known {
                        self.occurrence(
                            env.unit,
                            purpose.span,
                            Target::Variant {
                                owner: "TenderPurpose",
                                name: variant,
                            },
                            false,
                            false,
                        );
                        if !seen.insert(*variant) {
                            self.report(
                                env.unit,
                                Diagnostic::error(
                                    "duplicate-arm",
                                    purpose.span,
                                    Message::with(
                                        message!("{0} already has an arm"),
                                        [Arg::from(purpose.name.as_str())],
                                    ),
                                ),
                            );
                        }
                    } else {
                        let mut diagnostic = Diagnostic::error(
                            "unknown-variant",
                            purpose.span,
                            Message::with(
                                message!("{0} is not a tender purpose"),
                                [Arg::from(purpose.name.as_str())],
                            ),
                        );
                        if let Some(close) =
                            nearest(&purpose.name, ["Original", "Change", "Cancel"])
                        {
                            diagnostic =
                                diagnostic.with_fix(super::replace_fix(purpose.span, close));
                        }
                        self.report(env.unit, diagnostic);
                    }
                }
                Some(purpose) => self.report(
                    env.unit,
                    Diagnostic::error(
                        "flow-purpose",
                        purpose.span,
                        Message::with(
                            message!("{0} has no purpose; write a single -> action"),
                            [Arg::from(document.name)],
                        ),
                    ),
                ),
                None => {
                    if bare {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "duplicate-arm",
                                arm.span,
                                Message::new(message!("There is already an arm for every message")),
                            ),
                        );
                    }
                    bare = true;
                }
            }
            self.action(env, &arm.action, document);
        }
        if has_purpose && !bare && !receive.arms.is_empty() {
            let missing: Vec<&str> = ["Original", "Change", "Cancel"]
                .into_iter()
                .filter(|variant| !seen.contains(variant))
                .collect();
            if !missing.is_empty() {
                self.report(
                    env.unit,
                    Diagnostic::warning(
                        "flow-purpose-unhandled",
                        receive.set.span,
                        Message::with(
                            message!("Nothing says what a {0} does: it would be held"),
                            [Arg::from(missing.join(", "))],
                        ),
                    ),
                );
            }
        }
        let span = receive.span;
        self.scope_fact(env, span, None);
    }

    #[allow(clippy::too_many_lines)]
    fn action(&mut self, env: &mut Env, action: &Action, document: &'static schema::Record) {
        let tender = document.name == "loadTender";
        let only_tender = |checker: &mut Self, env: &Env, span: Span, what: &str| {
            if !tender {
                checker.report(
                    env.unit,
                    Diagnostic::error(
                        "flow-action",
                        span,
                        Message::with(
                            message!("{0} acts on a load and needs a loadTender; this is a {1}: use apply"),
                            [Arg::from(what), Arg::from(document.name)],
                        ),
                    ),
                );
            }
        };
        match action {
            Action::Create(span) => only_tender(self, env, *span, "create"),
            Action::Update {
                fields,
                keep,
                hold_when,
                span,
            } => {
                only_tender(self, env, *span, "update");
                for field in fields.iter().chain(keep) {
                    if let Some(found) = document.field(&field.name) {
                        self.occurrence(
                            env.unit,
                            field.span,
                            Target::Field {
                                record: document.name,
                                field: found.name,
                            },
                            false,
                            false,
                        );
                    } else {
                        let mut diagnostic = Diagnostic::error(
                            "unknown-field",
                            field.span,
                            Message::with(
                                message!("{0} has no field {1}"),
                                [Arg::from(document.name), Arg::from(field.name.as_str())],
                            ),
                        );
                        if let Some(close) =
                            nearest(&field.name, document.fields.iter().map(|field| field.name))
                        {
                            diagnostic = diagnostic.with_fix(super::replace_fix(field.span, close));
                        }
                        self.report(env.unit, diagnostic);
                    }
                }
                for field in keep {
                    if fields.iter().any(|updated| updated.name == field.name) {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "update-keep",
                                field.span,
                                Message::with(
                                    message!("{0} is both updated and kept"),
                                    [Arg::from(field.name.as_str())],
                                ),
                            ),
                        );
                    }
                }
                if fields.is_empty() {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "update-fields",
                            *span,
                            Message::new(message!("Name the fields to update: update stops, refs")),
                        ),
                    );
                }
                if let Some((cond, message)) = hold_when {
                    env.push();
                    env.bind("current", Type::Record(document.name), Span::default());
                    self.condition(env, cond);
                    self.expect_type(env, message, &Type::Text);
                    env.pop();
                }
            }
            Action::Cancel { unless, span } => {
                only_tender(self, env, *span, "cancel");
                if let Some((state, otherwise)) = unless {
                    if schema::STATES.iter().any(|(name, _)| *name == state.name) {
                        self.occurrence(
                            env.unit,
                            state.span,
                            Target::Builtin(format!("state.{}", state.name)),
                            false,
                            false,
                        );
                    } else {
                        let mut diagnostic = Diagnostic::error(
                            "unknown-state",
                            state.span,
                            Message::with(
                                message!("{0} is not a state: dispatched, delivered or invoiced"),
                                [Arg::from(state.name.as_str())],
                            ),
                        );
                        if let Some(close) =
                            nearest(&state.name, schema::STATES.iter().map(|(name, _)| *name))
                        {
                            diagnostic = diagnostic.with_fix(super::replace_fix(state.span, close));
                        }
                        self.report(env.unit, diagnostic);
                    }
                    if matches!(otherwise.as_ref(), Action::Cancel { .. }) {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "flow-action",
                                otherwise.span(),
                                Message::new(message!("Instead of cancelling, hold or ignore")),
                            ),
                        );
                    } else {
                        self.action(env, otherwise, document);
                    }
                }
            }
            Action::Apply(span) => {
                if tender {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "flow-action",
                            *span,
                            Message::new(message!("A tender is created, updated or cancelled; apply is for 990, 214 and 210")),
                        ),
                    );
                }
            }
            Action::Hold(message, _) => {
                self.expect_type(env, message, &Type::Text);
            }
            Action::Ignore(_) | Action::Error(_) => {}
        }
    }

    fn respond(&mut self, env: &mut Env, respond: &Respond, tender_received: bool) {
        if respond.set.name != "990" {
            self.report(
                env.unit,
                Diagnostic::error(
                    "unknown-set",
                    respond.set.span,
                    Message::new(message!("A tender is answered with a 990")),
                ),
            );
        }
        if !tender_received {
            self.report(
                env.unit,
                Diagnostic::error(
                    "respond-without-receive",
                    respond.span,
                    Message::new(message!(
                        "respond answers a received tender: add receive 204 as tender"
                    )),
                ),
            );
        }
        if let Some(info) =
            self.flow_map(env, respond.map.as_ref(), &respond.set, false, respond.span)
            && info.document != Some("tenderResponse")
            && let Some(map) = &respond.map
        {
            self.report(
                env.unit,
                Diagnostic::error(
                    "flow-map-document",
                    map.span,
                    Message::new(message!("A response is written from a tenderResponse")),
                ),
            );
        }
        env.push();
        env.bind("declined", Type::Bool, Span::default());
        env.bind("held", Type::Bool, Span::default());
        for cond in respond.accept.iter().chain(&respond.decline) {
            self.condition(env, cond);
        }
        match &respond.otherwise {
            Some((Otherwise::Planner { within, escalate }, _)) => {
                if let Some(within) = within {
                    self.expect_type(env, within, &Type::Duration);
                }
                if let Some(escalate) = escalate {
                    self.expect_type(env, escalate, &Type::Text);
                }
            }
            Some((Otherwise::Decline(message), _)) => {
                self.expect_type(env, message, &Type::Text);
            }
            Some((Otherwise::Accept, _)) => {}
            None => {
                if respond.accept.is_empty() && respond.decline.is_empty() {
                    self.report(
                        env.unit,
                        Diagnostic::warning(
                            "respond-planner",
                            respond.set.span,
                            Message::new(message!("Every tender waits for a planner; say otherwise planner decides to make it plain")),
                        ),
                    );
                }
            }
        }
        env.pop();
    }

    #[allow(clippy::too_many_lines)]
    fn send(&mut self, env: &mut Env, send: &Send) {
        self.set(env, &send.set);
        let info = self.flow_map(env, send.map.as_ref(), &send.set, false, send.span);
        let document = info.as_ref().and_then(|info| info.document);
        if send.events.is_empty() {
            self.report(
                env.unit,
                Diagnostic::error(
                    "send-events",
                    send.set.span,
                    Message::new(message!("Say when to send: on StopArrived, Delivered")),
                ),
            );
        }
        for event in send.events.iter().chain(&send.until) {
            if let Some(found) = schema::event(&event.name) {
                self.occurrence(
                    env.unit,
                    event.span,
                    Target::Event(found.name),
                    false,
                    false,
                );
                if let Some(document) = document
                    && found.document != document
                    && send.until.as_ref() != Some(event)
                {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "send-event-document",
                            event.span,
                            Message::with(
                                message!("{0} builds a {1}, and this map writes from a {2}"),
                                [
                                    Arg::from(found.name),
                                    Arg::from(found.document),
                                    Arg::from(document),
                                ],
                            ),
                        ),
                    );
                }
            } else {
                let mut diagnostic = Diagnostic::error(
                    "unknown-event",
                    event.span,
                    Message::with(
                        message!("{0} is not an event Trenova sends on"),
                        [Arg::from(event.name.as_str())],
                    ),
                );
                if let Some(close) =
                    nearest(&event.name, schema::EVENTS.iter().map(|item| item.name))
                {
                    diagnostic = diagnostic.with_fix(super::replace_fix(event.span, close));
                }
                self.report(env.unit, diagnostic);
            }
        }
        env.push();
        let saved = env.document;
        env.document = document;
        if let Some(document) = document {
            env.bind(
                document_binder(document),
                Type::Record(document),
                Span::default(),
            );
        }
        if let Some(filter) = &send.filter {
            self.condition(env, filter);
        }
        if let Some(key) = &send.key {
            self.expect_type(env, key, &Type::Text);
        }
        if let Some(batch) = &send.batch {
            self.expect_type(env, batch, &Type::Duration);
        }
        self.scope_fact(env, send.span, None);
        env.document = saved;
        env.pop();
    }

    fn expect_ack(&mut self, env: &mut Env, expect: &ExpectAck) {
        if !matches!(expect.set.name.as_str(), "997" | "999") {
            self.report(
                env.unit,
                Diagnostic::error(
                    "unknown-ack",
                    expect.set.span,
                    Message::new(message!("A flow expects a 997 or 999 for what it sends")),
                ),
            );
        }
        match &expect.within {
            Some(within) => {
                self.expect_type(env, within, &Type::Duration);
            }
            None => self.report(
                env.unit,
                Diagnostic::error(
                    "expect-within",
                    expect.set.span,
                    Message::new(message!("Say how long to wait: within 30 min")),
                ),
            ),
        }
        if let Some((count, span)) = expect.resend
            && count > 5
        {
            self.report(
                env.unit,
                Diagnostic::error(
                    "resend-limit",
                    span,
                    Message::new(message!("A message is resent at most 5 times")),
                ),
            );
        }
        env.push();
        env.bind(
            "message",
            Type::Struct(vec![
                ("set".into(), Type::Text),
                ("control".into(), Type::Text),
                ("key".into(), Type::Text),
            ]),
            Span::default(),
        );
        if let Some(escalate) = &expect.escalate {
            self.expect_type(env, escalate, &Type::Text);
        }
        env.pop();
    }
}
