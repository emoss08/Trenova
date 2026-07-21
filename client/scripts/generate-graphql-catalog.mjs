import { readdir, readFile, writeFile } from "node:fs/promises";
import { dirname, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { Kind, parse, print } from "graphql";

const clientRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const operationsDir = resolve(clientRoot, "src/graphql/operations");
const srcDir = resolve(clientRoot, "src");
const generatedDir = resolve(clientRoot, "src/graphql/generated");
const persistedDocumentsPath = resolve(generatedDir, "persisted-documents.json");
const outputFile = resolve(generatedDir, "operation-catalog.json");

function toPosix(absolutePath) {
  return relative(clientRoot, absolutePath).split(sep).join("/");
}

async function collectFiles(dir, predicate, acc = []) {
  const entries = await readdir(dir, { withFileTypes: true });
  for (const entry of entries) {
    const full = resolve(dir, entry.name);
    if (entry.isDirectory()) {
      await collectFiles(full, predicate, acc);
    } else if (predicate(full)) {
      acc.push(full);
    }
  }
  return acc;
}

function domainOf(sourceFile) {
  const rest = relative(operationsDir, resolve(clientRoot, sourceFile)).split(sep);
  return rest.length > 1 ? rest[0] : "root";
}

function collectSpreadNames(selectionSet, acc) {
  if (!selectionSet) {
    return acc;
  }
  for (const selection of selectionSet.selections) {
    if (selection.kind === Kind.FRAGMENT_SPREAD) {
      acc.add(selection.name.value);
    } else if (selection.kind === Kind.FIELD || selection.kind === Kind.INLINE_FRAGMENT) {
      collectSpreadNames(selection.selectionSet, acc);
    }
  }
  return acc;
}

function fragmentClosure(seedNames, fragmentMap) {
  const closure = new Set();
  const queue = [...seedNames];
  while (queue.length > 0) {
    const name = queue.shift();
    if (closure.has(name)) {
      continue;
    }
    const fragment = fragmentMap.get(name);
    if (!fragment) {
      continue;
    }
    closure.add(name);
    for (const child of collectSpreadNames(fragment.node.selectionSet, new Set())) {
      if (!closure.has(child)) {
        queue.push(child);
      }
    }
  }
  return closure;
}

function renderSdl(primaryNode, fragmentNames, fragmentMap) {
  const parts = [print(primaryNode)];
  for (const name of [...fragmentNames].sort((a, b) => a.localeCompare(b))) {
    const fragment = fragmentMap.get(name);
    if (fragment) {
      parts.push(print(fragment.node));
    }
  }
  return parts.join("\n\n");
}

function variableList(operationNode) {
  return (operationNode.variableDefinitions ?? []).map((definition) => ({
    name: definition.variable.name.value,
    type: print(definition.type),
    defaultValue: definition.defaultValue ? print(definition.defaultValue) : null,
  }));
}

function rootFieldNames(selectionSet) {
  const fields = [];
  for (const selection of selectionSet.selections) {
    if (selection.kind === Kind.FIELD) {
      fields.push(selection.name.value);
    }
  }
  return fields;
}

async function buildHashMap() {
  const manifest = JSON.parse(await readFile(persistedDocumentsPath, "utf8"));
  const hashes = new Map();
  const operationPattern = /\b(?:query|mutation|subscription)\s+(\w+)/;
  for (const [hash, document] of Object.entries(manifest)) {
    const match = operationPattern.exec(document);
    if (match) {
      hashes.set(match[1], hash);
    }
  }
  return hashes;
}

async function buildUsageIndex() {
  const files = await collectFiles(
    srcDir,
    (file) =>
      (file.endsWith(".ts") || file.endsWith(".tsx")) &&
      !file.startsWith(generatedDir) &&
      !file.endsWith(".d.ts"),
  );
  const operationUsage = new Map();
  const fragmentUsage = new Map();
  const tokenPattern = /\b([A-Z]\w*?)(Document|FragmentDoc)\b/g;

  for (const file of files) {
    const contents = await readFile(file, "utf8");
    const posix = toPosix(file);
    let match;
    tokenPattern.lastIndex = 0;
    while ((match = tokenPattern.exec(contents)) !== null) {
      const [, name, suffix] = match;
      const index = suffix === "Document" ? operationUsage : fragmentUsage;
      let bucket = index.get(name);
      if (!bucket) {
        bucket = new Set();
        index.set(name, bucket);
      }
      bucket.add(posix);
    }
  }

  return { operationUsage, fragmentUsage };
}

async function main() {
  const files = await collectFiles(operationsDir, (file) => file.endsWith(".graphql"));
  files.sort((a, b) => a.localeCompare(b));

  const fragmentMap = new Map();
  const operationRecords = [];

  for (const file of files) {
    const sourceFile = toPosix(file);
    const document = parse(await readFile(file, "utf8"), { noLocation: false });
    for (const definition of document.definitions) {
      if (definition.kind === Kind.FRAGMENT_DEFINITION) {
        if (!fragmentMap.has(definition.name.value)) {
          fragmentMap.set(definition.name.value, {
            node: definition,
            sourceFile,
            domain: domainOf(sourceFile),
            typeCondition: definition.typeCondition.name.value,
          });
        }
      } else if (definition.kind === Kind.OPERATION_DEFINITION && definition.name) {
        operationRecords.push({ node: definition, sourceFile });
      }
    }
  }

  const [hashMap, usageIndex] = await Promise.all([buildHashMap(), buildUsageIndex()]);

  const operations = operationRecords
    .map(({ node, sourceFile }) => {
      const name = node.name.value;
      const fragments = fragmentClosure(
        collectSpreadNames(node.selectionSet, new Set()),
        fragmentMap,
      );
      return {
        name,
        kind: node.operation,
        domain: domainOf(sourceFile),
        sourceFile,
        hash: hashMap.get(name) ?? null,
        rootFields: rootFieldNames(node.selectionSet),
        variables: variableList(node),
        fragments: [...fragments].sort((a, b) => a.localeCompare(b)),
        usages: [...(usageIndex.operationUsage.get(name) ?? [])].sort((a, b) => a.localeCompare(b)),
        sdl: renderSdl(node, fragments, fragmentMap),
      };
    })
    .sort((a, b) => a.name.localeCompare(b.name));

  const fragments = [...fragmentMap.entries()]
    .map(([name, fragment]) => {
      const nested = fragmentClosure(
        collectSpreadNames(fragment.node.selectionSet, new Set()),
        fragmentMap,
      );
      const usedByOperations = operations
        .filter((operation) => operation.fragments.includes(name))
        .map((operation) => operation.name);
      return {
        name,
        typeCondition: fragment.typeCondition,
        domain: fragment.domain,
        sourceFile: fragment.sourceFile,
        fragments: [...nested].filter((child) => child !== name).sort((a, b) => a.localeCompare(b)),
        usedByOperations,
        usages: [...(usageIndex.fragmentUsage.get(name) ?? [])].sort((a, b) => a.localeCompare(b)),
        sdl: renderSdl(fragment.node, nested, fragmentMap),
      };
    })
    .sort((a, b) => a.name.localeCompare(b.name));

  const catalog = {
    operationCount: operations.length,
    fragmentCount: fragments.length,
    operations,
    fragments,
  };

  await writeFile(outputFile, `${JSON.stringify(catalog, null, 2)}\n`, "utf8");
  process.stdout.write(
    `Wrote GraphQL catalog: ${operations.length} operations, ${fragments.length} fragments -> ${toPosix(outputFile)}\n`,
  );
}

main().catch((error) => {
  process.stderr.write(`Failed to generate GraphQL catalog: ${error?.stack ?? error}\n`);
  process.exitCode = 1;
});
