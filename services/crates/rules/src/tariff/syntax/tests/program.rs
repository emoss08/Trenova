//! Loading programs: standard and organisation modules, aliases, transitive imports, every
//! loader diagnostic and the limits.

use crate::tariff::context::Context;
use crate::tariff::context::Library;
use crate::tariff::syntax::ast::FileKind;
use crate::tariff::syntax::program::Decl;
use crate::tariff::syntax::program::MAX_MODULES;
use crate::tariff::syntax::program::MAX_PROGRAM_BYTES;
use crate::tariff::syntax::program::ModuleId;
use crate::tariff::syntax::program::Program;
use crate::tariff::syntax::program::Resolved;
use crate::tariff::syntax::program::load;

fn library(path: &str, source: &str) -> Library {
    Library {
        path: path.to_owned(),
        version: Some(format!("{path}@1")),
        source: source.to_owned(),
    }
}

fn context(libraries: Vec<Library>) -> Context {
    Context {
        libraries,
        ..Context::default()
    }
}

fn codes(program: &Program) -> Vec<(u32, &'static str)> {
    program
        .diagnostics
        .iter()
        .map(|(module, diagnostic)| (module.0, diagnostic.code))
        .collect()
}

fn tariff(body: &str) -> String {
    format!("tariff \"t\"\n\n{body}\n")
}

#[test]
fn loads_every_standard_module() {
    for (path, _) in crate::tariff::stdlib::MODULES {
        let program = load(&tariff(&format!("use {path}")), None, &Context::default());
        assert!(
            program.diagnostics.is_empty(),
            "{path}: {:?}",
            codes(&program)
        );
        let module = program.module_named(path).map(|module| module.id);
        assert_eq!(module, Some(ModuleId(1)));
        let alias = path.rsplit('.').next().unwrap_or_default();
        assert_eq!(
            program.resolve(ModuleId::ROOT, alias),
            module.map(Resolved::Module)
        );
    }
}

#[test]
fn imports_names_and_resolves_them_to_their_declarations() {
    let context = context(vec![library(
        "org.accessorials",
        "library \"a\"\n\n/// Hazmat.\nfn hazmat(miles: Distance) -> Money = $1/mi * miles\nconst FEE: Money = $5\ntype Stopover = { at: Int }\nenum Tier { Low, High }\ntest \"x\" {\n  expect true\n}\n",
    )]);
    let program = load(
        &tariff("use org.accessorials.{ hazmat, FEE, Stopover, Tier }"),
        Some(FileKind::Tariff),
        &context,
    );
    assert!(program.diagnostics.is_empty(), "{:?}", codes(&program));
    let library = ModuleId(1);
    assert_eq!(
        program.resolve(ModuleId::ROOT, "hazmat"),
        Some(Resolved::Decl {
            module: library,
            decl: Decl::Fn(0)
        })
    );
    assert_eq!(
        program.resolve(ModuleId::ROOT, "FEE"),
        Some(Resolved::Decl {
            module: library,
            decl: Decl::Const(1)
        })
    );
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "Stopover"),
        Some(Resolved::Decl {
            decl: Decl::Type(2),
            ..
        })
    ));
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "Tier"),
        Some(Resolved::Decl {
            decl: Decl::Enum(3),
            ..
        })
    ));
    let module = program.module(library);
    assert_eq!(
        module.and_then(|m| m.version.clone()),
        Some(String::from("org.accessorials@1"))
    );
    assert!(module.is_some_and(|m| m.tests.contains_key("x")));
    assert!(program.resolve(ModuleId::ROOT, "accessorials").is_none());
    assert_eq!(program.root().map(|m| m.path.clone()), Some(None));
}

#[test]
fn names_a_module_by_its_last_part_or_its_alias() {
    let program = load(
        &tariff("use std.fuel as f\nuse std.ltl.{ deficit } as l\nuse std.time"),
        None,
        &Context::default(),
    );
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "f"),
        Some(Resolved::Module(_))
    ));
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "l"),
        Some(Resolved::Module(_))
    ));
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "time"),
        Some(Resolved::Module(_))
    ));
    assert!(program.resolve(ModuleId::ROOT, "fuel").is_none());
    assert!(program.resolve(ModuleId::ROOT, "ltl").is_none());
}

#[test]
fn loads_transitive_imports_once_in_dependency_order() {
    let context = context(vec![
        library(
            "org.a",
            "library \"a\"\n\nuse org.b.{ two }\nuse org.c\nfn one() -> Int = two()\n",
        ),
        library(
            "org.b",
            "library \"b\"\n\nuse org.c.{ three }\nfn two() -> Int = three()\n",
        ),
        library("org.c", "library \"c\"\n\nfn three() -> Int = 3\n"),
    ]);
    let program = load(&tariff("use org.a.{ one }\nuse org.c"), None, &context);
    assert!(program.diagnostics.is_empty(), "{:?}", codes(&program));
    assert_eq!(program.modules.len(), 4);
    let order: Vec<Option<String>> = program
        .in_dependency_order()
        .map(|module| module.path.clone())
        .collect();
    assert_eq!(
        order,
        vec![
            Some(String::from("org.c")),
            Some(String::from("org.b")),
            Some(String::from("org.a")),
            None
        ]
    );
    let a = program
        .module_named("org.a")
        .map(|m| m.id)
        .unwrap_or_default();
    assert!(matches!(
        program.resolve(a, "two"),
        Some(Resolved::Decl {
            decl: Decl::Fn(1),
            ..
        })
    ));
}

#[test]
fn reports_an_unknown_module_with_a_suggestion() {
    let program = load(&tariff("use std.feul"), None, &Context::default());
    let (_, diagnostic) = &program.diagnostics[0];
    assert_eq!(diagnostic.code, "unknown-module");
    assert_eq!(
        diagnostic.message.english(),
        "There is no module `std.feul`. Did you mean `std.fuel`?"
    );
    assert_eq!(diagnostic.fixes[0].changes[0].insert, "std.fuel");
    let program = load(&tariff("use org.nothingLikeIt"), None, &Context::default());
    assert_eq!(
        program.diagnostics[0].1.message.english(),
        "There is no module `org.nothingLikeIt`"
    );
}

#[test]
fn reports_a_path_outside_std_and_org() {
    let program = load(&tariff("use lib.fuel\nuse std"), None, &Context::default());
    assert_eq!(
        codes(&program),
        vec![(0, "module-path"), (0, "module-path")]
    );
}

#[test]
fn reports_an_import_cycle_naming_it() {
    let context = context(vec![
        library("org.a", "library \"a\"\n\nuse org.b\n"),
        library("org.b", "library \"b\"\n\nuse org.a\n"),
    ]);
    let program = load(&tariff("use org.a"), None, &context);
    assert_eq!(codes(&program), vec![(2, "import-cycle")]);
    assert_eq!(
        program.diagnostics[0].1.message.english(),
        "These modules import each other in a cycle: org.a → org.b → org.a"
    );
    assert_eq!(program.modules.len(), 3);
}

#[test]
fn reports_a_module_importing_itself() {
    let context = context(vec![library("org.a", "library \"a\"\n\nuse org.a\n")]);
    let program = load(&tariff("use org.a"), None, &context);
    assert_eq!(codes(&program), vec![(1, "import-cycle")]);
}

#[test]
fn reports_importing_a_tariff_or_a_condition() {
    let context = context(vec![
        library("org.t", "tariff \"t\"\n\nlet a = 1\n"),
        library("org.c", "condition \"c\"\n\ntrue\n"),
    ]);
    let program = load(&tariff("use org.t\nuse org.c"), None, &context);
    assert_eq!(
        codes(&program),
        vec![(0, "import-not-library"), (0, "import-not-library")]
    );
    assert_eq!(
        program.diagnostics[0].1.message.english(),
        "`org.t` is a `tariff`; only a library can be imported"
    );
}

#[test]
fn reports_unknown_and_private_imports() {
    let context = context(vec![library(
        "org.a",
        "library \"a\"\n\nfn residential() -> Money = $1\nprivate fn secret() -> Int = 1\n",
    )]);
    let program = load(&tariff("use org.a.{ residentail, secret }"), None, &context);
    assert_eq!(
        codes(&program),
        vec![(0, "unknown-import"), (0, "private-import")]
    );
    assert_eq!(
        program.diagnostics[0].1.message.english(),
        "`org.a` does not declare `residentail`. Did you mean `residential`?"
    );
    assert_eq!(
        program.diagnostics[0].1.fixes[0].changes[0].insert,
        "residential"
    );
    assert!(program.resolve(ModuleId::ROOT, "secret").is_none());
}

#[test]
fn reports_a_name_imported_twice_and_a_collision_with_a_declaration() {
    let context = context(vec![
        library(
            "org.a",
            "library \"a\"\n\nfn f() -> Int = 1\nfn g() -> Int = 1\n",
        ),
        library("org.b", "library \"b\"\n\nfn f() -> Int = 2\n"),
    ]);
    let program = load(
        &tariff("use org.a.{ f, g }\nuse org.b.{ f }\n\nfn g() -> Int = 3"),
        None,
        &context,
    );
    assert_eq!(
        codes(&program),
        vec![(0, "import-collision"), (0, "duplicate-import")]
    );
    let collision = &program.diagnostics[0].1;
    assert_eq!(collision.related.len(), 1);
    let duplicate = &program.diagnostics[1].1;
    assert_eq!(duplicate.related.len(), 1);
    assert!(matches!(
        program.resolve(ModuleId::ROOT, "g"),
        Some(Resolved::Decl {
            module: ModuleId::ROOT,
            ..
        })
    ));
}

#[test]
fn reports_an_alias_colliding_with_an_import() {
    let context = context(vec![library(
        "org.fuel",
        "library \"f\"\n\nfn x() -> Int = 1\n",
    )]);
    let program = load(&tariff("use std.ltl as fuel\nuse org.fuel"), None, &context);
    assert_eq!(codes(&program), vec![(0, "duplicate-import")]);
}

#[test]
fn reports_duplicate_declarations_with_the_first() {
    let program = load(
        "library \"l\"\n\nfn f() -> Int = 1\nconst f: Int = 2\ntest \"t\" {}\ntest \"t\" {}\n",
        None,
        &Context::default(),
    );
    assert_eq!(
        codes(&program),
        vec![(0, "duplicate-declaration"), (0, "duplicate-declaration")]
    );
    let first = &program.diagnostics[0].1;
    assert_eq!(first.related[0].0.start, 16);
}

#[test]
fn reports_what_a_library_may_not_hold() {
    let context = context(vec![library(
        "org.a",
        "library \"a\"\n\ninput rate: Money\nlet x = 1\ncharge \"x\" = $1\nfn ok() -> Int = 1\n",
    )]);
    let program = load(&tariff("use org.a"), None, &context);
    assert_eq!(
        codes(&program),
        vec![
            (1, "library-item"),
            (1, "library-item"),
            (1, "library-item")
        ]
    );
}

#[test]
fn reports_a_root_of_the_wrong_kind() {
    let program = load(
        "library \"l\"\n",
        Some(FileKind::Tariff),
        &Context::default(),
    );
    assert_eq!(codes(&program), vec![(0, "file-kind")]);
    assert_eq!(
        program.diagnostics[0].1.message.english(),
        "This file must be a `tariff`, not a `library`"
    );
    let program = load(
        "condition \"c\"\n\ntrue\n",
        Some(FileKind::Condition),
        &Context::default(),
    );
    assert!(program.diagnostics.is_empty());
}

#[test]
fn carries_each_module_syntax_diagnostics() {
    let context = context(vec![library("org.a", "library \"a\"\n\nfn f( = 1\n")]);
    let program = load(&tariff("let a = \nuse org.a"), None, &context);
    let modules: Vec<u32> = program.diagnostics.iter().map(|(m, _)| m.0).collect();
    assert!(
        modules.contains(&0) && modules.contains(&1),
        "{:?}",
        codes(&program)
    );
    assert!(program.diagnostics.iter().all(|(_, d)| d.code == "syntax"));
}

#[test]
fn stops_at_the_module_limit() {
    let libraries: Vec<Library> = (0..MAX_MODULES + 2)
        .map(|index| {
            let next = format!("use org.m{}\n", index + 1);
            library(
                &format!("org.m{index}"),
                &format!("library \"m\"\n\n{next}"),
            )
        })
        .collect();
    let program = load(&tariff("use org.m0"), None, &context(libraries));
    assert_eq!(program.modules.len(), MAX_MODULES);
    assert!(codes(&program).iter().any(|(_, code)| *code == "limit"));
}

#[test]
fn stops_at_the_source_limit() {
    let filler = "// padding padding padding padding padding padding padding padding padding\n";
    let big: String = filler.repeat((200 * 1024) / filler.len());
    let libraries: Vec<Library> = (0..6)
        .map(|index| {
            library(
                &format!("org.big{index}"),
                &format!("library \"b\"\n\n{big}"),
            )
        })
        .collect();
    let uses = (0..6)
        .map(|index| format!("use org.big{index}"))
        .collect::<Vec<_>>()
        .join("\n");
    let program = load(&tariff(&uses), None, &context(libraries));
    let loaded: usize = program.modules.iter().map(|m| m.source.len()).sum();
    assert!(loaded <= MAX_PROGRAM_BYTES);
    assert!(codes(&program).iter().any(|(_, code)| *code == "limit"));
}

#[test]
fn ignores_a_use_whose_path_is_not_written_yet() {
    let program = load(&tariff("use std."), None, &Context::default());
    assert_eq!(codes(&program), vec![(0, "syntax")]);
    assert_eq!(program.modules.len(), 1);
}
