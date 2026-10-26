//! The header and the file-level declarations: `use`, `input`, `const`, `type`, `enum`, `fn`
//! and `test`.

use super::Parser;
use crate::span::Span;
use crate::tariff::diagnostic::Change;
use crate::tariff::diagnostic::Diagnostic;
use crate::tariff::diagnostic::Fix;
use crate::tariff::syntax::ast::Const;
use crate::tariff::syntax::ast::EnumDecl;
use crate::tariff::syntax::ast::File;
use crate::tariff::syntax::ast::FileKind;
use crate::tariff::syntax::ast::FnBody;
use crate::tariff::syntax::ast::FnDecl;
use crate::tariff::syntax::ast::Header;
use crate::tariff::syntax::ast::Input;
use crate::tariff::syntax::ast::Item;
use crate::tariff::syntax::ast::ItemKind;
use crate::tariff::syntax::ast::Name;
use crate::tariff::syntax::ast::Param;
use crate::tariff::syntax::ast::Property;
use crate::tariff::syntax::ast::Test;
use crate::tariff::syntax::ast::TypeDecl;
use crate::tariff::syntax::ast::TypeExprKind;
use crate::tariff::syntax::ast::Use;
use crate::tariff::syntax::ast::Variant;
use crate::tariff::syntax::token::TokenKind;
use trenova_module_kit::message;
use trenova_module_kit::message::Arg;
use trenova_module_kit::message::Message;

use super::statements::BlockKind;

const HEADER_INSERT: &str = "tariff \"Untitled\"\n\n";

fn header_kind(word: &str) -> Option<FileKind> {
    match word {
        "tariff" => Some(FileKind::Tariff),
        "library" => Some(FileKind::Library),
        "condition" => Some(FileKind::Condition),
        _ => None,
    }
}

impl Parser<'_> {
    pub(super) fn file(&mut self) -> File {
        let mut header = None;
        let mut docs = None;
        let first_is_header =
            self.kind() == TokenKind::Keyword && header_kind(self.text()).is_some();
        if first_is_header {
            let parsed = self.header();
            self.end_statement();
            docs = self.take_file_doc(parsed.span.end);
            header = Some(parsed);
        } else if !self.has_header_later() {
            let span = Span::new(0, 0);
            self.report(
                Diagnostic::error(
                    "header",
                    self.here_or_start(),
                    Message::new(message!(
                        "A file starts with its header: `tariff \"Name\"`, `library \"name\"` or `condition \"Name\"`"
                    )),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Add a `tariff` header")),
                    changes: vec![Change {
                        span,
                        insert: String::from(HEADER_INSERT),
                    }],
                }),
            );
        }
        let mut items = Vec::new();
        while !self.at_eof() {
            if self.at_header_keyword() {
                self.misplaced_header(&mut header);
                continue;
            }
            let before = self.pos;
            items.push(self.item(BlockKind::File));
            self.end_statement();
            if self.pos == before {
                self.bump();
            }
        }
        if docs.is_none()
            && let Some(parsed) = &header
            && !first_is_header
        {
            docs = self.take_file_doc(parsed.span.end);
        }
        self.report_stray_docs();
        let mut diagnostics = std::mem::take(&mut self.diagnostics);
        diagnostics.sort_by_key(|diagnostic| (diagnostic.span.start, diagnostic.span.end));
        File {
            header,
            docs,
            items,
            diagnostics,
            tokens: std::mem::take(&mut self.tokens),
            node_count: 0,
        }
    }

    fn here_or_start(&self) -> Span {
        let token = self.peek();
        if token.kind == TokenKind::Eof {
            Span::new(0, 0)
        } else {
            token.span
        }
    }

    fn at_header_keyword(&self) -> bool {
        self.kind() == TokenKind::Keyword && header_kind(self.text()).is_some()
    }

    fn has_header_later(&self) -> bool {
        self.sig.iter().any(|index| {
            self.tokens.get(*index).is_some_and(|token| {
                token.kind == TokenKind::Keyword && header_kind(self.slice(token.span)).is_some()
            })
        })
    }

    fn header(&mut self) -> Header {
        let keyword = self.bump();
        let kind = header_kind(self.slice(keyword.span)).unwrap_or(FileKind::Tariff);
        let name = self.literal_name(Message::with(
            message!("Expected the {0}'s name in quotes after `{0}`"),
            [Arg::from(kind.keyword())],
        ));
        Header {
            kind,
            span: keyword.span.cover(name.span),
            name,
        }
    }

    /// A header written after the first line: it is reported with a fix moving it to the top,
    /// and kept when it is the only one.
    fn misplaced_header(&mut self, header: &mut Option<Header>) {
        let line_start = self.line_start(self.peek().span.start);
        let parsed = self.header();
        let line_end = self.line_end(parsed.span.end);
        let text = self
            .slice(Span::new(line_start, line_end))
            .trim()
            .to_owned();
        if header.is_none() {
            self.report(
                Diagnostic::error(
                    "header",
                    parsed.span,
                    Message::new(message!("The header must be the file's first line")),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Move the header to the top")),
                    changes: vec![
                        Change {
                            span: Span::new(0, 0),
                            insert: format!("{text}\n\n"),
                        },
                        Change {
                            span: Span::new(line_start, line_end),
                            insert: String::new(),
                        },
                    ],
                }),
            );
            *header = Some(parsed);
        } else {
            self.report(
                Diagnostic::error(
                    "header",
                    parsed.span,
                    Message::new(message!("A file has only one header")),
                )
                .with_fix(Fix {
                    label: Message::new(message!("Remove this header")),
                    changes: vec![Change {
                        span: Span::new(line_start, line_end),
                        insert: String::new(),
                    }],
                }),
            );
        }
        self.end_statement();
    }

    fn line_start(&self, at: u32) -> u32 {
        let before = self.source.get(..at as usize).unwrap_or_default();
        before
            .rfind('\n')
            .map_or(0, |index| u32::try_from(index + 1).unwrap_or(u32::MAX))
    }

    /// The offset just past the newline that ends the line containing `at`.
    fn line_end(&self, at: u32) -> u32 {
        let after = self.source.get(at as usize..).unwrap_or_default();
        let length = after.find('\n').map_or(after.len(), |index| index + 1);
        at + u32::try_from(length).unwrap_or(0)
    }

    /// A quoted name with no interpolation (a header's or a test's). Its span covers the quotes.
    pub(super) fn literal_name(&mut self, message: Message) -> Name {
        if self.kind() != TokenKind::Text || self.newline_before() {
            self.expected(message);
            return self.missing_name();
        }
        let text = self.text_literal();
        if text
            .parts
            .iter()
            .any(|part| matches!(part, crate::tariff::syntax::ast::TextPart::Expr(_)))
        {
            self.error(
                text.span,
                Message::new(message!("A name in quotes can't interpolate values")),
            );
        }
        let decoded = text
            .parts
            .iter()
            .filter_map(|part| match part {
                crate::tariff::syntax::ast::TextPart::Literal(literal, _) => Some(literal.as_str()),
                crate::tariff::syntax::ast::TextPart::Expr(_) => None,
            })
            .collect::<String>();
        Name {
            text: decoded,
            span: text.span,
        }
    }

    /// One file-level item (or, in a block, a misplaced one), with its doc comment.
    pub(super) fn item(&mut self, block: BlockKind) -> Item {
        let docs = self.take_doc();
        let start = self.peek().span;
        let private = self.eat_word("private");
        if let Some(span) = private
            && !(self.at_word("const")
                || self.at_word("type")
                || self.at_word("enum")
                || self.at_word("fn"))
        {
            self.error(
                span,
                Message::new(message!(
                    "Only a `fn`, `const`, `type` or `enum` can be `private`"
                )),
            );
        }
        let outer = std::mem::replace(&mut self.statement_failed, false);
        let kind = if let Some(kind) = self.declaration() {
            kind
        } else {
            ItemKind::Stmt(self.statement(block))
        };
        self.skip_rest_if_failed(outer);
        let end = self.previous_span().end.max(start.start);
        Item {
            docs,
            private,
            span: Span::new(start.start, end),
            kind,
        }
    }

    /// A declaration, when the current token starts one.
    fn declaration(&mut self) -> Option<ItemKind> {
        if self.at_labelled("property") {
            return Some(ItemKind::Property(self.property_item()));
        }
        if self.kind() != TokenKind::Keyword {
            return None;
        }
        Some(match self.text() {
            "use" => ItemKind::Use(self.use_item()),
            "input" => ItemKind::Input(self.input_item()),
            "const" => ItemKind::Const(self.const_item()),
            "type" => ItemKind::Type(self.type_item()),
            "enum" => ItemKind::Enum(self.enum_item()),
            "fn" => ItemKind::Fn(self.fn_item()),
            "test" => ItemKind::Test(self.test_item()),
            _ => return None,
        })
    }

    fn use_item(&mut self) -> Use {
        self.bump();
        let mut path = Vec::new();
        let mut names = None;
        loop {
            if let Some(name) = self.same_line_name() {
                path.push(name);
            } else {
                self.expected(Message::new(message!(
                    "Expected a module path after `use`, like `std.fuel`"
                )));
                path.push(self.missing_name());
                break;
            }
            if !self.at(".") || self.newline_before() {
                break;
            }
            self.bump();
            if self.at("{") {
                names = Some(self.use_names());
                break;
            }
        }
        let alias = if self.at_word("as") && !self.newline_before() {
            self.eat_word("as");
            Some(self.value_name(Message::new(message!(
                "Expected the name to call the module after `as`"
            ))))
        } else {
            None
        };
        Use { path, names, alias }
    }

    fn use_names(&mut self) -> Vec<Name> {
        let open = self.bump().span;
        let mut names = Vec::new();
        while !self.at("}") && !self.at_eof() && !self.at_statement_line() {
            if let Some(name) = self.any_name() {
                names.push(name);
            } else {
                self.expected(Message::new(message!(
                    "Expected a name to import, or a closing brace"
                )));
                self.recover_in_list("}");
                continue;
            }
            if self.eat(",").is_none() && !self.at_list_end("}") {
                self.expected(Message::new(message!(
                    "Expected `,` between the names, or a closing brace"
                )));
                self.recover_in_list("}");
            }
        }
        if self.eat("}").is_none() {
            self.error(
                open,
                Message::new(message!("This list of names is missing its closing brace")),
            );
        }
        names
    }

    /// Skips to the next `,` (consumed) or `close` (not consumed) at the current nesting. A
    /// closing bracket that is not `close` ends the skip too, and is consumed when it is the
    /// first token, so the list always moves on.
    pub(super) fn recover_in_list(&mut self, close: &str) {
        let mut depth = 0usize;
        let start = self.pos;
        while !self.at_eof() && !self.at_statement_line() {
            let token = self.peek();
            if token.kind == TokenKind::Punct {
                match self.slice(token.span) {
                    "(" | "[" | "{" => depth += 1,
                    text if depth == 0 && text == close => return,
                    "," if depth == 0 => {
                        self.bump();
                        return;
                    }
                    ")" | "]" | "}" if depth == 0 => {
                        if self.pos == start {
                            self.bump();
                        }
                        return;
                    }
                    ")" | "]" | "}" => depth -= 1,
                    _ => {}
                }
            }
            self.bump();
        }
    }

    fn input_item(&mut self) -> Input {
        self.bump();
        let name = self.value_name(Message::new(message!(
            "Expected the input's name after `input`"
        )));
        let ty = if self.eat(":").is_some() {
            self.type_expr()
        } else {
            self.expected(Message::with(
                message!("Expected `:` and the input's type after `{0}`"),
                [Arg::from(name.text.clone())],
            ));
            self.missing_type()
        };
        let default = self.eat("=").map(|_| self.expr());
        let constraint = if self.at_word("where") && !self.newline_before() {
            self.bump();
            Some(self.expr())
        } else {
            None
        };
        Input {
            name,
            ty,
            default,
            constraint,
        }
    }

    fn const_item(&mut self) -> Const {
        self.bump();
        let capitals = !self.text().chars().any(|c| c.is_ascii_lowercase());
        let name = match self.kind() {
            TokenKind::Code | TokenKind::Name if !self.newline_before() => {
                let token = self.bump();
                self.name_of(&token)
            }
            TokenKind::TypeName if capitals && !self.newline_before() => {
                self.mark(TokenKind::Code);
                let token = self.bump();
                self.name_of(&token)
            }
            _ => self.value_name(Message::new(message!(
                "Expected the constant's name after `const`"
            ))),
        };
        let ty = self.eat(":").map(|_| self.type_expr());
        let value = if self.eat("=").is_some() {
            self.expr()
        } else {
            self.expected(Message::with(
                message!("Expected `=` and the value of `{0}`"),
                [Arg::from(name.text.clone())],
            ));
            self.missing()
        };
        Const { name, ty, value }
    }

    /// A `PascalCase` name for a declared type or enum, on the same line.
    fn type_name(&mut self, what: Message) -> Name {
        if self.newline_before() {
            self.expected(what);
            return self.missing_name();
        }
        self.pascal_name(what)
    }

    /// A `PascalCase` name.
    fn pascal_name(&mut self, what: Message) -> Name {
        match self.kind() {
            TokenKind::TypeName | TokenKind::Code => {
                let token = self.bump();
                self.name_of(&token)
            }
            TokenKind::Name => {
                let token = self.bump();
                let name = self.name_of(&token);
                let fixed = upper_first(&name.text);
                self.report(
                    Diagnostic::error(
                        "syntax",
                        token.span,
                        Message::with(
                            message!("A type's name starts with a capital letter: `{0}`"),
                            [Arg::from(fixed.clone())],
                        ),
                    )
                    .with_fix(super::rename_fix(token.span, fixed)),
                );
                self.mark_at(self.pos - 1, TokenKind::TypeName);
                name
            }
            _ => {
                self.expected(what);
                self.missing_name()
            }
        }
    }

    fn type_item(&mut self) -> TypeDecl {
        self.bump();
        let name = self.type_name(Message::new(message!(
            "Expected the type's name after `type`"
        )));
        if self.eat("=").is_none() {
            self.expected(Message::new(message!(
                "Expected `=` and the type's fields in braces, like `type Break = '{' from: Weight '}'`"
            )));
            if !self.at("{") {
                return TypeDecl {
                    name,
                    fields: Vec::new(),
                };
            }
        }
        let ty = self.type_expr();
        let fields = match ty.kind {
            TypeExprKind::Record(fields) => fields,
            TypeExprKind::Missing => Vec::new(),
            _ => {
                self.error(
                    ty.span,
                    Message::new(message!(
                        "A type declaration lists its fields in braces, like `type Break = '{' from: Weight '}'`"
                    )),
                );
                Vec::new()
            }
        };
        TypeDecl { name, fields }
    }

    fn enum_item(&mut self) -> EnumDecl {
        self.bump();
        let name = self.type_name(Message::new(message!(
            "Expected the enum's name after `enum`"
        )));
        let mut variants = Vec::new();
        let Some(open) = self.eat("{") else {
            self.expected(Message::new(message!(
                "Expected the enum's variants in braces, like `enum Zone '{' A, B '}'`"
            )));
            return EnumDecl { name, variants };
        };
        while !self.at("}") && !self.at_eof() && !self.at_statement_line() {
            let docs = self.take_doc();
            if !matches!(
                self.kind(),
                TokenKind::TypeName | TokenKind::Code | TokenKind::Name
            ) {
                self.expected(Message::new(message!(
                    "Expected a variant's name, or a closing brace"
                )));
                self.recover_in_list("}");
                continue;
            }
            let variant = self.pascal_name(Message::new(message!("Expected a variant's name")));
            variants.push(Variant {
                docs,
                name: variant,
            });
            if self.eat(",").is_none() && !self.at_list_end("}") {
                let at = self.previous_span().end;
                self.report(
                    Diagnostic::error(
                        "syntax",
                        self.here(),
                        Message::new(message!("Expected `,` between the variants")),
                    )
                    .with_fix(Fix {
                        label: Message::new(message!("Add a comma")),
                        changes: vec![Change {
                            span: Span::new(at, at),
                            insert: String::from(","),
                        }],
                    }),
                );
                if !matches!(self.kind(), TokenKind::TypeName | TokenKind::Code) {
                    self.recover_in_list("}");
                }
            }
        }
        if self.eat("}").is_none() {
            self.error(
                open,
                Message::new(message!("This enum is missing its closing brace")),
            );
        }
        EnumDecl { name, variants }
    }

    fn fn_item(&mut self) -> FnDecl {
        self.bump();
        let name = self.value_name(Message::new(message!(
            "Expected the function's name after `fn`"
        )));
        let params = if self.at("(") {
            self.params()
        } else {
            self.expected(Message::with(
                message!("Expected `(` and the parameters of `{0}`"),
                [Arg::from(name.text.clone())],
            ));
            Vec::new()
        };
        let ret = self.eat("->").map(|_| self.type_expr());
        let body = if self.at("{") {
            FnBody::Block(self.block(BlockKind::Function))
        } else if self.eat("=").is_some() {
            FnBody::Expr(self.expr())
        } else {
            self.expected(Message::new(message!(
                "Expected the function's body: a block in braces, or `=` and a value"
            )));
            FnBody::Expr(self.missing())
        };
        FnDecl {
            name,
            params,
            ret,
            body,
        }
    }

    fn params(&mut self) -> Vec<Param> {
        let open = self.bump().span;
        let mut params = Vec::new();
        while !self.at(")") && !self.at_eof() && !self.at_statement_line() {
            let start = self.peek().span.start;
            let name = if self.kind() == TokenKind::Name {
                let token = self.bump();
                self.name_of(&token)
            } else {
                self.value_name(Message::new(message!(
                    "Expected a parameter's name, or `)`"
                )))
            };
            if name.text.is_empty() {
                self.recover_in_list(")");
                continue;
            }
            let ty = if self.eat(":").is_some() {
                self.type_expr()
            } else {
                self.expected(Message::with(
                    message!("Expected `:` and the type of `{0}`"),
                    [Arg::from(name.text.clone())],
                ));
                self.missing_type()
            };
            let default = self.eat("=").map(|_| self.expr());
            let end = self.previous_span().end;
            params.push(Param {
                name,
                ty,
                default,
                span: Span::new(start, end),
            });
            if self.eat(",").is_none() && !self.at_list_end(")") {
                self.expected(Message::new(message!(
                    "Expected `,` between the parameters, or `)`"
                )));
                self.recover_in_list(")");
            }
        }
        if self.eat(")").is_none() {
            self.error(
                open,
                Message::new(message!("These parameters are missing their closing `)`")),
            );
        }
        params
    }

    fn test_item(&mut self) -> Test {
        self.bump();
        let name = self.literal_name(Message::new(message!(
            "Expected the test's name in quotes after `test`"
        )));
        let body = if self.at("{") {
            self.block(BlockKind::Test)
        } else {
            self.expected(Message::new(message!("Expected the test's body in braces")));
            let at = self.previous_span().end;
            crate::tariff::syntax::ast::Block {
                stmts: Vec::new(),
                span: Span::new(at, at),
            }
        };
        Test { name, body }
    }

    fn property_item(&mut self) -> Property {
        self.mark(TokenKind::Keyword);
        self.bump();
        let name = self.literal_name(Message::new(message!(
            "Expected the property's name in quotes after `property`"
        )));
        let body = if self.at("{") {
            self.block(BlockKind::Property)
        } else {
            self.expected(Message::new(message!(
                "Expected the property's body in braces"
            )));
            let at = self.previous_span().end;
            crate::tariff::syntax::ast::Block {
                stmts: Vec::new(),
                span: Span::new(at, at),
            }
        };
        Property { name, body }
    }
}

/// `word` with its first letter in upper case.
fn upper_first(word: &str) -> String {
    let mut characters = word.chars();
    characters.next().map_or_else(String::new, |first| {
        first.to_ascii_uppercase().to_string() + characters.as_str()
    })
}
