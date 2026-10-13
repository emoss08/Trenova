import type { EditorView } from "@codemirror/view";
import {
  WORD_OPERATORS,
  parseExpression,
  type Node,
  type Program,
  type Statement,
} from "./expr-parser";

/*
 * The formatter: one statement per line, `let` lines before the amount, a
 * single space around every operator, and a long line broken where it reads
 * best — a sum one term per line, a call one argument per line, a ternary
 * one branch per line. It prints the tree the parser built, so the author's
 * parentheses, literals and operator spellings (`and` stays `and`) survive;
 * only whitespace changes. A formula that does not parse, or that holds a
 * comment the tree would drop, is left exactly as it is.
 */

export type FormatOptions = {
  /** The column the formatter breaks a line before. */
  width?: number;
  /** Word operators the schema publishes beyond the built-in four. */
  operators?: ReadonlySet<string>;
};

export type FormatRefusal = "empty" | "syntax" | "comments";

export type FormatResult =
  | { ok: true; text: string; changed: boolean }
  | { ok: false; reason: FormatRefusal };

export const DEFAULT_FORMAT_WIDTH = 80;
const INDENT = "  ";

function pad(columns: number): string {
  return " ".repeat(columns);
}

/** The last line of a printed fragment, as a column count. */
function lastLineLength(text: string): number {
  const at = text.lastIndexOf("\n");
  return at === -1 ? text.length : text.length - at - 1;
}

function hasMissing(program: Program): boolean {
  const visit = (node: Node): boolean => {
    if (node.kind === "missing") return true;
    switch (node.kind) {
      case "member":
        return visit(node.object);
      case "index":
        return visit(node.object) || visit(node.index);
      case "slice":
        return (
          visit(node.object) ||
          (node.start !== null && visit(node.start)) ||
          (node.end !== null && visit(node.end))
        );
      case "call":
        return visit(node.callee) || node.args.some(visit);
      case "unary":
        return visit(node.operand);
      case "binary":
        return visit(node.left) || visit(node.right);
      case "ternary":
      case "if":
        return visit(node.test) || visit(node.consequent) || visit(node.alternate);
      case "array":
        return node.items.some(visit);
      case "map":
        return node.entries.some((entry) => visit(entry.value));
      case "predicate":
        return visit(node.body);
      default:
        return false;
    }
  };
  return program.statements.some((statement) => visit(statement.value));
}

/** The binary precedence groups a chain is flattened across; `..` and `**` stay nested. */
const CHAIN_GROUPS: Record<string, string> = {
  "+": "add",
  "-": "add",
  "*": "mul",
  "/": "mul",
  "%": "mul",
  "&&": "and",
  and: "and",
  "||": "or",
  or: "or",
  "??": "coalesce",
};

/** A `not` written between the operands: `a not in b`, `name not contains "x"`. */
function isInfixNot(node: Node & { kind: "unary" }): boolean {
  return (
    node.op === "not" &&
    node.operand.kind === "binary" &&
    !node.operand.grouped &&
    node.opRange.from > node.operand.left.from
  );
}

/** A call written with the pipe: `value | round(2)`. */
function isPipe(node: Node & { kind: "call" }): boolean {
  return node.args.length > 0 && node.callee.from > node.args[0].from;
}

class Printer {
  private readonly source: string;
  private readonly width: number;

  constructor(source: string, width: number) {
    this.source = source;
    this.width = width;
  }

  program(program: Program): string {
    const lines: string[] = [];
    program.statements.forEach((statement, index) => {
      if (index > 0) {
        const gap = this.source.slice(program.statements[index - 1].to, statement.from);
        if (/\n[ \t]*\r?\n/.test(gap)) lines.push("");
      }
      lines.push(this.statement(statement));
    });
    return lines.join("\n");
  }

  private statement(statement: Statement): string {
    if (statement.kind === "result") {
      return this.node(statement.value, 0, 0);
    }
    // i18n-ignore: formula source code, not text
    const head = `let ${statement.name} = `;
    const flat = this.flat(statement.value);
    if (head.length + flat.length + 1 <= this.width) {
      return `${head}${flat};`;
    }
    if (INDENT.length + flat.length + 1 <= this.width) {
      // i18n-ignore: formula source code, not text
      return `let ${statement.name} =\n${INDENT}${flat};`;
    }
    return `${head}${this.node(statement.value, head.length, 0)};`;
  }

  /** The node at a column, broken over lines indented past `indent` when it does not fit. */
  private node(node: Node, column: number, indent: number): string {
    const flat = this.flat(node);
    if (column + flat.length <= this.width) return flat;
    return this.broken(node, column, indent);
  }

  private broken(node: Node, column: number, indent: number): string {
    if (node.grouped) {
      const inner: Node = { ...node, grouped: node.grouped - 1 };
      return `(${this.broken(inner, column + 1, indent)})`;
    }
    switch (node.kind) {
      case "binary":
        return this.brokenChain(node, column, indent);
      case "ternary":
        return this.brokenTernary(node, indent);
      case "call":
        return this.brokenCall(node, column, indent);
      case "unary": {
        if (isInfixNot(node))
          return this.brokenChain(node.operand as Node & { kind: "binary" }, column, indent, node);
        const op = node.op === "not" ? "not " : node.op;
        // i18n-ignore: formula source code, not text
        return `${op}${this.node(node.operand, column + op.length, indent)}`;
      }
      case "if":
        return this.brokenIf(node, indent);
      case "array": {
        if (node.items.length === 0) return "[]";
        const inner = indent + INDENT.length;
        const items = node.items.map((item) => pad(inner) + this.node(item, inner, inner));
        return `[\n${items.join(",\n")}\n${pad(indent)}]`;
      }
      case "map": {
        if (node.entries.length === 0) return "{}";
        const inner = indent + INDENT.length;
        const entries = node.entries.map((entry) => {
          const head = `${pad(inner)}${entry.key}: `;
          return head + this.node(entry.value, head.length, inner);
        });
        return `{\n${entries.join(",\n")}\n${pad(indent)}}`;
      }
      case "predicate":
        return `{${this.node(node.body, column + 1, indent)}}`;
      case "member": {
        const object = this.node(node.object, column, indent);
        return `${object}${node.optional ? "?." : "."}${node.property}`;
      }
      case "index": {
        const object = this.node(node.object, column, indent);
        const index = this.node(node.index, lastLineLength(object) + 1, indent);
        return `${object}[${index}]`;
      }
      default:
        return this.flat(node);
    }
  }

  /**
   * A chain of one precedence group, one operand per line with the operator
   * leading the continuation, the way a column of charges reads.
   */
  private brokenChain(
    node: Node & { kind: "binary" },
    column: number,
    indent: number,
    negated?: Node & { kind: "unary" },
  ): string {
    const group = CHAIN_GROUPS[node.op];
    if (!group || negated) {
      const left = this.node(node.left, column, indent);
      // i18n-ignore: formula source code, not text
      const op = negated ? `not ${node.op}` : node.op;
      const next = indent + INDENT.length;
      return `${left}\n${pad(next)}${op} ${this.node(node.right, next + op.length + 1, next)}`;
    }
    const operands: Node[] = [];
    const operators: string[] = [];
    let current: Node = node;
    while (current.kind === "binary" && !current.grouped && CHAIN_GROUPS[current.op] === group) {
      operands.unshift(current.right);
      operators.unshift(current.op);
      current = current.left;
    }
    operands.unshift(current);
    const next = indent + INDENT.length;
    const lines = [this.node(operands[0], column, next)];
    operators.forEach((op, index) => {
      const operand = operands[index + 1];
      lines.push(`${pad(next)}${op} ${this.node(operand, next + op.length + 1, next)}`);
    });
    return lines.join("\n");
  }

  /** `test ? a` over `: test ? b` over `: c`, each case on its own line. */
  private brokenTernary(node: Node & { kind: "ternary" }, indent: number): string {
    const next = indent + INDENT.length;
    const cases: string[] = [];
    let current: Node = node;
    let first = true;
    while (current.kind === "ternary" && !current.grouped) {
      const column = first ? indent : next + 2;
      const test = this.node(current.test, column, next);
      const branch =
        current.consequent === current.test
          ? `${test} ?:`
          : `${test} ? ${this.node(current.consequent, lastLineLength(test) + 3, next)}`;
      cases.push(first ? branch : `${pad(next)}: ${branch}`);
      current = current.alternate;
      first = false;
    }
    cases.push(`${pad(next)}: ${this.node(current, next + 2, next)}`);
    return cases.join("\n");
  }

  private brokenCall(node: Node & { kind: "call" }, column: number, indent: number): string {
    if (isPipe(node)) {
      const next = indent + INDENT.length;
      const left = this.node(node.args[0], column, indent);
      const rest = node.args.slice(1);
      const callee = this.flat(node.callee);
      const call =
        node.parens.from === node.callee.from
          ? callee
          : `${callee}(${rest.map((arg) => this.flat(arg)).join(", ")})`;
      return `${left}\n${pad(next)}| ${call}`;
    }
    const callee = this.flat(node.callee);
    const inner = indent + INDENT.length;
    const args = node.args.map((arg) => pad(inner) + this.node(arg, inner, inner));
    return `${callee}(\n${args.join(",\n")}\n${pad(indent)})`;
  }

  private brokenIf(node: Node & { kind: "if" }, indent: number): string {
    const inner = indent + INDENT.length;
    const test = this.node(node.test, indent + 3, indent);
    const consequent = pad(inner) + this.node(node.consequent, inner, inner);
    const alternate =
      node.alternate.kind === "if" && !node.alternate.grouped
        ? this.brokenIf(node.alternate, indent)
        : `{\n${pad(inner)}${this.node(node.alternate, inner, inner)}\n${pad(indent)}}`;
    // i18n-ignore: formula source code, not text
    return `if ${test} {\n${consequent}\n${pad(indent)}} else ${alternate}`;
  }

  /** A literal as the author wrote it, without the parentheses around it. */
  private literal(node: Node): string {
    let text = this.source.slice(node.from, node.to).trim();
    for (let layer = node.grouped ?? 0; layer > 0; layer -= 1) {
      if (text.startsWith("(") && text.endsWith(")")) text = text.slice(1, -1).trim();
    }
    return text;
  }

  /** The node on one line. */
  flat(node: Node): string {
    const inner = this.flatInner(node);
    const layers = node.grouped ?? 0;
    return layers > 0 ? `${"(".repeat(layers)}${inner}${")".repeat(layers)}` : inner;
  }

  private flatInner(node: Node): string {
    switch (node.kind) {
      case "number":
      case "string":
        return this.literal(node);
      case "bool":
        return node.value ? "true" : "false";
      case "nil":
        return "nil";
      case "pointer":
        return "#";
      case "ident":
        return node.name;
      case "member": {
        if (node.object.kind === "pointer" && node.object.from === node.object.to) {
          return `.${node.property}`;
        }
        return `${this.flat(node.object)}${node.optional ? "?." : "."}${node.property}`;
      }
      case "index":
        return `${this.flat(node.object)}[${this.flat(node.index)}]`;
      case "slice":
        return `${this.flat(node.object)}[${node.start ? this.flat(node.start) : ""}:${
          node.end ? this.flat(node.end) : ""
        }]`;
      case "call": {
        const callee = this.flat(node.callee);
        if (isPipe(node)) {
          const rest = node.args.slice(1).map((arg) => this.flat(arg));
          const call =
            node.parens.from === node.callee.from ? callee : `${callee}(${rest.join(", ")})`;
          return `${this.flat(node.args[0])} | ${call}`;
        }
        return `${callee}(${node.args.map((arg) => this.flat(arg)).join(", ")})`;
      }
      case "unary": {
        if (isInfixNot(node)) {
          const operand = node.operand as Node & { kind: "binary" };
          // i18n-ignore: formula source code, not text
          return `${this.flat(operand.left)} not ${operand.op} ${this.flat(operand.right)}`;
        }
        const operand = this.flat(node.operand);
        // i18n-ignore: formula source code, not text
        if (node.op === "not") return `not ${operand}`;
        const spaced =
          node.operand.kind === "unary" && !node.operand.grouped && node.operand.op === node.op;
        return `${node.op}${spaced ? " " : ""}${operand}`;
      }
      case "binary": {
        const left = this.flat(node.left);
        const right = this.flat(node.right);
        if (node.op === "..") return `${left}..${right}`;
        return `${left} ${node.op} ${right}`;
      }
      case "ternary": {
        const test = this.flat(node.test);
        const alternate = this.flat(node.alternate);
        if (node.consequent === node.test) return `${test} ?: ${alternate}`;
        return `${test} ? ${this.flat(node.consequent)} : ${alternate}`;
      }
      case "if": {
        const alternate =
          node.alternate.kind === "if" && !node.alternate.grouped
            ? this.flatInner(node.alternate)
            : `{ ${this.flat(node.alternate)} }`;
        // i18n-ignore: formula source code, not text
        return `if ${this.flat(node.test)} { ${this.flat(node.consequent)} } else ${alternate}`;
      }
      case "array":
        return `[${node.items.map((item) => this.flat(item)).join(", ")}]`;
      case "map":
        return node.entries.length === 0
          ? "{}"
          : `{${node.entries.map((entry) => `${entry.key}: ${this.flat(entry.value)}`).join(", ")}}`;
      case "predicate":
        return `{${this.flat(node.body)}}`;
      case "missing":
        return "";
      default:
        return "";
    }
  }
}

/**
 * Formats a formula. Refuses, leaving the text alone, when it is empty, when
 * it does not parse (formatting a broken tree would drop what the parser
 * could not place) or when it holds a comment.
 */
export function formatExpression(source: string, options: FormatOptions = {}): FormatResult {
  const program = parseExpression(source, options.operators ?? WORD_OPERATORS);
  if (program.empty) return { ok: false, reason: "empty" };
  if (program.comments.length > 0) return { ok: false, reason: "comments" };
  if (program.diagnostics.length > 0 || hasMissing(program)) {
    return { ok: false, reason: "syntax" };
  }
  const text = new Printer(source, options.width ?? DEFAULT_FORMAT_WIDTH).program(program);
  return { ok: true, text, changed: text !== source };
}

/** Formats the whole document in place, keeping the cursor on the same line where it can. */
export function formatEditorDocument(view: EditorView, options: FormatOptions = {}): FormatResult {
  const source = view.state.doc.toString();
  const result = formatExpression(source, options);
  if (!result.ok || !result.changed) return result;
  const line = view.state.doc.lineAt(view.state.selection.main.head).number;
  view.dispatch({
    changes: { from: 0, to: source.length, insert: result.text },
    userEvent: "format",
  });
  const lines = view.state.doc.lines;
  const target = view.state.doc.line(Math.min(line, lines));
  view.dispatch({ selection: { anchor: target.to } });
  return result;
}
