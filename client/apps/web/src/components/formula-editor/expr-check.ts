import { translate } from "@trenova/shared/i18n/runtime";
import {
  isKnownFunction,
  isOperatorWord,
  type DisabledFunction,
  type FunctionShape,
  type KnownIdentifiers,
  type VariableDoc,
} from "./known-identifiers";
import {
  memberPath,
  parseExpression,
  type LetEntry,
  type Node,
  type Program,
  type Range,
  type SyntaxDiagnostic,
} from "./expr-parser";

/*
 * The language service's checks over a parsed expression: every name must be
 * a shipment field, one of the template's inputs or a `let` of its own; every
 * call a known function with the right number of arguments; every operator
 * applied to values of the kind it takes; a field that can be empty guarded;
 * and the whole formula coming to an amount. Each finding carries the span it
 * is about and, where one exists, the one-click fix that resolves it.
 */

export type ExprType =
  | "Number"
  | "String"
  | "Boolean"
  | "Date"
  | "Array"
  | "Object"
  | "Nil"
  | "Any";

export type DiagnosticSeverity = "error" | "warning" | "info";

export type DiagnosticCode =
  | "syntax"
  | "unknown-variable"
  | "unknown-field"
  | "unknown-function"
  | "disabled-function"
  | "arity"
  | "type"
  | "nullable"
  | "let-unused"
  | "let-duplicate"
  | "let-before-definition"
  | "let-shadows"
  | "let-last"
  | "result-type"
  | "discarded"
  | "divide-by-zero"
  | "engine";

/** A change the editor can make to resolve a finding, or an input to declare. */
export type ExprFix =
  | { kind: "replace"; label: string; changes: { from: number; to: number; insert: string }[] }
  | { kind: "declare"; label: string; name: string; type: "Number" | "String" | "Boolean" };

export type ExprDiagnostic = Range & {
  severity: DiagnosticSeverity;
  code: DiagnosticCode;
  message: string;
  /** The identifier the finding is about, when it is about one. */
  name?: string;
  fixes: ExprFix[];
};

export type LetBinding = Range & {
  name: string;
  nameRange: Range;
  type: ExprType;
  /** The value is a whole number to the engine, so `%` takes it. */
  whole: boolean;
  reads: number;
};

export type ExprAnalysis = {
  program: Program;
  diagnostics: ExprDiagnostic[];
  /** Schema variables the expression reads, by the path it reads them as. */
  schemaReads: Set<string>;
  /** The template's own inputs the expression reads. */
  inputReads: Set<string>;
  functionCalls: Set<string>;
  /** Every name read, inputs first, in the order they first appear. */
  reads: string[];
  bindings: Map<string, LetBinding>;
  resultType: ExprType;
  /** The engine's schema has not loaded yet, so names were not checked. */
  pending: boolean;
};

const BOOLEAN_PREFIX = /^(is|has|requires|uses|needs|allows|can)[A-Z_]/;

/** The type an input added from an unknown name starts as: a flag for is/has/requires names. */
export function inferInputType(name: string): "Number" | "String" | "Boolean" {
  return BOOLEAN_PREFIX.test(name) ? "Boolean" : "Number";
}

export function normalizeType(raw: string | undefined): ExprType {
  switch ((raw ?? "").toLowerCase()) {
    case "number":
    case "integer":
    case "int":
    case "decimal":
    case "float":
    case "float64":
      return "Number";
    case "string":
    case "text":
      return "String";
    case "boolean":
    case "bool":
      return "Boolean";
    case "date":
    case "datetime":
    case "time":
    case "timestamp":
      return "Date";
    case "array":
    case "list":
      return "Array";
    case "object":
    case "map":
      return "Object";
    default:
      return (raw ?? "").startsWith("[]") ? "Array" : "Any";
  }
}

/** What a value of the type is called in a sentence. */
export function typeWords(type: ExprType): string {
  switch (type) {
    case "Number":
      return translate("a number");
    case "String":
      return translate("text");
    case "Boolean":
      return translate("true/false");
    case "Date":
      return translate("a date");
    case "Array":
      return translate("a list");
    case "Object":
      return translate("a record");
    case "Nil":
      return translate("empty");
    default:
      return translate("a value");
  }
}

function levenshtein(a: string, b: string): number {
  if (Math.abs(a.length - b.length) > 3) return Number.MAX_SAFE_INTEGER;
  const rows = a.length + 1;
  const cols = b.length + 1;
  const distances = new Array<number>(rows * cols).fill(0);
  for (let i = 0; i < rows; i++) distances[i * cols] = i;
  for (let j = 0; j < cols; j++) distances[j] = j;
  for (let i = 1; i < rows; i++) {
    for (let j = 1; j < cols; j++) {
      const cost = a[i - 1] === b[j - 1] ? 0 : 1;
      distances[i * cols + j] = Math.min(
        distances[(i - 1) * cols + j] + 1,
        distances[i * cols + j - 1] + 1,
        distances[(i - 1) * cols + j - 1] + cost,
      );
    }
  }
  return distances[rows * cols - 1];
}

/** The closest known name, when one is within a couple of edits or differs only in case. */
export function nearestKnown(name: string, candidates: Iterable<string>): string | null {
  let best: string | null = null;
  let bestDistance = 3;
  const lower = name.toLowerCase();
  for (const candidate of candidates) {
    if (candidate === name) continue;
    const distance =
      candidate.toLowerCase() === lower ? 0 : levenshtein(lower, candidate.toLowerCase());
    if (distance < bestDistance) {
      best = candidate;
      bestDistance = distance;
    }
  }
  return best;
}

const quote = (name: string) => `\`${name}\``;

type Scope = {
  known: KnownIdentifiers;
  variables: Map<string, VariableDoc>;
  roots: Map<string, VariableDoc[]>;
  bindings: Map<string, LetBinding>;
  diagnostics: ExprDiagnostic[];
  schemaReads: Set<string>;
  inputReads: Set<string>;
  functionCalls: Set<string>;
  order: string[];
  /** Spans that sit under a guard (`??`, coalesce, a nil test), by the name they guard. */
  guarded: Set<Node>;
  /** Fields that can be empty and were flagged, so one is reported once. */
  flaggedNullable: Set<string>;
  /** Names of `let` bindings as they are defined, in order, for before-definition checks. */
  defined: Set<string>;
  /** `#` is in scope inside a predicate. */
  pointerDepth: number;
  /** Names the enclosing ternary tested for nil, so reads of them in its branches are guarded. */
  guardedByTest?: Set<string>;
  /** `#acc` is in scope inside reduce(). */
  reduceDepth: number;
  /** The expression's text, for fixes that rewrite a span from its own parts. */
  source: string;
  /** The type worked out for each node, so a chained comparison can read its middle operand's. */
  types: Map<Node, ExprType>;
};

function report(
  scope: Scope,
  range: Range,
  severity: DiagnosticSeverity,
  code: DiagnosticCode,
  message: string,
  extra: { name?: string; fixes?: ExprFix[] } = {},
) {
  scope.diagnostics.push({
    from: range.from,
    to: range.to,
    severity,
    code,
    message,
    name: extra.name,
    fixes: extra.fixes ?? [],
  });
}

function join(a: ExprType, b: ExprType): ExprType {
  if (a === b) return a;
  if (a === "Nil" || a === "Any") return b;
  if (b === "Nil" || b === "Any") return a;
  return "Any";
}

function isNumeric(type: ExprType): boolean {
  return type === "Number" || type === "Any";
}

function describeValue(node: Node, type: ExprType): string {
  const name = memberPath(node);
  if (name) return translate("{0} is {1}", quote(name), typeWords(type));
  if (node.kind === "string") return translate("'{0}' is text", node.value);
  if (node.kind === "bool") return translate("'{0}' is true/false", String(node.value));
  return translate("this part is {0}", typeWords(type));
}

/** The guard a value of this type reads naturally with: 0 for a number, "" for text, false for a flag. */
function guardFallback(type: ExprType): string {
  switch (type) {
    case "String":
      return '""';
    case "Boolean":
      return "false";
    default:
      return "0";
  }
}

function markGuarded(scope: Scope, node: Node) {
  scope.guarded.add(node);
  if (node.kind === "member") markGuarded(scope, node.object);
}

function checkNullable(scope: Scope, node: Node, doc: VariableDoc, path: string) {
  if (!doc.nullable || scope.guarded.has(node) || scope.flaggedNullable.has(path)) return;
  scope.flaggedNullable.add(path);
  const fallback = guardFallback(normalizeType(doc.type));
  report(
    scope,
    node,
    "warning",
    "nullable",
    translate(
      "{0} is empty on some shipments, and the formula would fail to rate them. Give it a fallback: ({0} ?? {1})",
      quote(path),
      fallback,
    ),
    {
      name: path,
      fixes: [
        {
          kind: "replace",
          label: translate("Use ({0} ?? {1})", path, fallback),
          changes: [{ from: node.from, to: node.to, insert: `(${path} ?? ${fallback})` }],
        },
      ],
    },
  );
}

function typeOfIdent(scope: Scope, node: Node & { kind: "ident" }): ExprType {
  const { name } = node;
  const binding = scope.bindings.get(name);
  if (binding) {
    if (!scope.defined.has(name)) {
      report(
        scope,
        node,
        "error",
        "let-before-definition",
        translate(
          "{0} is defined further down; a 'let' must come before the lines that read it",
          quote(name),
        ),
        { name },
      );
    }
    binding.reads += 1;
    return binding.type;
  }
  const doc = scope.variables.get(name);
  if (doc) {
    recordRead(scope, name, doc.custom === true);
    checkNullable(scope, node, doc, name);
    return normalizeType(doc.type);
  }
  if (scope.roots.has(name)) {
    recordRead(scope, name, false);
    return "Object";
  }
  if (isOperatorWord(scope.known, name)) {
    report(
      scope,
      node,
      "error",
      "syntax",
      translate("'{0}' is an operator: write text {0} \"value\"", name),
    );
    return "Any";
  }
  const suggestion = nearestKnown(name, [...scope.known.variablePaths, ...scope.bindings.keys()]);
  const inferred = inferInputType(name);
  const fixes: ExprFix[] = [];
  if (suggestion) {
    fixes.push({
      kind: "replace",
      label: translate("Change to {0}", suggestion),
      changes: [{ from: node.from, to: node.to, insert: suggestion }],
    });
  }
  fixes.push({
    kind: "declare",
    label: translate("Declare {0} as a {1} input", name, translate(inferred)),
    name,
    type: inferred,
  });
  report(
    scope,
    node,
    "error",
    "unknown-variable",
    suggestion
      ? translate(
          "{0} isn't a shipment field or one of this template's inputs. Did you mean {1}?",
          quote(name),
          quote(suggestion),
        )
      : translate(
          "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference.",
          quote(name),
        ),
    { name, fixes },
  );
  return "Any";
}

function recordRead(scope: Scope, name: string, custom: boolean) {
  const set = custom ? scope.inputReads : scope.schemaReads;
  if (!set.has(name)) {
    set.add(name);
    scope.order.push(name);
  }
}

function typeOfMember(scope: Scope, node: Node & { kind: "member" }): ExprType {
  const path = memberPath(node);
  const rootName = memberPath(rootOf(node));
  if (node.object.kind === "pointer") {
    if (scope.pointerDepth === 0) {
      report(
        scope,
        node,
        "error",
        "syntax",
        translate(
          "{0} reads a field of the current item, so it only works inside filter(), map() and the like",
          quote(`.${node.property}`),
        ),
      );
    }
    return "Any";
  }
  if (path && scope.variables.has(path)) {
    const doc = scope.variables.get(path)!;
    recordRead(scope, path, false);
    checkNullable(scope, node, doc, path);
    return normalizeType(doc.type);
  }
  if (path && rootName && scope.roots.has(rootName) && !scope.bindings.has(rootName)) {
    const siblings = scope.roots.get(rootName)!;
    const exact = siblings.find((doc) => path.startsWith(`${doc.name}.`));
    if (exact) {
      recordRead(scope, exact.name, false);
      return "Any";
    }
    const candidates = siblings.map((doc) => doc.name);
    const suggestion = nearestKnown(path, candidates);
    report(
      scope,
      node.propertyRange,
      "error",
      "unknown-field",
      suggestion
        ? translate(
            "{0} has no field {1}. Did you mean {2}?",
            quote(rootName),
            quote(node.property),
            quote(suggestion),
          )
        : translate("{0} has no field {1}", quote(rootName), quote(node.property)),
      {
        name: path,
        fixes: suggestion
          ? [
              {
                kind: "replace",
                label: translate("Change to {0}", suggestion),
                changes: [{ from: node.from, to: node.to, insert: suggestion }],
              },
            ]
          : [],
      },
    );
    recordRead(scope, rootName, false);
    return "Any";
  }
  const objectType = typeOf(scope, node.object);
  if (objectType === "Number" || objectType === "String" || objectType === "Boolean") {
    report(
      scope,
      node.propertyRange,
      "error",
      "type",
      translate(
        "{0}; it has no field {1}",
        describeValue(node.object, objectType),
        quote(node.property),
      ),
    );
  }
  return "Any";
}

function rootOf(node: Node): Node {
  return node.kind === "member" ? rootOf(node.object) : node;
}

const RATING_TIME = "ratingTime";

function reportDisabledFunction(
  scope: Scope,
  node: Node & { kind: "call" },
  disabled: DisabledFunction,
) {
  const { name, replacement } = disabled;
  switch (disabled.reason) {
    case "nondeterministic":
      report(
        scope,
        node.callee,
        "error",
        "disabled-function",
        translate(
          "{0} isn't available in formulas: a formula must price a shipment the same way every time it runs. Use {1}, the date the contract terms are read at",
          quote(`${name}()`),
          quote(replacement ?? RATING_TIME),
        ),
        {
          name,
          fixes:
            node.args.length === 0 && replacement
              ? [
                  {
                    kind: "replace",
                    label: translate("Change to {0}", replacement),
                    changes: [{ from: node.from, to: node.to, insert: replacement }],
                  },
                ]
              : [],
        },
      );
      return;
    case "textBuilding":
      report(
        scope,
        node.callee,
        "error",
        "disabled-function",
        translate(
          "{0} isn't available in formulas: formulas compare codes and do not build text",
          quote(name),
        ),
        { name },
      );
      return;
    default:
      report(
        scope,
        node.callee,
        "error",
        "disabled-function",
        translate(
          "{0} isn't available in formulas: formulas read the shipment's fields directly and do not convert them to or from JSON",
          quote(name),
        ),
        { name },
      );
  }
}

/** Whether an argument reads the current item (`#`, `.field`) outside a call that binds one. */
function readsItem(scope: Scope, node: Node): boolean {
  if (node.kind === "pointer" || node.kind === "predicate") return true;
  if (node.kind === "call") {
    const name = memberPath(node.callee);
    if (name !== null && scope.known.shapes.get(name)?.predicate) {
      return node.args.length > 0 && readsItem(scope, node.args[0]);
    }
  }
  return (
    childrenFor(node).some((child) => readsItem(scope, child)) ||
    (node.kind === "member" && readsItem(scope, node.object))
  );
}

function arityText(shape: FunctionShape): string {
  return shape.maxArgs === Number.POSITIVE_INFINITY
    ? translate("at least {0, plural, one {# argument} other {# arguments}}", shape.minArgs)
    : shape.minArgs === shape.maxArgs
      ? translate("{0, plural, one {# argument} other {# arguments}}", shape.minArgs)
      : translate("{0} to {1} arguments", shape.minArgs, shape.maxArgs);
}

function typeOfCall(scope: Scope, node: Node & { kind: "call" }): ExprType {
  const name = memberPath(node.callee);
  if (name === null || node.callee.kind !== "ident") {
    if (node.callee.kind === "member") typeOf(scope, node.callee.object);
    for (const arg of node.args) typeOf(scope, arg);
    return "Any";
  }
  const shape = scope.known.shapes.get(name);
  const disabled = scope.known.disabled.get(name);
  if (!shape && disabled && !scope.variables.has(name) && !scope.bindings.has(name)) {
    reportDisabledFunction(scope, node, disabled);
    for (const arg of node.args) typeOf(scope, arg);
    return "Any";
  }
  if (!shape || !isKnownFunction(scope.known, name)) {
    const suggestion = nearestKnown(name, scope.known.functionNames);
    const isVariable = scope.variables.has(name) || scope.bindings.has(name);
    report(
      scope,
      node.callee,
      "error",
      "unknown-function",
      isVariable
        ? translate("{0} is a value, not a function; it can't be called with ( )", quote(name))
        : suggestion
          ? translate(
              "{0} isn't a formula function. Did you mean {1}?",
              quote(name),
              quote(suggestion),
            )
          : translate("{0} isn't a formula function; pick one from the reference", quote(name)),
      {
        name,
        fixes: suggestion
          ? [
              {
                kind: "replace",
                label: translate("Change to {0}", suggestion),
                changes: [{ from: node.callee.from, to: node.callee.to, insert: suggestion }],
              },
            ]
          : [],
      },
    );
    for (const arg of node.args) typeOf(scope, arg);
    return "Any";
  }
  scope.functionCalls.add(name);
  if (name === "coalesce") {
    for (const arg of node.args) markGuarded(scope, arg);
  }

  const predicateArg = shape.predicate ? 1 : -1;
  const argTypes = node.args.map((arg, index) => {
    if (index === predicateArg) {
      scope.pointerDepth += 1;
      if (name === "reduce") scope.reduceDepth += 1;
      const type = typeOf(scope, arg);
      if (name === "reduce") scope.reduceDepth -= 1;
      scope.pointerDepth -= 1;
      return type;
    }
    if (!shape.predicate && index > 0 && readsItem(scope, arg)) {
      reportPredicateOnAFunction(scope, node, name, arg);
      scope.pointerDepth += 1;
      const type = typeOf(scope, arg);
      scope.pointerDepth -= 1;
      return type;
    }
    return typeOf(scope, arg);
  });

  if (node.args.length < shape.minArgs || node.args.length > shape.maxArgs) {
    report(
      scope,
      node.parens,
      "error",
      "arity",
      translate(
        "{0} takes {1}; {2, plural, one {# was} other {# were}} given",
        shape.signature,
        arityText(shape),
        node.args.length,
      ),
      { name },
    );
  }
  if (shape.numericArgs) {
    node.args.forEach((arg, index) => {
      const type = argTypes[index];
      if (type !== undefined && !isNumeric(type) && type !== "Nil") {
        report(
          scope,
          arg,
          "error",
          "type",
          translate("{0} takes numbers, but {1}", name, describeValue(arg, type)),
        );
      }
    });
  }
  if (name === "coalesce") {
    return argTypes.reduce<ExprType>((acc, type) => join(acc, type), "Nil");
  }
  if (scope.known.functions.some((fn) => fn.name === name && fn.category === "rateTable")) {
    const first = node.args[0];
    if (first && first.kind !== "string") {
      report(
        scope,
        first,
        "warning",
        "type",
        translate('{0} names its rate table in quotes: {0}("zone_rates", …)', name),
      );
    }
  }
  return normalizeType(shape.returns);
}

/** A predicate given to a function that takes values, such as `sum(stops, .weight)`. */
function reportPredicateOnAFunction(
  scope: Scope,
  node: Node & { kind: "call" },
  name: string,
  arg: Node,
) {
  const list = node.args[0];
  const source = scope.source;
  const fixes: ExprFix[] =
    node.args.length === 2 && list
      ? [
          {
            kind: "replace",
            label: translate("Use {0}(map(…))", name),
            changes: [
              {
                from: node.from,
                to: node.to,
                // i18n-ignore: formula source code, not text
                insert: `${name}(map(${source.slice(list.from, list.to)}, ${source.slice(arg.from, arg.to)}))`,
              },
            ],
          },
        ]
      : [];
  report(
    scope,
    arg,
    "error",
    "type",
    translate(
      "{0} adds up values and does not take a predicate; map the list first: {0}(map(list, .field))",
      name,
    ),
    { name, fixes },
  );
}

/** Whether a value is a whole number to the engine, which is what `%` takes. */
function isWhole(scope: Scope, node: Node): boolean {
  switch (node.kind) {
    case "number":
      return node.whole;
    case "ident": {
      const binding = scope.bindings.get(node.name);
      if (binding) return binding.whole;
      const doc = scope.variables.get(node.name);
      return doc !== undefined && isWholeType(doc.type);
    }
    case "member": {
      const path = memberPath(node);
      const doc = path ? scope.variables.get(path) : undefined;
      return doc !== undefined && isWholeType(doc.type);
    }
    case "call": {
      const name = memberPath(node.callee);
      return name !== null && scope.known.shapes.get(name)?.returns === "Integer";
    }
    case "unary":
      return (node.op === "-" || node.op === "+") && isWhole(scope, node.operand);
    case "binary":
      return (
        (node.op === "+" || node.op === "-" || node.op === "*" || node.op === "%") &&
        isWhole(scope, node.left) &&
        isWhole(scope, node.right)
      );
    case "ternary":
    case "if":
      return isWhole(scope, node.consequent) && isWhole(scope, node.alternate);
    case "sequence":
      return isWhole(scope, node.body);
    default:
      return false;
  }
}

function isWholeType(type: string): boolean {
  const lower = type.toLowerCase();
  return lower === "integer" || lower === "int";
}

/** A relation's field (`origin.zip`): the engine reads it untyped, so it compiles against anything. */
function isUntypedToTheEngine(scope: Scope, node: Node): boolean {
  if (node.kind !== "member") return false;
  const root = memberPath(node.object);
  return root !== null && scope.roots.has(root.split(".")[0]);
}

const COMPARISONS = new Set(["<", ">", "<=", ">="]);

function checkArithmetic(
  scope: Scope,
  node: Node & { kind: "binary" },
  left: ExprType,
  right: ExprType,
) {
  for (const [operand, type] of [
    [node.left, left],
    [node.right, right],
  ] as const) {
    if (isNumeric(type) || type === "Nil") continue;
    const name = memberPath(operand);
    let hint = "";
    const fixes: ExprFix[] = [];
    if (type === "Boolean") {
      hint = name
        ? translate(" Write ({0} ? amount : 0) to charge when it is true.", name)
        : translate(" Write (test ? amount : 0) to charge when a test holds.");
      if (name) {
        fixes.push({
          kind: "replace",
          label: translate("Change to ({0} ? 1 : 0)", name),
          changes: [{ from: operand.from, to: operand.to, insert: `(${name} ? 1 : 0)` }],
        });
      }
    } else if (type === "String") {
      hint = translate(" Compare text with ==, or look its rate up with lookup().");
    }
    report(
      scope,
      operand,
      "error",
      "type",
      translate("'{0}' works on numbers, but {1}.{2}", node.op, describeValue(operand, type), hint),
      { fixes },
    );
  }
}

function typeOfBinary(scope: Scope, node: Node & { kind: "binary" }): ExprType {
  const { op } = node;
  if (op === "??") {
    markGuarded(scope, node.left);
    const left = typeOf(scope, node.left);
    const right = typeOf(scope, node.right);
    return join(left, right);
  }
  const chained =
    COMPARISONS.has(op) &&
    node.left.kind === "binary" &&
    !node.left.grouped &&
    COMPARISONS.has(node.left.op);
  const leftOfChain = typeOf(scope, node.left);
  const left =
    chained && node.left.kind === "binary"
      ? (scope.types.get(node.left.right) ?? "Any")
      : leftOfChain;
  const right = typeOf(scope, node.right);
  switch (op) {
    case "+":
      if (left === "String" && right === "String") return "String";
      if ((left === "String" && right === "Number") || (left === "Number" && right === "String")) {
        report(
          scope,
          node.opRange,
          "error",
          "type",
          translate(
            "'+' adds two numbers or joins two texts; here one side is text and the other a number. Convert with string() or float().",
          ),
        );
        return "Any";
      }
      if (left === "Date" || right === "Date") {
        checkDateArithmetic(scope, node, left, right);
        return "Date";
      }
      checkArithmetic(scope, node, left, right);
      return "Number";
    case "-":
    case "*":
    case "/":
    case "%":
    case "**":
    case "^":
      if (op === "/" && node.right.kind === "number" && node.right.value === 0) {
        report(
          scope,
          node.right,
          "error",
          "divide-by-zero",
          translate("Dividing by 0 fails on every load"),
        );
      }
      if (op === "-" && left === "Date") {
        if (right !== "Date") checkDateArithmetic(scope, node, left, right);
        return right === "Date" ? "Any" : "Date";
      }
      checkArithmetic(scope, node, left, right);
      if (op === "%") checkWholeOperands(scope, node, left, right);
      return "Number";
    case "<":
    case ">":
    case "<=":
    case ">=": {
      const comparable = (type: ExprType) =>
        type === "Number" ||
        type === "Date" ||
        type === "String" ||
        type === "Any" ||
        type === "Nil";
      if (
        !comparable(left) ||
        !comparable(right) ||
        (left !== "Any" && right !== "Any" && left !== "Nil" && right !== "Nil" && left !== right)
      ) {
        const bad = !comparable(left) ? node.left : !comparable(right) ? node.right : node.right;
        const badType = bad === node.left ? left : right;
        report(
          scope,
          bad,
          "error",
          "type",
          badType === "Boolean"
            ? translate(
                "'{0}' compares numbers, but {1}. Use it as the test itself.",
                op,
                describeValue(bad, badType),
              )
            : translate(
                "'{0}' compares values of the same kind, but {1} and {2}",
                op,
                describeValue(node.left, left),
                describeValue(node.right, right),
              ),
        );
      }
      return "Boolean";
    }
    case "==":
    case "!=":
      if (
        left !== "Any" &&
        right !== "Any" &&
        left !== "Nil" &&
        right !== "Nil" &&
        left !== right
      ) {
        const untyped =
          isUntypedToTheEngine(scope, node.left) || isUntypedToTheEngine(scope, node.right);
        report(
          scope,
          node.opRange,
          untyped ? "warning" : "error",
          "type",
          untyped
            ? translate(
                "{0} and {1}, so this is never {2}",
                describeValue(node.left, left),
                describeValue(node.right, right),
                op === "==" ? translate("equal") : translate("different"),
              )
            : translate(
                "{0} and {1}; the engine compares only values of the same kind",
                describeValue(node.left, left),
                describeValue(node.right, right),
              ),
        );
      }
      return "Boolean";
    case "and":
    case "&&":
    case "or":
    case "||":
      for (const [operand, type] of [
        [node.left, left],
        [node.right, right],
      ] as const) {
        if (type === "Boolean" || type === "Any" || type === "Nil") continue;
        const name = memberPath(operand);
        report(
          scope,
          operand,
          "error",
          "type",
          type === "Number" && name
            ? translate(
                "'{0}' joins two true/false tests, but {1}. Compare it, such as {2} > 0",
                op,
                describeValue(operand, type),
                name,
              )
            : translate(
                "'{0}' joins two true/false tests, but {1}",
                op,
                describeValue(operand, type),
              ),
        );
      }
      return "Boolean";
    case "in":
      if (
        right !== "Array" &&
        right !== "Object" &&
        right !== "String" &&
        right !== "Any" &&
        right !== "Nil"
      ) {
        report(
          scope,
          node.right,
          "error",
          "type",
          translate(
            "'in' looks inside a list, a record or text, but {0}",
            describeValue(node.right, right),
          ),
        );
      }
      return "Boolean";
    case "contains":
    case "startsWith":
    case "endsWith":
    case "matches":
      checkTextOperands(scope, node, left, right);
      return "Boolean";
    case "..":
      return "Array";
    default:
      if (scope.known.operatorNames.has(op)) {
        checkTextOperands(scope, node, left, right);
        return "Boolean";
      }
      return "Any";
  }
}

/** A date moves by a duration (`duration("2h")`), not by a bare number. */
function checkDateArithmetic(
  scope: Scope,
  node: Node & { kind: "binary" },
  left: ExprType,
  right: ExprType,
) {
  const number = left === "Number" ? node.left : right === "Number" ? node.right : null;
  if (!number) return;
  report(
    scope,
    number,
    "error",
    "type",
    translate(
      'A date moves by a duration, not a number: write {0} {1} duration("24h")',
      memberPath(node.left === number ? node.right : node.left) ?? translate("the date"),
      node.op,
    ),
  );
}

/** `%` takes whole numbers only; a decimal field, an input or a quotient is refused by the engine. */
function checkWholeOperands(
  scope: Scope,
  node: Node & { kind: "binary" },
  left: ExprType,
  right: ExprType,
) {
  for (const [operand, type] of [
    [node.left, left],
    [node.right, right],
  ] as const) {
    if (type !== "Number" || isWhole(scope, operand)) continue;
    const name = memberPath(operand);
    report(
      scope,
      operand,
      "error",
      "type",
      name
        ? translate(
            "'%' takes whole numbers, but {0} can hold a fraction. Use int({1}) to take its whole part",
            quote(name),
            name,
          )
        : translate("'%' takes whole numbers, but this part can come to a fraction"),
      {
        fixes: name
          ? [
              {
                kind: "replace",
                label: translate("Change to int({0})", name),
                // i18n-ignore: formula source code, not text
                changes: [{ from: operand.from, to: operand.to, insert: `int(${name})` }],
              },
            ]
          : [],
      },
    );
  }
}

function checkTextOperands(
  scope: Scope,
  node: Node & { kind: "binary" },
  left: ExprType,
  right: ExprType,
) {
  for (const [operand, type] of [
    [node.left, left],
    [node.right, right],
  ] as const) {
    if (
      type === "String" ||
      type === "Any" ||
      type === "Nil" ||
      (node.op === "contains" && type === "Array")
    ) {
      continue;
    }
    report(
      scope,
      operand,
      "error",
      "type",
      translate("'{0}' works on text, but {1}", node.op, describeValue(operand, type)),
    );
  }
}

function typeOf(scope: Scope, node: Node): ExprType {
  const type = typeOfNode(scope, node);
  scope.types.set(node, type);
  return type;
}

function typeOfNode(scope: Scope, node: Node): ExprType {
  switch (node.kind) {
    case "number":
      return "Number";
    case "string":
      return "String";
    case "bool":
      return "Boolean";
    case "nil":
      return "Nil";
    case "pointer":
      if (scope.pointerDepth === 0) {
        report(
          scope,
          node,
          "error",
          "syntax",
          translate("'#' stands for the current item inside filter(), map() and the like"),
        );
      } else if (node.name === "acc" && scope.reduceDepth === 0) {
        report(
          scope,
          node,
          "error",
          "syntax",
          translate("'#acc' is the running total inside reduce() and nowhere else"),
        );
      } else if (node.name !== "" && node.name !== "acc" && node.name !== "index") {
        report(
          scope,
          node,
          "error",
          "syntax",
          translate("'#{0}' isn't a pointer; use #, #index or, inside reduce(), #acc", node.name),
        );
      }
      return "Any";
    case "missing":
      return "Any";
    case "ident":
      return typeOfIdent(scope, node);
    case "member":
      return typeOfMember(scope, node);
    case "index": {
      const object = typeOf(scope, node.object);
      typeOf(scope, node.index);
      if (object === "Number" || object === "Boolean") {
        report(
          scope,
          node,
          "error",
          "type",
          translate("{0}; it can't be indexed with [ ]", describeValue(node.object, object)),
        );
      }
      return "Any";
    }
    case "slice": {
      typeOf(scope, node.object);
      if (node.start) typeOf(scope, node.start);
      if (node.end) typeOf(scope, node.end);
      return "Any";
    }
    case "call":
      return typeOfCall(scope, node);
    case "sequence":
      return typeOfSequence(scope, node);
    case "unary": {
      const operand = typeOf(scope, node.operand);
      if (node.op === "-" || node.op === "+") {
        if (!isNumeric(operand) && operand !== "Nil") {
          report(
            scope,
            node.operand,
            "error",
            "type",
            translate(
              "'{0}' works on numbers, but {1}",
              node.op,
              describeValue(node.operand, operand),
            ),
          );
        }
        return "Number";
      }
      if (operand !== "Boolean" && operand !== "Any" && operand !== "Nil") {
        report(
          scope,
          node.operand,
          "error",
          "type",
          translate(
            "'{0}' flips a true/false test, but {1}",
            node.op,
            describeValue(node.operand, operand),
          ),
        );
      }
      return "Boolean";
    }
    case "binary":
      return typeOfBinary(scope, node);
    case "ternary":
    case "if": {
      markNilTests(scope, node.test);
      const test = typeOf(scope, node.test);
      if (test !== "Boolean" && test !== "Any" && test !== "Nil") {
        const name = memberPath(node.test);
        report(
          scope,
          node.test,
          "error",
          "type",
          name && test === "Number"
            ? translate(
                "The test before '?' must be true/false, but {0}. Compare it: {1} > 0 ? …",
                describeValue(node.test, test),
                name,
              )
            : translate(
                "The test before '?' must be true/false, but {0}",
                describeValue(node.test, test),
              ),
        );
      }
      const consequent = typeOf(scope, node.consequent);
      const alternate = typeOf(scope, node.alternate);
      return join(consequent, alternate);
    }
    case "array":
      for (const item of node.items) typeOf(scope, item);
      return "Array";
    case "map":
      for (const entry of node.entries) typeOf(scope, entry.value);
      return "Object";
    case "predicate":
      scope.pointerDepth += 1;
      typeOf(scope, node.body);
      scope.pointerDepth -= 1;
      return "Any";
    default:
      return "Any";
  }
}

/**
 * `let` lines inside parentheses, an `if` block or a predicate. Each name is
 * scoped to the value after it; the engine refuses one that a field, an input,
 * a function or an enclosing `let` already holds.
 */
function typeOfSequence(scope: Scope, node: Node & { kind: "sequence" }): ExprType {
  const added: string[] = [];
  for (const entry of node.lets) {
    const type = typeOf(scope, entry.value);
    if (entry.name === "") continue;
    if (scope.bindings.has(entry.name)) {
      report(
        scope,
        entry.nameRange,
        "error",
        "let-duplicate",
        translate("{0} is already defined above; use another name", quote(entry.name)),
        { name: entry.name },
      );
      continue;
    }
    checkLetName(scope, entry);
    scope.bindings.set(entry.name, {
      name: entry.name,
      nameRange: entry.nameRange,
      type,
      whole: isWhole(scope, entry.value),
      reads: 0,
      from: entry.from,
      to: entry.to,
    });
    scope.defined.add(entry.name);
    added.push(entry.name);
  }
  const type = typeOf(scope, node.body);
  for (const name of added) {
    const binding = scope.bindings.get(name);
    if (binding && binding.reads === 0) {
      report(
        scope,
        binding.nameRange,
        "warning",
        "let-unused",
        translate("{0} is never used", quote(name)),
        { name },
      );
    }
    scope.bindings.delete(name);
    scope.defined.delete(name);
  }
  return type;
}

/** The engine refuses a `let` that would hide a field, an input or a function. */
function checkLetName(scope: Scope, entry: Pick<LetEntry, "name" | "nameRange">) {
  const { name } = entry;
  if (scope.variables.has(name) || scope.roots.has(name)) {
    const custom = scope.variables.get(name)?.custom === true;
    report(
      scope,
      entry.nameRange,
      "error",
      "let-shadows",
      custom
        ? translate(
            "{0} is one of this template's inputs, and a 'let' can't redeclare it. Use another name",
            quote(name),
          )
        : translate(
            "{0} is already a shipment field, and a 'let' can't redeclare it. Use another name",
            quote(name),
          ),
      { name },
    );
    return;
  }
  if (scope.known.functionNames.has(name)) {
    report(
      scope,
      entry.nameRange,
      "error",
      "let-shadows",
      translate(
        "{0} is the name of a formula function, and a 'let' can't take it. Use another name",
        quote(name),
      ),
      { name },
    );
  }
}

/** A ternary whose test asks whether a value is empty guards that value in both branches. */
function markNilTests(scope: Scope, test: Node) {
  const names = new Set<string>();
  const collect = (node: Node) => {
    if (node.kind === "binary" && (node.op === "==" || node.op === "!=")) {
      const [a, b] = [node.left, node.right];
      const path = b.kind === "nil" ? memberPath(a) : a.kind === "nil" ? memberPath(b) : null;
      if (path) names.add(path);
    } else if (
      node.kind === "binary" &&
      (node.op === "and" || node.op === "&&" || node.op === "or" || node.op === "||")
    ) {
      collect(node.left);
      collect(node.right);
    }
  };
  collect(test);
  scope.guardedByTest = names;
}

/** Walks the tree marking any read of a name the surrounding ternary tested for nil as guarded. */
function markTestedReads(scope: Scope, node: Node, names: Set<string>) {
  const path = memberPath(node);
  if (path && names.has(path)) markGuarded(scope, node);
  switch (node.kind) {
    case "member":
      break;
    default:
      for (const child of childrenFor(node)) markTestedReads(scope, child, names);
  }
}

function childrenFor(node: Node): Node[] {
  switch (node.kind) {
    case "index":
      return [node.object, node.index];
    case "slice":
      return [node.object, ...(node.start ? [node.start] : []), ...(node.end ? [node.end] : [])];
    case "call":
      return node.args;
    case "unary":
      return [node.operand];
    case "binary":
      return [node.left, node.right];
    case "ternary":
    case "if":
      return [node.test, node.consequent, node.alternate];
    case "array":
      return node.items;
    case "map":
      return node.entries.map((entry) => entry.value);
    case "predicate":
      return [node.body];
    case "sequence":
      return [...node.lets.map((entry) => entry.value), node.body];
    default:
      return [];
  }
}

function syntaxToDiagnostic(diagnostic: SyntaxDiagnostic): ExprDiagnostic {
  return {
    from: diagnostic.from,
    to: diagnostic.to,
    severity: "error",
    code: "syntax",
    message: diagnostic.message,
    fixes: diagnostic.fix
      ? [
          {
            kind: "replace",
            label: diagnostic.fix.label,
            changes: [{ from: diagnostic.from, to: diagnostic.to, insert: diagnostic.fix.insert }],
          },
        ]
      : [],
  };
}

function preMarkGuards(scope: Scope, node: Node) {
  // Reads under a `?? fallback`, inside coalesce(), or inside a branch of a
  // ternary that tested the same name for nil are guarded before any read is
  // typed, so a guard that comes later in the tree still counts.
  if (node.kind === "binary" && node.op === "??") markGuarded(scope, node.left);
  if (node.kind === "binary" && (node.op === "==" || node.op === "!=")) {
    if (node.right.kind === "nil") markGuarded(scope, node.left);
    if (node.left.kind === "nil") markGuarded(scope, node.right);
  }
  if (node.kind === "call" && memberPath(node.callee) === "coalesce") {
    for (const arg of node.args) markGuarded(scope, arg);
  }
  if (node.kind === "ternary" || node.kind === "if") {
    markNilTests(scope, node.test);
    const names = scope.guardedByTest ?? new Set<string>();
    if (names.size > 0) {
      markTestedReads(scope, node.consequent, names);
      markTestedReads(scope, node.alternate, names);
    }
  }
  for (const child of childrenFor(node)) preMarkGuards(scope, child);
  if (node.kind === "member") preMarkGuards(scope, node.object);
}

/** Checks a parsed expression against what the schema and the template know. */
export function checkExpression(expression: string, known: KnownIdentifiers): ExprAnalysis {
  const program = parseExpression(expression, known.operatorNames);
  const scope: Scope = {
    known,
    variables: new Map(known.variables.map((doc) => [doc.name, doc])),
    roots: new Map(),
    bindings: new Map(),
    diagnostics: program.diagnostics.map(syntaxToDiagnostic),
    schemaReads: new Set(),
    inputReads: new Set(),
    functionCalls: new Set(),
    order: [],
    guarded: new Set(),
    flaggedNullable: new Set(),
    defined: new Set(),
    pointerDepth: 0,
    reduceDepth: 0,
    types: new Map(),
    source: expression,
  };
  for (const doc of known.variables) {
    const dot = doc.name.indexOf(".");
    if (dot === -1) continue;
    const root = doc.name.slice(0, dot);
    const list = scope.roots.get(root) ?? [];
    list.push(doc);
    scope.roots.set(root, list);
  }

  const statements = program.statements;
  for (const statement of statements) {
    if (statement.kind !== "let") continue;
    if (scope.bindings.has(statement.name)) {
      report(
        scope,
        statement.nameRange,
        "error",
        "let-duplicate",
        translate("{0} is already defined above; use another name", quote(statement.name)),
        { name: statement.name },
      );
      continue;
    }
    scope.bindings.set(statement.name, {
      name: statement.name,
      nameRange: statement.nameRange,
      type: "Any",
      whole: false,
      reads: 0,
      from: statement.from,
      to: statement.to,
    });
  }

  let resultType: ExprType = "Any";
  statements.forEach((statement, index) => {
    preMarkGuards(scope, statement.value);
    if (statement.kind === "let") {
      if (statement.name !== "") checkLetName(scope, statement);
      const type = typeOf(scope, statement.value);
      const binding = scope.bindings.get(statement.name);
      if (binding && binding.nameRange.from === statement.nameRange.from) {
        binding.type = type;
        binding.whole = isWhole(scope, statement.value);
        scope.defined.add(statement.name);
      }
      if (index === statements.length - 1 && statement.name !== "") {
        report(
          scope,
          statement,
          "error",
          "let-last",
          translate("The formula ends on a 'let'; add the amount it charges on the last line"),
          {
            fixes: [
              {
                kind: "replace",
                label: translate("Return {0}", statement.name),
                changes: [{ from: statement.to, to: statement.to, insert: `\n${statement.name}` }],
              },
            ],
          },
        );
      }
      return;
    }
    const type = typeOf(scope, statement.value);
    if (index < statements.length - 1) {
      report(
        scope,
        statement,
        "warning",
        "discarded",
        translate(
          "This line's value is worked out and thrown away; only the last line is the charge. Bind it with 'let' to use it.",
        ),
      );
      return;
    }
    resultType = type;
    if (type === "Boolean") {
      const name = memberPath(statement.value);
      report(
        scope,
        statement.value,
        "error",
        "result-type",
        translate(
          "The formula must come to an amount, but this comes to true/false. Write {0} ? amount : 0",
          name ?? translate("test"),
        ),
      );
    } else if (type === "String" || type === "Array" || type === "Object" || type === "Date") {
      report(
        scope,
        statement.value,
        "error",
        "result-type",
        translate("The formula must come to an amount, but this comes to {0}", typeWords(type)),
      );
    }
  });

  for (const binding of scope.bindings.values()) {
    if (binding.reads === 0 && binding.name !== "") {
      report(
        scope,
        binding.nameRange,
        "warning",
        "let-unused",
        translate("{0} is never used", quote(binding.name)),
        { name: binding.name },
      );
    }
  }

  const found = known.ready
    ? scope.diagnostics
    : scope.diagnostics.filter((diagnostic) => !SCHEMA_CODES.has(diagnostic.code));
  const diagnostics = dedupe(found).sort((a, b) => a.from - b.from || a.to - b.to);
  const reads = [
    ...scope.order.filter((name) => scope.inputReads.has(name)),
    ...scope.order.filter((name) => !scope.inputReads.has(name)),
  ];

  return {
    program,
    diagnostics,
    schemaReads: scope.schemaReads,
    inputReads: scope.inputReads,
    functionCalls: scope.functionCalls,
    reads,
    bindings: scope.bindings,
    resultType,
    pending: !known.ready,
  };
}

/** Findings that depend on the engine's schema; none is reported until it has loaded. */
const SCHEMA_CODES: ReadonlySet<DiagnosticCode> = new Set<DiagnosticCode>([
  "unknown-variable",
  "unknown-field",
  "unknown-function",
  "disabled-function",
  "arity",
  "let-shadows",
]);

function dedupe(diagnostics: ExprDiagnostic[]): ExprDiagnostic[] {
  const seen = new Set<string>();
  return diagnostics.filter((diagnostic) => {
    const key = `${diagnostic.from}:${diagnostic.to}:${diagnostic.code}:${diagnostic.message}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

/**
 * Parses the position expr's engine writes into its errors, "(line:column)",
 * onto a span. The line and column are 1-based and the column counts
 * characters (runes), not the UTF-16 units the editor's offsets are in.
 */
export function engineErrorRange(message: string, expression: string): Range | null {
  const match = /\((\d+):(\d+)\)/.exec(message);
  if (!match) return null;
  const line = Number(match[1]);
  const column = Number(match[2]);
  const lines = expression.split("\n");
  if (line < 1 || line > lines.length) return null;
  let offset = 0;
  for (let index = 0; index < line - 1; index += 1) offset += lines[index].length + 1;
  const text = lines[line - 1];
  const runes = Array.from(text);
  const start = runes.slice(0, Math.max(0, column - 1)).join("").length;
  const from = Math.min(offset + start, offset + text.length);
  const rest = text.slice(start);
  const word = /^[A-Za-z_][A-Za-z0-9_.]*|^\S/.exec(rest);
  return { from, to: from + Math.max(1, word?.[0].length ?? 1) };
}

/** The engine's message without its wrapping and position: what a person should read. */
export function engineErrorText(message: string): string {
  let text = message.trim();
  const during = text.lastIndexOf("during ");
  if (text.startsWith("schema error") && during !== -1) {
    const colon = text.indexOf(": ", during);
    if (colon !== -1) text = text.slice(colon + 2);
  }
  text = text.replace(/\s*\(\d+:\d+\)[\s\S]*$/, "").trim();
  return text.charAt(0).toUpperCase() + text.slice(1);
}
