import { translate } from "@trenova/shared/i18n/runtime";

/*
 * A parser for the formula language (expr-lang, as the formula engine runs
 * it), kept in step with the engine's grammar: `let name = value;` lines
 * before one result expression, `??`, the ternary, `if … { } else { }`,
 * calls, member access, indexes and slices, predicates and the word
 * operators. It never throws: a malformed expression yields a tree with
 * `missing` nodes and the diagnostics that explain what is wrong, each
 * anchored to the span it is about, so the editor can underline it. Offsets
 * are UTF-16 code units, which is what CodeMirror counts in.
 */

export type Range = { from: number; to: number };

export type TokenKind =
  | "number"
  | "string"
  | "ident"
  | "keyword"
  | "op"
  | "punct"
  | "pointer"
  | "comment"
  | "error"
  | "eof";

export type Token = Range & { kind: TokenKind; text: string };

export const KEYWORDS = new Set([
  "true",
  "false",
  "nil",
  "in",
  "not",
  "and",
  "or",
  "let",
  "if",
  "else",
]);

export const WORD_OPERATORS = new Set(["startsWith", "endsWith", "contains", "matches"]);

/* Binary precedence, from expr's operator table; higher binds tighter. */
const BINARY: Record<string, { precedence: number; right: boolean }> = {
  "|": { precedence: 0, right: false },
  or: { precedence: 10, right: false },
  "||": { precedence: 10, right: false },
  and: { precedence: 15, right: false },
  "&&": { precedence: 15, right: false },
  "==": { precedence: 20, right: false },
  "!=": { precedence: 20, right: false },
  "<": { precedence: 20, right: false },
  ">": { precedence: 20, right: false },
  ">=": { precedence: 20, right: false },
  "<=": { precedence: 20, right: false },
  in: { precedence: 20, right: false },
  matches: { precedence: 20, right: false },
  contains: { precedence: 20, right: false },
  startsWith: { precedence: 20, right: false },
  endsWith: { precedence: 20, right: false },
  "..": { precedence: 25, right: false },
  "+": { precedence: 30, right: false },
  "-": { precedence: 30, right: false },
  "*": { precedence: 60, right: false },
  "/": { precedence: 60, right: false },
  "%": { precedence: 60, right: false },
  "**": { precedence: 100, right: true },
  "^": { precedence: 100, right: true },
  "??": { precedence: 500, right: false },
};

const NEGATABLE = new Set(["contains", "matches", "startsWith", "endsWith", "in"]);

const UNARY_PRECEDENCE: Record<string, number> = { not: 50, "!": 50, "-": 90, "+": 90 };

const TOKEN =
  /(\s+)|(\/\/[^\n]*|\/\*[\s\S]*?(?:\*\/|$))|(\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|\.\d+(?:[eE][+-]?\d+)?)|("(?:\\.|[^"\\\n])*"?|'(?:\\.|[^'\\\n])*'?|`(?:\\.|[^`\\])*`?)|([A-Za-z_][A-Za-z0-9_]*)|(\?\?|\?\.|==|!=|<=|>=|&&|\|\||\*\*|\.\.|[-+*/%<>!?:^=|])|([(),;[\]{}.])|(#)|(.)/gy;

export type Tokenized = { tokens: Token[]; diagnostics: SyntaxDiagnostic[] };

export type SyntaxDiagnostic = Range & {
  message: string;
  /** A one-line replacement that fixes it, when there is one. */
  fix?: { label: string; insert: string };
};

/** Splits the expression into tokens; comments and spaces are dropped. */
export function tokenize(
  source: string,
  operators: ReadonlySet<string> = WORD_OPERATORS,
): Tokenized {
  const tokens: Token[] = [];
  const diagnostics: SyntaxDiagnostic[] = [];
  const pattern = new RegExp(TOKEN.source, "gy");
  let match: RegExpExecArray | null;
  while ((match = pattern.exec(source)) !== null) {
    const text = match[0];
    const from = match.index;
    const to = from + text.length;
    if (text === "") break;
    if (match[1] !== undefined || match[2] !== undefined) {
      if (match[2] !== undefined && text.startsWith("/*") && !text.endsWith("*/")) {
        diagnostics.push({ from, to, message: translate("This comment is never closed") });
      }
      continue;
    }
    if (match[3] !== undefined) {
      tokens.push({ kind: "number", text, from, to });
    } else if (match[4] !== undefined) {
      const quote = text[0];
      if (text.length < 2 || text[text.length - 1] !== quote || text.endsWith(`\\${quote}`)) {
        diagnostics.push({ from, to, message: translate("This text is never closed") });
      }
      tokens.push({ kind: "string", text, from, to });
    } else if (match[5] !== undefined) {
      tokens.push({
        kind:
          KEYWORDS.has(text) || WORD_OPERATORS.has(text) || operators.has(text)
            ? "keyword"
            : "ident",
        text,
        from,
        to,
      });
    } else if (match[6] !== undefined) {
      tokens.push({ kind: "op", text, from, to });
    } else if (match[7] !== undefined) {
      tokens.push({ kind: "punct", text, from, to });
    } else if (match[8] !== undefined) {
      tokens.push({ kind: "pointer", text, from, to });
    } else {
      diagnostics.push({ from, to, message: translate("Unexpected character '{0}'", text) });
      tokens.push({ kind: "error", text, from, to });
    }
  }
  tokens.push({ kind: "eof", text: "", from: source.length, to: source.length });
  return { tokens, diagnostics };
}

export type Node =
  | (Range & { kind: "number"; value: number })
  | (Range & { kind: "string"; value: string })
  | (Range & { kind: "bool"; value: boolean })
  | (Range & { kind: "nil" })
  | (Range & { kind: "pointer" })
  | (Range & { kind: "ident"; name: string })
  | (Range & {
      kind: "member";
      object: Node;
      property: string;
      propertyRange: Range;
      optional: boolean;
    })
  | (Range & { kind: "index"; object: Node; index: Node })
  | (Range & { kind: "slice"; object: Node; start: Node | null; end: Node | null })
  | (Range & { kind: "call"; callee: Node; args: Node[]; parens: Range })
  | (Range & { kind: "unary"; op: string; operand: Node; opRange: Range })
  | (Range & { kind: "binary"; op: string; left: Node; right: Node; opRange: Range })
  | (Range & { kind: "ternary"; test: Node; consequent: Node; alternate: Node })
  | (Range & { kind: "if"; test: Node; consequent: Node; alternate: Node })
  | (Range & { kind: "array"; items: Node[] })
  | (Range & { kind: "map"; entries: { key: string; value: Node }[] })
  | (Range & { kind: "predicate"; body: Node })
  | (Range & { kind: "missing" });

export type Statement =
  | (Range & { kind: "let"; name: string; nameRange: Range; value: Node })
  | (Range & { kind: "result"; value: Node });

export type Program = {
  statements: Statement[];
  diagnostics: SyntaxDiagnostic[];
  tokens: Token[];
  empty: boolean;
};

/** A node's full text span is what its range says; a call's name is its callee. */
export function calleeName(node: Node): string | null {
  if (node.kind === "ident") return node.name;
  if (node.kind === "member") {
    const object = calleeName(node.object);
    return object === null ? null : `${object}.${node.property}`;
  }
  return null;
}

/** The dotted path a member chain reads, such as `customer.name`, when it is one. */
export function memberPath(node: Node): string | null {
  return calleeName(node);
}

const OPEN_FOR_CLOSE: Record<string, string> = { ")": "(", "]": "[", "}": "{" };

class Parser {
  private index = 0;
  private readonly tokens: Token[];
  private readonly operators: ReadonlySet<string>;
  readonly diagnostics: SyntaxDiagnostic[] = [];

  constructor(tokens: Token[], operators: ReadonlySet<string>) {
    this.tokens = tokens;
    this.operators = operators;
  }

  private binary(token: Token): { precedence: number; right: boolean } | undefined {
    if (token.kind !== "op" && token.kind !== "keyword") return undefined;
    const known = BINARY[token.text];
    if (known) return known;
    return this.operators.has(token.text) ? { precedence: 20, right: false } : undefined;
  }

  private get current(): Token {
    return this.tokens[this.index];
  }

  private peek(offset = 1): Token {
    return this.tokens[Math.min(this.index + offset, this.tokens.length - 1)];
  }

  private advance(): Token {
    const token = this.current;
    if (this.index < this.tokens.length - 1) this.index += 1;
    return token;
  }

  private is(kind: TokenKind, text?: string): boolean {
    return this.current.kind === kind && (text === undefined || this.current.text === text);
  }

  private isOp(text: string): boolean {
    return (
      (this.current.kind === "op" || this.current.kind === "keyword") && this.current.text === text
    );
  }

  private report(range: Range, message: string, fix?: SyntaxDiagnostic["fix"]) {
    const overlaps = this.diagnostics.some((d) => d.from === range.from && d.message === message);
    if (!overlaps) this.diagnostics.push({ from: range.from, to: range.to, message, fix });
  }

  private missing(at: number): Node {
    return { kind: "missing", from: at, to: at };
  }

  private describe(token: Token): string {
    return token.kind === "eof" ? translate("the end of the formula") : `'${token.text}'`;
  }

  /** Skips to the `;` that ends the statement, or the end, after an error. */
  private recover() {
    let depth = 0;
    while (!this.is("eof")) {
      const token = this.current;
      if (token.kind === "punct") {
        if (token.text === "(" || token.text === "[" || token.text === "{") depth += 1;
        else if (token.text === ")" || token.text === "]" || token.text === "}") depth -= 1;
        else if (token.text === ";" && depth <= 0) return;
      }
      this.advance();
    }
  }

  parseProgram(): Statement[] {
    const statements: Statement[] = [];
    while (!this.is("eof")) {
      const start = this.current.from;
      if (this.is("keyword", "let")) {
        const statement = this.parseLet();
        statements.push(statement);
        continue;
      }
      if (this.is("punct", ";")) {
        this.report(this.current, translate("Unexpected ';'"));
        this.advance();
        continue;
      }
      const value = this.parseExpression(0);
      const end = Math.max(this.tokens[this.index - 1]?.to ?? start, value.to);
      statements.push({ kind: "result", value, from: start, to: end });
      if (this.is("punct", ";")) {
        this.advance();
        continue;
      }
      if (!this.is("eof")) {
        this.report(this.current, translate("Unexpected {0}", this.describe(this.current)));
        this.recover();
        if (this.is("punct", ";")) this.advance();
      }
    }
    return statements;
  }

  private parseLet(): Statement {
    const letToken = this.advance();
    let name = "";
    let nameRange: Range = { from: letToken.to, to: letToken.to };
    if (this.is("ident")) {
      const nameToken = this.advance();
      name = nameToken.text;
      nameRange = nameToken;
    } else if (this.is("keyword") && !this.isOp("=")) {
      const nameToken = this.advance();
      this.report(nameToken, translate("'{0}' is a keyword and cannot be a name", nameToken.text));
      name = nameToken.text;
      nameRange = nameToken;
    } else {
      this.report(letToken, translate("'let' needs a name: let fuel = …"));
    }
    if (this.isOp("=")) {
      this.advance();
    } else if (this.isOp("==")) {
      const token = this.advance();
      this.report(token, translate("'let' assigns with one '='"), {
        label: translate("Use '='"),
        insert: "=",
      });
    } else {
      this.report(
        nameRange,
        name === ""
          ? translate("'let' needs a name, '=' and a value")
          : translate("'let {0}' needs '=' and a value", name),
      );
    }
    const value =
      this.is("punct", ";") || this.is("eof")
        ? this.missing(this.current.from)
        : this.parseExpression(0);
    if (value.kind === "missing") {
      this.report(
        nameRange,
        name === ""
          ? translate("'let' needs a value after '='")
          : translate("'let {0}' needs a value after '='", name),
      );
    }
    const end = this.tokens[this.index - 1]?.to ?? letToken.to;
    if (this.is("punct", ";")) {
      this.advance();
    } else if (this.is("eof")) {
      this.report(
        { from: end, to: end },
        translate("A 'let' line ends with ';', then the formula's amount follows on the next line"),
        { label: translate("Add ';'"), insert: ";" },
      );
    } else {
      this.report({ from: end, to: end }, translate("A 'let' line ends with ';'"), {
        label: translate("Add ';'"),
        insert: ";",
      });
    }
    return { kind: "let", name, nameRange, value, from: letToken.from, to: end };
  }

  private parseExpression(precedence: number): Node {
    if (precedence === 0 && this.is("keyword", "if")) {
      return this.parseIf();
    }
    let left = this.parseUnary();
    let previousOp = "";

    for (;;) {
      let token = this.current;
      let negate: Token | null = null;
      if (
        token.kind === "keyword" &&
        token.text === "not" &&
        (NEGATABLE.has(this.peek().text) || this.operators.has(this.peek().text))
      ) {
        negate = token;
        token = this.peek();
      }
      if (token.kind === "op" && token.text === "=" && !negate) {
        this.report(token, translate("'=' assigns; use '==' to compare"), {
          label: translate("Use '=='"),
          insert: "==",
        });
        token = { ...token, text: "==" };
      }
      const op = this.binary(token);
      if (!op || op.precedence < precedence) break;
      if (negate) this.advance();
      this.advance();

      if (token.text === "|") {
        const name = this.current;
        if (!this.is("ident")) {
          this.report(token, translate("'|' pipes into a function: value | round(2)"));
          return left;
        }
        this.advance();
        const args = this.is("punct", "(") ? this.parseArguments() : { args: [], parens: name };
        left = {
          kind: "call",
          callee: { kind: "ident", name: name.text, from: name.from, to: name.to },
          args: [left, ...args.args],
          parens: args.parens,
          from: left.from,
          to: args.parens.to,
        };
        continue;
      }

      if (previousOp === "??" && token.text !== "??") {
        this.report(
          token,
          translate("Wrap the '??' part in parentheses before '{0}': (a ?? 0) {0} b", token.text),
        );
      }

      const right = this.parseExpression(op.right ? op.precedence : op.precedence + 1);
      if (right.kind === "missing") {
        this.report(token, translate("'{0}' needs a value on its right", token.text));
      }
      const node: Node = {
        kind: "binary",
        op: token.text,
        left,
        right,
        opRange: token,
        from: left.from,
        to: right.to,
      };
      left = negate
        ? {
            kind: "unary",
            op: "not",
            operand: node,
            opRange: negate,
            from: negate.from,
            to: node.to,
          }
        : node;
      previousOp = token.text;
    }

    if (precedence <= 0 && this.isOp("?")) {
      return this.parseTernary(left);
    }
    return left;
  }

  private parseTernary(test: Node): Node {
    this.advance();
    let consequent: Node;
    let alternate: Node;
    if (this.isOp(":")) {
      this.advance();
      consequent = test;
      alternate = this.parseExpression(0);
    } else {
      consequent = this.parseExpression(0);
      if (this.isOp(":")) {
        this.advance();
        alternate = this.parseExpression(0);
      } else {
        this.report(
          this.current,
          translate("'?' needs ':' and the value for the other case: test ? a : b"),
        );
        alternate = this.missing(this.current.from);
      }
    }
    if (consequent.kind === "missing") {
      this.report(this.current, translate("'?' needs a value for when the test holds"));
    }
    return { kind: "ternary", test, consequent, alternate, from: test.from, to: alternate.to };
  }

  private parseIf(): Node {
    const ifToken = this.advance();
    const test = this.parseExpression(0);
    const consequent = this.parseBlock("if");
    let alternate: Node;
    if (this.is("keyword", "else")) {
      this.advance();
      alternate = this.is("keyword", "if") ? this.parseIf() : this.parseBlock("else");
    } else {
      this.report(ifToken, translate("'if' needs an 'else' with the other value"));
      alternate = this.missing(this.current.from);
    }
    return { kind: "if", test, consequent, alternate, from: ifToken.from, to: alternate.to };
  }

  private parseBlock(after: string): Node {
    if (!this.is("punct", "{")) {
      this.report(this.current, translate("'{0}' needs its value in braces after it", after));
      return this.missing(this.current.from);
    }
    const open = this.advance();
    const body = this.is("punct", "}") ? this.missing(this.current.from) : this.parseExpression(0);
    if (this.is("punct", "}")) {
      this.advance();
    } else {
      this.report(open, translate("'{0}' opened here is never closed", "{"));
    }
    return body;
  }

  private parseUnary(): Node {
    const token = this.current;
    const precedence =
      token.kind === "op" || token.kind === "keyword" ? UNARY_PRECEDENCE[token.text] : undefined;
    if (precedence !== undefined) {
      this.advance();
      const operand = this.parseExpression(precedence);
      if (operand.kind === "missing") {
        this.report(token, translate("'{0}' needs a value after it", token.text));
      }
      return {
        kind: "unary",
        op: token.text,
        operand,
        opRange: token,
        from: token.from,
        to: operand.to,
      };
    }
    return this.parsePostfix(this.parsePrimary());
  }

  private parsePostfix(node: Node): Node {
    for (;;) {
      if (this.is("punct", ".") || this.isOp("?.")) {
        const dot = this.advance();
        if (this.is("ident") || this.is("keyword")) {
          const property = this.advance();
          node = {
            kind: "member",
            object: node,
            property: property.text,
            propertyRange: property,
            optional: dot.text === "?.",
            from: node.from,
            to: property.to,
          };
          continue;
        }
        this.report(dot, translate("'.' needs a field name after it"));
        return node;
      }
      if (this.is("punct", "(")) {
        const { args, parens } = this.parseArguments();
        node = { kind: "call", callee: node, args, parens, from: node.from, to: parens.to };
        continue;
      }
      if (this.is("punct", "[")) {
        const open = this.advance();
        let start: Node | null = null;
        let end: Node | null = null;
        let slice = false;
        if (!this.isOp(":")) start = this.parseExpression(0);
        if (this.isOp(":")) {
          slice = true;
          this.advance();
          if (!this.is("punct", "]")) end = this.parseExpression(0);
        }
        const close = this.current;
        if (this.is("punct", "]")) {
          this.advance();
        } else {
          this.report(open, translate("'[' opened here is never closed"));
        }
        const to = this.is("eof") ? close.from : close.to;
        node = slice
          ? { kind: "slice", object: node, start, end, from: node.from, to }
          : {
              kind: "index",
              object: node,
              index: start ?? this.missing(open.to),
              from: node.from,
              to,
            };
        continue;
      }
      return node;
    }
  }

  private parseArguments(): { args: Node[]; parens: Range } {
    const open = this.advance();
    const args: Node[] = [];
    if (!this.is("punct", ")")) {
      for (;;) {
        if (this.is("punct", ")") || this.is("eof")) {
          this.report(
            this.tokens[this.index - 1],
            translate("Remove the ',' or add a value after it"),
          );
          break;
        }
        args.push(this.parseExpression(0));
        if (this.is("punct", ",")) {
          this.advance();
          continue;
        }
        break;
      }
    }
    if (this.is("punct", ")")) {
      const close = this.advance();
      return { args, parens: { from: open.from, to: close.to } };
    }
    this.report(open, translate("'(' opened here is never closed"));
    if (!this.is("eof") && !this.is("punct", ";")) {
      this.report(
        this.current,
        translate("Unexpected {0} inside the call", this.describe(this.current)),
      );
      this.recover();
    }
    const end = this.tokens[this.index - 1]?.to ?? open.to;
    return { args, parens: { from: open.from, to: end } };
  }

  private parsePrimary(): Node {
    const token = this.current;
    switch (token.kind) {
      case "number": {
        this.advance();
        return { kind: "number", value: Number(token.text), from: token.from, to: token.to };
      }
      case "string": {
        this.advance();
        const quote = token.text[0];
        const closed = token.text.length >= 2 && token.text.endsWith(quote);
        return {
          kind: "string",
          value: token.text.slice(1, closed ? -1 : undefined),
          from: token.from,
          to: token.to,
        };
      }
      case "pointer": {
        this.advance();
        return { kind: "pointer", from: token.from, to: token.to };
      }
      case "ident": {
        this.advance();
        return { kind: "ident", name: token.text, from: token.from, to: token.to };
      }
      case "keyword": {
        if (token.text === "true" || token.text === "false") {
          this.advance();
          return { kind: "bool", value: token.text === "true", from: token.from, to: token.to };
        }
        if (token.text === "nil") {
          this.advance();
          return { kind: "nil", from: token.from, to: token.to };
        }
        if (token.text === "if") {
          return this.parseIf();
        }
        if (token.text === "let") {
          this.report(
            token,
            translate("'let' lines come before the formula's amount, each ending with ';'"),
          );
          this.advance();
          return this.missing(token.to);
        }
        this.report(token, translate("Unexpected '{0}' here", token.text));
        this.advance();
        return this.missing(token.to);
      }
      case "punct": {
        if (token.text === "(") {
          this.advance();
          const inner = this.is("punct", ")")
            ? this.missing(this.current.from)
            : this.parseExpression(0);
          if (inner.kind === "missing") this.report(token, translate("'()' is empty"));
          if (this.is("punct", ")")) {
            const close = this.advance();
            return { ...inner, from: token.from, to: close.to } as Node;
          }
          this.report(token, translate("'(' opened here is never closed"));
          return inner;
        }
        if (token.text === "[") {
          this.advance();
          const items: Node[] = [];
          while (!this.is("punct", "]") && !this.is("eof")) {
            items.push(this.parseExpression(0));
            if (this.is("punct", ",")) this.advance();
            else break;
          }
          if (this.is("punct", "]")) {
            const close = this.advance();
            return { kind: "array", items, from: token.from, to: close.to };
          }
          this.report(token, translate("'[' opened here is never closed"));
          return { kind: "array", items, from: token.from, to: this.current.from };
        }
        if (token.text === "{") {
          return this.parseBraces();
        }
        if (
          token.text === "." &&
          (this.peek().kind === "ident" || this.peek().kind === "keyword")
        ) {
          // `.field` inside a predicate reads the current item's field.
          const dot = this.advance();
          const property = this.advance();
          return {
            kind: "member",
            object: { kind: "pointer", from: dot.from, to: dot.from },
            property: property.text,
            propertyRange: property,
            optional: false,
            from: dot.from,
            to: property.to,
          };
        }
        if (token.text === ")" || token.text === "]" || token.text === "}") {
          this.report(
            token,
            translate("'{0}' has no matching '{1}'", token.text, OPEN_FOR_CLOSE[token.text]),
          );
          this.advance();
          return this.parsePrimary();
        }
        this.report(token, translate("Unexpected '{0}'", token.text));
        this.advance();
        return this.missing(token.from);
      }
      case "op": {
        if (token.text === "=") {
          this.report(token, translate("'=' assigns; use '==' to compare"), {
            label: translate("Use '=='"),
            insert: "==",
          });
          this.advance();
          return this.missing(token.from);
        }
        this.report(token, translate("'{0}' needs a value before it", token.text));
        this.advance();
        return this.missing(token.from);
      }
      case "error": {
        this.advance();
        return this.missing(token.to);
      }
      case "comment":
      case "eof":
      default:
        return this.missing(token.from);
    }
  }

  private parseBraces(): Node {
    const open = this.advance();
    const entries: { key: string; value: Node }[] = [];
    const isMap =
      (this.is("ident") || this.is("string") || this.is("keyword") || this.is("number")) &&
      this.peek().text === ":";
    if (this.is("punct", "}")) {
      this.advance();
      return { kind: "map", entries, from: open.from, to: this.tokens[this.index - 1].to };
    }
    if (!isMap) {
      const body = this.parseExpression(0);
      if (this.is("punct", "}")) {
        const close = this.advance();
        return { kind: "predicate", body, from: open.from, to: close.to };
      }
      this.report(open, translate("'{0}' opened here is never closed", "{"));
      return { kind: "predicate", body, from: open.from, to: body.to };
    }
    while (!this.is("punct", "}") && !this.is("eof")) {
      const key = this.advance();
      if (!this.isOp(":")) {
        this.report(key, translate("A map entry is written key: value"));
        break;
      }
      this.advance();
      entries.push({ key: key.text, value: this.parseExpression(0) });
      if (this.is("punct", ",")) this.advance();
      else break;
    }
    if (this.is("punct", "}")) {
      const close = this.advance();
      return { kind: "map", entries, from: open.from, to: close.to };
    }
    this.report(open, translate("'{0}' opened here is never closed", "{"));
    return { kind: "map", entries, from: open.from, to: this.current.from };
  }
}

/** Parses the whole expression. Never throws; what is wrong is in `diagnostics`. */
export function parseExpression(
  source: string,
  operators: ReadonlySet<string> = WORD_OPERATORS,
): Program {
  const { tokens, diagnostics } = tokenize(source, operators);
  const empty = tokens.length === 1;
  if (empty) {
    return { statements: [], diagnostics, tokens, empty: true };
  }
  const parser = new Parser(tokens, operators);
  const statements = parser.parseProgram();
  const all = [...diagnostics, ...parser.diagnostics].sort((a, b) => a.from - b.from);
  return { statements, diagnostics: all, tokens, empty: false };
}

/** Every node of a tree, parents before children. */
export function* walk(node: Node): Generator<Node> {
  yield node;
  switch (node.kind) {
    case "member":
      yield* walk(node.object);
      break;
    case "index":
      yield* walk(node.object);
      yield* walk(node.index);
      break;
    case "slice":
      yield* walk(node.object);
      if (node.start) yield* walk(node.start);
      if (node.end) yield* walk(node.end);
      break;
    case "call":
      yield* walk(node.callee);
      for (const arg of node.args) yield* walk(arg);
      break;
    case "unary":
      yield* walk(node.operand);
      break;
    case "binary":
      yield* walk(node.left);
      yield* walk(node.right);
      break;
    case "ternary":
    case "if":
      yield* walk(node.test);
      yield* walk(node.consequent);
      yield* walk(node.alternate);
      break;
    case "array":
      for (const item of node.items) yield* walk(item);
      break;
    case "map":
      for (const entry of node.entries) yield* walk(entry.value);
      break;
    case "predicate":
      yield* walk(node.body);
      break;
    default:
      break;
  }
}

/** The innermost node whose span holds the position, with its ancestors outermost first. */
export function nodePath(program: Program, position: number): Node[] {
  const path: Node[] = [];
  const visit = (node: Node) => {
    if (position < node.from || position > node.to) return;
    path.push(node);
    for (const child of childrenOf(node)) {
      if (position >= child.from && position <= child.to) {
        visit(child);
        return;
      }
    }
  };
  for (const statement of program.statements) {
    if (position >= statement.from && position <= statement.to) {
      visit(statement.value);
      break;
    }
  }
  return path;
}

export function childrenOf(node: Node): Node[] {
  switch (node.kind) {
    case "member":
      return [node.object];
    case "index":
      return [node.object, node.index];
    case "slice":
      return [node.object, ...(node.start ? [node.start] : []), ...(node.end ? [node.end] : [])];
    case "call":
      return [node.callee, ...node.args];
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
    default:
      return [];
  }
}
