//! Checks tests and properties: that each step suits the file it is in, that samples, settings,
//! lookups and events exist, and that expectations are typed against what the file produces.

use trenova_module_kit::diagnostic::{Diagnostic, nearest};
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::env::Env;
use super::{Checker, Program};
use crate::lading::facts::Target;
use crate::lading::schema;
use crate::lading::syntax::ast::{Header, Item, TestStep, TextLit, Vary, VarySource};
use crate::lading::types::Type;

/// What a file's tests run.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Subject {
    /// A map that fills a document, and its rules for a rules file.
    Inbound(Option<&'static str>),
    /// A map that writes X12 from a document, and its rules for a rules file.
    Outbound(Option<&'static str>),
    /// A flow.
    Flow,
    /// A library's functions.
    Library,
    /// A guide's validation.
    Guide,
    /// Nothing tests can run.
    None,
}

/// What a file's tests run.
#[must_use]
pub fn subject(program: &Program, unit: usize) -> Subject {
    match program.units[unit].header() {
        Header::Map { name, .. } => match program.maps.get(&name.name) {
            Some(info) if info.inbound() => Subject::Inbound(info.document),
            Some(info) => Subject::Outbound(info.document),
            None => Subject::None,
        },
        Header::Rules { map, .. } => match program.maps.get(&map.name) {
            Some(info) if info.inbound() => Subject::Inbound(info.document),
            Some(info) => Subject::Outbound(info.document),
            None => Subject::None,
        },
        Header::Flow { .. } => Subject::Flow,
        Header::Library { .. } => Subject::Library,
        Header::Guide { .. } => Subject::Guide,
        Header::Partner { .. } | Header::Missing(_) => Subject::None,
    }
}

impl Checker<'_> {
    /// Checks every test and property of a file.
    pub(crate) fn check_tests(&mut self, unit: usize) {
        let program: &Program = self.program;
        let subject = subject(program, unit);
        for item in program.units[unit].items() {
            let (steps, varies, span) = match item {
                Item::Test(test) => (&test.steps, &[][..], test.name.span),
                Item::Property(property) => (
                    &property.steps,
                    property.varies.as_slice(),
                    property.name.span,
                ),
                _ => continue,
            };
            if subject == Subject::None {
                if !matches!(program.units[unit].header(), Header::Partner { .. }) {
                    continue;
                }
                self.report(
                    unit,
                    Diagnostic::error(
                        "test-here",
                        span,
                        Message::new(message!(
                            "Tests go in maps, rules, flows, libraries and guides"
                        )),
                    ),
                );
                continue;
            }
            let mut env = Env::new(unit);
            env.test = true;
            if let Subject::Inbound(document) | Subject::Outbound(document) = subject {
                env.document = document;
            }
            env.inbound = matches!(subject, Subject::Inbound(_));
            let mut input = false;
            for vary in varies {
                input |= self.vary(&mut env, vary, subject);
            }
            if let Subject::Flow = subject {
                self.bind_flow_names(&mut env);
            }
            for (step, step_span) in steps {
                input |= matches!(
                    step,
                    TestStep::GivenFile(_)
                        | TestStep::GivenText(_)
                        | TestStep::GivenDocument { .. }
                );
                self.step(&mut env, step, *step_span, subject);
            }
            if !input
                && matches!(
                    subject,
                    Subject::Inbound(_) | Subject::Outbound(_) | Subject::Guide
                )
            {
                self.report(
                    unit,
                    Diagnostic::error(
                        "test-input",
                        span,
                        match subject {
                            Subject::Outbound(_) => Message::new(message!(
                                "Give the test a document: given shipmentStatus '{' ... '}'"
                            )),
                            _ => Message::new(message!(
                                "Give the test a message: given file \"samples/...\""
                            )),
                        },
                    ),
                );
            }
        }
    }

    fn bind_flow_names(&mut self, env: &mut Env) {
        let program: &Program = self.program;
        for item in program.units[env.unit].items() {
            if let Item::Flow(crate::lading::syntax::ast::FlowDecl::Receive(receive)) = item {
                let ty = receive
                    .map
                    .as_ref()
                    .and_then(|map| program.maps.get(&map.name))
                    .and_then(|info| info.document)
                    .map_or(Type::Any, Type::Record);
                env.bind(&receive.binder.name, ty, receive.binder.span);
            }
        }
    }

    /// Binds a varied name; true when it is the test's input.
    fn vary(&mut self, env: &mut Env, vary: &Vary, subject: Subject) -> bool {
        let (ty, input) = match &vary.source {
            VarySource::Samples(glob) => {
                if !matches!(subject, Subject::Inbound(_) | Subject::Guide) {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "vary-samples",
                            glob.span,
                            Message::new(message!(
                                "Samples are read by a map from X12, CSV or JSON"
                            )),
                        ),
                    );
                }
                if !self
                    .program
                    .samples
                    .iter()
                    .any(|sample| glob_matches(&glob.value, &sample.path))
                {
                    self.report(
                        env.unit,
                        Diagnostic::warning(
                            "no-samples",
                            glob.span,
                            Message::with(
                                message!("No sample matches {0}"),
                                [Arg::from(glob.value.as_str())],
                            ),
                        ),
                    );
                }
                (Type::Text, true)
            }
            VarySource::List(list) => {
                let ty = self.expr(env, list, None);
                (ty.item().cloned().unwrap_or(Type::Any), false)
            }
            VarySource::Range(low, high) => {
                let low_ty = self.expr(env, low, None);
                let high_ty = self.expr(env, high, Some(&low_ty));
                let joined = low_ty
                    .join(&high_ty)
                    .filter(Type::is_ordered)
                    .unwrap_or_else(|| {
                        self.report(
                            env.unit,
                            super::expr::mismatch(high.span, &low_ty, &high_ty),
                        );
                        Type::Any
                    });
                (joined, false)
            }
            VarySource::Enum(name) => {
                if let Some(item) = schema::enumeration(&name.name) {
                    self.occurrence(
                        env.unit,
                        name.span,
                        Target::Type(item.name.to_owned()),
                        false,
                        false,
                    );
                    (Type::Enum(item.name), false)
                } else {
                    let mut diagnostic = Diagnostic::error(
                        "unknown-type",
                        name.span,
                        Message::with(
                            message!("There is no enum named {0}"),
                            [Arg::from(name.name.as_str())],
                        ),
                    );
                    if let Some(close) =
                        nearest(&name.name, schema::ENUMS.iter().map(|item| item.name))
                    {
                        diagnostic = diagnostic.with_fix(super::replace_fix(name.span, close));
                    }
                    self.report(env.unit, diagnostic);
                    (Type::Any, false)
                }
            }
        };
        self.bind_local(env, &vary.name, ty);
        input
    }

    fn sample(&mut self, env: &Env, path: &TextLit) {
        if self
            .program
            .samples
            .iter()
            .any(|sample| sample.path == path.value)
        {
            return;
        }
        let names: Vec<&str> = self
            .program
            .samples
            .iter()
            .map(|sample| sample.path.as_str())
            .collect();
        let mut diagnostic = Diagnostic::error(
            "unknown-sample",
            path.span,
            Message::with(
                message!("There is no sample {0}"),
                [Arg::from(path.value.as_str())],
            ),
        );
        if let Some(close) = nearest(&path.value, names) {
            diagnostic =
                diagnostic.with_fix(super::replace_fix(path.span, &format!("\"{close}\"")));
        }
        self.report(env.unit, diagnostic);
    }

    fn wrong_step(&mut self, env: &Env, span: Span, message: &'static str) {
        self.report(
            env.unit,
            Diagnostic::error("test-step", span, Message::new(message)),
        );
    }

    #[allow(clippy::too_many_lines)]
    fn step(&mut self, env: &mut Env, step: &TestStep, span: Span, subject: Subject) {
        let reads_message = matches!(subject, Subject::Inbound(_) | Subject::Guide);
        let flow = subject == Subject::Flow;
        match step {
            TestStep::GivenFile(path) => {
                if !reads_message {
                    self.wrong_step(env, span, message!("given file is for a map that reads a message; a flow test uses receive file"));
                }
                self.sample(env, path);
            }
            TestStep::GivenText(text) => {
                if !reads_message {
                    self.wrong_step(
                        env,
                        span,
                        message!("given text is for a map that reads a message"),
                    );
                }
                self.expect_type(env, text, &Type::Text);
            }
            TestStep::GivenDocument { document, value } => {
                let Subject::Outbound(expected) = subject else {
                    self.wrong_step(
                        env,
                        span,
                        message!("given with a document is for a map that writes X12"),
                    );
                    self.expr(env, value, None);
                    return;
                };
                if let Some(found) = schema::document(&document.name) {
                    self.occurrence(
                        env.unit,
                        document.span,
                        Target::Type(found.name.to_owned()),
                        false,
                        false,
                    );
                    if expected.is_some_and(|expected| expected != found.name) {
                        self.report(
                            env.unit,
                            Diagnostic::error(
                                "test-document",
                                document.span,
                                Message::with(
                                    message!("This map writes from a {0}"),
                                    [Arg::from(expected.unwrap_or_default())],
                                ),
                            ),
                        );
                    }
                    let saved = env.document.take();
                    self.expect_type(env, value, &Type::Record(found.name));
                    env.document = saved;
                } else {
                    self.report(
                        env.unit,
                        Diagnostic::error(
                            "unknown-document",
                            document.span,
                            Message::with(
                                message!("{0} is not a document"),
                                [Arg::from(document.name.as_str())],
                            ),
                        ),
                    );
                    self.expr(env, value, None);
                }
            }
            TestStep::GivenSetting { name, value } => {
                let setting = self
                    .program
                    .partner
                    .as_ref()
                    .and_then(|partner| {
                        partner
                            .settings
                            .iter()
                            .find(|setting| setting.name == name.name)
                    })
                    .map(|setting| setting.ty.clone());
                if let Some(ty) = setting {
                    self.occurrence(
                        env.unit,
                        name.span,
                        Target::Setting(name.name.clone()),
                        false,
                        false,
                    );
                    self.expect_type(env, value, &ty);
                } else {
                    let names: Vec<String> = self
                        .program
                        .partner
                        .as_ref()
                        .map(|partner| {
                            partner
                                .settings
                                .iter()
                                .map(|setting| setting.name.clone())
                                .collect()
                        })
                        .unwrap_or_default();
                    let mut diagnostic = Diagnostic::error(
                        "unknown-setting",
                        name.span,
                        Message::with(
                            message!("The partner has no setting {0}"),
                            [Arg::from(name.name.as_str())],
                        ),
                    );
                    if let Some(close) = nearest(&name.name, names.iter().map(String::as_str)) {
                        diagnostic = diagnostic.with_fix(super::replace_fix(name.span, close));
                    }
                    self.report(env.unit, diagnostic);
                    self.expr(env, value, None);
                }
            }
            TestStep::GivenLookup { kind, code, label } => {
                if let Some(found) = schema::lookup(&kind.name) {
                    self.occurrence(
                        env.unit,
                        kind.span,
                        Target::Type(found.name.to_owned()),
                        false,
                        false,
                    );
                } else {
                    let mut diagnostic = Diagnostic::error(
                        "unknown-lookup",
                        kind.span,
                        Message::with(
                            message!("{0} is not a lookup kind"),
                            [Arg::from(kind.name.as_str())],
                        ),
                    );
                    if let Some(close) =
                        nearest(&kind.name, schema::LOOKUPS.iter().map(|item| item.name))
                    {
                        diagnostic = diagnostic.with_fix(super::replace_fix(kind.span, close));
                    }
                    self.report(env.unit, diagnostic);
                }
                self.expect_type(env, code, &Type::Text);
                if let Some(label) = label {
                    self.expect_type(env, label, &Type::Text);
                }
            }
            TestStep::GivenNow(value) => {
                self.expect_type(env, value, &Type::DateTime);
            }
            TestStep::Expect(cond) => {
                if flow {
                    self.wrong_step(
                        env,
                        span,
                        message!(
                            "A flow test checks what the flow did: expect sent 990, expect held"
                        ),
                    );
                }
                self.condition(env, cond);
            }
            TestStep::ExpectX12 { text, .. } => {
                if !matches!(subject, Subject::Outbound(_)) {
                    self.wrong_step(
                        env,
                        span,
                        message!("expect x12 checks what a map to X12 wrote"),
                    );
                }
                self.expect_type(env, text, &Type::Text);
            }
            TestStep::ExpectOutcome {
                negated,
                what,
                set,
                detail,
            } => {
                self.expect_outcome(
                    env,
                    *negated,
                    &what.name,
                    what.span,
                    set.as_ref(),
                    detail.as_ref(),
                    subject,
                );
            }
            TestStep::Receive { file, source } => {
                if !flow {
                    self.wrong_step(env, span, message!("receive is a step of a flow test"));
                }
                if *file {
                    if let crate::lading::syntax::ast::ExprKind::Text(parts) = &source.kind
                        && let [crate::lading::syntax::ast::TextPart::Literal(path)] =
                            parts.as_slice()
                    {
                        self.sample(
                            env,
                            &TextLit {
                                value: path.clone(),
                                span: source.span,
                            },
                        );
                    }
                } else {
                    self.expect_type(env, source, &Type::Text);
                }
            }
            TestStep::Respond { reason, .. } => {
                if !flow {
                    self.wrong_step(env, span, message!("respond is a step of a flow test"));
                }
                if let Some(reason) = reason {
                    self.expect_type(env, reason, &Type::Text);
                }
            }
            TestStep::Event { kind, value } => {
                if !flow {
                    self.wrong_step(env, span, message!("event is a step of a flow test"));
                }
                if let Some(found) = schema::event(&kind.name) {
                    self.occurrence(env.unit, kind.span, Target::Event(found.name), false, false);
                    if let Some(value) = value {
                        self.expect_type(env, value, &Type::Record(found.document));
                    }
                } else {
                    let mut diagnostic = Diagnostic::error(
                        "unknown-event",
                        kind.span,
                        Message::with(
                            message!("{0} is not an event"),
                            [Arg::from(kind.name.as_str())],
                        ),
                    );
                    if let Some(close) =
                        nearest(&kind.name, schema::EVENTS.iter().map(|item| item.name))
                    {
                        diagnostic = diagnostic.with_fix(super::replace_fix(kind.span, close));
                    }
                    self.report(env.unit, diagnostic);
                }
            }
            TestStep::Ack(_) => {
                if !flow {
                    self.wrong_step(env, span, message!("ack is a step of a flow test"));
                }
            }
            TestStep::Advance(duration) => {
                if !flow {
                    self.wrong_step(env, span, message!("advance is a step of a flow test"));
                }
                self.expect_type(env, duration, &Type::Duration);
            }
        }
    }

    #[allow(clippy::too_many_arguments)]
    fn expect_outcome(
        &mut self,
        env: &mut Env,
        negated: bool,
        what: &str,
        span: Span,
        set: Option<&crate::lading::syntax::ast::Ident>,
        detail: Option<&crate::lading::syntax::ast::Expr>,
        subject: Subject,
    ) {
        let flow = subject == Subject::Flow;
        let plural = matches!(what, "holds" | "rejects" | "declines" | "warnings");
        let allowed = if flow {
            matches!(
                what,
                "sent"
                    | "held"
                    | "escalated"
                    | "created"
                    | "updated"
                    | "cancelled"
                    | "applied"
                    | "ignored"
                    | "declined"
                    | "rejected"
            )
        } else if subject == Subject::Guide {
            matches!(what, "rejects" | "rejected" | "warnings" | "warned")
        } else {
            matches!(what, "held" | "rejected" | "declined" | "warned") || plural
        };
        if !allowed {
            self.report(
                env.unit,
                Diagnostic::error(
                    "test-expect",
                    span,
                    if flow {
                        Message::with(message!("A flow test does not check {0}: sent, held, escalated, created, updated, cancelled, applied or ignored"), [Arg::from(what)])
                    } else {
                        Message::with(message!("Here a test checks held, rejected, declined, warned, or no holds, rejects, declines, warnings; not {0}"), [Arg::from(what)])
                    },
                ),
            );
        }
        if plural && !negated {
            self.report(
                env.unit,
                Diagnostic::error(
                    "test-expect",
                    span,
                    Message::with(message!("Write expect no {0}"), [Arg::from(what)]),
                ),
            );
        }
        if let Some(set) = set
            && !matches!(
                set.name.as_str(),
                "204" | "210" | "214" | "990" | "997" | "999"
            )
        {
            self.report(
                env.unit,
                Diagnostic::error(
                    "unknown-set",
                    set.span,
                    Message::with(
                        message!("{0} is not a set a flow sends"),
                        [Arg::from(set.name.as_str())],
                    ),
                ),
            );
        }
        if let Some(detail) = detail {
            if what == "sent" {
                if !matches!(&detail.kind, crate::lading::syntax::ast::ExprKind::Name(word) if word == "accept" || word == "decline")
                {
                    self.wrong_step(
                        env,
                        detail.span,
                        message!("expect sent 990 takes accept or decline"),
                    );
                }
            } else {
                self.expect_type(env, detail, &Type::Text);
            }
        }
    }
}

/// Whether a sample path matches a glob with `*` (within a folder) and `**` (any folders).
#[must_use]
pub fn glob_matches(glob: &str, path: &str) -> bool {
    fn matches(glob: &[u8], path: &[u8]) -> bool {
        match glob.first() {
            None => path.is_empty(),
            Some(b'*') if glob.get(1) == Some(&b'*') => {
                let rest = glob.get(2..).unwrap_or_default();
                let rest = rest.strip_prefix(b"/").unwrap_or(rest);
                (0..=path.len()).any(|index| {
                    (index == 0 || path[index - 1] == b'/') && matches(rest, &path[index..])
                }) || matches(rest, path)
            }
            Some(b'*') => {
                let rest = &glob[1..];
                (0..=path.len())
                    .take_while(|index| *index == 0 || path[index - 1] != b'/')
                    .any(|index| matches(rest, &path[index..]))
            }
            Some(byte) => path.first() == Some(byte) && matches(&glob[1..], &path[1..]),
        }
    }
    matches(glob.as_bytes(), path.as_bytes())
}

#[cfg(test)]
mod globs {
    use super::glob_matches;

    #[test]
    fn globs_match_folders_and_names() {
        assert!(glob_matches("samples/**/*.x12", "samples/a/b.x12"));
        assert!(glob_matches("samples/**/*.x12", "samples/b.x12"));
        assert!(glob_matches("samples/*.x12", "samples/b.x12"));
        assert!(!glob_matches("samples/*.x12", "samples/a/b.x12"));
        assert!(!glob_matches("samples/*.x12", "samples/b.csv"));
    }
}
