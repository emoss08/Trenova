//! `lading.flowStep`: a flow as a function from its state and one event to its new state and
//! the actions Trenova takes. The TMS runs one instance per key as a Temporal workflow that
//! feeds it events and carries out its actions; flow tests drive the same function.

use std::collections::BTreeMap;
use std::rc::Rc;

use serde::{Deserialize, Serialize};
use serde_json::Value as Json;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::check::Program;
use super::check::flow::document_binder;
use super::eval::interp::{Frame, Given, Interp, Outcome, OutcomeKind2};
use super::eval::json::{from_json, to_json};
use super::eval::value::{Value, display, instant_rfc3339};
use super::syntax::ast::{Action, Expr, FlowDecl, Item, Otherwise, Receive, Respond, Send};
use super::types::Type;
use super::values::ZoneValue;

/// The step budget of one flow step.
pub const FLOW_STEPS: u64 = 1_000_000;

/// The state of a load the flow acts on.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct LoadState {
    /// Whether it exists.
    pub exists: bool,
    /// `open`, `dispatched`, `delivered`, `invoiced` or `cancelled`.
    pub state: String,
}

/// A message sent that waits for its acknowledgment.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Pending {
    /// The send's identifier, as the action named it.
    pub id: String,
    /// The set sent.
    pub set: String,
    /// The control number, once the host reports it.
    #[serde(default)]
    pub control: String,
    /// When the acknowledgment is due.
    pub due: i64,
    /// Resends left.
    pub resends: u32,
}

/// Events of one send clause gathered to go out together.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Batch {
    /// The clause's index among the flow's clauses.
    pub clause: usize,
    /// The documents gathered.
    pub documents: Vec<Json>,
    /// When the batch goes out.
    pub due: i64,
}

/// A flow instance's state.
#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", default)]
pub struct FlowState {
    /// The instance's key.
    pub key: String,
    /// The last document received under each binder (`tender`), and the load as Trenova has
    /// it (`current`).
    pub documents: BTreeMap<String, Json>,
    /// The load.
    pub load: LoadState,
    /// Whether a tender is waiting for a response, and since when.
    pub awaiting_response: Option<i64>,
    /// When the planner's decision is due.
    pub response_due: Option<i64>,
    /// Whether the response deadline was escalated.
    pub escalated: bool,
    /// Whether the rules declined or held the last tender.
    pub declined: bool,
    /// Whether the rules held it.
    pub held: bool,
    /// What the last decline said.
    pub decline_reason: String,
    /// Sends waiting for acknowledgment.
    pub pending: Vec<Pending>,
    /// Batches being gathered.
    pub batches: Vec<Batch>,
    /// Send clauses that reached their `until` event.
    pub finished_sends: Vec<usize>,
    /// How many sends this instance has made, for naming them.
    pub sends: u64,
}

/// An event the flow handles.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase", tag = "kind")]
pub enum FlowEvent {
    /// A message from the partner, read by its map and rules (`lading.run`).
    Received {
        /// The set.
        set: String,
        /// The document.
        document: Json,
        /// What the map's elses and rules decided.
        #[serde(default)]
        outcomes: Vec<OutcomeInput>,
        /// The load as Trenova has it now, when there is one.
        #[serde(default)]
        current: Option<Json>,
        /// The load's state.
        #[serde(default)]
        load: Option<LoadState>,
    },
    /// Something happened in Trenova, with the document built for it.
    Trenova {
        /// The event (`StopArrived`).
        event: String,
        /// The document.
        document: Json,
    },
    /// A planner decided a tender.
    Planner {
        /// `accept` or `decline`.
        decision: String,
        /// Why a tender is declined.
        #[serde(default)]
        reason: String,
    },
    /// The host sent a message the flow asked for.
    Sent {
        /// The send's identifier.
        id: String,
        /// Its control number.
        control: String,
    },
    /// The partner acknowledged a message.
    Ack {
        /// The control number acknowledged.
        control: String,
        /// Whether it was accepted.
        accepted: bool,
    },
    /// A timer the flow asked for fired.
    Timer,
    /// The load changed state in Trenova.
    Load {
        /// The new state.
        load: LoadState,
    },
}

/// An outcome as the host passes it back from a run.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct OutcomeInput {
    /// `reject`, `decline`, `hold` or `warn`.
    pub kind: String,
    /// The message.
    #[serde(default)]
    pub message: String,
    /// A decline's code.
    #[serde(default)]
    pub code: String,
}

/// What Trenova does.
#[derive(Debug, Clone, PartialEq, Serialize)]
#[serde(rename_all = "camelCase", tag = "kind")]
pub enum FlowAction {
    /// Send a functional acknowledgment for the group a message came in.
    Acknowledge {
        /// `997` or `999`.
        set: String,
        /// Whether the transaction is rejected in it.
        rejected: bool,
    },
    /// Create a load from the document.
    Create {
        /// The document.
        document: Json,
    },
    /// Apply some fields of the document to the load.
    Update {
        /// The document.
        document: Json,
        /// The fields applied.
        fields: Vec<String>,
    },
    /// Cancel the load.
    Cancel,
    /// Hand the document to Trenova's handling for its set.
    Apply {
        /// The set.
        set: String,
        /// The document.
        document: Json,
    },
    /// Put the instance in the exception queue until a person releases it.
    Hold {
        /// Why.
        message: Message,
    },
    /// Do nothing with the message.
    Ignore,
    /// Ask a planner to decide a tender.
    AskPlanner {
        /// When the decision is due.
        #[serde(skip_serializing_if = "Option::is_none")]
        due: Option<i64>,
        /// The rules' decline, proposed.
        #[serde(skip_serializing_if = "String::is_empty")]
        proposal: String,
    },
    /// Write and send a message.
    Send {
        /// The send's identifier, reported back with a `sent` event.
        id: String,
        /// The set.
        set: String,
        /// The map.
        map: String,
        /// The document.
        document: Json,
    },
    /// Send a message again.
    Resend {
        /// The send's identifier.
        id: String,
    },
    /// Raise an exception for a person.
    Escalate {
        /// Why.
        message: Message,
    },
}

/// One step's result.
#[derive(Debug, Clone, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct FlowStep {
    /// The new state.
    pub state: FlowState,
    /// What Trenova does, in order.
    pub actions: Vec<FlowAction>,
    /// When to wake the instance with a timer event, if ever.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub wake: Option<i64>,
    /// Why the step could not run.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<Message>,
    /// The clauses and arms that ran: file and byte offset.
    #[serde(skip)]
    pub coverage: Vec<(usize, u32)>,
}

/// The clauses of a flow.
fn clauses(program: &Program, unit: usize) -> Vec<&FlowDecl> {
    program.units[unit]
        .items()
        .iter()
        .filter_map(|item| match item {
            Item::Flow(clause) => Some(clause),
            _ => None,
        })
        .collect()
}

/// The key a received message or event belongs to.
#[must_use]
pub fn route(program: &Program, flow: &str, event: &FlowEvent, now: i64) -> Option<String> {
    let unit = *program.flows.get(flow)?;
    let given = Given {
        now,
        ..Given::default()
    };
    let mut interp = Interp::new(program, &given, FLOW_STEPS);
    match event {
        FlowEvent::Received { set, document, .. } => {
            let receive = clauses(program, unit)
                .into_iter()
                .find_map(|clause| match clause {
                    FlowDecl::Receive(receive) if receive.set.name == *set => Some(receive),
                    _ => None,
                })?;
            let mut frame = receive_frame(program, unit, receive, document)?;
            let key = interp.eval(&mut frame, receive.key.as_ref()?).ok()?;
            (!key.is_none()).then(|| display(&key))
        }
        FlowEvent::Trenova { event, document } => {
            let send = clauses(program, unit)
                .into_iter()
                .find_map(|clause| match clause {
                    FlowDecl::Send(send) if send.events.iter().any(|name| name.name == *event) => {
                        Some(send)
                    }
                    _ => None,
                })?;
            let (mut frame, value) = send_frame(program, unit, send, document)?;
            let key = match &send.key {
                Some(key) => interp.eval(&mut frame, key).ok()?,
                None => value.field("key"),
            };
            (!key.is_none()).then(|| display(&key))
        }
        _ => None,
    }
}

fn receive_frame(
    program: &Program,
    unit: usize,
    receive: &Receive,
    document: &Json,
) -> Option<Frame> {
    let info = program.maps.get(&receive.map.as_ref()?.name)?;
    let value = from_json(program, document, &Type::Record(info.document?), "").ok()?;
    let mut frame = Frame::new(unit);
    frame.bind(&receive.binder.name, value);
    Some(frame)
}

fn send_frame(
    program: &Program,
    unit: usize,
    send: &Send,
    document: &Json,
) -> Option<(Frame, Value)> {
    let info = program.maps.get(&send.map.as_ref()?.name)?;
    let name = info.document?;
    let value = from_json(program, document, &Type::Record(name), "").ok()?;
    let mut frame = Frame::new(unit);
    frame.document = Some(value.clone());
    frame.bind(document_binder(name), value.clone());
    Some((frame, value))
}

/// Takes one step of a flow.
#[must_use]
pub fn step(
    program: &Program,
    flow: &str,
    state: FlowState,
    event: &FlowEvent,
    now: i64,
    settings: &Given,
) -> FlowStep {
    let Some(unit) = program.flows.get(flow).copied() else {
        return FlowStep {
            state,
            actions: Vec::new(),
            wake: None,
            error: Some(Message::with(
                message!("There is no flow named {0}"),
                [Arg::from(flow)],
            )),
            coverage: Vec::new(),
        };
    };
    let given = Given {
        now,
        settings: settings.settings.clone(),
        lookups: BTreeMap::new(),
        stand_in: false,
    };
    let mut stepper = Stepper {
        program,
        unit,
        interp: Interp::new(program, &given, FLOW_STEPS),
        state,
        actions: Vec::new(),
        now,
        coverage: Vec::new(),
    };
    match event {
        FlowEvent::Received {
            set,
            document,
            outcomes,
            current,
            load,
        } => stepper.received(set, document, outcomes, current.as_ref(), load.as_ref()),
        FlowEvent::Trenova { event, document } => stepper.trenova(event, document),
        FlowEvent::Planner { decision, reason } => stepper.planner(decision, reason),
        FlowEvent::Sent { id, control } => {
            if let Some(pending) = stepper
                .state
                .pending
                .iter_mut()
                .find(|pending| pending.id == *id)
            {
                pending.control.clone_from(control);
            }
        }
        FlowEvent::Ack { control, accepted } => stepper.ack(control, *accepted),
        FlowEvent::Timer => stepper.timer(),
        FlowEvent::Load { load } => stepper.state.load = load.clone(),
    }
    let wake = stepper.wake();
    FlowStep {
        state: stepper.state,
        actions: stepper.actions,
        wake,
        error: None,
        coverage: stepper.coverage,
    }
}

struct Stepper<'p> {
    program: &'p Program,
    unit: usize,
    interp: Interp<'p>,
    state: FlowState,
    actions: Vec<FlowAction>,
    now: i64,
    coverage: Vec<(usize, u32)>,
}

impl Stepper<'_> {
    /// A frame with every received document bound under its binder.
    fn frame(&self) -> Frame {
        let program = self.program;
        let mut frame = Frame::new(self.unit);
        for clause in clauses(program, self.unit) {
            if let FlowDecl::Receive(receive) = clause
                && let Some(document) = self.state.documents.get(&receive.binder.name)
                && let Some(info) = receive
                    .map
                    .as_ref()
                    .and_then(|map| program.maps.get(&map.name))
                && let Some(name) = info.document
                && let Ok(value) = from_json(program, document, &Type::Record(name), "")
            {
                frame.bind(&receive.binder.name, value);
            }
        }
        frame.bind("declined", Value::Bool(self.state.declined));
        frame.bind("held", Value::Bool(self.state.held));
        frame
    }

    fn text(&mut self, frame: &mut Frame, expr: &Expr) -> String {
        self.interp
            .eval(frame, expr)
            .map(|value| display(&value))
            .unwrap_or_default()
    }

    fn truth(&mut self, frame: &mut Frame, expr: &Expr) -> bool {
        self.interp
            .eval(frame, expr)
            .is_ok_and(|value| value.truthy())
    }

    fn duration(&mut self, frame: &mut Frame, expr: &Expr) -> i64 {
        match self.interp.eval(frame, expr) {
            Ok(Value::Duration(seconds)) => seconds,
            _ => 0,
        }
    }

    #[allow(clippy::too_many_lines)]
    fn received(
        &mut self,
        set: &str,
        document: &Json,
        outcomes: &[OutcomeInput],
        current: Option<&Json>,
        load: Option<&LoadState>,
    ) {
        let program = self.program;
        let Some(receive) =
            clauses(program, self.unit)
                .into_iter()
                .find_map(|clause| match clause {
                    FlowDecl::Receive(receive) if receive.set.name == set => Some(receive.clone()),
                    _ => None,
                })
        else {
            self.actions.push(FlowAction::Ignore);
            return;
        };
        if let Some(load) = load {
            self.state.load = load.clone();
        }
        if let Some(current) = current {
            self.state
                .documents
                .insert("current".to_owned(), current.clone());
        }
        let rejected = outcomes.iter().any(|outcome| outcome.kind == "reject");
        let ack = receive
            .acknowledge
            .as_ref()
            .map_or("997", |ack| ack.name.as_str());
        if ack != "none" {
            self.actions.push(FlowAction::Acknowledge {
                set: ack.to_owned(),
                rejected,
            });
        }
        if rejected {
            return;
        }
        self.state.declined = outcomes.iter().any(|outcome| outcome.kind == "decline");
        self.state.held = outcomes.iter().any(|outcome| outcome.kind == "hold");
        self.state.decline_reason = outcomes
            .iter()
            .find(|outcome| outcome.kind == "decline")
            .map(|outcome| outcome.message.clone())
            .unwrap_or_default();
        self.state
            .documents
            .insert(receive.binder.name.clone(), document.clone());
        let mut frame = self.frame();
        if let Some(key) = &receive.key {
            let key = self.text(&mut frame, key);
            if self.state.key.is_empty() {
                self.state.key = key;
            }
        }
        let purpose = document
            .get("purpose")
            .and_then(Json::as_str)
            .unwrap_or("Original")
            .to_owned();
        let arm = receive
            .arms
            .iter()
            .find(|arm| {
                arm.purpose
                    .as_ref()
                    .is_some_and(|name| name.name == purpose)
            })
            .or_else(|| receive.arms.iter().find(|arm| arm.purpose.is_none()));
        let Some(arm) = arm else {
            self.actions.push(FlowAction::Hold {
                message: Message::with(
                    message!("Nothing says what a {0} does"),
                    [Arg::from(purpose.as_str())],
                ),
            });
            return;
        };
        self.coverage.push((self.unit, receive.span.start));
        self.coverage.push((self.unit, arm.span.start));
        let action = arm.action.clone();
        self.act(&mut frame, &action, set, document);
        for outcome in outcomes.iter().filter(|outcome| outcome.kind == "hold") {
            self.actions.push(FlowAction::Hold {
                message: written(&outcome.message),
            });
        }
        if set == "204" && purpose == "Original" {
            let respond = clauses(program, self.unit)
                .into_iter()
                .find_map(|clause| match clause {
                    FlowDecl::Respond(respond) => Some(respond.clone()),
                    _ => None,
                });
            if let Some(respond) = respond {
                self.respond(&mut frame, &respond);
            }
        }
    }

    fn act(&mut self, frame: &mut Frame, action: &Action, set: &str, document: &Json) {
        match action {
            Action::Create(_) => {
                if self.state.load.exists {
                    self.actions.push(FlowAction::Hold {
                        message: Message::new(message!("A load already exists for this tender")),
                    });
                } else {
                    self.state.load = LoadState {
                        exists: true,
                        state: "open".to_owned(),
                    };
                    self.actions.push(FlowAction::Create {
                        document: document.clone(),
                    });
                }
            }
            Action::Update {
                fields, hold_when, ..
            } => {
                if !self.state.load.exists {
                    self.actions.push(FlowAction::Hold {
                        message: Message::new(message!("There is no load to update")),
                    });
                    return;
                }
                if let Some((cond, message)) = hold_when {
                    let current = self
                        .state
                        .documents
                        .get("current")
                        .cloned()
                        .or_else(|| self.state.documents.get("created").cloned());
                    frame.push();
                    if let Some(current) = current
                        && let Ok(value) =
                            from_json(self.program, &current, &Type::Record("loadTender"), "")
                    {
                        frame.bind("current", value);
                    }
                    let hold = self.truth(frame, cond);
                    let text = self.text(frame, message);
                    frame.pop();
                    if hold {
                        self.actions.push(FlowAction::Hold {
                            message: written(&text),
                        });
                        return;
                    }
                }
                self.actions.push(FlowAction::Update {
                    document: document.clone(),
                    fields: fields.iter().map(|field| field.name.clone()).collect(),
                });
                self.state
                    .documents
                    .insert("current".to_owned(), document.clone());
            }
            Action::Cancel { unless, .. } => {
                if let Some((state, otherwise)) = unless
                    && self.state.load.state == state.name
                {
                    let otherwise = otherwise.as_ref().clone();
                    self.act(frame, &otherwise, set, document);
                    return;
                }
                "cancelled".clone_into(&mut self.state.load.state);
                self.actions.push(FlowAction::Cancel);
            }
            Action::Apply(_) => self.actions.push(FlowAction::Apply {
                set: set.to_owned(),
                document: document.clone(),
            }),
            Action::Hold(message, _) => {
                let message = self.text(frame, message);
                self.actions.push(FlowAction::Hold {
                    message: written(&message),
                });
            }
            Action::Ignore(_) | Action::Error(_) => self.actions.push(FlowAction::Ignore),
        }
        if matches!(action, Action::Create(_)) {
            self.state
                .documents
                .insert("current".to_owned(), document.clone());
        }
    }

    fn respond(&mut self, frame: &mut Frame, respond: &Respond) {
        self.coverage.push((self.unit, respond.span.start));
        for cond in &respond.accept {
            if self.truth(frame, cond) {
                self.send_response(respond, "Accept", "");
                return;
            }
        }
        for cond in &respond.decline {
            if self.truth(frame, cond) {
                let reason = self.state.decline_reason.clone();
                self.send_response(respond, "Decline", &reason);
                return;
            }
        }
        match &respond.otherwise {
            Some((Otherwise::Accept, _)) => self.send_response(respond, "Accept", ""),
            Some((Otherwise::Decline(message), _)) => {
                let reason = self.text(frame, message);
                self.send_response(respond, "Decline", &reason);
            }
            Some((Otherwise::Planner { within, .. }, _)) => {
                let due = within
                    .as_ref()
                    .map(|within| self.now + self.duration(frame, within));
                self.state.awaiting_response = Some(self.now);
                self.state.response_due = due;
                self.state.escalated = false;
                self.actions.push(FlowAction::AskPlanner {
                    due,
                    proposal: if self.state.declined {
                        self.state.decline_reason.clone()
                    } else {
                        String::new()
                    },
                });
            }
            None => {
                self.state.awaiting_response = Some(self.now);
                self.actions.push(FlowAction::AskPlanner {
                    due: None,
                    proposal: if self.state.declined {
                        self.state.decline_reason.clone()
                    } else {
                        String::new()
                    },
                });
            }
        }
    }

    fn send_response(&mut self, respond: &Respond, decision: &str, reason: &str) {
        self.state.awaiting_response = None;
        self.state.response_due = None;
        let key = self
            .state
            .documents
            .values()
            .find_map(|document| {
                document
                    .get("key")
                    .and_then(Json::as_str)
                    .map(str::to_owned)
            })
            .unwrap_or_else(|| self.state.key.clone());
        let mut document = serde_json::json!({
            "key": key,
            "decision": decision,
            "respondedAt": { "at": instant_rfc3339(self.now, ZoneValue::UTC), "zone": "UTC" },
        });
        if !reason.is_empty() {
            document["reason"] = Json::String(reason.chars().take(200).collect());
        }
        let map = respond
            .map
            .as_ref()
            .map(|map| map.name.clone())
            .unwrap_or_default();
        self.send(respond.set.name.clone(), map, document);
    }

    fn send(&mut self, set: String, map: String, document: Json) {
        self.state.sends += 1;
        let id = format!("{}-{}", self.state.key, self.state.sends);
        let expect = clauses(self.program, self.unit)
            .into_iter()
            .find_map(|clause| match clause {
                FlowDecl::Expect(expect) => Some(expect.clone()),
                _ => None,
            });
        if let Some(expect) = expect {
            let mut frame = self.frame();
            let within = expect
                .within
                .as_ref()
                .map_or(1800, |within| self.duration(&mut frame, within));
            self.state.pending.push(Pending {
                id: id.clone(),
                set: set.clone(),
                control: String::new(),
                due: self.now + within,
                resends: expect.resend.map_or(0, |(count, _)| count),
            });
        }
        self.actions.push(FlowAction::Send {
            id,
            set,
            map,
            document,
        });
    }

    fn planner(&mut self, decision: &str, reason: &str) {
        if self.state.awaiting_response.is_none() {
            return;
        }
        let respond =
            clauses(self.program, self.unit)
                .into_iter()
                .find_map(|clause| match clause {
                    FlowDecl::Respond(respond) => Some(respond.clone()),
                    _ => None,
                });
        if let Some(respond) = respond {
            let decision = if decision.eq_ignore_ascii_case("decline") {
                "Decline"
            } else {
                "Accept"
            };
            self.send_response(&respond, decision, reason);
        }
    }

    fn trenova(&mut self, event: &str, document: &Json) {
        let program = self.program;
        let sends: Vec<(usize, Send)> = clauses(program, self.unit)
            .into_iter()
            .enumerate()
            .filter_map(|(index, clause)| match clause {
                FlowDecl::Send(send) if send.events.iter().any(|name| name.name == event) => {
                    Some((index, send.clone()))
                }
                _ => None,
            })
            .collect();
        for (index, send) in sends {
            if self.state.finished_sends.contains(&index) {
                continue;
            }
            if let Some(filter) = &send.filter {
                let Some((mut frame, _)) = send_frame(program, self.unit, &send, document) else {
                    continue;
                };
                if !self.truth(&mut frame, filter) {
                    continue;
                }
            }
            self.coverage.push((self.unit, send.span.start));
            let map = send
                .map
                .as_ref()
                .map(|map| map.name.clone())
                .unwrap_or_default();
            let mut frame = self.frame();
            match &send.batch {
                Some(batch) => {
                    let window = self.duration(&mut frame, batch);
                    match self
                        .state
                        .batches
                        .iter_mut()
                        .find(|batch| batch.clause == index)
                    {
                        Some(open) => open.documents.push(document.clone()),
                        None => self.state.batches.push(Batch {
                            clause: index,
                            documents: vec![document.clone()],
                            due: self.now + window,
                        }),
                    }
                }
                None => self.send(send.set.name.clone(), map, document.clone()),
            }
            if send.until.as_ref().is_some_and(|until| until.name == event) {
                self.flush(index);
                self.state.finished_sends.push(index);
            }
        }
    }

    /// Sends a batch now.
    fn flush(&mut self, clause: usize) {
        let Some(position) = self
            .state
            .batches
            .iter()
            .position(|batch| batch.clause == clause)
        else {
            return;
        };
        let batch = self.state.batches.remove(position);
        let Some(FlowDecl::Send(send)) = clauses(self.program, self.unit)
            .get(clause)
            .map(|clause| (*clause).clone())
        else {
            return;
        };
        let map = send
            .map
            .as_ref()
            .map(|map| map.name.clone())
            .unwrap_or_default();
        self.send(send.set.name.clone(), map, merge(&batch.documents));
    }

    fn ack(&mut self, control: &str, accepted: bool) {
        let Some(position) = self
            .state
            .pending
            .iter()
            .position(|pending| pending.control == control)
        else {
            return;
        };
        let pending = self.state.pending.remove(position);
        if !accepted {
            let message = self.escalation(&pending, true);
            self.actions.push(FlowAction::Escalate { message });
        }
    }

    fn escalation(&mut self, pending: &Pending, rejected: bool) -> Message {
        let expect = clauses(self.program, self.unit)
            .into_iter()
            .find_map(|clause| match clause {
                FlowDecl::Expect(expect) => Some(expect.clone()),
                _ => None,
            });
        let mut frame = self.frame();
        let mut fields = BTreeMap::new();
        fields.insert("set".to_owned(), Value::text(&pending.set));
        fields.insert("control".to_owned(), Value::text(&pending.control));
        fields.insert("key".to_owned(), Value::text(&self.state.key));
        frame.bind("message", Value::Struct(Rc::new(fields)));
        let text = expect
            .and_then(|expect| expect.escalate)
            .map(|escalate| self.text(&mut frame, &escalate))
            .unwrap_or_default();
        if rejected {
            Message::with(
                message!("The partner rejected {0} {1}: {2}"),
                [
                    Arg::from(pending.set.as_str()),
                    Arg::from(pending.control.as_str()),
                    Arg::from(text),
                ],
            )
        } else {
            written(&text)
        }
    }

    fn timer(&mut self) {
        if let Some(due) = self.state.response_due
            && due <= self.now
            && self.state.awaiting_response.is_some()
            && !self.state.escalated
        {
            let respond = clauses(self.program, self.unit)
                .into_iter()
                .find_map(|clause| match clause {
                    FlowDecl::Respond(respond) => Some(respond.clone()),
                    _ => None,
                });
            if let Some(Respond {
                otherwise:
                    Some((
                        Otherwise::Planner {
                            escalate: Some(escalate),
                            ..
                        },
                        _,
                    )),
                ..
            }) = respond
            {
                let mut frame = self.frame();
                let message = self.text(&mut frame, &escalate);
                self.actions.push(FlowAction::Escalate {
                    message: written(&message),
                });
            }
            self.state.escalated = true;
        }
        let due: Vec<usize> = self
            .state
            .batches
            .iter()
            .filter(|batch| batch.due <= self.now)
            .map(|batch| batch.clause)
            .collect();
        for clause in due {
            self.flush(clause);
        }
        let mut index = 0;
        while index < self.state.pending.len() {
            if self.state.pending[index].due > self.now {
                index += 1;
                continue;
            }
            if self.state.pending[index].resends > 0 {
                let within = self.state.pending[index].due - self.now;
                let pending = &mut self.state.pending[index];
                pending.resends -= 1;
                pending.due = self.now + within.max(1800);
                let id = pending.id.clone();
                self.actions.push(FlowAction::Resend { id });
                index += 1;
            } else {
                let pending = self.state.pending.remove(index);
                let message = self.escalation(&pending, false);
                self.actions.push(FlowAction::Escalate { message });
            }
        }
    }

    /// The earliest time something is due.
    fn wake(&self) -> Option<i64> {
        let response = self
            .state
            .response_due
            .filter(|_| self.state.awaiting_response.is_some() && !self.state.escalated);
        response
            .into_iter()
            .chain(self.state.batches.iter().map(|batch| batch.due))
            .chain(self.state.pending.iter().map(|pending| pending.due))
            .min()
    }
}

/// Text a project's author wrote, already in the words they chose, as a message.
fn written(text: &str) -> Message {
    Message::with(message!("{0}"), [Arg::from(text)])
}

/// Documents gathered in a batch as one: lists concatenated, the rest from the latest.
fn merge(documents: &[Json]) -> Json {
    let mut merged = serde_json::Map::new();
    for document in documents {
        let Json::Object(object) = document else {
            continue;
        };
        for (key, value) in object {
            match (merged.get_mut(key), value) {
                (Some(Json::Array(existing)), Json::Array(more)) => {
                    existing.extend(more.iter().cloned());
                }
                _ => {
                    merged.insert(key.clone(), value.clone());
                }
            }
        }
    }
    Json::Object(merged)
}

/// Outcomes of a run as a flow event takes them.
#[must_use]
pub fn outcome_inputs(outcomes: &[Outcome]) -> Vec<OutcomeInput> {
    outcomes
        .iter()
        .map(|outcome| OutcomeInput {
            kind: match outcome.kind {
                OutcomeKind2::Reject => "reject",
                OutcomeKind2::Decline => "decline",
                OutcomeKind2::Hold => "hold",
                OutcomeKind2::Warn => "warn",
            }
            .to_owned(),
            message: outcome.message.clone(),
            code: outcome.code.clone(),
        })
        .collect()
}

/// The JSON of a value, for documents flows build.
#[must_use]
pub fn document_json(value: &Value) -> Json {
    to_json(value)
}
