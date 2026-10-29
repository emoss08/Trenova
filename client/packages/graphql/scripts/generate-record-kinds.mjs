// generate-record-kinds.mjs mirrors the server's kinds of record into TypeScript.
//
// A kind of record is named in many places on both sides: where it opens
// (record-links.ts), what a page or an @ mention may name, which agent subject
// it is, which tab the @ search shows it under. Each of those lists was kept by
// hand and they drifted: a record link the server did not know refused every
// message asked from its page. The server owns what kinds exist, so this
// generator carries them to the client, where every table keyed by a kind is
// typed against them and a kind the server does not know fails the typecheck.
//
// Sources, all Go maps of plain literals:
//   permission/resource_gen.go   resource names
//   recordkind/recordid.go       the resources whose records are one kind, and their id prefixes
//   recordkind/recordkind.go     the kinds a resource's name cannot stand for, and their prefixes
//   agent/enums.go, agent/subjectrecord.go   the kind of record each agent subject is
//   assistantservice/assistantthreads/mentions.go the kinds the @ search offers, and the tab each is under
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(packageRoot, "../../..");

const PERMISSION = "services/tms/internal/modules/identity/domain/permission";
const RECORD_KIND = "services/tms/internal/modules/identity/domain/recordkind";
const AGENT = "services/tms/internal/modules/agent/domain/agent";
const SOURCES = {
  resources: `${PERMISSION}/resource_gen.go`,
  recordIDs: `${RECORD_KIND}/recordid.go`,
  recordKinds: `${RECORD_KIND}/recordkind.go`,
  subjectTypes: `${AGENT}/enums.go`,
  subjectKinds: `${AGENT}/subjectrecord.go`,
  mentions: "services/tms/internal/modules/agent/app/assistantservice/assistantthreads/mentions.go",
};
const OUTPUT = "client/packages/shared/src/types/generated/record-kinds.ts";

function fail(message) {
  throw new Error(
    `${message} Update the parser in scripts/generate-record-kinds.mjs if the Go ` +
      `declaration style changed; never hand-edit the generated output.`,
  );
}

/** The body of a top-level `var name = …{ … }` declaration, braces balanced. */
function block(source, file, name) {
  const start = source.search(new RegExp(`^var ${name} = [^{]*\\{`, "m"));
  if (start < 0) {
    fail(`No var ${name} in ${file}.`);
  }
  const open = source.indexOf("{", start);
  let depth = 0;
  for (let i = open; i < source.length; i += 1) {
    if (source[i] === "{") depth += 1;
    if (source[i] === "}") depth -= 1;
    if (depth === 0) {
      return source.slice(open + 1, i);
    }
  }

  return fail(`Unbalanced braces in var ${name} in ${file}.`);
}

function constants(source, file, pattern) {
  const found = new Map();
  for (const match of source.matchAll(pattern)) {
    found.set(match[1], match[2]);
  }
  if (found.size === 0) {
    fail(`No constants matching ${pattern} in ${file}.`);
  }

  return found;
}

function entries(text, file, pattern, what) {
  const found = [...text.matchAll(pattern)];
  if (found.length === 0) {
    fail(`No ${what} entries in ${file}.`);
  }

  return found;
}

async function read(file) {
  return readFile(resolve(repoRoot, file), "utf8");
}

const [resourcesGo, recordIDsGo, recordKindsGo, subjectTypesGo, subjectKindsGo, mentionsGo] =
  await Promise.all(Object.values(SOURCES).map(read));

const resources = constants(
  resourcesGo,
  SOURCES.resources,
  /^\s*Resource(\w+)\s+Resource\s*=\s*"([^"]+)"/gm,
);
const kindNames = constants(
  recordKindsGo,
  SOURCES.recordKinds,
  /^\s*(\w+)\s+Kind\s*=\s*"([^"]+)"/gm,
);
const subjectTypes = constants(
  subjectTypesGo,
  SOURCES.subjectTypes,
  /^\s*Subject(\w+)\s*=\s*SubjectType\("([^"]+)"\)/gm,
);

const prefixes = new Map();
function addKind(kind, prefix, from) {
  if (prefixes.has(kind)) {
    fail(`Record kind "${kind}" is declared twice (${from}).`);
  }
  prefixes.set(kind, prefix);
}

for (const [, name, prefix] of entries(
  block(recordIDsGo, SOURCES.recordIDs, "recordIDPrefixes"),
  SOURCES.recordIDs,
  /Resource(\w+):\s*"([^"]+)"/g,
  "recordIDPrefixes",
)) {
  const kind =
    resources.get(name) ?? fail(`Resource${name} is not declared in ${SOURCES.resources}.`);
  addKind(kind, prefix, SOURCES.recordIDs);
}
for (const [, name, prefix] of entries(
  block(recordKindsGo, SOURCES.recordKinds, "recordKinds"),
  SOURCES.recordKinds,
  /(\w+):\s*\{\s*"([^"]+)"/g,
  "recordKinds",
)) {
  const kind = kindNames.get(name) ?? fail(`${name} is not declared in ${SOURCES.recordKinds}.`);
  addKind(kind, prefix, SOURCES.recordKinds);
}

function knownKind(kind, from) {
  if (!prefixes.has(kind)) {
    fail(`${from} names "${kind}", which is not a record kind in ${RECORD_KIND}.`);
  }

  return kind;
}

const subjects = entries(
  block(subjectKindsGo, SOURCES.subjectKinds, "subjectRecordKinds"),
  SOURCES.subjectKinds,
  /Subject(\w+):\s*"([^"]+)"/g,
  "subjectRecordKinds",
).map(([, name, kind]) => [
  subjectTypes.get(name) ?? fail(`Subject${name} is not declared in ${SOURCES.subjectTypes}.`),
  knownKind(kind, SOURCES.subjectKinds),
]);

const mentions = entries(
  block(mentionsGo, SOURCES.mentions, "mentionKinds"),
  SOURCES.mentions,
  /\{\s*kind:\s*"([^"]+)"(?:,\s*under:\s*"([^"]+)")?(,\s*askedFor:\s*true)?\s*\}/g,
  "mentionKinds",
).map(([, kind, under, askedFor]) => ({
  kind: knownKind(kind, SOURCES.mentions),
  tab: under ?? kind,
  askedFor: askedFor !== undefined,
}));

function objectBody(pairs) {
  return pairs.map(([key, value]) => `  ${key}: "${value}",`).join("\n");
}

function render() {
  const kinds = [...prefixes.entries()].sort(([a], [b]) => a.localeCompare(b));
  const sortedSubjects = [...subjects].sort(([a], [b]) => a.localeCompare(b));
  const askedFor = mentions.filter((mention) => mention.askedFor).map(({ kind }) => `"${kind}"`);

  return `// Code generated by client/packages/graphql/scripts/generate-record-kinds.mjs. DO NOT EDIT.
//
// Source of truth: ${RECORD_KIND} (record kinds), ${SOURCES.subjectKinds}
// (agent subjects) and ${SOURCES.mentions} (@ search).
//
// Regenerate with: pnpm --filter @trenova/graphql codegen
// Drift between these values and the Go source fails the codegen:check CI gate.

/** Every kind of record the server knows, with the prefix each of its ids carries. */
export const RecordKindIDPrefix = {
${objectBody(kinds)}
} as const;

export type RecordKind = keyof typeof RecordKindIDPrefix;

/** The kind of record each agent subject (\`AgentSubjectType\`) is. */
export const AgentSubjectRecordKind = {
${objectBody(sortedSubjects)}
} as const satisfies Record<string, RecordKind>;

/** The kinds of record the @ search offers, in the order offered, and the tab each sits under. */
export const MentionKindTab = {
${objectBody(mentions.map(({ kind, tab }) => [kind, tab]))}
} as const satisfies Partial<Record<RecordKind, string>>;

export type MentionKind = keyof typeof MentionKindTab;
export type MentionTab = (typeof MentionKindTab)[MentionKind];

/** Kinds the @ search offers only when their own tab is asked for, never across every kind. */
export const MentionKindsAskedFor: readonly MentionKind[] = [${askedFor.join(", ")}];
`;
}

const outputPath = resolve(repoRoot, OUTPUT);
await mkdir(dirname(outputPath), { recursive: true });
await writeFile(outputPath, render(), "utf8");

process.stdout.write(
  `record kinds: ${prefixes.size}, agent subjects: ${subjects.length}, mention kinds: ` +
    `${mentions.length} -> ${relative(repoRoot, outputPath)}\n`,
);
