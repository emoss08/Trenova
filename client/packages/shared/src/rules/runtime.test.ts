import { describe, expect, it } from "vitest";
import {
  RULES_VERSION_OP,
  RulesRequestError,
  RulesRuntime,
  RulesTrapError,
  isRulesLoaded,
  rulesSync,
} from "./runtime";
import { TARIFF_OPERATIONS } from "./tariff";
import { RULES_WASM_BYTES, RULES_WASM_SHA256 } from "./test-module";

// A module that speaks the ABI and traps on an instance's first call, then answers {"ok":1}:
// a call that traps again proves the runtime threw the trapped instance away.
// (module (memory (export "memory") 1) (global $calls (mut i32) (i32.const 0))
//   (data (i32.const 16) "{\"ok\":1}")
//   (func (export "rules_alloc") (param i32) (result i32) (i32.const 1024))
//   (func (export "rules_free") (param i32 i32))
//   (func (export "rules_call") (param i32 i32) (result i64)
//     (if (i32.eqz (global.get $calls)) (then (global.set $calls (i32.const 1)) (unreachable)))
//     (i64.or (i64.shl (i64.const 16) (i64.const 32)) (i64.const 8))))
const TRAPS_ON_FIRST_CALL = new Uint8Array([
  0, 97, 115, 109, 1, 0, 0, 0, 1, 17, 3, 96, 1, 127, 1, 127, 96, 2, 127, 127, 0, 96, 2, 127, 127, 1,
  126, 3, 4, 3, 0, 1, 2, 5, 3, 1, 0, 1, 6, 6, 1, 127, 1, 65, 0, 11, 7, 50, 4, 6, 109, 101, 109, 111,
  114, 121, 2, 0, 11, 114, 117, 108, 101, 115, 95, 97, 108, 108, 111, 99, 0, 0, 10, 114, 117, 108,
  101, 115, 95, 102, 114, 101, 101, 0, 1, 10, 114, 117, 108, 101, 115, 95, 99, 97, 108, 108, 0, 2,
  10, 32, 3, 5, 0, 65, 128, 8, 11, 2, 0, 11, 21, 0, 35, 0, 69, 4, 64, 65, 1, 36, 0, 0, 11, 66, 16,
  66, 32, 134, 66, 8, 132, 11, 11, 14, 1, 0, 65, 16, 11, 8, 123, 34, 111, 107, 34, 58, 49, 125,
]);

function captured(run: () => unknown): unknown {
  try {
    run();
  } catch (error) {
    return error;
  }
  throw new Error("expected the call to throw");
}

describe("rules.wasm", () => {
  it("matches its .sha256 sidecar", async () => {
    const digest = new Uint8Array(await crypto.subtle.digest("SHA-256", RULES_WASM_BYTES));
    const sum = Array.from(digest, (byte) => byte.toString(16).padStart(2, "0")).join("");
    expect(RULES_WASM_SHA256).toBe(`${sum}  rules.wasm\n`);
  });
});

describe("RulesRuntime", () => {
  it("is loaded by the test setup and answers rules.version", () => {
    expect(isRulesLoaded()).toBe(true);

    const version = rulesSync().version();
    expect(version.version).toMatch(/^\d+\.\d+\.\d+$/);
    expect(version.operations).toContain(RULES_VERSION_OP);
    expect(version.operations).toEqual(expect.arrayContaining([...TARIFF_OPERATIONS]));
    expect(version.operations.filter((op) => op.startsWith("formula."))).toEqual([]);
  });

  it("returns the output as JSON text from callRaw", () => {
    const raw = rulesSync().callRaw(RULES_VERSION_OP, "{}");
    expect(raw.startsWith('{"version":"')).toBe(true);
    expect(JSON.parse(raw)).toEqual(rulesSync().version());
  });

  it("throws a request error with the module's code and detail for an unknown operation", () => {
    const error = captured(() => rulesSync().call("nope.never", { a: 1 }));
    expect(error).toBeInstanceOf(RulesRequestError);
    expect(error).toMatchObject({
      op: "nope.never",
      code: "unknown_op",
      detail: 'no operation is named "nope.never"',
    });
  });

  it("throws a malformed request error for input that is not JSON", () => {
    const error = captured(() => rulesSync().callRaw(RULES_VERSION_OP, '{"a":'));
    expect(error).toBeInstanceOf(RulesRequestError);
    expect(error).toMatchObject({ op: RULES_VERSION_OP, code: "malformed" });
    expect(rulesSync().version().operations).toContain(RULES_VERSION_OP);
  });

  it("carries text through UTF-8 both ways", () => {
    const error = captured(() => rulesSync().call("nope.ünïcødé"));
    expect(error).toMatchObject({ detail: 'no operation is named "nope.ünïcødé"' });
  });

  it("replaces an instance that trapped", async () => {
    const { module, instance } = await WebAssembly.instantiate(TRAPS_ON_FIRST_CALL, {});
    const runtime = new RulesRuntime(module, instance);

    const first = captured(() => runtime.call("any.op"));
    expect(first).toBeInstanceOf(RulesTrapError);
    expect(first).toMatchObject({ op: "any.op" });
    expect((first as RulesTrapError).cause).toBeInstanceOf(WebAssembly.RuntimeError);

    expect(captured(() => runtime.call("any.op"))).toBeInstanceOf(RulesTrapError);
  });

  it("keeps a healthy instance between calls", async () => {
    const { module, instance } = await WebAssembly.instantiate(TRAPS_ON_FIRST_CALL, {});
    const runtime = new RulesRuntime(module, instance);
    expect(() => (instance.exports.rules_call as (a: number, b: number) => bigint)(0, 0)).toThrow(
      WebAssembly.RuntimeError,
    );

    expect(runtime.call("any.op")).toBe(1);
    expect(runtime.call("any.op")).toBe(1);
  });

  it("refuses a module without the rules exports", async () => {
    const empty = new Uint8Array([0, 97, 115, 109, 1, 0, 0, 0]);
    const { module, instance } = await WebAssembly.instantiate(empty, {});
    expect(() => new RulesRuntime(module, instance)).toThrow(TypeError);
  });
});
