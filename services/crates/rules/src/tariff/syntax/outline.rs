//! The outline and folding ranges the Studio shows for a file.
//!
//! The outline lists the header, inputs, constants, types, enums, functions, charges, total
//! clauses and tests in source order. A charge written inside an `if`, `for` or `match` is still
//! listed, with how many of them enclose it. Folding covers every block, record, record type,
//! list, enum body, argument list, doc comment, run of `//` comment lines and run of `use` lines
//! that spans more than one line.

use super::ast::ArmBody;
use super::ast::Block;
use super::ast::Expr;
use super::ast::ExprKind;
use super::ast::File;
use super::ast::FnBody;
use super::ast::Item;
use super::ast::ItemKind;
use super::ast::Stmt;
use super::ast::StmtKind;
use super::ast::Text;
use super::ast::TotalClause;
use super::ast::TypeExpr;
use super::ast::TypeExprKind;
use super::token::TokenKind;
use super::walk;
use crate::span::Span;

/// What an outline entry is.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum OutlineKind {
    /// The `tariff`, `library` or `condition` header.
    Header,
    /// An `input`.
    Input,
    /// A `const`.
    Const,
    /// A `type`.
    Type,
    /// An `enum`.
    Enum,
    /// A `fn`.
    Function,
    /// A `charge`.
    Charge,
    /// A `minimum`, `maximum` or `round` clause.
    Total,
    /// A `test`.
    Test,
    /// A `property`.
    Property,
    /// A `best` or `worst`.
    Choice,
    /// An `option` of a choice.
    Option,
}

impl OutlineKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Header => "header",
            Self::Input => "input",
            Self::Const => "const",
            Self::Type => "type",
            Self::Enum => "enum",
            Self::Function => "function",
            Self::Charge => "charge",
            Self::Total => "total",
            Self::Test => "test",
            Self::Property => "property",
            Self::Choice => "choice",
            Self::Option => "option",
        }
    }
}

/// One entry of the outline.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct OutlineItem {
    /// What it is.
    pub kind: OutlineKind,
    /// Its name: the declared name, a charge's label or a test's name as written (an
    /// interpolation shown as its source, `Detention, stop {stop.sequence}`), or the total
    /// clause's keyword.
    pub name: String,
    /// A charge's code, when it has one.
    pub code: Option<String>,
    /// How many `if`, `for` and `match` bodies enclose it (charges and total clauses).
    pub depth: u32,
    /// The whole entry, for revealing it.
    pub span: Span,
    /// The part to select when the entry is picked: its name or label.
    pub selection: Span,
}

/// What a folding range folds.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum FoldKind {
    /// A block of statements, the arms of a `match`, or an enum's variants.
    Block,
    /// A record literal or a record type.
    Record,
    /// A list or an argument list.
    List,
    /// A doc comment.
    Doc,
    /// Consecutive `//` comment lines.
    Comment,
    /// Consecutive `use` lines.
    Imports,
}

impl FoldKind {
    /// The kind's wire name.
    #[must_use]
    pub const fn as_str(self) -> &'static str {
        match self {
            Self::Block => "block",
            Self::Record => "record",
            Self::List => "list",
            Self::Doc => "doc",
            Self::Comment => "comment",
            Self::Imports => "imports",
        }
    }
}

/// A file's outline and folding ranges.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Outline {
    /// The entries, in source order.
    pub items: Vec<OutlineItem>,
    /// The folding ranges, by start.
    pub folding: Vec<(Span, FoldKind)>,
}

/// The outline of a parsed file.
#[must_use]
pub fn outline(source: &str, file: &File) -> Outline {
    let mut builder = Builder {
        source,
        outline: Outline::default(),
    };
    if let Some(header) = &file.header {
        builder.push(OutlineItem {
            kind: OutlineKind::Header,
            name: header.name.text.clone(),
            code: None,
            depth: 0,
            span: header.span,
            selection: header.name.span,
        });
    }
    for item in &file.items {
        builder.item(item);
    }
    builder.comments(file, TokenKind::DocComment, FoldKind::Doc);
    builder.comments(file, TokenKind::Comment, FoldKind::Comment);
    builder.imports(file);
    walk::exprs(file, &mut |expr| builder.fold_expr(expr));
    builder.outline.items.sort_by_key(|item| item.span.start);
    builder
        .outline
        .folding
        .sort_by_key(|(span, _)| (span.start, span.end));
    builder.outline.folding.dedup();
    builder.outline
}

struct Builder<'a> {
    source: &'a str,
    outline: Outline,
}

impl Builder<'_> {
    fn push(&mut self, item: OutlineItem) {
        self.outline.items.push(item);
    }

    fn fold(&mut self, span: Span, kind: FoldKind) {
        let text = self
            .source
            .get(span.start as usize..span.end as usize)
            .unwrap_or_default();
        if text.contains('\n') {
            self.outline.folding.push((span, kind));
        }
    }

    fn declared(&mut self, kind: OutlineKind, name: &super::ast::Name, item: &Item) {
        self.push(OutlineItem {
            kind,
            name: name.text.clone(),
            code: None,
            depth: 0,
            span: item.span,
            selection: name.span,
        });
    }

    fn item(&mut self, item: &Item) {
        match &item.kind {
            ItemKind::Use(_) => {}
            ItemKind::Input(input) => {
                self.declared(OutlineKind::Input, &input.name, item);
                self.fold_type(&input.ty);
            }
            ItemKind::Const(constant) => {
                self.declared(OutlineKind::Const, &constant.name, item);
                if let Some(ty) = &constant.ty {
                    self.fold_type(ty);
                }
            }
            ItemKind::Type(decl) => {
                self.declared(OutlineKind::Type, &decl.name, item);
                self.fold(item.span, FoldKind::Record);
                for field in &decl.fields {
                    self.fold_type(&field.ty);
                }
            }
            ItemKind::Enum(decl) => {
                self.declared(OutlineKind::Enum, &decl.name, item);
                self.fold(item.span, FoldKind::Block);
            }
            ItemKind::Fn(function) => {
                self.declared(OutlineKind::Function, &function.name, item);
                for param in &function.params {
                    self.fold_type(&param.ty);
                }
                if let Some(ret) = &function.ret {
                    self.fold_type(ret);
                }
                if let FnBody::Block(block) = &function.body {
                    self.block(block, 0);
                }
            }
            ItemKind::Test(test) => {
                self.push(OutlineItem {
                    kind: OutlineKind::Test,
                    name: test.name.text.clone(),
                    code: None,
                    depth: 0,
                    span: item.span,
                    selection: test.name.span,
                });
                self.block(&test.body, 0);
            }
            ItemKind::Property(property) => {
                self.push(OutlineItem {
                    kind: OutlineKind::Property,
                    name: property.name.text.clone(),
                    code: None,
                    depth: 0,
                    span: item.span,
                    selection: property.name.span,
                });
                self.block(&property.body, 0);
            }
            ItemKind::Stmt(stmt) => self.stmt(stmt, 0),
        }
    }

    fn block(&mut self, block: &Block, depth: u32) {
        self.fold(block.span, FoldKind::Block);
        for stmt in &block.stmts {
            self.stmt(stmt, depth);
        }
    }

    fn label(&self, text: &Text) -> (String, Span) {
        let inner = Span::new(
            text.span.start.saturating_add(1).min(text.span.end),
            text.span.end.saturating_sub(1).max(text.span.start),
        );
        let raw = self
            .source
            .get(inner.start as usize..inner.end as usize)
            .unwrap_or_default();
        let name = text.literal().unwrap_or_else(|| raw.to_owned());
        (name, text.span)
    }

    fn stmt(&mut self, stmt: &Stmt, depth: u32) {
        match &stmt.kind {
            StmtKind::Charge(charge) => {
                let (name, selection) = self.label(&charge.label);
                self.push(OutlineItem {
                    kind: OutlineKind::Charge,
                    name,
                    code: charge.code.as_ref().map(|code| code.text.clone()),
                    depth,
                    span: stmt.span,
                    selection,
                });
            }
            StmtKind::Total(clause) => {
                let name = match clause {
                    TotalClause::Minimum(_) => "minimum",
                    TotalClause::Maximum(_) => "maximum",
                    TotalClause::Round { .. } => "round",
                };
                let keyword = Span::new(
                    stmt.span.start,
                    stmt.span.start + u32::try_from(name.len()).unwrap_or(0),
                );
                self.push(OutlineItem {
                    kind: OutlineKind::Total,
                    name: name.to_owned(),
                    code: None,
                    depth,
                    span: stmt.span,
                    selection: keyword,
                });
            }
            StmtKind::If(statement) => {
                for (_, block) in &statement.branches {
                    self.block(block, depth + 1);
                }
                if let Some(block) = &statement.otherwise {
                    self.block(block, depth + 1);
                }
            }
            StmtKind::For(statement) => self.block(&statement.body, depth + 1),
            StmtKind::Choose(choose) => {
                self.push(OutlineItem {
                    kind: OutlineKind::Choice,
                    name: choose.label.text.clone(),
                    code: None,
                    depth,
                    span: choose.span,
                    selection: choose.label.span,
                });
                self.fold(choose.span, FoldKind::Block);
                for option in &choose.options {
                    self.push(OutlineItem {
                        kind: OutlineKind::Option,
                        name: option.label.text.clone(),
                        code: None,
                        depth: depth + 1,
                        span: option.span,
                        selection: option.label.span,
                    });
                    self.block(&option.body, depth + 2);
                }
            }
            StmtKind::Match(statement) => {
                self.fold(statement.span, FoldKind::Block);
                for arm in &statement.arms {
                    if let ArmBody::Block(block) = &arm.body {
                        self.block(block, depth + 1);
                    }
                }
            }
            StmtKind::Let(binding) | StmtKind::Var(binding) => {
                if let Some(ty) = &binding.ty {
                    self.fold_type(ty);
                }
            }
            StmtKind::Item(item) => self.item(item),
            _ => {}
        }
    }

    fn fold_type(&mut self, ty: &TypeExpr) {
        match &ty.kind {
            TypeExprKind::Record(fields) => {
                self.fold(ty.span, FoldKind::Record);
                for field in fields {
                    self.fold_type(&field.ty);
                }
            }
            TypeExprKind::Named { args, .. } => {
                for arg in args {
                    self.fold_type(arg);
                }
            }
            TypeExprKind::Optional(inner) => self.fold_type(inner),
            TypeExprKind::Quotient { .. } | TypeExprKind::Missing => {}
        }
    }

    fn fold_expr(&mut self, expr: &Expr) {
        match &expr.kind {
            ExprKind::Record(_) => self.fold(expr.span, FoldKind::Record),
            ExprKind::List(_) => self.fold(expr.span, FoldKind::List),
            ExprKind::Match { .. } => self.fold(expr.span, FoldKind::Block),
            ExprKind::Call { callee, args } if !args.is_empty() => {
                let start = callee.span.end;
                self.fold(Span::new(start, expr.span.end), FoldKind::List);
            }
            _ => {}
        }
    }

    /// Doc comments of more than one line.
    /// Folds each run of consecutive comment lines of one kind.
    fn comments(&mut self, file: &File, comment: TokenKind, kind: FoldKind) {
        let mut run: Option<Span> = None;
        let mut previous_was_newline = false;
        for token in &file.tokens {
            match token.kind {
                found if found == comment => {
                    run = Some(match run {
                        Some(span) if previous_was_newline => span.cover(token.span),
                        Some(span) => {
                            self.fold(span, kind);
                            token.span
                        }
                        None => token.span,
                    });
                    previous_was_newline = false;
                }
                TokenKind::Newline if !previous_was_newline => previous_was_newline = true,
                _ => {
                    if let Some(span) = run.take() {
                        self.fold(span, kind);
                    }
                    previous_was_newline = false;
                }
            }
        }
        if let Some(span) = run {
            self.fold(span, kind);
        }
    }

    /// Folds each run of consecutive `use` items.
    fn imports(&mut self, file: &File) {
        let mut run: Option<Span> = None;
        for item in &file.items {
            if matches!(item.kind, ItemKind::Use(_)) {
                run = Some(run.map_or(item.span, |span| span.cover(item.span)));
            } else if let Some(span) = run.take() {
                self.fold(span, FoldKind::Imports);
            }
        }
        if let Some(span) = run {
            self.fold(span, FoldKind::Imports);
        }
    }
}
