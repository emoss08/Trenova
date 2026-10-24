//! Real scripts through the real SDK, answered by a fake host.

use serde_json::{Value, json};
use trenova_scripts::{Done, EndStatus, HostFunction, Limit, Progress, Start, Step, resume, start};

struct Run {
    done: Done,
    calls: Vec<(HostFunction, Value)>,
}

fn limits() -> Value {
    json!({"outputChars": 8000, "memoryBytes": 256 << 20, "cpuMs": 5000, "recursion": 200, "calls": 60})
}

fn index() -> Value {
    json!([
        {"name": "shipment_list", "description": "List shipments.", "parameters": {
            "type": "object",
            "properties": {"status": {"type": "string", "description": "The status to match."}},
            "required": ["status"]
        }},
        {"name": "customer_get", "description": "Get one customer."},
        {"name": "receivable_get_bank_receipt", "description": "One bank receipt."},
        {"name": "shipment_add_note", "description": "Add a note.", "writes": true}
    ])
}

fn request(code: &str, overrides: &Value) -> Start {
    let mut body = json!({
        "code": code,
        "index": index().to_string(),
        "inputs": json!({"region": "West"}).to_string(),
        "clock": {"unixSeconds": 1_793_000_000, "offsetSeconds": -18_000, "zone": "CDT"},
        "limits": limits(),
    });
    if let (Some(body), Some(overrides)) = (body.as_object_mut(), overrides.as_object()) {
        for (key, value) in overrides {
            body.insert(key.clone(), value.clone());
        }
    }
    serde_json::from_value(body).expect("a valid start request")
}

fn host(function: HostFunction, payload: &Value) -> Value {
    match function {
        HostFunction::Read => match payload["tool"].as_str() {
            Some("shipment_list") => json!({"status": "ok", "data": [
                {"pro": "P1", "customerId": "c1", "amount": 120},
                {"pro": "P2", "customerId": "c2", "amount": 80},
                {"pro": "P3", "customerId": "c1", "amount": 40}
            ]}),
            Some("receivable_get_bank_receipt") => {
                json!({"status": "refused", "message": "you may not read bank receipts"})
            }
            _ => json!({"status": "ok", "data": {"echo": payload["args"]}}),
        },
        HostFunction::Gather => {
            let calls = payload.as_array().cloned().unwrap_or_default();
            Value::Array(
                calls
                    .iter()
                    .map(|call| {
                        let id = call["args"]["id"].as_str().unwrap_or_default();
                        json!({"status": "ok", "data": {"id": id, "name": format!("Customer {id}")}})
                    })
                    .collect(),
            )
        }
        HostFunction::Result => json!({"status": "ok"}),
    }
}

fn run(code: &str) -> Run {
    run_with(code, &json!({}))
}

fn run_with(code: &str, overrides: &Value) -> Run {
    let mut calls = Vec::new();
    let mut progress = start(&request(code, overrides));
    loop {
        match progress {
            Progress::Done(Step::Done(done)) => return Run { done, calls },
            Progress::Paused(session, Step::Call { function, payload }) => {
                let payload: Value = serde_json::from_str(&payload).expect("JSON from the SDK");
                let reply = host(function, &payload);
                calls.push((function, payload));
                progress = resume(*session, reply.to_string());
            }
            other => panic!("an impossible step: {:?}", other.step()),
        }
    }
}

fn ok(run: &Run) {
    assert_eq!(
        run.done.status,
        EndStatus::Ok,
        "{:?}\n{}",
        run.done.error,
        run.done.output
    );
}

#[test]
fn reads_join_and_print() {
    let run = run(r#"
loads = trenova.shipment.list(status="Delayed")
customers = trenova.gather(*[trenova.customer.get.defer(id=l["customerId"]) for l in loads])
totals = {}
for load, customer in zip(loads, customers):
    totals[customer["name"]] = totals.get(customer["name"], 0) + load["amount"]
for name in sorted(totals):
    print(name, totals[name])
"#);
    ok(&run);
    assert_eq!(run.done.output, "Customer c1 160\nCustomer c2 80\n");
    assert_eq!(run.calls.len(), 2);
    assert_eq!(run.calls[0].0, HostFunction::Read);
    assert_eq!(
        run.calls[0].1,
        json!({"tool": "shipment_list", "args": {"status": "Delayed"}})
    );
    assert_eq!(run.calls[1].0, HostFunction::Gather);
    assert_eq!(run.calls[1].1.as_array().map(Vec::len), Some(3));
}

#[test]
fn a_refused_read_raises_refused_with_the_tool_status_and_message() {
    let run = run(r#"
try:
    trenova.receivable.get_bank_receipt(id="r1")
except trenova.Refused as refused:
    print(refused)
"#);
    ok(&run);
    assert_eq!(
        run.done.output,
        "receivable_get_bank_receipt: refused: you may not read bank receipts\n"
    );
}

#[test]
fn an_uncaught_refusal_ends_the_script_with_its_traceback() {
    let run = run("x = 1\ntrenova.receivable.get_bank_receipt(id='r1')\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    let error = run.done.error.expect("the refusal");
    assert_eq!(error.name, "OSError");
    assert!(
        error.traceback.contains("File \"script.py\", line 2"),
        "{}",
        error.traceback
    );
    assert!(
        !error.traceback.contains("python-input"),
        "{}",
        error.traceback
    );
}

#[test]
fn gather_can_return_refusals_instead_of_raising() {
    let run = run(r#"
answers = trenova.gather(trenova.customer.get.defer(id="c1"), return_exceptions=True)
print(len(answers), answers[0]["name"])
"#);
    ok(&run);
    assert_eq!(run.done.output, "1 Customer c1\n");
}

#[test]
fn gather_takes_only_deferred_calls() {
    let run = run("trenova.gather(1)\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    assert_eq!(run.done.error.map(|e| e.name), Some("TypeError".to_owned()));
    assert!(run.calls.is_empty());
}

#[test]
fn result_is_sent_once() {
    let run = run("trenova.result({'total': 3})\ntrenova.result(1)\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    assert_eq!(run.calls, vec![(HostFunction::Result, json!({"total": 3}))]);
    assert_eq!(
        run.done.error.map(|e| e.value),
        Some("result() may be called once".to_owned())
    );
}

#[test]
fn inputs_tools_help_and_reprs() {
    let run = run(r#"
print(trenova.inputs["region"], inputs["region"])
print(trenova.tools())
print(trenova.shipment.list)
print(trenova.shipment.add_note)
help(trenova.shipment.list)
"#);
    ok(&run);
    assert!(
        run.done.output.starts_with("West West\n"),
        "{}",
        run.done.output
    );
    assert!(run.done.output.contains("'receivable_get_bank_receipt'"));
    assert!(run.done.output.contains("<trenova read shipment_list>"));
    assert!(
        run.done
            .output
            .contains("<trenova change shipment_add_note>")
    );
    assert!(
        run.done
            .output
            .contains("status (required): The status to match.")
    );
}

#[test]
fn a_read_the_turn_does_not_hold_is_an_attribute_error() {
    let run = run("trenova.shipment.cancel(id='s1')\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    assert_eq!(
        run.done.error.map(|e| e.name),
        Some("AttributeError".to_owned())
    );
    assert!(run.calls.is_empty());
}

#[test]
fn the_clock_is_the_hosts_in_the_organizations_zone() {
    let run = run("import datetime\nprint(datetime.date.today(), datetime.datetime.now().hour)\n");
    ok(&run);
    assert_eq!(run.done.output, "2026-10-26 2\n");
}

#[test]
fn nothing_outside_is_reachable() {
    for code in [
        "open('/etc/passwd').read()",
        "import os\nos.getenv('HOME')",
        "import pathlib\npathlib.Path('/etc/passwd').read_text()",
    ] {
        let run = run(code);
        assert_eq!(run.done.status, EndStatus::Raised, "{code}");
        let error = run.done.error.expect("an error");
        assert_eq!(error.name, "OSError", "{code}: {error:?}");
        assert!(
            error.value.contains("cannot reach files"),
            "{code}: {error:?}"
        );
        assert!(run.calls.is_empty());
    }
}

#[test]
fn a_module_monty_does_not_ship_cannot_be_imported() {
    for code in ["import socket", "import subprocess", "import requests"] {
        let run = run(code);
        assert_eq!(run.done.status, EndStatus::Raised, "{code}");
        assert_eq!(
            run.done.error.map(|e| e.name),
            Some("ModuleNotFoundError".to_owned()),
            "{code}"
        );
    }
}

#[test]
fn an_unknown_function_is_a_name_error_not_a_host_call() {
    let run = run("fetch_everything()\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    assert_eq!(run.done.error.map(|e| e.name), Some("NameError".to_owned()));
    assert!(run.calls.is_empty());
}

#[test]
fn host_functions_take_one_string() {
    let run = run("_trenova_read(1, 2)\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    assert_eq!(run.done.error.map(|e| e.name), Some("TypeError".to_owned()));
    assert!(run.calls.is_empty());
}

#[test]
fn a_spinning_script_hits_the_cpu_limit() {
    let mut limits = limits();
    limits["cpuMs"] = json!(50);
    let run = run_with("while True:\n    pass\n", &json!({"limits": limits}));
    assert_eq!(run.done.status, EndStatus::Limit);
    assert_eq!(run.done.limit, Some(Limit::Cpu));
}

#[test]
fn the_cpu_limit_cannot_be_caught() {
    let mut limits = limits();
    limits["cpuMs"] = json!(50);
    let code = "try:\n    while True:\n        pass\nexcept BaseException:\n    print('caught')\n";
    let run = run_with(code, &json!({"limits": limits}));
    assert_eq!(run.done.limit, Some(Limit::Cpu));
    assert_eq!(run.done.output, "");
}

#[test]
fn host_calls_stop_at_the_limit() {
    let mut limits = limits();
    limits["calls"] = json!(3);
    let run = run_with(
        "for i in range(10):\n    trenova.customer.get(id=str(i))\n",
        &json!({"limits": limits}),
    );
    assert_eq!(run.done.status, EndStatus::Limit);
    assert_eq!(run.done.limit, Some(Limit::Calls));
    assert_eq!(run.calls.len(), 3);
}

#[test]
fn printed_output_keeps_its_end() {
    let mut limits = limits();
    limits["outputChars"] = json!(10);
    let run = run_with(
        "for i in range(1000):\n    print(i)\n",
        &json!({"limits": limits}),
    );
    ok(&run);
    assert!(run.done.output_truncated);
    assert!(
        run.done.output.ends_with("998\n999\n"),
        "{:?}",
        run.done.output
    );
    assert!(run.done.output.chars().count() <= 10);
}

#[test]
fn a_syntax_error_is_raised_with_its_line() {
    let run = run("x = 1\ny = (\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    let error = run.done.error.expect("the error");
    assert_eq!(error.name, "SyntaxError");
}

#[test]
fn deep_recursion_is_a_recursion_error() {
    let run = run("def f(n):\n    return f(n + 1)\nf(0)\n");
    assert_eq!(run.done.status, EndStatus::Raised);
    assert_eq!(
        run.done.error.map(|e| e.name),
        Some("RecursionError".to_owned())
    );
}

#[test]
fn the_sdk_names_no_tool() {
    for name in [
        "shipment_",
        "invoice_",
        "customer_",
        "report_",
        "_list(",
        "_get(",
    ] {
        assert!(
            !trenova_scripts::session::SDK_SOURCE.contains(name),
            "the SDK is generic; reads come from the index, but it names {name}"
        );
    }
}
