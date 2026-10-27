//! `tariff.references`: every place the name under the cursor is declared, read or written.
//!
//! The name resolves the way rename resolves it ([`super::occurrences`]): a checker symbol
//! (bindings, `var`s, inputs, functions, constants, types, enums, parameters, lambda and loop
//! variables, `if let`, `require let` and pattern bindings, module aliases and imported names),
//! a local the checker did not reach (a property's `vary` name), a field of a declared record
//! type or an enum variant. Unlike rename it refuses nothing it can name: a name a library
//! declares has its declaration in that library, with the library's `module`.
//!
//! Nothing is answered for a position on no name, a keyword or contextual keyword, a unit, a
//! schema root or a field of the record a tariff rates: they have no declaration to list.

use super::cursor::{Scopes, is_word, token_at};
use super::occurrences::{
    FieldOwner, Occurrence, Subject, Use, VariantOwner, field_at, input_target_at, local_at,
    local_subject, occurrences,
};
use super::session::Session;
use crate::tariff::check::SymbolKind;
use crate::tariff::syntax::token::TokenKind;

/// Every occurrence of the name at byte `offset` of the root file, its declaration left out
/// unless `declaration`.
#[must_use]
pub fn references(session: &Session<'_>, offset: u32, declaration: bool) -> Vec<Occurrence> {
    let file = session.file();
    let Some(token) = token_at(&file.tokens, offset).and_then(|index| file.tokens.get(index))
    else {
        return Vec::new();
    };
    if !is_word(token.kind) || matches!(token.kind, TokenKind::Unit | TokenKind::Keyword) {
        return Vec::new();
    }
    let span = token.span;
    let scopes = Scopes::of(session.source(), file);
    session.tick(scopes.exprs.len());
    let Some(subject) = subject(session, &scopes, offset, span) else {
        return Vec::new();
    };
    occurrences(session, &scopes, &subject)
        .into_iter()
        .filter(|occurrence| declaration || occurrence.kind != Use::Declaration)
        .collect()
}

/// What the name at `span` names, when it has occurrences to list.
fn subject(
    session: &Session<'_>,
    scopes: &Scopes<'_>,
    offset: u32,
    span: trenova_module_kit::span::Span,
) -> Option<Subject> {
    if let Some((_, id)) = session.symbol_at(offset) {
        let symbol = session.symbol(id)?;
        return (!matches!(symbol.kind, SymbolKind::Root | SymbolKind::Test))
            .then_some(Subject::Symbol(id));
    }
    if let Some(id) = input_target_at(session, span) {
        return Some(Subject::Symbol(id));
    }
    if let Some(local) = local_at(session, scopes, span) {
        return Some(local_subject(session, local));
    }
    if let Some(owner) = field_at(session, scopes, span) {
        return match owner {
            FieldOwner::Root(record, field) | FieldOwner::Library(record, field) => {
                Some(Subject::Field(record, field))
            }
            FieldOwner::Schema(_) => None,
        };
    }
    match super::occurrences::variant_at(session, scopes, span)? {
        VariantOwner::Root(enumeration, variant)
        | VariantOwner::Library(enumeration, variant)
        | VariantOwner::Schema(enumeration, variant) => {
            Some(Subject::Variant(enumeration, variant))
        }
    }
}
