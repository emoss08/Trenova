//! The static formula operations: catalog, check, format, signature help and hover.
//!
//! Each handler decodes its input, runs the language service on byte spans, and converts every
//! position to UTF-16 code units on the way out (and the cursor position from them on the way
//! in), which is what the editor and the server's traces count. The shapes are documented in
//! `docs/engineering/rules-engine.md`.

use crate::formula::catalog;
use crate::formula::catalog::Function;
use crate::formula::check;
use crate::formula::check::Analysis;
use crate::formula::check::Diagnostic;
use crate::formula::check::Expect;
use crate::formula::check::Fix;
use crate::formula::check::RateTable;
use crate::formula::check::Schema;
use crate::formula::syntax;
use crate::formula::syntax::FormatOutcome;
use crate::formula::syntax::StatementKindInfo;
use crate::message::Message;
use crate::ops::OpResult;
use crate::ops::decode;
use crate::ops::encode;
use crate::span::Utf16Index;
use serde::Deserialize;
use serde::Serialize;
use serde_json::value::RawValue;

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireFunction {
    name: &'static str,
    signature: &'static str,
    description: Message,
    example: Message,
    category: &'static str,
    operator: bool,
    min_args: u8,
    max_args: i32,
    returns: &'static str,
    numeric_args: bool,
    predicate: bool,
}

impl WireFunction {
    fn new(function: &Function) -> Self {
        Self {
            name: function.name,
            signature: function.signature,
            description: Message::new(function.description),
            example: Message::new(function.example),
            category: function.category.as_str(),
            operator: function.operator,
            min_args: function.min_args,
            max_args: function.max_args.map_or(-1, i32::from),
            returns: function.returns.as_str(),
            numeric_args: function.numeric_args,
            predicate: function.predicate,
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireDisabled {
    name: &'static str,
    reason: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    replacement: Option<&'static str>,
    message: Message,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct CatalogOutput {
    functions: Vec<WireFunction>,
    builtins: Vec<WireFunction>,
    disabled: Vec<WireDisabled>,
    reserved: Vec<&'static str>,
}

/// Lists every function, built-in, disabled and reserved name.
pub fn catalog(_: &RawValue) -> OpResult {
    let (functions, builtins): (Vec<&Function>, Vec<&Function>) = catalog::FUNCTIONS
        .iter()
        .partition(|function| function.documented);
    encode(&CatalogOutput {
        functions: functions.into_iter().map(WireFunction::new).collect(),
        builtins: builtins.into_iter().map(WireFunction::new).collect(),
        disabled: catalog::DISABLED
            .iter()
            .map(|disabled| WireDisabled {
                name: disabled.name,
                reason: disabled.reason.as_str(),
                replacement: disabled.replacement,
                message: disabled.message(),
            })
            .collect(),
        reserved: catalog::reserved(),
    })
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct CheckInput {
    source: String,
    #[serde(default)]
    expect: Expect,
    #[serde(default)]
    schema: Option<Schema>,
    #[serde(default)]
    tables: Option<Vec<RateTable>>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireChange {
    from: u32,
    to: u32,
    insert: String,
}

#[derive(Serialize)]
#[serde(tag = "kind", rename_all = "camelCase")]
enum WireFix {
    Replace {
        label: Message,
        changes: Vec<WireChange>,
    },
    Declare {
        label: Message,
        name: String,
        #[serde(rename = "type")]
        input_type: &'static str,
    },
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireDiagnostic {
    from: u32,
    to: u32,
    severity: &'static str,
    code: &'static str,
    #[serde(flatten)]
    message: Message,
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
    fixes: Vec<WireFix>,
}

impl WireDiagnostic {
    fn new(diagnostic: &Diagnostic, index: &Utf16Index) -> Self {
        let range = index.range(diagnostic.span);
        Self {
            from: range.from,
            to: range.to,
            severity: diagnostic.severity.as_str(),
            code: diagnostic.code.as_str(),
            message: diagnostic.message.clone(),
            name: diagnostic.name.clone(),
            fixes: diagnostic
                .fixes
                .iter()
                .map(|fix| match fix {
                    Fix::Replace { label, changes } => WireFix::Replace {
                        label: label.clone(),
                        changes: changes
                            .iter()
                            .map(|change| {
                                let range = index.range(change.span);
                                WireChange {
                                    from: range.from,
                                    to: range.to,
                                    insert: change.insert.clone(),
                                }
                            })
                            .collect(),
                    },
                    Fix::Declare {
                        label,
                        name,
                        input_type,
                    } => WireFix::Declare {
                        label: label.clone(),
                        name: name.clone(),
                        input_type: input_type.as_str(),
                    },
                })
                .collect(),
        }
    }
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireBinding {
    name: String,
    #[serde(rename = "type")]
    ty: &'static str,
    reads: usize,
    from: u32,
    to: u32,
    name_from: u32,
    name_to: u32,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireStatement {
    index: usize,
    kind: &'static str,
    #[serde(skip_serializing_if = "Option::is_none")]
    name: Option<String>,
    code: String,
    from: u32,
    to: u32,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireTable {
    name: String,
    function: &'static str,
    from: u32,
    to: u32,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct CheckOutput {
    empty: bool,
    pending: bool,
    diagnostics: Vec<WireDiagnostic>,
    result_type: &'static str,
    reads: Vec<String>,
    schema_reads: Vec<String>,
    input_reads: Vec<String>,
    function_calls: Vec<String>,
    bindings: Vec<WireBinding>,
    statements: Vec<WireStatement>,
    lookup_tables: Vec<WireTable>,
}

impl CheckOutput {
    fn new(analysis: Analysis, source: &str) -> Self {
        let index = Utf16Index::new(source);
        Self {
            empty: analysis.empty,
            pending: analysis.pending,
            diagnostics: analysis
                .diagnostics
                .iter()
                .map(|diagnostic| WireDiagnostic::new(diagnostic, &index))
                .collect(),
            result_type: analysis.result_type.as_str(),
            reads: analysis.reads,
            schema_reads: analysis.schema_reads,
            input_reads: analysis.input_reads,
            function_calls: analysis.function_calls,
            bindings: analysis
                .bindings
                .into_iter()
                .map(|binding| {
                    let range = index.range(binding.span);
                    let name = index.range(binding.name_span);
                    WireBinding {
                        name: binding.name,
                        ty: binding.ty.as_str(),
                        reads: binding.reads,
                        from: range.from,
                        to: range.to,
                        name_from: name.from,
                        name_to: name.to,
                    }
                })
                .collect(),
            statements: analysis
                .statements
                .into_iter()
                .map(|statement| {
                    let range = index.range(statement.span);
                    WireStatement {
                        index: statement.index,
                        kind: match statement.kind {
                            StatementKindInfo::Let => "let",
                            StatementKindInfo::Result => "result",
                        },
                        name: statement.name,
                        code: statement.code,
                        from: range.from,
                        to: range.to,
                    }
                })
                .collect(),
            lookup_tables: analysis
                .lookup_tables
                .into_iter()
                .map(|table| {
                    let range = index.range(table.span);
                    WireTable {
                        name: table.name,
                        function: table.function,
                        from: range.from,
                        to: range.to,
                    }
                })
                .collect(),
        }
    }
}

/// Checks one formula against a schema and returns its analysis.
pub fn check(input: &RawValue) -> OpResult {
    let input: CheckInput = decode(input)?;
    let analysis = check::check(
        &input.source,
        input.expect,
        input.schema.as_ref(),
        input.tables.as_deref(),
    );
    encode(&CheckOutput::new(analysis, &input.source))
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct FormatInput {
    source: String,
    #[serde(default)]
    width: Option<usize>,
    #[serde(default)]
    operators: Vec<String>,
}

#[derive(Serialize)]
#[serde(untagged)]
enum FormatOutput {
    Formatted {
        ok: bool,
        text: String,
        changed: bool,
    },
    Refused {
        ok: bool,
        reason: &'static str,
    },
}

/// Formats one formula.
pub fn format(input: &RawValue) -> OpResult {
    let input: FormatInput = decode(input)?;
    let operators: Vec<&str> = input.operators.iter().map(String::as_str).collect();
    let width = input.width.unwrap_or(syntax::DEFAULT_WIDTH).max(1);
    encode(&match syntax::format(&input.source, width, &operators) {
        FormatOutcome::Formatted { text, changed } => FormatOutput::Formatted {
            ok: true,
            text,
            changed,
        },
        FormatOutcome::Refused(reason) => FormatOutput::Refused {
            ok: false,
            reason: reason.as_str(),
        },
    })
}

#[derive(Deserialize)]
#[serde(rename_all = "camelCase")]
struct CursorInput {
    source: String,
    position: u32,
    #[serde(default)]
    operators: Vec<String>,
    #[serde(default)]
    schema: Option<Schema>,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireSignature {
    name: &'static str,
    active: usize,
    from: u32,
    to: u32,
}

/// Finds the call and argument at a cursor.
pub fn signature(input: &RawValue) -> OpResult {
    let input: CursorInput = decode(input)?;
    let index = Utf16Index::new(&input.source);
    let position = index.to_byte(&input.source, input.position);
    let operators: Vec<&str> = input.operators.iter().map(String::as_str).collect();
    let found = check::signature_at(&input.source, position, &operators).map(|signature| {
        let range = index.range(signature.parens);
        WireSignature {
            name: signature.name,
            active: signature.active,
            from: range.from,
            to: range.to,
        }
    });
    encode(&found)
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct WireIdentifier {
    name: String,
    kind: &'static str,
    from: u32,
    to: u32,
}

/// Finds the identifier at a cursor.
pub fn identifier(input: &RawValue) -> OpResult {
    let input: CursorInput = decode(input)?;
    let index = Utf16Index::new(&input.source);
    let position = index.to_byte(&input.source, input.position);
    let found =
        check::identifier_at(&input.source, position, input.schema.as_ref()).map(|identifier| {
            let range = index.range(identifier.span);
            WireIdentifier {
                name: identifier.name,
                kind: identifier.kind.as_str(),
                from: range.from,
                to: range.to,
            }
        });
    encode(&found)
}
