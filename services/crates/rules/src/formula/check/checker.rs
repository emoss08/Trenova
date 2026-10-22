//! The walk over a parsed formula that types every node and reports what is wrong.
//!
//! Guards come first: before a statement is typed, every read that sits under a guard (the left
//! of `??`, an argument of `coalesce`, an operand compared with `nil`, a branch of a conditional
//! whose test compares that name with `nil`) is marked, so a guard written after the read still
//! counts. Reads through `?.` are guarded where they are read. Then each statement is typed in
//! order, with the `let` bindings in scope as a stack: a nested sequence's names are popped when
//! its value has been typed, so sibling scopes may reuse a name and an inner scope may not
//! redeclare one.
//!
//! The walk recurses over the tree, which the parser keeps at most `MAX_DEPTH` deep, and does a
//! bounded amount of work per node, so checking is linear in the formula's size.

use super::Analysis;
use super::Binding;
use super::Change;
use super::Code;
use super::Diagnostic;
use super::Expect;
use super::Fix;
use super::Severity;
use super::TableRef;
use super::knowledge::Doc;
use super::knowledge::Knowledge;
use super::knowledge::RateTable;
use super::knowledge::Schema;
use super::suggest::infer_input_type;
use super::suggest::nearest_known;
use super::types::Type;
use crate::formula::catalog;
use crate::formula::catalog::Function;
use crate::formula::catalog::Param;
use crate::formula::syntax;
use crate::formula::syntax::ast::Binary;
use crate::formula::syntax::ast::BinaryOp;
use crate::formula::syntax::ast::Call;
use crate::formula::syntax::ast::Chain;
use crate::formula::syntax::ast::CompareOp;
use crate::formula::syntax::ast::Expr;
use crate::formula::syntax::ast::ExprKind;
use crate::formula::syntax::ast::Index;
use crate::formula::syntax::ast::Member;
use crate::formula::syntax::ast::Method;
use crate::formula::syntax::ast::Pointer;
use crate::formula::syntax::ast::Sequence;
use crate::formula::syntax::ast::Slice;
use crate::formula::syntax::ast::StatementKind;
use crate::formula::syntax::ast::SyntaxCode;
use crate::formula::syntax::ast::Unary;
use crate::formula::syntax::ast::UnaryOp;
use crate::formula::syntax::split_statements;
use crate::message;
use crate::message::Arg;
use crate::message::Message;
use crate::span::Span;
use std::collections::BTreeMap;
use std::collections::BTreeSet;

fn quote(name: &str) -> String {
    format!("`{name}`")
}

/// An inner phrase passed as an argument, in English: [`Arg`] has no nested message, so the
/// phrase is filled in here and the outer template is what the caller translates.
fn phrase(message: &Message) -> Arg {
    Arg::from(message.clone())
}

fn replace(label: Message, span: Span, insert: String) -> Fix {
    Fix::Replace {
        label,
        changes: vec![Change { span, insert }],
    }
}

/// The guard that keeps a formula pricing when a field is empty: `coalesce(field, 0)`, or for a
/// field under an object `(object?.field ?? 0)`, which also holds when the object is empty.
fn guard_for(path: &str, zero: &str) -> String {
    if path.contains('.') {
        format!("({} ?? {zero})", path.replace('.', "?."))
    } else {
        format!("coalesce({path}, {zero})")
    }
}

/// Whether a member chain reads through `?.` anywhere.
fn reads_optionally(expr: &Expr) -> bool {
    let mut current = expr;
    while let ExprKind::Member(member) = &current.kind {
        if member.optional {
            return true;
        }
        current = &member.object;
    }
    false
}

fn member_chain(expr: &Expr) -> (&Expr, Vec<(&Expr, &Member)>) {
    let mut members = Vec::new();
    let mut current = expr;
    while let ExprKind::Member(member) = &current.kind {
        members.push((current, member));
        current = &member.object;
    }
    members.reverse();
    (current, members)
}

struct Checker<'a> {
    source: &'a str,
    knowledge: &'a Knowledge,
    tables: Option<&'a [RateTable]>,
    bindings: Vec<Binding>,
    binding_index: BTreeMap<String, usize>,
    defined: BTreeSet<String>,
    diagnostics: Vec<Diagnostic>,
    order: Vec<String>,
    read: BTreeSet<String>,
    input_reads: BTreeSet<String>,
    function_calls: Vec<String>,
    called: BTreeSet<String>,
    guarded: BTreeSet<Span>,
    flagged: BTreeSet<String>,
    pointer_depth: u32,
    reduce_depth: u32,
    elements: Vec<Option<String>>,
    lookup_tables: Vec<TableRef>,
}

pub(super) fn run(
    source: &str,
    expect: Expect,
    schema: Option<&Schema>,
    tables: Option<&[RateTable]>,
) -> Analysis {
    let knowledge = schema.map(Knowledge::new).unwrap_or_default();
    let operators: Vec<&str> = schema
        .map(|schema| schema.operators.iter().map(String::as_str).collect())
        .unwrap_or_default();
    let program = syntax::parse(source, &operators);
    let mut checker = Checker {
        source,
        knowledge: &knowledge,
        tables,
        bindings: Vec::new(),
        binding_index: BTreeMap::new(),
        defined: BTreeSet::new(),
        diagnostics: Vec::new(),
        order: Vec::new(),
        read: BTreeSet::new(),
        input_reads: BTreeSet::new(),
        function_calls: Vec::new(),
        called: BTreeSet::new(),
        guarded: BTreeSet::new(),
        flagged: BTreeSet::new(),
        pointer_depth: 0,
        reduce_depth: 0,
        elements: Vec::new(),
        lookup_tables: Vec::new(),
    };
    for diagnostic in &program.diagnostics {
        checker.diagnostics.push(Diagnostic {
            span: diagnostic.span,
            severity: Severity::Error,
            code: match diagnostic.code {
                SyntaxCode::Syntax => Code::Syntax,
                SyntaxCode::Limit => Code::Limit,
            },
            message: diagnostic.message.clone(),
            name: None,
            fixes: diagnostic
                .fix
                .iter()
                .map(|fix| replace(fix.label.clone(), diagnostic.span, fix.insert.clone()))
                .collect(),
        });
    }
    let mut result_type = Type::Any;
    if program.empty {
        let end = u32::try_from(source.len()).unwrap_or(u32::MAX);
        checker.error(
            Span::new(0, end),
            Code::Empty,
            Message::new(message!("The formula is empty")),
        );
    } else {
        result_type = checker.statements(&program, expect);
    }
    let pending = schema.is_none();
    let mut seen = BTreeSet::new();
    let mut diagnostics: Vec<Diagnostic> = checker
        .diagnostics
        .drain(..)
        .filter(|diagnostic| !(pending && diagnostic.code.needs_schema()))
        .filter(|diagnostic| {
            seen.insert((
                diagnostic.span,
                diagnostic.code,
                diagnostic.message.message,
                diagnostic.message.english(),
            ))
        })
        .collect();
    diagnostics.sort_by_key(|diagnostic| (diagnostic.span.start, diagnostic.span.end));
    let input_reads: Vec<String> = checker
        .order
        .iter()
        .filter(|name| checker.input_reads.contains(*name))
        .cloned()
        .collect();
    let schema_reads: Vec<String> = checker
        .order
        .iter()
        .filter(|name| !checker.input_reads.contains(*name))
        .cloned()
        .collect();
    let reads = input_reads.iter().chain(&schema_reads).cloned().collect();
    let statements = split_statements(&program, source);
    Analysis {
        empty: program.empty,
        pending,
        diagnostics,
        result_type,
        reads,
        schema_reads,
        input_reads,
        function_calls: checker.function_calls,
        bindings: checker.bindings,
        statements,
        lookup_tables: checker.lookup_tables,
        program,
    }
}

impl Checker<'_> {
    fn report(
        &mut self,
        span: Span,
        severity: Severity,
        code: Code,
        message: Message,
        name: Option<String>,
        fixes: Vec<Fix>,
    ) {
        self.diagnostics.push(Diagnostic {
            span,
            severity,
            code,
            message,
            name,
            fixes,
        });
    }

    fn error(&mut self, span: Span, code: Code, message: Message) {
        self.report(span, Severity::Error, code, message, None, Vec::new());
    }

    fn slice(&self, span: Span) -> &str {
        self.source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default()
    }

    /// The path an item's field is read by, such as `.weight` or `#.location.city`.
    fn item_path(&self, expr: &Expr) -> Option<String> {
        let (base, members) = member_chain(expr);
        if members.is_empty()
            || !matches!(
                base.kind,
                ExprKind::Pointer(Pointer::Implicit | Pointer::Element)
            )
        {
            return None;
        }
        let fields: Vec<&str> = members
            .iter()
            .map(|(_, member)| member.property.text.as_str())
            .collect();
        Some(format!(
            "{}.{}",
            self.slice(base.span).trim(),
            fields.join(".")
        ))
    }

    /// `` `x` is a number ``, `'abc' is text`, `this part is a list`.
    fn describe(&self, expr: &Expr, ty: Type) -> Message {
        if let Some(path) = expr.path().or_else(|| self.item_path(expr)) {
            return Message::with(
                message!("{0} is {1}"),
                [Arg::from(quote(&path)), phrase(&ty.words())],
            );
        }
        match &expr.kind {
            ExprKind::String(value) => {
                Message::with(message!("'{0}' is text"), [Arg::from(value.as_str())])
            }
            ExprKind::Bool(value) => Message::with(
                message!("'{0}' is true/false"),
                [Arg::from(if *value { "true" } else { "false" })],
            ),
            _ => Message::with(message!("this part is {0}"), [phrase(&ty.words())]),
        }
    }

    fn record_read(&mut self, name: &str, input: bool) {
        if self.read.insert(name.to_owned()) {
            self.order.push(name.to_owned());
            if input {
                self.input_reads.insert(name.to_owned());
            }
        }
    }

    fn record_call(&mut self, name: &str) {
        if self.called.insert(name.to_owned()) {
            self.function_calls.push(name.to_owned());
        }
    }

    fn statements(&mut self, program: &syntax::Program, expect: Expect) -> Type {
        self.register_lets(program);

        let mut result_type = Type::Any;
        let last = program.statements.len().saturating_sub(1);
        for (index, statement) in program.statements.iter().enumerate() {
            self.pre_mark_guards(&statement.value);
            match &statement.kind {
                StatementKind::Let(name) => {
                    if !name.text.is_empty() {
                        self.check_let_name(&name.text, name.span);
                    }
                    let ty = self.type_of(&statement.value);
                    if let Some(&at) = self.binding_index.get(&name.text)
                        && let Some(binding) = self.bindings.get_mut(at)
                        && binding.name_span == name.span
                    {
                        binding.ty = ty;
                        self.defined.insert(name.text.clone());
                    }
                    if index == last && !name.text.is_empty() {
                        let (at, insert) = match statement.semicolon {
                            Some(semicolon) => (semicolon.end, format!("\n{}", name.text)),
                            None => (statement.span.end, format!(";\n{}", name.text)),
                        };
                        self.report(
                            statement.span,
                            Severity::Error,
                            Code::LetLast,
                            Message::new(message!(
                                "The formula ends on a 'let'; add the amount it charges on the last line"
                            )),
                            None,
                            vec![replace(
                                Message::with(
                                    message!("Return {0}"),
                                    [Arg::from(name.text.as_str())],
                                ),
                                Span::new(at, at),
                                insert,
                            )],
                        );
                    }
                }
                StatementKind::Result => {
                    let ty = self.type_of(&statement.value);
                    if index < last {
                        self.report(
                            statement.span,
                            Severity::Warning,
                            Code::Discarded,
                            Message::new(message!(
                                "This line's value is worked out and thrown away; only the last line is the charge. Bind it with 'let' to use it."
                            )),
                            None,
                            Vec::new(),
                        );
                    } else {
                        result_type = ty;
                        self.check_result(&statement.value, ty, expect);
                    }
                }
            }
        }

        let unused: Vec<(Span, String)> = self
            .bindings
            .iter()
            .filter(|binding| binding.reads == 0)
            .map(|binding| (binding.name_span, binding.name.clone()))
            .collect();
        for (span, name) in unused {
            self.report_unused(span, name);
        }
        result_type
    }

    fn report_unused(&mut self, span: Span, name: String) {
        self.report(
            span,
            Severity::Warning,
            Code::LetUnused,
            Message::with(message!("{0} is never used"), [Arg::from(quote(&name))]),
            Some(name),
            Vec::new(),
        );
    }

    fn check_result(&mut self, value: &Expr, ty: Type, expect: Expect) {
        match expect {
            Expect::Amount => {
                if ty == Type::Boolean {
                    let name = value
                        .path()
                        .map_or_else(|| Arg::from(Message::new(message!("test"))), Arg::from);
                    self.error(
                        value.span,
                        Code::ResultType,
                        Message::with(
                            message!(
                                "The formula must come to an amount, but this comes to true/false. Write {0} ? amount : 0"
                            ),
                            [name],
                        ),
                    );
                } else if !ty.is_numeric() {
                    self.error(
                        value.span,
                        Code::ResultType,
                        Message::with(
                            message!("The formula must come to an amount, but this comes to {0}"),
                            [phrase(&ty.words())],
                        ),
                    );
                }
            }
            Expect::Predicate => {
                if !(ty.is_numeric() || ty == Type::Boolean) {
                    self.error(
                        value.span,
                        Code::ResultType,
                        Message::with(
                            message!(
                                "The formula must come to true/false or a number, but this comes to {0}"
                            ),
                            [phrase(&ty.words())],
                        ),
                    );
                }
            }
        }
    }

    /// A `let` may not hide a field, an input or a function.
    fn check_let_name(&mut self, name: &str, span: Span) {
        let knowledge = self.knowledge;
        let doc = knowledge.top.doc(name);
        if doc.is_some() || knowledge.top.is_root(name) {
            let message = if doc.is_some_and(|doc| doc.input) {
                message!(
                    "{0} is one of this template's inputs, and a 'let' can't redeclare it. Use another name"
                )
            } else {
                message!(
                    "{0} is already a shipment field, and a 'let' can't redeclare it. Use another name"
                )
            };
            self.report(
                span,
                Severity::Error,
                Code::LetShadows,
                Message::with(message, [Arg::from(quote(name))]),
                Some(name.to_owned()),
                Vec::new(),
            );
            return;
        }
        if catalog::callable(name).is_some() {
            self.report(
                span,
                Severity::Error,
                Code::LetShadows,
                Message::with(
                    message!(
                        "{0} is the name of a formula function, and a 'let' can't take it. Use another name"
                    ),
                    [Arg::from(quote(name))],
                ),
                Some(name.to_owned()),
                Vec::new(),
            );
        }
    }

    fn type_of(&mut self, expr: &Expr) -> Type {
        match &expr.kind {
            ExprKind::Number(number) => {
                if number.whole {
                    Type::Integer
                } else {
                    Type::Number
                }
            }
            ExprKind::String(_) => Type::String,
            ExprKind::Bool(_) => Type::Boolean,
            ExprKind::Nil => Type::Nil,
            ExprKind::Missing => Type::Any,
            ExprKind::Pointer(pointer) => self.type_pointer(expr, pointer),
            ExprKind::Ident(name) => self.type_ident(expr, name),
            ExprKind::Member(member) => self.type_member(expr, member),
            ExprKind::Index(index) => self.type_index(expr, index),
            ExprKind::Slice(slice) => self.type_slice(expr, slice),
            ExprKind::Call(call) => self.type_call(expr, call),
            ExprKind::Method(method) => self.type_method(method),
            ExprKind::Sequence(sequence) => self.type_sequence(sequence),
            ExprKind::Unary(unary) => self.type_unary(unary),
            ExprKind::Binary(binary) => self.type_binary(binary),
            ExprKind::Chain(chain) => self.type_chain(chain),
            ExprKind::Ternary(ternary) => {
                let test = self.type_of(&ternary.test);
                self.check_test(&ternary.test, test);
                let then = match &ternary.then {
                    Some(then) => self.type_of(then),
                    None => test,
                };
                let otherwise = self.type_of(&ternary.otherwise);
                then.join(otherwise)
            }
            ExprKind::If(branch) => {
                let test = self.type_of(&branch.test);
                self.check_test(&branch.test, test);
                let then = self.type_of(&branch.then);
                let otherwise = self.type_of(&branch.otherwise);
                then.join(otherwise)
            }
            ExprKind::Array(items) => {
                for item in items {
                    self.type_of(item);
                }
                Type::Array
            }
            ExprKind::Map(entries) => {
                for entry in entries {
                    self.type_of(&entry.value);
                }
                Type::Object
            }
            ExprKind::Predicate(predicate) => {
                self.in_item(None, false, |checker| checker.type_of(&predicate.body));
                Type::Any
            }
        }
    }

    /// Runs `body` with `#` and `.field` reading an item of `list`.
    fn in_item(
        &mut self,
        list: Option<String>,
        reduce: bool,
        body: impl FnOnce(&mut Self) -> Type,
    ) -> Type {
        self.pointer_depth += 1;
        if reduce {
            self.reduce_depth += 1;
        }
        self.elements.push(list);
        let ty = body(self);
        self.elements.pop();
        if reduce {
            self.reduce_depth -= 1;
        }
        self.pointer_depth -= 1;
        ty
    }

    fn not_indexable(&mut self, expr: &Expr, object: &Expr, ty: Type) {
        self.error(
            expr.span,
            Code::Type,
            Message::with(
                message!("{0}; it can't be indexed with [ ]"),
                [phrase(&self.describe(object, ty))],
            ),
        );
    }

    fn check_test(&mut self, test: &Expr, ty: Type) {
        if ty == Type::Boolean || ty.is_open() {
            return;
        }
        let message = match test.path() {
            Some(name) if ty.is_number() => Message::with(
                message!(
                    "The test before '?' must be true/false, but {0}. Compare it: {1} > 0 ? …"
                ),
                [phrase(&self.describe(test, ty)), Arg::from(name)],
            ),
            _ => Message::with(
                message!("The test before '?' must be true/false, but {0}"),
                [phrase(&self.describe(test, ty))],
            ),
        };
        self.error(test.span, Code::Type, message);
    }

    fn type_pointer(&mut self, expr: &Expr, pointer: &Pointer) -> Type {
        if self.pointer_depth == 0 {
            self.error(
                expr.span,
                Code::Syntax,
                Message::new(message!(
                    "'#' stands for the current item inside filter(), map() and the like"
                )),
            );
            return Type::Any;
        }
        match pointer {
            Pointer::Accumulator if self.reduce_depth == 0 => {
                self.error(
                    expr.span,
                    Code::Syntax,
                    Message::new(message!(
                        "'#acc' is the running total inside reduce() and nowhere else"
                    )),
                );
                Type::Any
            }
            Pointer::Unknown(name) => {
                self.error(
                    expr.span,
                    Code::Syntax,
                    Message::with(
                        message!("'#{0}' isn't a pointer; use #, #index or, inside reduce(), #acc"),
                        [Arg::from(name.as_str())],
                    ),
                );
                Type::Any
            }
            Pointer::Index => Type::Integer,
            _ => Type::Any,
        }
    }

    fn type_ident(&mut self, expr: &Expr, name: &str) -> Type {
        if let Some(&at) = self.binding_index.get(name) {
            if !self.defined.contains(name) {
                self.report(
                    expr.span,
                    Severity::Error,
                    Code::LetBeforeDefinition,
                    Message::with(
                        message!(
                            "{0} is defined further down; a 'let' must come before the lines that read it"
                        ),
                        [Arg::from(quote(name))],
                    ),
                    Some(name.to_owned()),
                    Vec::new(),
                );
            }
            if let Some(binding) = self.bindings.get_mut(at) {
                binding.reads += 1;
                return binding.ty;
            }
        }
        let knowledge = self.knowledge;
        if let Some(doc) = knowledge.top.doc(name) {
            self.record_read(name, doc.input);
            self.check_nullable(expr, doc, name, name, guard_for(name, doc.zero));
            return doc.ty;
        }
        if knowledge.top.is_root(name) {
            self.record_read(name, false);
            return Type::Object;
        }
        let suggestion = nearest_known(
            name,
            knowledge
                .top
                .names()
                .chain(self.bindings.iter().map(|binding| binding.name.as_str())),
        )
        .map(str::to_owned);
        let inferred = infer_input_type(name);
        let mut fixes = Vec::new();
        if let Some(suggestion) = &suggestion {
            fixes.push(replace(
                Message::with(message!("Change to {0}"), [Arg::from(suggestion.as_str())]),
                expr.span,
                suggestion.clone(),
            ));
        }
        fixes.push(Fix::Declare {
            label: Message::with(
                message!("Declare {0} as a {1} input"),
                [Arg::from(name), Arg::from(inferred.as_str())],
            ),
            name: name.to_owned(),
            input_type: inferred,
        });
        let message = match &suggestion {
            Some(suggestion) => Message::with(
                message!(
                    "{0} isn't a shipment field or one of this template's inputs. Did you mean {1}?"
                ),
                [Arg::from(quote(name)), Arg::from(quote(suggestion))],
            ),
            None => Message::with(
                message!(
                    "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference."
                ),
                [Arg::from(quote(name))],
            ),
        };
        self.report(
            expr.span,
            Severity::Error,
            Code::UnknownVariable,
            message,
            Some(name.to_owned()),
            fixes,
        );
        Type::Any
    }

    /// Warns once per field when a field that can be empty is read without a guard.
    fn check_nullable(&mut self, expr: &Expr, doc: &Doc, display: &str, key: &str, guard: String) {
        if !doc.nullable
            || self.guarded.contains(&expr.span)
            || reads_optionally(expr)
            || self.flagged.contains(key)
        {
            return;
        }
        self.flagged.insert(key.to_owned());
        self.report(
            expr.span,
            Severity::Warning,
            Code::Nullable,
            Message::with(
                message!(
                    "{0} is empty on some shipments, and the formula would fail to rate them. Give it a fallback: {1}"
                ),
                [Arg::from(quote(display)), Arg::from(guard.as_str())],
            ),
            Some(display.to_owned()),
            vec![replace(
                Message::with(message!("Use {0}"), [Arg::from(guard.as_str())]),
                expr.span,
                guard,
            )],
        );
    }

    fn type_member(&mut self, expr: &Expr, member: &Member) -> Type {
        let (base, members) = member_chain(expr);
        match &base.kind {
            ExprKind::Pointer(Pointer::Implicit | Pointer::Element) => {
                if self.pointer_depth == 0 {
                    let first = members.first().map_or(expr, |(node, _)| *node);
                    let property = members
                        .first()
                        .map_or(member.property.text.as_str(), |(_, m)| {
                            m.property.text.as_str()
                        });
                    self.error(
                        first.span,
                        Code::Syntax,
                        Message::with(
                            message!(
                                "{0} reads a field of the current item, so it only works inside filter(), map() and the like"
                            ),
                            [Arg::from(quote(&format!(".{property}")))],
                        ),
                    );
                    return Type::Any;
                }
                let list = self.elements.last().cloned().flatten();
                return self.resolve_element(list, base, &members);
            }
            ExprKind::Ident(name)
                if !self.binding_index.contains_key(name) && self.knowledge.top.is_root(name) =>
            {
                return self.resolve_top(name, base, &members);
            }
            _ => {}
        }
        if let Some(list) = self.element_of(base) {
            self.type_of(base);
            return self.resolve_element(Some(list), base, &members);
        }
        let object = self.type_of(&member.object);
        self.member_on(object, &member.object, member)
    }

    fn member_on(&mut self, ty: Type, object: &Expr, member: &Member) -> Type {
        if ty.is_scalar() || ty == Type::Array {
            self.error(
                member.property.span,
                Code::Type,
                Message::with(
                    message!("{0}; it has no field {1}"),
                    [
                        phrase(&self.describe(object, ty)),
                        Arg::from(quote(&member.property.text)),
                    ],
                ),
            );
        }
        Type::Any
    }

    /// A member chain from a record root, such as `customer.name`.
    fn resolve_top(&mut self, root: &str, base: &Expr, members: &[(&Expr, &Member)]) -> Type {
        let knowledge = self.knowledge;
        let namespace = &knowledge.top;
        let path_upto = |count: usize| {
            let mut path = root.to_owned();
            for (_, member) in members.iter().take(count) {
                path.push('.');
                path.push_str(&member.property.text);
            }
            path
        };
        let count = members.len();
        let full = path_upto(count);
        let Some((outer, _)) = members.last() else {
            return Type::Any;
        };
        if let Some(doc) = namespace.doc(&full) {
            self.record_read(&full, doc.input);
            self.check_nullable(outer, doc, &full, &full, guard_for(&full, doc.zero));
            return doc.ty;
        }
        if namespace.is_prefix(&full) {
            self.record_read(&full, false);
            return Type::Object;
        }
        for known in (1..count).rev() {
            let path = path_upto(known);
            if let Some(doc) = namespace.doc(&path) {
                self.record_read(&path, doc.input);
                let (node, _) = members[known - 1];
                self.check_nullable(node, doc, &path, &path, guard_for(&path, doc.zero));
                let mut ty = doc.ty;
                for (_, member) in members.iter().skip(known) {
                    ty = self.member_on(ty, &member.object, member);
                }
                return ty;
            }
        }
        let known = (1..count)
            .rev()
            .find(|known| namespace.is_prefix(&path_upto(*known)))
            .unwrap_or(0);
        let (bad_node, bad) = members[known];
        let attempted = path_upto(known + 1);
        let under = format!("{root}.");
        let suggestion = nearest_known(
            &attempted,
            namespace.names().filter(|name| name.starts_with(&under)),
        )
        .map(str::to_owned);
        self.report_unknown_field(
            &quote(&path_upto(known)),
            bad,
            suggestion.as_deref(),
            Span::new(base.span.start, bad_node.span.end),
            full,
            false,
        );
        self.record_read(root, false);
        Type::Any
    }

    fn report_unknown_field(
        &mut self,
        owner: &str,
        bad: &Member,
        suggestion: Option<&str>,
        fix_span: Span,
        name: String,
        item: bool,
    ) {
        let property = quote(&bad.property.text);
        let message = match (suggestion, item) {
            (Some(suggestion), false) => Message::with(
                message!("{0} has no field {1}. Did you mean {2}?"),
                [
                    Arg::from(owner),
                    Arg::from(property),
                    Arg::from(quote(suggestion)),
                ],
            ),
            (None, false) => Message::with(
                message!("{0} has no field {1}"),
                [Arg::from(owner), Arg::from(property)],
            ),
            (Some(suggestion), true) => Message::with(
                message!("An item of {0} has no field {1}. Did you mean {2}?"),
                [
                    Arg::from(owner),
                    Arg::from(property),
                    Arg::from(quote(suggestion)),
                ],
            ),
            (None, true) => Message::with(
                message!("An item of {0} has no field {1}"),
                [Arg::from(owner), Arg::from(property)],
            ),
        };
        let fixes = suggestion
            .map(|suggestion| {
                vec![replace(
                    Message::with(message!("Change to {0}"), [Arg::from(suggestion)]),
                    fix_span,
                    suggestion.to_owned(),
                )]
            })
            .unwrap_or_default();
        self.report(
            bad.property.span,
            Severity::Error,
            Code::UnknownField,
            message,
            Some(name),
            fixes,
        );
    }

    /// A member chain on an item of a list: `.weight`, `#.weight`, `stops[0].weight`.
    fn resolve_element(
        &mut self,
        list: Option<String>,
        base: &Expr,
        members: &[(&Expr, &Member)],
    ) -> Type {
        let knowledge = self.knowledge;
        let Some(list) = list else {
            return Type::Any;
        };
        let Some(namespace) = knowledge.element(&list) else {
            return Type::Any;
        };
        let segments: Vec<&str> = members
            .iter()
            .map(|(_, member)| member.property.text.as_str())
            .collect();
        let base_text = self.slice(base.span).trim().to_owned();
        let display = |count: usize| format!("{base_text}.{}", segments[..count].join("."));
        let guard = |count: usize, zero: &str| {
            let mut path = format!(
                "{base_text}.{}",
                segments.first().copied().unwrap_or_default()
            );
            for segment in segments.iter().take(count).skip(1) {
                path.push_str("?.");
                path.push_str(segment);
            }
            format!("({path} ?? {zero})")
        };
        let count = segments.len();
        let full = segments.join(".");
        let Some((outer, _)) = members.last() else {
            return Type::Any;
        };
        if let Some(doc) = namespace.doc(&full) {
            let key = format!("{list}[].{full}");
            self.check_nullable(outer, doc, &display(count), &key, guard(count, doc.zero));
            return doc.ty;
        }
        if namespace.is_prefix(&full) {
            return Type::Object;
        }
        for known in (1..count).rev() {
            let path = segments[..known].join(".");
            if let Some(doc) = namespace.doc(&path) {
                let (node, _) = members[known - 1];
                let key = format!("{list}[].{path}");
                self.check_nullable(node, doc, &display(known), &key, guard(known, doc.zero));
                let mut ty = doc.ty;
                for (_, member) in members.iter().skip(known) {
                    ty = self.member_on(ty, &member.object, member);
                }
                return ty;
            }
        }
        let known = (1..count)
            .rev()
            .find(|known| namespace.is_prefix(&segments[..*known].join(".")))
            .unwrap_or(0);
        let (_, bad) = members[known];
        let attempted = segments[..=known].join(".");
        let suggestion = nearest_known(&attempted, namespace.names()).map(str::to_owned);
        let first = members
            .first()
            .map_or(bad.property.span, |(_, m)| m.property.span);
        let owner = if known == 0 {
            quote(&list)
        } else {
            quote(&display(known))
        };
        self.report_unknown_field(
            &owner,
            bad,
            suggestion.as_deref(),
            Span::new(first.start, bad.property.span.end),
            format!("{list}[].{full}"),
            known == 0,
        );
        Type::Any
    }

    /// The list a value's items come from, when the schema knows that list's item fields.
    fn element_source(&self, expr: &Expr) -> Option<String> {
        match &expr.kind {
            ExprKind::Ident(_) | ExprKind::Member(_) => {
                if let Some(path) = expr.path()
                    && !self
                        .binding_index
                        .contains_key(path.split('.').next().unwrap_or_default())
                {
                    let doc = self.knowledge.top.doc(&path)?;
                    return (doc.ty == Type::Array && self.knowledge.element(&path).is_some())
                        .then_some(path);
                }
                let (base, members) = member_chain(expr);
                let list = self.element_of(base)?;
                let fields: Vec<&str> = members
                    .iter()
                    .map(|(_, member)| member.property.text.as_str())
                    .collect();
                let key = format!("{list}[].{}", fields.join("."));
                self.knowledge.element(&key).is_some().then_some(key)
            }
            ExprKind::Call(call)
                if matches!(
                    call.function.text.as_str(),
                    "filter" | "sortBy" | "take" | "reverse" | "uniq"
                ) =>
            {
                call.args.first().and_then(|list| self.element_source(list))
            }
            ExprKind::Slice(slice) => self.element_source(&slice.object),
            _ => None,
        }
    }

    /// The list a value is one item of, when known: `#`, `.field`'s item, `stops[0]`,
    /// `first(stops)`, `find(stops, ...)`.
    fn element_of(&self, expr: &Expr) -> Option<String> {
        match &expr.kind {
            ExprKind::Pointer(Pointer::Element | Pointer::Implicit) => {
                self.elements.last().cloned().flatten()
            }
            ExprKind::Index(index) => self.element_source(&index.object),
            ExprKind::Call(call)
                if matches!(
                    call.function.text.as_str(),
                    "first" | "last" | "find" | "findLast"
                ) =>
            {
                call.args.first().and_then(|list| self.element_source(list))
            }
            _ => None,
        }
    }

    /// Whether an argument reads the current item (`#`, `.field`) outside a call that binds one.
    fn reads_item(expr: &Expr) -> bool {
        match &expr.kind {
            ExprKind::Pointer(_) | ExprKind::Predicate(_) => return true,
            ExprKind::Call(call)
                if catalog::callable(&call.function.text).is_some_and(|f| f.predicate) =>
            {
                return call.args.first().is_some_and(Self::reads_item);
            }
            _ => {}
        }
        let mut found = false;
        expr.for_each_child(|child| found = found || Self::reads_item(child));
        found
    }

    fn type_call(&mut self, expr: &Expr, call: &Call) -> Type {
        let name = call.function.text.as_str();
        let knowledge = self.knowledge;
        let is_value = knowledge.top.doc(name).is_some() || self.binding_index.contains_key(name);
        let Some(function) = catalog::callable(name) else {
            if let Some(disabled) = catalog::disabled(name).filter(|_| !is_value) {
                let fixes = match disabled.replacement {
                    Some(replacement) if call.args.is_empty() => vec![replace(
                        Message::with(message!("Change to {0}"), [Arg::from(replacement)]),
                        expr.span,
                        replacement.to_owned(),
                    )],
                    _ => Vec::new(),
                };
                self.report(
                    call.function.span,
                    Severity::Error,
                    Code::DisabledFunction,
                    disabled.message(),
                    Some(name.to_owned()),
                    fixes,
                );
            } else {
                self.unknown_function(call, is_value);
            }
            for arg in &call.args {
                self.type_of(arg);
            }
            return Type::Any;
        };
        self.record_call(name);

        let mut types = Vec::with_capacity(call.args.len());
        for (index, arg) in call.args.iter().enumerate() {
            let ty = if function.predicate && index == 1 {
                let list = call.args.first().and_then(|list| self.element_source(list));
                let body = match &arg.kind {
                    ExprKind::Predicate(predicate) => &*predicate.body,
                    _ => arg,
                };
                self.in_item(list, name == "reduce", |checker| checker.type_of(body))
            } else if !function.predicate
                && index > 0
                && self.pointer_depth == 0
                && Self::reads_item(arg)
            {
                self.predicate_on_a_function(call, expr, name, arg);
                self.in_item(None, false, |checker| checker.type_of(arg))
            } else {
                self.type_of(arg)
            };
            types.push(ty);
        }

        if !function.takes(call.args.len()) {
            self.report(
                call.parens,
                Severity::Error,
                Code::Arity,
                arity_message(function.signature, function, call.args.len()),
                Some(name.to_owned()),
                Vec::new(),
            );
        }
        for (index, (arg, ty)) in call.args.iter().zip(&types).enumerate() {
            self.check_param(function, function.param(index), arg, *ty);
        }
        if function.reads_table() {
            self.check_table(function, call);
        }
        if name == "coalesce" {
            return types.iter().fold(Type::Nil, |joined, ty| joined.join(*ty));
        }
        Type::from_returns(function.returns)
    }

    fn unknown_function(&mut self, call: &Call, is_value: bool) {
        let name = call.function.text.as_str();
        let suggestion = if is_value {
            None
        } else {
            nearest_known(
                name,
                catalog::FUNCTIONS
                    .iter()
                    .filter(|function| !function.operator)
                    .map(|function| function.name),
            )
        };
        let message = if is_value {
            Message::with(
                message!("{0} is a value, not a function; it can't be called with ( )"),
                [Arg::from(quote(name))],
            )
        } else if let Some(suggestion) = suggestion {
            Message::with(
                message!("{0} isn't a formula function. Did you mean {1}?"),
                [Arg::from(quote(name)), Arg::from(quote(suggestion))],
            )
        } else {
            Message::with(
                message!("{0} isn't a formula function; pick one from the reference"),
                [Arg::from(quote(name))],
            )
        };
        let fixes = suggestion
            .map(|suggestion| {
                vec![replace(
                    Message::with(message!("Change to {0}"), [Arg::from(suggestion)]),
                    call.function.span,
                    suggestion.to_owned(),
                )]
            })
            .unwrap_or_default();
        self.report(
            call.function.span,
            Severity::Error,
            Code::UnknownFunction,
            message,
            Some(name.to_owned()),
            fixes,
        );
    }

    /// A predicate given to a function that takes values, such as `sum(stops, .weight)`.
    fn predicate_on_a_function(&mut self, call: &Call, expr: &Expr, name: &str, arg: &Expr) {
        let mut fixes = Vec::new();
        if call.args.len() == 2
            && let Some(list) = call.args.first()
        {
            let insert = format!(
                "{name}(map({}, {}))",
                self.slice(list.span),
                self.slice(arg.span)
            );
            fixes.push(replace(
                Message::with(message!("Use {0}(map(…))"), [Arg::from(name)]),
                expr.span,
                insert,
            ));
        }
        self.report(
            arg.span,
            Severity::Error,
            Code::Type,
            Message::with(
                message!(
                    "{0} adds up values and does not take a predicate; map the list first: {0}(map(list, .field))"
                ),
                [Arg::from(name)],
            ),
            Some(name.to_owned()),
            fixes,
        );
    }

    fn check_param(&mut self, function: &Function, param: Param, arg: &Expr, ty: Type) {
        if ty.is_open() {
            return;
        }
        let name = function.name;
        let template = match param {
            Param::Any | Param::Predicate | Param::Table => return,
            Param::Number if ty.is_number() => return,
            Param::Aggregate if ty.is_number() || ty == Type::Array => return,
            Param::Integer if ty == Type::Integer => return,
            Param::Integer if ty == Type::Number => {
                self.not_whole(name, arg);
                return;
            }
            Param::Number | Param::Aggregate | Param::Integer => {
                message!("{0} takes numbers, but {1}")
            }
            Param::Text if ty == Type::String => return,
            Param::Text => message!("{0} takes text, but {1}"),
            Param::List if ty == Type::Array => return,
            Param::List => message!("{0} takes a list, but {1}"),
            Param::Record if ty == Type::Object => return,
            Param::Record => message!("{0} takes a record, but {1}"),
            Param::Sized if matches!(ty, Type::String | Type::Array | Type::Object) => return,
            Param::Sized => message!("{0} takes text, a list or a record, but {1}"),
            Param::Convertible if ty == Type::String || ty.is_number() => return,
            Param::Convertible => message!("{0} takes text or a number, but {1}"),
        };
        self.error(
            arg.span,
            Code::Type,
            Message::with(template, [Arg::from(name), phrase(&self.describe(arg, ty))]),
        );
    }

    /// A number that can hold a fraction where a whole number is needed.
    fn not_whole(&mut self, what: &str, operand: &Expr) {
        let path = operand.path();
        let message = match &path {
            Some(path) => Message::with(
                message!(
                    "{0} takes whole numbers, but {1} can hold a fraction. Use int({2}) to take its whole part"
                ),
                [
                    Arg::from(what),
                    Arg::from(quote(path)),
                    Arg::from(path.as_str()),
                ],
            ),
            None => Message::with(
                message!("{0} takes whole numbers, but this part can come to a fraction"),
                [Arg::from(what)],
            ),
        };
        let fixes = path
            .map(|path| {
                vec![replace(
                    Message::with(message!("Change to int({0})"), [Arg::from(path.as_str())]),
                    operand.span,
                    format!("int({path})"),
                )]
            })
            .unwrap_or_default();
        self.report(
            operand.span,
            Severity::Error,
            Code::Type,
            message,
            None,
            fixes,
        );
    }

    fn check_table(&mut self, function: &Function, call: &Call) {
        let Some(first) = call.args.first() else {
            return;
        };
        let ExprKind::String(table) = &first.kind else {
            if !first.is_missing() {
                self.error(
                    first.span,
                    Code::RateTable,
                    Message::with(
                        message!("{0} names its rate table in quotes: {0}(\"zone_rates\", …)"),
                        [Arg::from(function.name)],
                    ),
                );
            }
            return;
        };
        self.lookup_tables.push(TableRef {
            name: table.clone(),
            function: function.name,
            span: first.span,
        });
        let Some(tables) = self.tables else {
            return;
        };
        let Some(found) = tables.iter().find(|candidate| candidate.name == *table) else {
            let suggestion = nearest_known(
                table,
                tables.iter().map(|candidate| candidate.name.as_str()),
            )
            .map(str::to_owned);
            let message = match &suggestion {
                Some(suggestion) => Message::with(
                    message!("There is no rate table named {0}. Did you mean {1}?"),
                    [Arg::from(quote(table)), Arg::from(quote(suggestion))],
                ),
                None => Message::with(
                    message!("There is no rate table named {0}"),
                    [Arg::from(quote(table))],
                ),
            };
            let fixes = suggestion
                .map(|suggestion| {
                    vec![replace(
                        Message::with(message!("Change to {0}"), [Arg::from(suggestion.as_str())]),
                        first.span,
                        format!("\"{suggestion}\""),
                    )]
                })
                .unwrap_or_default();
            self.report(
                first.span,
                Severity::Error,
                Code::RateTable,
                message,
                Some(table.clone()),
                fixes,
            );
            return;
        };
        let two_axes = matches!(function.name, "lookup2" | "lookup2Or");
        let message = if two_axes && found.axes == 1 {
            Some(Message::with(
                message!("{0} has one axis; read it with lookup() or lookupOr()"),
                [Arg::from(quote(table))],
            ))
        } else if !two_axes && found.axes == 2 {
            Some(Message::with(
                message!("{0} has two axes; read it with lookup2() or lookup2Or()"),
                [Arg::from(quote(table))],
            ))
        } else if matches!(function.name, "lookupInterp" | "deficitWeight") && !found.banded {
            Some(Message::with(
                message!("{0} has no bands, so {1}() can't read it"),
                [Arg::from(quote(table)), Arg::from(function.name)],
            ))
        } else {
            None
        };
        if let Some(message) = message {
            self.report(
                first.span,
                Severity::Error,
                Code::RateTable,
                message,
                Some(table.clone()),
                Vec::new(),
            );
        }
    }

    fn type_method(&mut self, method: &Method) -> Type {
        let receiver = self.type_of(&method.receiver);
        for arg in &method.args {
            self.type_of(arg);
        }
        let name = method.method.text.as_str();
        let (methods, result, template) = match receiver {
            Type::Date => (
                catalog::DATE_METHODS,
                Type::Integer,
                message!(
                    "A date has no method {0}; it has Year(), Month(), Day(), Hour(), Minute(), Second(), Weekday(), YearDay() and Unix()"
                ),
            ),
            Type::Duration => (
                catalog::DURATION_METHODS,
                Type::Number,
                message!("A duration has no method {0}; it has Hours(), Minutes() and Seconds()"),
            ),
            Type::Any | Type::Nil => return Type::Any,
            other => {
                self.error(
                    method.method.span,
                    Code::Type,
                    Message::with(
                        message!("{0}; it has no method {1}"),
                        [
                            phrase(&self.describe(&method.receiver, other)),
                            Arg::from(quote(name)),
                        ],
                    ),
                );
                return Type::Any;
            }
        };
        if !methods.contains(&name) {
            let suggestion = nearest_known(name, methods.iter().copied());
            let fixes = suggestion
                .map(|suggestion| {
                    vec![replace(
                        Message::with(message!("Change to {0}"), [Arg::from(suggestion)]),
                        method.method.span,
                        suggestion.to_owned(),
                    )]
                })
                .unwrap_or_default();
            self.report(
                method.method.span,
                Severity::Error,
                Code::UnknownFunction,
                Message::with(template, [Arg::from(quote(name))]),
                Some(name.to_owned()),
                fixes,
            );
            return Type::Any;
        }
        if !method.args.is_empty() {
            self.report(
                method.parens,
                Severity::Error,
                Code::Arity,
                Message::with(
                    message!(
                        "{0} takes {1, plural, one {# argument} other {# arguments}}; {2, plural, one {# was} other {# were}} given"
                    ),
                    [
                        Arg::from(format!("{name}()")),
                        Arg::from(0usize),
                        Arg::from(method.args.len()),
                    ],
                ),
                Some(name.to_owned()),
                Vec::new(),
            );
        }
        result
    }

    /// `let` lines inside parentheses, an `if` block or a predicate: each name is scoped to the
    /// value after it, and may not redeclare a name in scope.
    fn type_sequence(&mut self, sequence: &Sequence) -> Type {
        let mut added = 0usize;
        for entry in &sequence.lets {
            let ty = self.type_of(&entry.value);
            let name = &entry.name.text;
            if name.is_empty() {
                continue;
            }
            if self.binding_index.contains_key(name) {
                self.report(
                    entry.name.span,
                    Severity::Error,
                    Code::LetDuplicate,
                    Message::with(
                        message!("{0} is already defined above; use another name"),
                        [Arg::from(quote(name))],
                    ),
                    Some(name.clone()),
                    Vec::new(),
                );
                continue;
            }
            self.check_let_name(name, entry.name.span);
            self.binding_index.insert(name.clone(), self.bindings.len());
            self.bindings.push(Binding {
                name: name.clone(),
                name_span: entry.name.span,
                span: entry.span,
                ty,
                reads: 0,
            });
            self.defined.insert(name.clone());
            added += 1;
        }
        let ty = self.type_of(&sequence.body);
        let start = self.bindings.len().saturating_sub(added);
        let scoped: Vec<Binding> = self.bindings.drain(start..).collect();
        for binding in scoped {
            self.binding_index.remove(&binding.name);
            self.defined.remove(&binding.name);
            if binding.reads == 0 {
                self.report_unused(binding.name_span, binding.name);
            }
        }
        ty
    }

    fn type_chain(&mut self, chain: &Chain) -> Type {
        let types: Vec<Type> = chain
            .operands
            .iter()
            .map(|operand| self.type_of(operand))
            .collect();
        for (index, (op, _)) in chain.operators.iter().enumerate() {
            if let (Some(left), Some(right)) =
                (chain.operands.get(index), chain.operands.get(index + 1))
            {
                self.check_comparison(*op, left, types[index], right, types[index + 1]);
            }
        }
        Type::Boolean
    }

    fn check_comparison(&mut self, op: CompareOp, left: &Expr, l: Type, right: &Expr, r: Type) {
        let comparable = |ty: Type| {
            ty.is_number()
                || matches!(ty, Type::Date | Type::Duration | Type::String)
                || ty.is_open()
        };
        if comparable(l) && comparable(r) && l.compatible(r) {
            return;
        }
        let (bad, bad_type) = if comparable(l) { (right, r) } else { (left, l) };
        let message = if bad_type == Type::Boolean {
            Message::with(
                message!("'{0}' compares numbers, but {1}. Use it as the test itself."),
                [Arg::from(op.text()), phrase(&self.describe(bad, bad_type))],
            )
        } else {
            Message::with(
                message!("'{0}' compares values of the same kind, but {1} and {2}"),
                [
                    Arg::from(op.text()),
                    phrase(&self.describe(left, l)),
                    phrase(&self.describe(right, r)),
                ],
            )
        };
        self.error(bad.span, Code::Type, message);
    }

    fn type_binary(&mut self, binary: &Binary) -> Type {
        let left = self.type_of(&binary.left);
        let right = self.type_of(&binary.right);
        match &binary.op {
            BinaryOp::Coalesce => left.join(right),
            BinaryOp::And { .. } | BinaryOp::Or { .. } => {
                self.check_logic(binary, left, right);
                Type::Boolean
            }
            BinaryOp::Equal | BinaryOp::NotEqual => {
                if !left.compatible(right) {
                    self.error(
                        binary.op_span,
                        Code::Type,
                        Message::with(
                            message!(
                                "{0} and {1}; the engine compares only values of the same kind"
                            ),
                            [
                                phrase(&self.describe(&binary.left, left)),
                                phrase(&self.describe(&binary.right, right)),
                            ],
                        ),
                    );
                }
                Type::Boolean
            }
            BinaryOp::Compare(op) => {
                self.check_comparison(*op, &binary.left, left, &binary.right, right);
                Type::Boolean
            }
            BinaryOp::In => {
                if !(matches!(right, Type::Array | Type::Object | Type::String) || right.is_open())
                {
                    self.error(
                        binary.right.span,
                        Code::Type,
                        Message::with(
                            message!("'in' looks inside a list, a record or text, but {0}"),
                            [phrase(&self.describe(&binary.right, right))],
                        ),
                    );
                }
                Type::Boolean
            }
            BinaryOp::Matches
            | BinaryOp::Contains
            | BinaryOp::StartsWith
            | BinaryOp::EndsWith
            | BinaryOp::Word(_) => {
                self.check_text(binary, left, right);
                Type::Boolean
            }
            BinaryOp::Range => {
                if !self.check_arithmetic(binary, left, right) {
                    for (operand, ty) in [(&*binary.left, left), (&*binary.right, right)] {
                        if ty == Type::Number {
                            self.not_whole("'..'", operand);
                        }
                    }
                }
                Type::Array
            }
            _ => self.type_arithmetic(binary, left, right),
        }
    }

    fn type_arithmetic(&mut self, binary: &Binary, left: Type, right: Type) -> Type {
        self.check_divisor(binary);
        let number = |l: Type, r: Type| {
            if l == Type::Integer && r == Type::Integer {
                Type::Integer
            } else {
                Type::Number
            }
        };
        match (&binary.op, left, right) {
            (BinaryOp::Add, Type::String, Type::String | Type::Any)
            | (BinaryOp::Add, Type::Any, Type::String) => Type::String,
            (BinaryOp::Add, Type::String, other) | (BinaryOp::Add, other, Type::String)
                if other.is_number() =>
            {
                self.error(
                    binary.op_span,
                    Code::Type,
                    Message::new(message!(
                        "'+' adds two numbers or joins two texts; here one side is text and the other a number. Convert with string() or float()."
                    )),
                );
                Type::Any
            }
            (BinaryOp::Add | BinaryOp::Subtract, Type::Date, Type::Duration)
            | (BinaryOp::Add, Type::Duration, Type::Date) => Type::Date,
            (BinaryOp::Subtract, Type::Date, Type::Date)
            | (BinaryOp::Add | BinaryOp::Subtract, Type::Duration, Type::Duration) => {
                Type::Duration
            }
            (BinaryOp::Add | BinaryOp::Subtract, Type::Date, other)
            | (BinaryOp::Add, other, Type::Date)
                if other.is_number() =>
            {
                self.moved_by_a_number(
                    binary,
                    message!(
                        "A date moves by a duration, not a number: write {0} {1} duration(\"24h\")"
                    ),
                    left,
                );
                Type::Date
            }
            (BinaryOp::Add, Type::Date, Type::Date) => {
                self.error(
                    binary.op_span,
                    Code::Type,
                    Message::new(message!(
                        "Two dates can't be added; subtract one from the other to get a duration"
                    )),
                );
                Type::Any
            }
            (BinaryOp::Add | BinaryOp::Subtract, Type::Duration, other)
            | (BinaryOp::Add, other, Type::Duration)
                if other.is_number() =>
            {
                self.moved_by_a_number(
                    binary,
                    message!(
                        "A duration adds to another duration, not a number: write {0} {1} duration(\"1h\")"
                    ),
                    left,
                );
                Type::Duration
            }
            (BinaryOp::Multiply | BinaryOp::Divide, Type::Duration, other)
            | (BinaryOp::Multiply, other, Type::Duration)
                if other.is_numeric() || other == Type::Nil =>
            {
                Type::Duration
            }
            (BinaryOp::Divide, Type::Duration, Type::Duration) => Type::Number,
            (BinaryOp::Modulo, l, r) => {
                if !self.check_arithmetic(binary, l, r) {
                    self.check_modulo(binary, l, r);
                }
                number(l, r)
            }
            (BinaryOp::Add | BinaryOp::Subtract | BinaryOp::Multiply, l, r) => {
                self.check_arithmetic(binary, l, r);
                number(l, r)
            }
            (_, l, r) => {
                self.check_arithmetic(binary, l, r);
                Type::Number
            }
        }
    }

    /// A date or a duration moved by a bare number: the number is the operand to fix.
    fn moved_by_a_number(&mut self, binary: &Binary, template: &'static str, left: Type) {
        let (number, other) = if left.is_number() {
            (&*binary.left, &*binary.right)
        } else {
            (&*binary.right, &*binary.left)
        };
        let fallback = if template.starts_with("A date") {
            Message::new(message!("the date"))
        } else {
            Message::new(message!("the duration"))
        };
        let subject = other.path().map_or_else(|| Arg::from(fallback), Arg::from);
        self.error(
            number.span,
            Code::Type,
            Message::with(template, [subject, Arg::from(binary.op.text())]),
        );
    }

    /// Reports each operand of an arithmetic operator that is not a number; whether any was.
    fn check_arithmetic(&mut self, binary: &Binary, left: Type, right: Type) -> bool {
        let mut wrong = false;
        for (operand, ty) in [(&*binary.left, left), (&*binary.right, right)] {
            if ty.is_numeric() || ty == Type::Nil {
                continue;
            }
            wrong = true;
            let name = operand.path();
            let mut fixes = Vec::new();
            let hint = match (ty, &name) {
                (Type::Boolean, Some(name)) => {
                    fixes.push(replace(
                        Message::with(
                            message!("Change to ({0} ? 1 : 0)"),
                            [Arg::from(name.as_str())],
                        ),
                        operand.span,
                        format!("({name} ? 1 : 0)"),
                    ));
                    Arg::from(Message::with(
                        message!(" Write ({0} ? amount : 0) to charge when it is true."),
                        [Arg::from(name.as_str())],
                    ))
                }
                (Type::Boolean, None) => Arg::from(Message::new(message!(
                    " Write (test ? amount : 0) to charge when a test holds."
                ))),
                (Type::String, _) => Arg::from(Message::new(message!(
                    " Compare text with ==, or look its rate up with lookup()."
                ))),
                _ => Arg::from(String::new()),
            };
            self.report(
                operand.span,
                Severity::Error,
                Code::Type,
                Message::with(
                    message!("'{0}' works on numbers, but {1}.{2}"),
                    [
                        Arg::from(binary.op.text()),
                        phrase(&self.describe(operand, ty)),
                        hint,
                    ],
                ),
                None,
                fixes,
            );
        }
        wrong
    }

    /// `%` takes whole numbers only; a decimal field, an input or a quotient is refused.
    fn check_modulo(&mut self, binary: &Binary, left: Type, right: Type) {
        for (operand, ty) in [(&*binary.left, left), (&*binary.right, right)] {
            if ty != Type::Number {
                continue;
            }
            let path = operand.path();
            let message = match &path {
                Some(path) => Message::with(
                    message!(
                        "'%' takes whole numbers, but {0} can hold a fraction. Use int({1}) to take its whole part"
                    ),
                    [Arg::from(quote(path)), Arg::from(path.as_str())],
                ),
                None => Message::new(message!(
                    "'%' takes whole numbers, but this part can come to a fraction"
                )),
            };
            let fixes = path
                .map(|path| {
                    vec![replace(
                        Message::with(message!("Change to int({0})"), [Arg::from(path.as_str())]),
                        operand.span,
                        format!("int({path})"),
                    )]
                })
                .unwrap_or_default();
            self.report(
                operand.span,
                Severity::Error,
                Code::Type,
                message,
                None,
                fixes,
            );
        }
    }

    /// Binds every top-level `let` before any is typed, so a read of one defined further down is
    /// reported as such.
    fn register_lets(&mut self, program: &syntax::Program) {
        for statement in &program.statements {
            let StatementKind::Let(name) = &statement.kind else {
                continue;
            };
            if name.text.is_empty() {
                continue;
            }
            if self.binding_index.contains_key(&name.text) {
                self.report(
                    name.span,
                    Severity::Error,
                    Code::LetDuplicate,
                    Message::with(
                        message!("{0} is already defined above; use another name"),
                        [Arg::from(quote(&name.text))],
                    ),
                    Some(name.text.clone()),
                    Vec::new(),
                );
                continue;
            }
            self.binding_index
                .insert(name.text.clone(), self.bindings.len());
            self.bindings.push(Binding {
                name: name.text.clone(),
                name_span: name.span,
                span: statement.span,
                ty: Type::Any,
                reads: 0,
            });
        }
    }

    fn type_index(&mut self, expr: &Expr, index: &Index) -> Type {
        let object = self.type_of(&index.object);
        self.type_of(&index.index);
        if object.is_scalar() && object != Type::String {
            self.not_indexable(expr, &index.object, object);
        }
        if object == Type::String {
            Type::String
        } else {
            Type::Any
        }
    }

    fn type_slice(&mut self, expr: &Expr, slice: &Slice) -> Type {
        let object = self.type_of(&slice.object);
        if let Some(start) = &slice.start {
            self.type_of(start);
        }
        if let Some(end) = &slice.end {
            self.type_of(end);
        }
        if object.is_scalar() && object != Type::String {
            self.not_indexable(expr, &slice.object, object);
        }
        match object {
            Type::String => Type::String,
            Type::Array => Type::Array,
            _ => Type::Any,
        }
    }

    fn type_unary(&mut self, unary: &Unary) -> Type {
        let operand = self.type_of(&unary.operand);
        if let UnaryOp::Not { .. } = unary.op {
            if !(operand == Type::Boolean || operand.is_open()) {
                self.error(
                    unary.operand.span,
                    Code::Type,
                    Message::with(
                        message!("'{0}' flips a true/false test, but {1}"),
                        [
                            Arg::from(unary.op.text()),
                            phrase(&self.describe(&unary.operand, operand)),
                        ],
                    ),
                );
            }
            return Type::Boolean;
        }
        match operand {
            Type::Integer => Type::Integer,
            Type::Duration => Type::Duration,
            other => {
                if !(other.is_numeric() || other == Type::Nil) {
                    self.error(
                        unary.operand.span,
                        Code::Type,
                        Message::with(
                            message!("'{0}' works on numbers, but {1}"),
                            [
                                Arg::from(unary.op.text()),
                                phrase(&self.describe(&unary.operand, other)),
                            ],
                        ),
                    );
                }
                Type::Number
            }
        }
    }

    fn check_logic(&mut self, binary: &Binary, left: Type, right: Type) {
        for (operand, ty) in [(&*binary.left, left), (&*binary.right, right)] {
            if ty == Type::Boolean || ty.is_open() {
                continue;
            }
            let message = match operand.path() {
                Some(name) if ty.is_number() => Message::with(
                    message!(
                        "'{0}' joins two true/false tests, but {1}. Compare it, such as {2} > 0"
                    ),
                    [
                        Arg::from(binary.op.text()),
                        phrase(&self.describe(operand, ty)),
                        Arg::from(name),
                    ],
                ),
                _ => Message::with(
                    message!("'{0}' joins two true/false tests, but {1}"),
                    [
                        Arg::from(binary.op.text()),
                        phrase(&self.describe(operand, ty)),
                    ],
                ),
            };
            self.error(operand.span, Code::Type, message);
        }
    }

    fn check_text(&mut self, binary: &Binary, left: Type, right: Type) {
        for (operand, ty) in [(&*binary.left, left), (&*binary.right, right)] {
            if ty == Type::String || ty.is_open() {
                continue;
            }
            self.error(
                operand.span,
                Code::Type,
                Message::with(
                    message!("'{0}' works on text, but {1}"),
                    [
                        Arg::from(binary.op.text()),
                        phrase(&self.describe(operand, ty)),
                    ],
                ),
            );
        }
    }

    fn check_divisor(&mut self, binary: &Binary) {
        if binary.op == BinaryOp::Divide
            && let ExprKind::Number(number) = &binary.right.kind
            && number.value.is_some_and(|value| value.is_zero())
        {
            self.report(
                binary.right.span,
                Severity::Warning,
                Code::DivideByZero,
                Message::new(message!("Dividing by 0 fails on every load")),
                None,
                Vec::new(),
            );
        }
    }

    fn mark_guarded(&mut self, expr: &Expr) {
        let mut current = expr;
        loop {
            self.guarded.insert(current.span);
            match &current.kind {
                ExprKind::Member(member) => current = &member.object,
                ExprKind::Predicate(predicate) if !predicate.braces => current = &predicate.body,
                _ => return,
            }
        }
    }

    /// Marks every read under a guard before the statement is typed, so a guard written after a
    /// read still counts.
    fn pre_mark_guards(&mut self, expr: &Expr) {
        match &expr.kind {
            ExprKind::Binary(binary)
                if matches!(
                    binary.op,
                    BinaryOp::Coalesce | BinaryOp::Equal | BinaryOp::NotEqual
                ) =>
            {
                match binary.op {
                    BinaryOp::Coalesce => self.mark_guarded(&binary.left),
                    BinaryOp::Equal | BinaryOp::NotEqual => {
                        if binary.right.kind == ExprKind::Nil {
                            self.mark_guarded(&binary.left);
                        }
                        if binary.left.kind == ExprKind::Nil {
                            self.mark_guarded(&binary.right);
                        }
                    }
                    _ => {}
                }
            }
            ExprKind::Call(call) if call.function.text == "coalesce" => {
                for arg in &call.args {
                    self.mark_guarded(arg);
                }
            }
            ExprKind::Binary(binary) if matches!(binary.op, BinaryOp::And { .. }) => {
                let names = nil_tests(&binary.left, false);
                self.mark_tested_reads(&binary.right, &names);
            }
            ExprKind::Binary(binary) if matches!(binary.op, BinaryOp::Or { .. }) => {
                let names = nil_tests(&binary.left, true);
                self.mark_tested_reads(&binary.right, &names);
            }
            ExprKind::Ternary(ternary) => {
                let names = nil_tests(&ternary.test, None);
                if !names.is_empty() {
                    if let Some(then) = &ternary.then {
                        self.mark_tested_reads(then, &names);
                    }
                    self.mark_tested_reads(&ternary.otherwise, &names);
                }
            }
            ExprKind::If(branch) => {
                let names = nil_tests(&branch.test, None);
                if !names.is_empty() {
                    self.mark_tested_reads(&branch.then, &names);
                    self.mark_tested_reads(&branch.otherwise, &names);
                }
            }
            _ => {}
        }
        expr.for_each_child(|child| self.pre_mark_guards(child));
    }

    /// Marks any read of a name the surrounding conditional tested for `nil` as guarded.
    fn mark_tested_reads(&mut self, expr: &Expr, names: &BTreeSet<String>) {
        if expr.path().is_some_and(|path| names.contains(&path)) {
            self.mark_guarded(expr);
        }
        if matches!(expr.kind, ExprKind::Member(_)) {
            return;
        }
        expr.for_each_child(|child| self.mark_tested_reads(child, names));
    }
}

/// The paths a test compares with `nil`. For a conditional (`empty` is `None`) any comparison
/// counts, through `and` and `or`, since each branch is taken knowing the answer. For the left
/// of `and` (`Some(false)`) the paths it requires to be set (`x != nil`, through `and`) guard its
/// right; for the left of `or` (`Some(true)`) the paths it lets through when empty (`x == nil`,
/// through `or`) do.
fn nil_tests(test: &Expr, empty: impl Into<Option<bool>>) -> BTreeSet<String> {
    fn collect(expr: &Expr, empty: Option<bool>, names: &mut BTreeSet<String>) {
        let ExprKind::Binary(binary) = &expr.kind else {
            return;
        };
        match (&binary.op, empty) {
            (BinaryOp::Equal, None | Some(true)) | (BinaryOp::NotEqual, None | Some(false)) => {
                let path = if binary.right.kind == ExprKind::Nil {
                    binary.left.path()
                } else if binary.left.kind == ExprKind::Nil {
                    binary.right.path()
                } else {
                    None
                };
                names.extend(path);
            }
            (BinaryOp::And { .. }, None | Some(false))
            | (BinaryOp::Or { .. }, None | Some(true)) => {
                collect(&binary.left, empty, names);
                collect(&binary.right, empty, names);
            }
            _ => {}
        }
    }
    let mut names = BTreeSet::new();
    collect(test, empty.into(), &mut names);
    names
}

/// "round(x, decimals?) takes 1 to 2 arguments; 3 were given".
fn arity_message(signature: &str, function: &Function, given: usize) -> Message {
    let min = usize::from(function.min_args);
    match function.max_args {
        None => Message::with(
            message!(
                "{0} takes at least {1, plural, one {# argument} other {# arguments}}; {2, plural, one {# was} other {# were}} given"
            ),
            [Arg::from(signature), Arg::from(min), Arg::from(given)],
        ),
        Some(max) if usize::from(max) == min => Message::with(
            message!(
                "{0} takes {1, plural, one {# argument} other {# arguments}}; {2, plural, one {# was} other {# were}} given"
            ),
            [Arg::from(signature), Arg::from(min), Arg::from(given)],
        ),
        Some(max) => Message::with(
            message!(
                "{0} takes {1} to {2} arguments; {3, plural, one {# was} other {# were}} given"
            ),
            [
                Arg::from(signature),
                Arg::from(min.to_string()),
                Arg::from(usize::from(max).to_string()),
                Arg::from(given),
            ],
        ),
    }
}
