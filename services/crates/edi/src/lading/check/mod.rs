//! Checks a whole project and builds the program a run executes.
//!
//! The checker reads every file, compiles the guides, types every map, rule, flow, function and
//! test against the guides and Trenova's documents, and reports what is wrong with fixes. What
//! it learns about names and types is kept as [`Facts`] for the editor; what a run needs (the
//! files, the guides, the partner, which map is which) is the [`Program`].

mod calls;
mod decls;
mod env;
mod expr;
pub mod flow;
mod guide;
mod inbound;
mod outbound;
mod rules;
pub mod tests;
mod types;
pub mod x12;

use std::collections::{BTreeMap, BTreeSet};

use trenova_module_kit::diagnostic::Diagnostic;
use trenova_module_kit::message;
use trenova_module_kit::message::{Arg, Message};
use trenova_module_kit::span::Span;

use super::facts::{Facts, Occurrence, Target};
use super::project::{Context, Project, Sample};
use super::schema;
use super::syntax::ast::{CsvColumn, Endpoint, Header, Item, SettingDecl, TypeExpr};
use super::syntax::parser::{Parsed, parse};
use super::types::Type;
use crate::x12::guide::compiled::CompiledGuide;

pub use env::Env;

/// A file-level declaration.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Decl {
    /// `fn`, by item index.
    Fn(usize),
    /// `crosswalk`, by item index.
    Crosswalk(usize),
    /// `const`, by item index.
    Const(usize),
}

/// One file of the project.
#[derive(Debug, Clone)]
pub struct Unit {
    /// Its path.
    pub path: String,
    /// Its text.
    pub source: String,
    /// Its tree.
    pub parsed: Parsed,
    /// What is wrong with it.
    pub diagnostics: Vec<Diagnostic>,
    /// Its own declarations by name.
    pub decls: BTreeMap<String, Decl>,
}

impl Unit {
    /// The header.
    #[must_use]
    pub fn header(&self) -> &Header {
        &self.parsed.file.header
    }

    /// The items.
    #[must_use]
    pub fn items(&self) -> &[Item] {
        &self.parsed.file.items
    }
}

/// A guide the program reads or writes with.
#[derive(Debug, Clone)]
pub struct GuideEntry {
    /// Its name (`x12.004010.204`, `acme.204`).
    pub name: String,
    /// The guide.
    pub compiled: CompiledGuide,
    /// The file it is written in, for a partner guide.
    pub unit: Option<usize>,
}

/// A setting of the partner.
#[derive(Debug, Clone)]
pub struct Setting {
    /// Its name.
    pub name: String,
    /// Its type.
    pub ty: Type,
    /// The file and item index of its declaration.
    pub unit: usize,
    /// The item index.
    pub item: usize,
}

/// The partner file's facts.
#[derive(Debug, Clone, Default)]
pub struct Partner {
    /// Its file.
    pub unit: usize,
    /// Its name.
    pub name: String,
    /// What people call it.
    pub label: String,
    /// Its zone.
    pub zone: Option<String>,
    /// Its currency.
    pub currency: Option<String>,
    /// Its settings.
    pub settings: Vec<Setting>,
}

/// What a map reads.
#[derive(Debug, Clone)]
pub enum Source {
    /// X12, with a guide.
    X12(usize),
    /// A document.
    Document(&'static str),
    /// A CSV file.
    Csv(Vec<(String, Type)>),
    /// JSON.
    Json,
}

/// One map's facts.
#[derive(Debug, Clone)]
pub struct MapInfo {
    /// Its file.
    pub unit: usize,
    /// What it reads.
    pub source: Source,
    /// The guide it writes, for an outbound map.
    pub target_guide: Option<usize>,
    /// The document it fills or reads.
    pub document: Option<&'static str>,
}

impl MapInfo {
    /// Whether it turns a message into a document.
    #[must_use]
    pub fn inbound(&self) -> bool {
        !matches!(self.source, Source::Document(_))
    }

    /// The guide it reads or writes.
    #[must_use]
    pub fn guide(&self) -> Option<usize> {
        match self.source {
            Source::X12(guide) => Some(guide),
            _ => self.target_guide,
        }
    }
}

/// A function's signature.
#[derive(Debug, Clone)]
pub struct FnSig {
    /// The parameters: name, type, and whether it has a default.
    pub params: Vec<(String, Type, bool)>,
    /// What it returns.
    pub ret: Type,
}

/// A checked project, ready to run.
#[derive(Debug, Clone, Default)]
pub struct Program {
    /// The files.
    pub units: Vec<Unit>,
    /// The samples.
    pub samples: Vec<Sample>,
    /// The guides, by index.
    pub guides: Vec<GuideEntry>,
    /// Guide indexes by name.
    pub guide_names: BTreeMap<String, usize>,
    /// The partner.
    pub partner: Option<Partner>,
    /// Maps by name.
    pub maps: BTreeMap<String, MapInfo>,
    /// Rules files by name: their file and their map.
    pub rules: BTreeMap<String, (usize, String)>,
    /// Flows by name: their file.
    pub flows: BTreeMap<String, usize>,
    /// Libraries by name: their file.
    pub libraries: BTreeMap<String, usize>,
    /// Function signatures by file and name.
    pub fns: BTreeMap<(usize, String), FnSig>,
    /// Crosswalk types by file and name.
    pub crosswalks: BTreeMap<(usize, String), (Type, Type)>,
    /// Constant types by file and name.
    pub consts: BTreeMap<(usize, String), Type>,
    /// What Trenova said about the organisation.
    pub context: Context,
    /// What the checker learned, for the editor.
    pub facts: Facts,
}

impl Program {
    /// The partner's interchange block.
    #[must_use]
    pub fn interchange(&self) -> Option<&super::syntax::ast::Interchange> {
        let partner = self.partner.as_ref()?;
        self.units[partner.unit]
            .items()
            .iter()
            .find_map(|item| match item {
                Item::Interchange(interchange) => Some(interchange),
                _ => None,
            })
    }

    /// The separators Trenova writes for the partner: element, segment, component and
    /// repetition.
    #[must_use]
    pub fn partner_separators(&self) -> Vec<char> {
        let written: Vec<char> = self
            .interchange()
            .map(|interchange| {
                interchange
                    .separators
                    .iter()
                    .filter_map(|separator| separator.value.chars().next())
                    .collect()
            })
            .unwrap_or_default();
        if written.len() == 4 {
            written
        } else {
            vec!['*', '~', '>', '^']
        }
    }

    /// Whether any file has an error.
    #[must_use]
    pub fn has_errors(&self) -> bool {
        self.units.iter().any(|unit| {
            unit.diagnostics.iter().any(|diagnostic| {
                diagnostic.severity == trenova_module_kit::diagnostic::Severity::Error
            })
        })
    }

    /// Whether any file has an error outside its tests and properties: such a project does not
    /// run, while one whose only errors are in tests (a sample the host left out of a run, say)
    /// runs as it would in production.
    #[must_use]
    pub fn has_code_errors(&self) -> bool {
        self.units.iter().any(|unit| {
            let tests: Vec<Span> = unit
                .items()
                .iter()
                .filter(|item| matches!(item, Item::Test(_) | Item::Property(_)))
                .map(Item::span)
                .collect();
            unit.diagnostics.iter().any(|diagnostic| {
                diagnostic.severity == trenova_module_kit::diagnostic::Severity::Error
                    && !tests.iter().any(|test| {
                        test.start <= diagnostic.span.start && diagnostic.span.end <= test.end
                    })
            })
        })
    }

    /// Whether a file declares a map of this name, even one whose header has an error and so
    /// was not registered: that error is reported in its own file, not at each use.
    #[must_use]
    pub fn declares_map(&self, name: &str) -> bool {
        self.units.iter().any(|unit| {
            matches!(unit.header(), Header::Map { name: declared, .. } if declared.name == name)
        })
    }

    /// A file by its header's name.
    #[must_use]
    pub fn unit_named(&self, name: &str) -> Option<usize> {
        self.units
            .iter()
            .position(|unit| unit.header().name().is_some_and(|ident| ident.name == name))
    }

    /// A file by path.
    #[must_use]
    pub fn unit_at(&self, path: &str) -> Option<usize> {
        self.units.iter().position(|unit| unit.path == path)
    }

    /// The declaration a name stands for in a file: its own, then its libraries'.
    #[must_use]
    pub fn decl(&self, unit: usize, name: &str) -> Option<(usize, Decl)> {
        let own = self.units.get(unit)?;
        if let Some(decl) = own.decls.get(name) {
            return Some((unit, *decl));
        }
        for library in own.header().uses() {
            if let Some(&library_unit) = self.libraries.get(&library.name)
                && let Some(decl) = self.units[library_unit].decls.get(name)
            {
                return Some((library_unit, *decl));
            }
        }
        None
    }

    /// A guide by name.
    #[must_use]
    pub fn guide(&self, name: &str) -> Option<&GuideEntry> {
        self.guide_names.get(name).map(|index| &self.guides[*index])
    }

    /// The type `partner` has.
    #[must_use]
    pub fn partner_type(&self) -> Type {
        let mut fields = vec![
            ("id".to_owned(), Type::Text),
            ("name".to_owned(), Type::Text),
            ("code".to_owned(), Type::Text),
            ("zone".to_owned(), Type::Zone),
            ("currency".to_owned(), Type::Text),
        ];
        if let Some(partner) = &self.partner {
            for setting in &partner.settings {
                fields.push((setting.name.clone(), setting.ty.clone()));
            }
        }
        Type::Struct(fields)
    }

    /// The type `org` has.
    #[must_use]
    pub fn org_type() -> Type {
        Type::Struct(vec![
            ("name".to_owned(), Type::Text),
            ("scac".to_owned(), Type::Text),
            ("zone".to_owned(), Type::Zone),
        ])
    }

    /// The type of a record's custom fields.
    #[must_use]
    pub fn custom_type(&self, record: &str) -> Type {
        let fields = self
            .context
            .custom_fields
            .get(record)
            .map(|fields| {
                fields
                    .iter()
                    .map(|field| {
                        let ty = match field.ty.as_str() {
                            "Int" => Type::Int,
                            "Decimal" => Type::Decimal,
                            "Bool" => Type::Bool,
                            "Date" => Type::Date,
                            "DateTime" => Type::DateTime,
                            "Texts" => Type::Text.list(),
                            _ => Type::Text,
                        };
                        (field.name.clone(), ty.optional())
                    })
                    .collect()
            })
            .unwrap_or_default();
        Type::Struct(fields)
    }

    /// A schema field's type, custom fields included.
    #[must_use]
    pub fn field_type(&self, field: &schema::Field) -> Type {
        match field.spec {
            schema::Spec::Custom(record) => self.custom_type(record),
            spec => spec.ty(),
        }
    }

    /// The type of reading a field of a document or record: optional unless it is required,
    /// and lists, references by qualifier and custom fields read as empty rather than none.
    #[must_use]
    pub fn read_type(&self, field: &schema::Field) -> Type {
        let ty = self.field_type(field);
        let never_none = field.required
            || matches!(
                field.spec,
                schema::Spec::Records(_)
                    | schema::Spec::Texts
                    | schema::Spec::Refs
                    | schema::Spec::Custom(_)
            );
        if never_none { ty } else { ty.optional() }
    }
}

/// Checks a project.
#[must_use]
pub fn check(project: &Project, context: &Context) -> Program {
    let mut program = Program {
        samples: project.samples.clone(),
        context: context.clone(),
        ..Program::default()
    };
    load_units(&mut program, project);
    guide::compile_guides(&mut program);
    collect_partner(&mut program);
    collect_headers(&mut program);
    collect_signatures(&mut program);
    let mut checker = Checker::new(&program);
    checker.check_all();
    let (diagnostics, facts) = checker.finish();
    for (unit, found) in program.units.iter_mut().zip(diagnostics) {
        unit.diagnostics.extend(found);
        unit.diagnostics
            .sort_by_key(|diagnostic| (diagnostic.span.start, diagnostic.span.end));
        unit.diagnostics
            .dedup_by(|a, b| a.span == b.span && a.message == b.message);
    }
    program.facts = facts;
    program
}

fn load_units(program: &mut Program, project: &Project) {
    let mut names: BTreeMap<String, (usize, Span)> = BTreeMap::new();
    let mut partners = 0usize;
    for file in &project.files {
        let parsed = parse(&file.source);
        let mut diagnostics = parsed.diagnostics.clone();
        let mut decls = BTreeMap::new();
        for (index, item) in parsed.file.items.iter().enumerate() {
            let (name, decl) = match item {
                Item::Fn(item) => (&item.name, Decl::Fn(index)),
                Item::Crosswalk(item) => (&item.name, Decl::Crosswalk(index)),
                Item::Const(item) => (&item.name, Decl::Const(index)),
                _ => continue,
            };
            if decls.insert(name.name.clone(), decl).is_some() {
                diagnostics.push(Diagnostic::error(
                    "duplicate-name",
                    name.span,
                    Message::with(
                        message!("{0} is declared twice in this file"),
                        [Arg::from(name.name.as_str())],
                    ),
                ));
            }
        }
        let unit_index = program.units.len();
        if let Some(name) = parsed.file.header.name() {
            if matches!(parsed.file.header, Header::Partner { .. }) {
                partners += 1;
                if partners > 1 {
                    diagnostics.push(Diagnostic::error(
                        "duplicate-partner",
                        name.span,
                        Message::new(message!("A project has one partner file")),
                    ));
                }
            }
            if let Some((other, _)) = names.get(&name.name) {
                let other_path = project.files[*other].path.clone();
                diagnostics.push(Diagnostic::error(
                    "duplicate-name",
                    name.span,
                    Message::with(
                        message!("{0} is already the name of {1}"),
                        [Arg::from(name.name.as_str()), Arg::from(other_path)],
                    ),
                ));
            } else {
                names.insert(name.name.clone(), (unit_index, name.span));
            }
        }
        program.units.push(Unit {
            path: file.path.clone(),
            source: file.source.clone(),
            parsed,
            diagnostics,
            decls,
        });
    }
    if partners == 0
        && let Some(first) = program.units.first_mut()
    {
        first.diagnostics.push(Diagnostic::error(
            "missing-partner",
            Span::new(0, 0),
            Message::new(message!(
                "The project has no partner file: add one that starts with partner"
            )),
        ));
    }
}

fn collect_partner(program: &mut Program) {
    let Some(unit_index) = program
        .units
        .iter()
        .position(|unit| matches!(unit.header(), Header::Partner { .. }))
    else {
        return;
    };
    let unit = &program.units[unit_index];
    let Header::Partner { name, label } = unit.header() else {
        return;
    };
    let mut partner = Partner {
        unit: unit_index,
        name: name.name.clone(),
        label: label
            .as_ref()
            .map(|label| label.value.clone())
            .unwrap_or_default(),
        ..Partner::default()
    };
    let mut diagnostics = Vec::new();
    for (index, item) in unit.items().iter().enumerate() {
        match item {
            Item::Zone(zone) => {
                if crate::lading::tz::Zone::named(&zone.name).is_none() {
                    diagnostics.push(Diagnostic::error(
                        "unknown-zone",
                        zone.span,
                        Message::with(
                            message!("{0} is not a time zone Trenova knows"),
                            [Arg::from(zone.name.as_str())],
                        ),
                    ));
                }
                partner.zone = Some(zone.name.clone());
            }
            Item::Currency(currency) => partner.currency = Some(currency.value.clone()),
            Item::Setting(setting) => {
                let (ty, found) = types::resolve(setting_type(setting));
                diagnostics.extend(found);
                if partner
                    .settings
                    .iter()
                    .any(|existing| existing.name == setting.name.name)
                {
                    diagnostics.push(Diagnostic::error(
                        "duplicate-name",
                        setting.name.span,
                        Message::with(
                            message!("{0} is declared twice in this file"),
                            [Arg::from(setting.name.name.as_str())],
                        ),
                    ));
                }
                if !matches!(
                    ty.unwrapped(),
                    Type::Bool
                        | Type::Int
                        | Type::Decimal
                        | Type::Text
                        | Type::Percent
                        | Type::Money
                        | Type::Duration
                        | Type::Measure(_)
                        | Type::Zone
                        | Type::Any
                ) {
                    diagnostics.push(Diagnostic::error(
                        "setting-type",
                        setting.ty.span,
                        Message::new(message!(
                            "A setting is a Bool, Int, Decimal, Text, Percent, Money, Duration, Zone or a measure"
                        )),
                    ));
                }
                partner.settings.push(Setting {
                    name: setting.name.name.clone(),
                    ty,
                    unit: unit_index,
                    item: index,
                });
            }
            _ => {}
        }
    }
    program.units[unit_index].diagnostics.extend(diagnostics);
    program.partner = Some(partner);
}

fn setting_type(setting: &SettingDecl) -> &TypeExpr {
    &setting.ty
}

fn collect_headers(program: &mut Program) {
    for unit_index in 0..program.units.len() {
        let header = program.units[unit_index].header().clone();
        let mut diagnostics = Vec::new();
        match &header {
            Header::Map { name, from, to, .. } => {
                let info = map_info(program, unit_index, from, to, &mut diagnostics);
                if let Some(info) = info {
                    program.maps.insert(name.name.clone(), info);
                }
            }
            Header::Rules { name, map, .. } => {
                program
                    .rules
                    .insert(name.name.clone(), (unit_index, map.name.clone()));
            }
            Header::Flow { name, .. } => {
                program.flows.insert(name.name.clone(), unit_index);
            }
            Header::Library { name, .. } => {
                program.libraries.insert(name.name.clone(), unit_index);
            }
            _ => {}
        }
        program.units[unit_index].diagnostics.extend(diagnostics);
    }
    for unit_index in 0..program.units.len() {
        let uses = program.units[unit_index].header().uses().to_vec();
        for library in uses {
            if !program.libraries.contains_key(&library.name) {
                let names: Vec<&str> = program.libraries.keys().map(String::as_str).collect();
                let mut diagnostic = Diagnostic::error(
                    "unknown-library",
                    library.span,
                    Message::with(
                        message!("There is no library named {0}"),
                        [Arg::from(library.name.as_str())],
                    ),
                );
                if let Some(close) = trenova_module_kit::diagnostic::nearest(&library.name, names) {
                    diagnostic = diagnostic.with_fix(replace_fix(library.span, close));
                }
                program.units[unit_index].diagnostics.push(diagnostic);
            }
        }
    }
}

/// A fix that writes `with` over `span`.
impl Checker<'_> {
    /// The library, unused by `unit`, that declares `name` as the kind `wanted` matches: what a
    /// missing function or crosswalk is usually missing.
    pub(crate) fn unused_library_with(
        &self,
        unit: usize,
        name: &str,
        wanted: fn(&Decl) -> bool,
    ) -> Option<String> {
        let program: &Program = self.program;
        let used = program.units[unit].header().uses();
        program
            .libraries
            .iter()
            .filter(|(library, _)| !used.iter().any(|ident| ident.name == **library))
            .find(|(_, library_unit)| {
                program.units[**library_unit]
                    .decls
                    .get(name)
                    .is_some_and(wanted)
            })
            .map(|(library, _)| library.clone())
    }

    /// A fix that adds `use library` to `unit`'s header.
    pub(crate) fn use_fix(
        &self,
        unit: usize,
        library: &str,
    ) -> trenova_module_kit::diagnostic::Fix {
        let program: &Program = self.program;
        let source = &program.units[unit].source;
        let header = program.units[unit].header();
        let (at, insert) = if let Some(last) = header.uses().last() {
            (last.span.end, format!(", {library}"))
        } else {
            let name_end = header.name().map_or(0, |name| name.span.end as usize);
            let line_end = source[name_end..]
                .find('\n')
                .map_or(source.len(), |at| name_end + at);
            let header_end = source[line_end..]
                .find("\n\n")
                .map_or(source.len(), |at| line_end + at);
            let word = if matches!(header, Header::Map { .. }) {
                "use "
            } else {
                "use"
            };
            (
                u32::try_from(header_end).unwrap_or(0),
                format!("\n  {word} {library}"),
            )
        };
        trenova_module_kit::diagnostic::Fix {
            label: Message::with(message!("Use {0}"), [Arg::from(library)]),
            changes: vec![trenova_module_kit::diagnostic::Change {
                span: Span::new(at, at),
                insert,
            }],
        }
    }
}

pub(crate) fn replace_fix(span: Span, with: &str) -> trenova_module_kit::diagnostic::Fix {
    trenova_module_kit::diagnostic::Fix {
        label: Message::with(message!("Change to {0}"), [Arg::from(with)]),
        changes: vec![trenova_module_kit::diagnostic::Change {
            span,
            insert: with.to_owned(),
        }],
    }
}

fn map_info(
    program: &mut Program,
    unit: usize,
    from: &Endpoint,
    to: &Endpoint,
    diagnostics: &mut Vec<Diagnostic>,
) -> Option<MapInfo> {
    let source = match from {
        Endpoint::Guide(name) => Source::X12(guide::guide_index(program, name, diagnostics)?),
        Endpoint::Document(name) => Source::Document(document(name, diagnostics)?),
        Endpoint::Csv { columns, .. } => Source::Csv(csv_columns(columns, diagnostics)),
        Endpoint::Json(_) => Source::Json,
        Endpoint::Missing(_) => return None,
    };
    let inbound = !matches!(source, Source::Document(_));
    let (target_guide, document_name) = match to {
        Endpoint::Guide(name) if !inbound => {
            (Some(guide::guide_index(program, name, diagnostics)?), None)
        }
        Endpoint::Document(name) if inbound => (None, Some(document(name, diagnostics)?)),
        Endpoint::Missing(_) => return None,
        other => {
            diagnostics.push(Diagnostic::error(
                "map-direction",
                other.span(),
                if inbound {
                    Message::new(message!(
                        "A map from X12, CSV or JSON writes a document such as loadTender"
                    ))
                } else {
                    Message::new(message!(
                        "A map from a document writes X12 with a guide such as x12.004010.214"
                    ))
                },
            ));
            return None;
        }
    };
    let document_name = match source {
        Source::Document(name) => Some(name),
        _ => document_name,
    };
    if let (Some(document_name), Some(guide_index)) = (
        document_name,
        match source {
            Source::X12(guide) => Some(guide),
            _ => target_guide,
        },
    ) {
        let set = program.guides[guide_index]
            .compiled
            .guide()
            .transaction_set
            .clone();
        let expected = schema::document(document_name).and_then(|record| record.set);
        if expected.is_some_and(|expected| expected != set) {
            diagnostics.push(Diagnostic::warning(
                "map-set",
                from.span().cover(to.span()),
                Message::with(
                    message!("{0} is a {1}, but the guide is for a {2}"),
                    [
                        Arg::from(document_name),
                        Arg::from(expected.unwrap_or_default()),
                        Arg::from(set),
                    ],
                ),
            ));
        }
    }
    Some(MapInfo {
        unit,
        source,
        target_guide,
        document: document_name,
    })
}

fn document(
    name: &super::syntax::ast::Ident,
    diagnostics: &mut Vec<Diagnostic>,
) -> Option<&'static str> {
    if let Some(record) = schema::document(&name.name) {
        return Some(record.name);
    }
    let mut diagnostic = Diagnostic::error(
        "unknown-document",
        name.span,
        Message::with(
            message!(
                "{0} is not a document: loadTender, tenderResponse, shipmentStatus or freightInvoice"
            ),
            [Arg::from(name.name.as_str())],
        ),
    );
    if let Some(close) = trenova_module_kit::diagnostic::nearest(
        &name.name,
        schema::DOCUMENTS.iter().map(|record| record.name),
    ) {
        diagnostic = diagnostic.with_fix(replace_fix(name.span, close));
    }
    diagnostics.push(diagnostic);
    None
}

fn csv_columns(columns: &[CsvColumn], diagnostics: &mut Vec<Diagnostic>) -> Vec<(String, Type)> {
    let mut seen = BTreeSet::new();
    columns
        .iter()
        .map(|column| {
            let (ty, found) = types::resolve(&column.ty);
            diagnostics.extend(found);
            if !seen.insert(column.name.name.to_lowercase()) {
                diagnostics.push(Diagnostic::error(
                    "duplicate-name",
                    column.name.span,
                    Message::with(
                        message!("The column {0} is listed twice"),
                        [Arg::from(column.name.name.as_str())],
                    ),
                ));
            }
            (column.name.name.clone(), ty.optional())
        })
        .collect()
}

fn collect_signatures(program: &mut Program) {
    for unit_index in 0..program.units.len() {
        let mut diagnostics = Vec::new();
        let items = program.units[unit_index].items().to_vec();
        for item in &items {
            match item {
                Item::Fn(decl) => {
                    let params = decl
                        .params
                        .iter()
                        .map(|param| {
                            let (ty, found) = types::resolve(&param.ty);
                            diagnostics.extend(found);
                            (param.name.name.clone(), ty, param.default.is_some())
                        })
                        .collect();
                    let (ret, found) = types::resolve(&decl.ret);
                    diagnostics.extend(found);
                    program
                        .fns
                        .insert((unit_index, decl.name.name.clone()), FnSig { params, ret });
                }
                Item::Crosswalk(decl) => {
                    let (left, found) = types::resolve(&decl.left);
                    diagnostics.extend(found);
                    let (right, found) = types::resolve(&decl.right);
                    diagnostics.extend(found);
                    program
                        .crosswalks
                        .insert((unit_index, decl.name.name.clone()), (left, right));
                }
                _ => {}
            }
        }
        program.units[unit_index].diagnostics.extend(diagnostics);
    }
}

/// The checker's working state: the program it reads, and what it reports.
pub(crate) struct Checker<'p> {
    pub(crate) program: &'p Program,
    pub(crate) diagnostics: Vec<Vec<Diagnostic>>,
    pub(crate) facts: Facts,
    pub(crate) const_types: BTreeMap<(usize, String), Type>,
    calls: BTreeMap<(usize, String), BTreeSet<(usize, String)>>,
}

impl<'p> Checker<'p> {
    fn new(program: &'p Program) -> Self {
        Self {
            program,
            diagnostics: vec![Vec::new(); program.units.len()],
            facts: Facts::default(),
            const_types: BTreeMap::new(),
            calls: BTreeMap::new(),
        }
    }

    fn finish(self) -> (Vec<Vec<Diagnostic>>, Facts) {
        (self.diagnostics, self.facts)
    }

    pub(crate) fn report(&mut self, unit: usize, diagnostic: Diagnostic) {
        if let Some(found) = self.diagnostics.get_mut(unit) {
            found.push(diagnostic);
        }
    }

    pub(crate) fn occurrence(
        &mut self,
        unit: usize,
        span: Span,
        target: Target,
        declaration: bool,
        write: bool,
    ) {
        self.facts.occurrences.push(Occurrence {
            unit,
            span,
            target,
            declaration,
            write,
        });
    }

    pub(crate) fn record_call(&mut self, from: (usize, String), to: (usize, String)) {
        self.calls.entry(from).or_default().insert(to);
    }

    fn check_all(&mut self) {
        let program = self.program;
        for unit in 0..program.units.len() {
            self.declare_file(unit);
        }
        for unit in 0..program.units.len() {
            self.check_decls(unit);
        }
        for unit in 0..program.units.len() {
            match program.units[unit].header() {
                Header::Map { name, .. } => {
                    if let Some(info) = program.maps.get(&name.name) {
                        if info.inbound() {
                            self.check_inbound(unit, info);
                        } else {
                            self.check_outbound(unit, info);
                        }
                    }
                }
                Header::Rules { map, .. } => self.check_rules_file(unit, map),
                Header::Flow { partner, .. } => self.check_flow(unit, partner),
                Header::Partner { .. } => self.check_partner_items(unit),
                _ => {}
            }
            self.check_tests(unit);
        }
        self.check_flow_routing();
        self.check_call_cycles();
    }

    fn declare_file(&mut self, unit: usize) {
        let program = self.program;
        let header = program.units[unit].header();
        if let Some(name) = header.name() {
            self.occurrence(
                unit,
                name.span,
                Target::File(name.name.clone()),
                true,
                false,
            );
        }
        for library in header.uses() {
            self.occurrence(
                unit,
                library.span,
                Target::File(library.name.clone()),
                false,
                false,
            );
        }
        for item in program.units[unit].items() {
            let (name, kind) = match item {
                Item::Fn(item) => (&item.name, ()),
                Item::Crosswalk(item) => (&item.name, ()),
                Item::Const(item) => (&item.name, ()),
                Item::Setting(item) => {
                    self.occurrence(
                        unit,
                        item.name.span,
                        Target::Setting(item.name.name.clone()),
                        true,
                        false,
                    );
                    continue;
                }
                _ => continue,
            };
            let () = kind;
            self.occurrence(
                unit,
                name.span,
                Target::Decl {
                    unit,
                    name: name.name.clone(),
                },
                true,
                false,
            );
        }
    }

    fn check_call_cycles(&mut self) {
        let mut reported = BTreeSet::new();
        let nodes: Vec<(usize, String)> = self.calls.keys().cloned().collect();
        for start in nodes {
            let mut stack = vec![(start.clone(), vec![start.clone()])];
            let mut seen = BTreeSet::new();
            while let Some((node, path)) = stack.pop() {
                if !seen.insert(node.clone()) {
                    continue;
                }
                let nexts: Vec<(usize, String)> = self
                    .calls
                    .get(&node)
                    .into_iter()
                    .flatten()
                    .cloned()
                    .collect();
                for next in &nexts {
                    if *next == start {
                        if reported.insert(start.clone()) {
                            let names: Vec<String> =
                                path.iter().map(|(_, name)| name.clone()).collect();
                            let span = self.decl_span(&start);
                            self.report(
                                start.0,
                                Diagnostic::error(
                                    "call-cycle",
                                    span,
                                    Message::with(
                                        message!("{0} calls itself through {1}: functions may not recurse"),
                                        [Arg::from(start.1.as_str()), Arg::from(names.join(" -> "))],
                                    ),
                                ),
                            );
                        }
                    } else {
                        let mut next_path = path.clone();
                        next_path.push(next.clone());
                        stack.push((next.clone(), next_path));
                    }
                }
            }
        }
    }

    fn decl_span(&self, decl: &(usize, String)) -> Span {
        self.program.units[decl.0]
            .items()
            .iter()
            .find_map(|item| match item {
                Item::Fn(item) if item.name.name == decl.1 => Some(item.name.span),
                _ => None,
            })
            .unwrap_or_default()
    }
}

/// The ACME project of `testdata/lading/acme`, for tests across the crate.
#[cfg(test)]
pub(crate) mod corpus {
    use std::path::Path;

    use crate::lading::project::{Context, CustomField, Org, Project, Sample, SourceFile};

    fn walk(root: &Path, dir: &Path, project: &mut Project) {
        let mut entries: Vec<_> = std::fs::read_dir(dir).expect("corpus").flatten().collect();
        entries.sort_by_key(std::fs::DirEntry::path);
        for entry in entries {
            let path = entry.path();
            if path.is_dir() {
                walk(root, &path, project);
                continue;
            }
            let relative = path
                .strip_prefix(root)
                .expect("relative")
                .to_string_lossy()
                .replace('\\', "/");
            let text = std::fs::read_to_string(&path).expect("text");
            if relative.ends_with(".lading") {
                project.files.push(SourceFile {
                    path: relative,
                    source: text,
                });
            } else {
                project.samples.push(Sample {
                    path: relative,
                    content: text,
                });
            }
        }
    }

    fn load(name: &str) -> Project {
        let root = Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("testdata/lading")
            .join(name);
        let mut project = Project::default();
        walk(&root, &root, &mut project);
        project
    }

    /// The project.
    pub fn acme() -> Project {
        load("acme")
    }

    /// The project that uses every feature of the language, for Acme Manufacturing.
    pub fn acme_mfg() -> Project {
        load("acme-mfg")
    }

    /// The organisation Acme Manufacturing's project runs for, with the shipment custom fields
    /// its maps write.
    pub fn acme_mfg_context() -> Context {
        let field = |name: &str, ty: &str, label: &str| CustomField {
            name: name.into(),
            ty: ty.into(),
            label: label.into(),
        };
        Context {
            org: Org {
                name: "Trenova Logistics".into(),
                scac: "TRNV".into(),
                zone: "America/Chicago".into(),
            },
            custom_fields: [(
                "shipment".to_owned(),
                vec![
                    field("po_number", "Text", "PO number"),
                    field("declared_value", "Decimal", "Declared value"),
                    field("appointment_required", "Bool", "Appointment required"),
                ],
            )]
            .into_iter()
            .collect(),
        }
    }

    /// The organisation it runs for.
    pub fn context() -> Context {
        Context {
            org: Org {
                name: "Trenova Freight".into(),
                scac: "TRNV".into(),
                zone: "America/Chicago".into(),
            },
            ..Context::default()
        }
    }
}

#[cfg(test)]
mod check_tests {
    use super::corpus::{acme, acme_mfg, acme_mfg_context, context};
    use trenova_module_kit::diagnostic::Severity;

    use super::{Program, check};
    use crate::lading::testing::run_tests;

    #[test]
    fn the_acme_project_checks_clean() {
        assert_clean(&check(&acme(), &context()), Severity::Info);
    }

    #[test]
    fn the_project_using_every_feature_checks_clean_and_passes_its_tests() {
        let program = check(&acme_mfg(), &acme_mfg_context());
        assert_clean(&program, Severity::Info);
        let report = run_tests(&program);
        let failed: Vec<String> = report
            .tests
            .iter()
            .filter(|test| !test.passed)
            .map(|test| {
                let failures: Vec<String> = test
                    .failures
                    .iter()
                    .map(|failure| failure.message.english())
                    .collect();
                format!(
                    "{} {}: {}",
                    program.units[test.unit].path,
                    test.name,
                    failures.join("; ")
                )
            })
            .collect();
        assert!(failed.is_empty(), "{}", failed.join("\n"));
        assert_eq!(report.tests.len(), 53);
    }

    /// Asserts the program has no finding as severe as `least`, or more.
    fn assert_clean(program: &Program, least: Severity) {
        let mut found = Vec::new();
        for unit in &program.units {
            for diagnostic in &unit.diagnostics {
                if rank(diagnostic.severity) < rank(least) {
                    continue;
                }
                let line = unit.source[..diagnostic.span.start as usize]
                    .matches('\n')
                    .count()
                    + 1;
                found.push(format!(
                    "{}:{line}: {} {}: {}",
                    unit.path,
                    diagnostic.severity.as_str(),
                    diagnostic.code,
                    diagnostic.message.english()
                ));
            }
        }
        assert!(found.is_empty(), "{}", found.join("\n"));
    }

    fn rank(severity: Severity) -> u8 {
        match severity {
            Severity::Info => 0,
            Severity::Warning => 1,
            Severity::Error => 2,
        }
    }
}

#[cfg(test)]
mod findings {
    use super::check;
    use super::corpus::{acme, context};
    use crate::lading::project::SourceFile;

    /// The finding codes after replacing `old` with `new` in the file at `path`.
    fn codes(path: &str, old: &str, new: &str) -> Vec<&'static str> {
        let mut project = acme();
        let file = project
            .files
            .iter_mut()
            .find(|file| file.path == path)
            .expect("file");
        assert!(file.source.contains(old), "{old}");
        file.source = file.source.replacen(old, new, 1);
        let program = check(&project, &context());
        program
            .units
            .iter()
            .flat_map(|unit| unit.diagnostics.iter().map(|found| found.code))
            .collect()
    }

    #[test]
    #[allow(clippy::too_many_lines)]
    fn mistakes_are_found() {
        let map = "maps/tender-in.lading";
        let cases: &[(&str, &str, &str, &str)] = &[
            (map, "stop.S501", "S501", "x12-unqualified"),
            (map, "key      <- B204", "key      <- B299", "x12-unknown"),
            (
                map,
                "key      <- B204",
                "key      <- L11[02 == \"BM\"].01",
                "optional-unhandled",
            ),
            (map, "key      <- B204", "", "field-unassigned"),
            (
                map,
                "purpose  <- B2A01 via purposes",
                "purpose  <- B2A01",
                "type-mismatch",
            ),
            (
                map,
                "type     <- stop.S502 via stopTypes",
                "type     <- stop.S502 via stopTypez",
                "unknown-crosswalk",
            ),
            (
                map,
                "name     <- party.N102.titleCase()",
                "name     <- party.N102.titelCase()",
                "unknown-function",
            ),
            (
                map,
                "bol      <- L11[02 == \"BM\"]",
                "bol      <- L11[02 == \"BX\"]",
                "code-not-in-list",
            ),
            (
                map,
                "sequence <- stop.S501",
                "sequence <- stop.S502",
                "type-mismatch",
            ),
            (
                "lib/common.lading",
                "  \"04\" <-> Change\n  \"05\" -> Change\n",
                "",
                "crosswalk-misses-code",
            ),
            (
                "maps/status-out.lading",
                "  else -> \"X6\"\n",
                "",
                "crosswalk-incomplete",
            ),
            (
                "maps/status-out.lading",
                ", 03 <- org.scac }",
                " }",
                "x12-mandatory-unwritten",
            ),
            (
                "maps/status-out.lading",
                "LX { 01 <- index + 1 }",
                "LX { 01 <- event.at }",
                "type-mismatch",
            ),
            ("maps/status-out.lading", "B10 {", "AT7 {", "x12-unknown"),
            (
                "rules/tender-checks.lading",
                "require (weight ?? 0 lb) <= partner.maxWeight",
                "require true",
                "rule-always-passes",
            ),
            (
                "maps/response-out.lading",
                "B1 {",
                "B1 {\n  05 <- key",
                "x12-unknown",
            ),
            (
                "maps/tender-in.lading",
                "else reject \"Stop {stop.S501} has no type\"",
                "",
                "optional-unhandled",
            ),
            (
                "flows/tender.lading",
                "on Cancel   ->",
                "on Cancelled ->",
                "unknown-variant",
            ),
            (
                "flows/tender.lading",
                "via statusOut",
                "via statusOutt",
                "unknown-map",
            ),
            (
                "flows/tender.lading",
                "on InvoicePosted",
                "on StopArrived",
                "send-event-document",
            ),
            (
                "lib/common.lading",
                "fn zip5(postal: Text) -> Text = postal.digits().take(5)",
                "fn zip5(postal: Text) -> Text = zip6(postal)\nfn zip6(postal: Text) -> Text = zip5(postal)",
                "call-cycle",
            ),
            (
                "partner.lading",
                "version 004010",
                "version 003050",
                "interchange-version",
            ),
            (
                "guides/acme-204.lading",
                "L11 max 10",
                "L11 max 100",
                "guide-widens",
            ),
            (
                "guides/acme-204.lading",
                "0300.0310.N1 {",
                "N1 {",
                "guide-ambiguous-path",
            ),
            (
                "flows/tender.lading",
                "send 210 via invoiceOut on InvoicePosted",
                "send 210 via invoiceOut on InvoicePosted\nsend 210 via invoiceOut on InvoicePosted",
                "duplicate-send",
            ),
            (
                "flows/tender.lading",
                "respond 990 via responseOut",
                "respond 990 via responseOut\n  otherwise accept\n\nrespond 990 via responseOut",
                "flow-respond-taken",
            ),
        ];
        let mut failures = Vec::new();
        for (path, old, new, code) in cases {
            let found = codes(path, old, new);
            if !found.contains(code) {
                failures.push(format!(
                    "{path}: {old:?} -> {new:?}: expected {code}, found {found:?}"
                ));
            }
        }
        assert!(failures.is_empty(), "{}", failures.join("\n"));
    }
    #[test]
    fn an_unknown_custom_field_says_where_custom_fields_come_from() {
        let mut project = acme();
        let file = project
            .files
            .iter_mut()
            .find(|file| file.path == "maps/tender-in.lading")
            .expect("map");
        file.source = file.source.replacen(
            "use  common\n",
            "use  common\n\ncustom.po_number <- B204\n",
            1,
        );
        let messages = |program: &super::Program| -> Vec<String> {
            program
                .units
                .iter()
                .flat_map(|unit| unit.diagnostics.iter())
                .filter(|found| found.code == "unknown-field")
                .map(|found| found.message.english())
                .collect()
        };
        let none = messages(&check(&project, &context()));
        assert_eq!(
            none,
            [
                "The organization has no custom field po_number: add it in Settings, under Custom fields"
            ]
        );
        let mut other = context();
        other.custom_fields.insert(
            "shipment".to_owned(),
            vec![crate::lading::project::CustomField {
                name: "pro_ref".into(),
                ty: "Text".into(),
                label: "PRO reference".into(),
            }],
        );
        assert_eq!(
            messages(&check(&project, &other)),
            ["There is no custom field po_number; the organization's are pro_ref"]
        );
        let mut many = context();
        many.custom_fields.insert(
            "shipment".to_owned(),
            ["aa", "bb", "po_numbr", "cc", "dd", "ee"]
                .into_iter()
                .map(|name| crate::lading::project::CustomField {
                    name: name.into(),
                    ty: "Text".into(),
                    label: String::new(),
                })
                .collect(),
        );
        assert_eq!(
            messages(&check(&project, &many)),
            [
                "There is no custom field po_number; the organization has 6 custom fields, such as po_numbr, aa, bb, cc, dd"
            ]
        );
    }

    #[test]
    fn a_map_whose_guide_is_unknown_is_reported_once_not_at_its_tests_rules_and_flows() {
        let found = codes("maps/tender-in.lading", "from acme.204", "from acme.990");
        assert!(found.contains(&"unknown-guide"), "{found:?}");
        assert!(!found.contains(&"test-here"), "{found:?}");
        assert!(!found.contains(&"unknown-map"), "{found:?}");
    }

    #[test]
    fn a_set_or_event_a_second_flow_would_take_is_found_there() {
        let mut project = acme();
        project.files.push(SourceFile {
            path: "flows/second.lading".into(),
            source: "flow second with acme\n\nreceive 204 as tender\n  via tenderIn\n  key tender.key\n  on Original -> create\n\nsend 214 via statusOut on Delivered\n".into(),
        });
        let program = check(&project, &context());
        let second: Vec<&str> = program
            .units
            .iter()
            .filter(|unit| unit.path == "flows/second.lading")
            .flat_map(|unit| unit.diagnostics.iter().map(|found| found.code))
            .collect();
        assert!(second.contains(&"flow-receive-taken"), "{second:?}");
        assert!(second.contains(&"flow-event-taken"), "{second:?}");
        let first: Vec<&str> = program
            .units
            .iter()
            .filter(|unit| unit.path == "flows/tender.lading")
            .flat_map(|unit| unit.diagnostics.iter().map(|found| found.code))
            .collect();
        assert!(first.is_empty(), "the first flow keeps its sets: {first:?}");
    }

    #[test]
    fn a_function_from_an_unused_library_names_it_and_offers_to_use_it() {
        let mut project = acme();
        let rules = project
            .files
            .iter_mut()
            .find(|file| file.path == "rules/tender-checks.lading")
            .expect("rules");
        rules.source = rules.source.replacen(
            "rule \"hazmat needs eyes\" {",
            "rule \"hazmat needs eyes\" {\n  let opens = window([], partner.zone)",
            1,
        );
        let program = check(&project, &context());
        let unit = program
            .units
            .iter()
            .find(|unit| unit.path == "rules/tender-checks.lading")
            .expect("unit");
        let found = unit
            .diagnostics
            .iter()
            .find(|found| found.code == "unknown-function")
            .expect("unknown-function");
        assert_eq!(
            found.message.english(),
            "window is in the library common, which this file does not use"
        );
        let change = &found.fixes[0].changes[0];
        assert_eq!(change.insert, "\n  use common");
        assert_eq!(
            &unit.source[..change.span.start as usize],
            "rules tenderChecks for tenderIn"
        );
    }

    #[test]
    fn a_guide_line_names_a_segment_found_in_one_loop_by_its_id() {
        let found = codes("guides/acme-204.lading", "0300.S5 {", "S5 {");
        assert!(found.is_empty(), "{found:?}");
    }
}
