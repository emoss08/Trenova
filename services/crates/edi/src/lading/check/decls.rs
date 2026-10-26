//! Checks file-level declarations: function bodies, crosswalk pairs, constants, and the
//! partner's interchange and settings.

use std::collections::BTreeSet;

use trenova_module_kit::diagnostic::Diagnostic;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};

use super::env::Env;
use super::{Checker, Program};
use crate::lading::facts::Target;
use crate::lading::syntax::ast::{
    CrosswalkDecl, Direction, Expr, ExprKind, FnDecl, Interchange, Item, SettingDecl, TextPart,
};
use crate::lading::types::Type;

impl Checker<'_> {
    /// Checks a file's functions, crosswalks and constants.
    pub(crate) fn check_decls(&mut self, unit: usize) {
        let program: &Program = self.program;
        for (index, item) in program.units[unit].items().iter().enumerate() {
            match item {
                Item::Fn(decl) => self.function(unit, decl),
                Item::Crosswalk(decl) => self.crosswalk(unit, decl),
                Item::Const(_) => {
                    self.const_type_of(unit, index);
                }
                _ => {}
            }
        }
    }

    fn function(&mut self, unit: usize, decl: &FnDecl) {
        let Some(signature) = self
            .program
            .fns
            .get(&(unit, decl.name.name.clone()))
            .cloned()
        else {
            return;
        };
        let mut env = Env::new(unit);
        env.function = Some((unit, decl.name.name.clone()));
        let mut seen = BTreeSet::new();
        for (param, (_, ty, _)) in decl.params.iter().zip(&signature.params) {
            if !seen.insert(param.name.name.clone()) {
                self.report(
                    unit,
                    Diagnostic::error(
                        "duplicate-name",
                        param.name.span,
                        Message::with(
                            message!("{0} is declared twice"),
                            [Arg::from(param.name.name.as_str())],
                        ),
                    ),
                );
            }
            if let Some(default) = &param.default {
                self.expect_type(&mut env, default, ty);
            }
            env.bind(&param.name.name, ty.clone(), param.name.span);
            self.occurrence(
                unit,
                param.name.span,
                Target::Local {
                    unit,
                    span: param.name.span,
                },
                true,
                false,
            );
        }
        for binding in &decl.body.lets {
            let ty = self.expr(&mut env, &binding.value, None);
            self.bind_local(&mut env, &binding.name, ty);
        }
        let ty = self.expr(&mut env, &decl.body.value, Some(&signature.ret));
        if !ty.fits(&signature.ret) {
            let diagnostic = if ty.is_optional() && ty.unwrapped().fits(&signature.ret) {
                Diagnostic::error(
                    "optional-unhandled",
                    decl.body.value.span,
                    Message::with(
                        message!(
                            "This may be none, and {0} returns {1}: return {1}? or give a default with ??"
                        ),
                        [
                            Arg::from(decl.name.name.as_str()),
                            Arg::from(signature.ret.to_string()),
                        ],
                    ),
                )
            } else {
                super::expr::mismatch(decl.body.value.span, &signature.ret, &ty)
            };
            self.report(unit, diagnostic);
        }
        let span = decl.span;
        self.scope_fact(&env, span, None);
    }

    fn crosswalk(&mut self, unit: usize, decl: &CrosswalkDecl) {
        let Some((left, right)) = self
            .program
            .crosswalks
            .get(&(unit, decl.name.name.clone()))
            .cloned()
        else {
            return;
        };
        let mut env = Env::new(unit);
        let mut forward = BTreeSet::new();
        let mut backward = BTreeSet::new();
        for pair in &decl.pairs {
            for value in &pair.left {
                self.crosswalk_value(&mut env, value, &left);
            }
            for value in &pair.right {
                self.crosswalk_value(&mut env, value, &right);
            }
            if matches!(pair.direction, Direction::Both | Direction::Forward) {
                for value in &pair.left {
                    self.duplicate_value(unit, value, &mut forward);
                }
            }
            if matches!(pair.direction, Direction::Both | Direction::Backward) {
                for value in &pair.right {
                    self.duplicate_value(unit, value, &mut backward);
                }
            }
        }
        let mut directions = BTreeSet::new();
        for (direction, value) in &decl.otherwise {
            let (target, name) = match direction {
                Direction::Backward => (&left, "<-"),
                _ => (&right, "->"),
            };
            if !directions.insert(name) {
                self.report(
                    unit,
                    Diagnostic::error(
                        "duplicate-else",
                        value.span,
                        Message::new(message!("This direction already has an else")),
                    ),
                );
            }
            self.crosswalk_value(&mut env, value, target);
        }
        if left == right && !matches!(left, Type::Any) {
            self.report(
                unit,
                Diagnostic {
                    severity: trenova_module_kit::diagnostic::Severity::Info,
                    ..Diagnostic::error(
                        "crosswalk-same-type",
                        decl.name.span,
                        Message::with(
                            message!("Both sides are {0}: via {1} reads left to right, via {1}.back right to left"),
                            [Arg::from(left.to_string()), Arg::from(decl.name.name.as_str())],
                        ),
                    )
                },
            );
        }
    }

    fn crosswalk_value(&mut self, env: &mut Env, value: &Expr, ty: &Type) {
        self.expect_type(env, value, ty);
    }

    fn duplicate_value(&mut self, unit: usize, value: &Expr, seen: &mut BTreeSet<String>) {
        let key = match &value.kind {
            ExprKind::Text(parts) => match parts.as_slice() {
                [TextPart::Literal(text)] => text.clone(),
                [] => String::new(),
                _ => return,
            },
            ExprKind::Type(name) | ExprKind::Number(name) => name.clone(),
            _ => return,
        };
        if !seen.insert(key.clone()) {
            self.report(
                unit,
                Diagnostic::error(
                    "crosswalk-duplicate",
                    value.span,
                    Message::with(
                        message!("{0} is already paired in this direction"),
                        [Arg::from(key)],
                    ),
                ),
            );
        }
    }

    fn const_type_of(&mut self, unit: usize, index: usize) {
        let _ = self.const_type(unit, index);
    }

    /// Checks the partner file's interchange and settings.
    pub(crate) fn check_partner_items(&mut self, unit: usize) {
        let program: &Program = self.program;
        let mut interchanges = 0;
        let mut zone = false;
        for item in program.units[unit].items() {
            match item {
                Item::Interchange(interchange) => {
                    interchanges += 1;
                    if interchanges > 1 {
                        self.report(
                            unit,
                            Diagnostic::error(
                                "duplicate-interchange",
                                interchange.span,
                                Message::new(message!("A partner has one interchange")),
                            ),
                        );
                    }
                    self.interchange(unit, interchange);
                }
                Item::Setting(setting) => self.setting(unit, setting),
                Item::Zone(_) => zone = true,
                Item::Fn(_)
                | Item::Crosswalk(_)
                | Item::Const(_)
                | Item::Currency(_)
                | Item::Error(_) => {}
                other => self.report(
                    unit,
                    Diagnostic::error(
                        "partner-item",
                        other.span(),
                        Message::new(message!(
                            "A partner file holds its interchange, zone, currency and settings"
                        )),
                    ),
                ),
            }
        }
        let header = program.units[unit]
            .header()
            .name()
            .map(|name| name.span)
            .unwrap_or_default();
        if interchanges == 0 {
            self.report(
                unit,
                Diagnostic::error(
                    "interchange-missing",
                    header,
                    Message::new(message!(
                        "Say who the partner is on the wire: interchange {{ them ZZ \"ID\" ... }}"
                    )),
                ),
            );
        }
        if !zone {
            self.report(
                unit,
                Diagnostic::warning(
                    "zone-missing",
                    header,
                    Message::new(message!(
                        "Without a zone, times with no zone of their own are read in UTC"
                    )),
                ),
            );
        }
    }

    #[allow(clippy::too_many_lines)]
    fn interchange(&mut self, unit: usize, interchange: &Interchange) {
        let span = interchange.span;
        for (party, label) in [(&interchange.them, "them"), (&interchange.us, "us")] {
            match party {
                Some(party) => {
                    if party.qualifier.name.len() != 2 {
                        self.report(
                            unit,
                            Diagnostic::error(
                                "interchange-qualifier",
                                party.qualifier.span,
                                Message::new(message!(
                                    "An ISA qualifier is two characters, such as ZZ or 01"
                                )),
                            ),
                        );
                    }
                    if party.id.value.is_empty() || party.id.value.len() > 15 {
                        self.report(
                            unit,
                            Diagnostic::error(
                                "interchange-id",
                                party.id.span,
                                Message::new(message!("An interchange ID is 1 to 15 characters")),
                            ),
                        );
                    }
                }
                None => self.report(
                    unit,
                    Diagnostic::error(
                        "interchange-party",
                        span,
                        Message::with(
                            message!("The interchange needs {0} with a qualifier and an ID"),
                            [Arg::from(label)],
                        ),
                    ),
                ),
            }
        }
        if let Some((theirs, ours)) = &interchange.group {
            for code in [theirs, ours] {
                if code.value.is_empty() || code.value.len() > 15 {
                    self.report(
                        unit,
                        Diagnostic::error(
                            "interchange-group",
                            code.span,
                            Message::new(message!("A GS application code is 1 to 15 characters")),
                        ),
                    );
                }
            }
        }
        match &interchange.version {
            Some(version) if matches!(version.name.as_str(), "004010" | "005010") => {}
            Some(version) => self.report(
                unit,
                Diagnostic::error(
                    "interchange-version",
                    version.span,
                    Message::new(message!("Trenova reads and writes X12 004010 and 005010")),
                ),
            ),
            None => self.report(
                unit,
                Diagnostic::error(
                    "interchange-version",
                    span,
                    Message::new(message!("Name the X12 version: version 004010")),
                ),
            ),
        }
        if !interchange.separators.is_empty() {
            let mut seen = BTreeSet::new();
            if interchange.separators.len() != 4 {
                self.report(
                    unit,
                    Diagnostic::error(
                        "interchange-separators",
                        span,
                        Message::new(message!(
                            "separators names four: element, segment, component and repetition"
                        )),
                    ),
                );
            }
            for separator in &interchange.separators {
                let mut chars = separator.value.chars();
                let single = chars
                    .next()
                    .is_some_and(|c| !c.is_alphanumeric() && c != ' ')
                    && chars.next().is_none();
                if !single || !seen.insert(separator.value.clone()) {
                    self.report(
                        unit,
                        Diagnostic::error(
                            "interchange-separators",
                            separator.span,
                            Message::new(message!("Each separator is one character that is not a letter or digit, and no two are the same")),
                        ),
                    );
                }
            }
        }
        if let Some(usage) = &interchange.usage
            && !matches!(usage.name.as_str(), "test" | "production")
        {
            self.report(
                unit,
                Diagnostic::error(
                    "interchange-usage",
                    usage.span,
                    Message::new(message!("usage is test or production")),
                ),
            );
        }
    }

    fn setting(&mut self, unit: usize, setting: &SettingDecl) {
        let Some(ty) = self
            .program
            .partner
            .as_ref()
            .and_then(|partner| {
                partner
                    .settings
                    .iter()
                    .find(|found| found.name == setting.name.name)
            })
            .map(|found| found.ty.clone())
        else {
            return;
        };
        let mut env = Env::new(unit);
        match &setting.default {
            Some(default) => {
                self.expect_type(&mut env, default, &ty);
            }
            None => {
                if !ty.is_optional() {
                    self.report(
                        unit,
                        Diagnostic::error(
                            "setting-default",
                            setting.name.span,
                            Message::new(message!(
                                "A setting needs a default: setting name: Type = value"
                            )),
                        ),
                    );
                }
            }
        }
        if let Some((low, high)) = &setting.range {
            if !ty.is_ordered() {
                self.report(
                    unit,
                    Diagnostic::error(
                        "setting-range",
                        low.span.cover(high.span),
                        Message::with(
                            message!("A {0} setting has no range"),
                            [Arg::from(ty.to_string())],
                        ),
                    ),
                );
            }
            self.expect_type(&mut env, low, &ty);
            self.expect_type(&mut env, high, &ty);
        }
    }
}
