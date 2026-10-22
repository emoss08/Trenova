// Formula libraries: the organisation's Tariff `library` files, imported by templates as
// `use org.<name>`. GraphQL reads (the list, a library with its versions and reviews) and the REST
// writes under /formula-libraries/, as formulalibraryhandler serves them. A library is versioned
// and reviewed like a template; templates pin the approved version they were checked against, so
// editing or archiving a library never changes what an approved template version rates.
import { ORG, USER, userRecord } from "../fixtures/formula.mjs";
import { RulesError } from "../rules.mjs";
import {
  applyEdit,
  approvedVersion,
  business,
  checkOf,
  conflict,
  connection,
  diffContent,
  invalid,
  libraryPath,
  notFound,
  pinsOf,
  problem,
  publicReview,
  readinessOf,
  reviewDiffOf,
  reviewers,
  reviewFlow,
  rollbackRecord,
  studio,
  tariffContext,
} from "./formula.mjs";

const BASE = "/api/v1/formula-libraries";
const ID = "fl_[A-Za-z0-9_]+";
const NAME = /^[a-z][A-Za-z0-9]*$/;
const MAX_NAME = 64;
const KEYWORDS = new Set([
  "tariff",
  "library",
  "condition",
  "use",
  "input",
  "let",
  "var",
  "const",
  "fn",
  "type",
  "enum",
  "private",
  "charge",
  "if",
  "else",
  "for",
  "in",
  "where",
  "match",
  "return",
  "require",
  "test",
  "given",
  "expect",
  "minimum",
  "maximum",
  "round",
  "note",
  "true",
  "false",
  "none",
  "and",
  "or",
  "not",
]);

const libraryIdOf = (path) => new RegExp(`/formula-libraries/(${ID})`).exec(path)?.[1];

function libraryOf(state, path) {
  return studio(state).libraries.get(libraryIdOf(path)) ?? null;
}

const contextFor = (state, library) => tariffContext(state, { except: library?.id });

function derived(state, library, source) {
  const check = checkOf(source, "library", contextFor(state, library));
  return {
    imports: pinsOf(state, check),
    testCount: check.outline?.items.filter((item) => item.kind === "test").length ?? 0,
  };
}

function publicLibrary(state, library) {
  return {
    __typename: "FormulaLibrary",
    id: library.id,
    organizationId: ORG.id,
    businessUnitId: USER.businessUnitId,
    name: library.name,
    path: libraryPath(library),
    description: library.description,
    source: library.source,
    ...derived(state, library, library.source),
    status: library.status,
    submittedById: library.submittedById,
    submittedAt: library.submittedAt,
    approvedById: library.approvedById,
    approvedAt: library.approvedAt,
    reviewComment: library.reviewComment,
    version: library.version,
    sourceLibraryId: library.sourceLibraryId,
    sourceVersionNumber: library.sourceVersionNumber,
    currentVersionNumber: library.currentVersionNumber,
    createdAt: library.createdAt,
    updatedAt: library.updatedAt,
  };
}

function publicVersion(state, library, version) {
  const previous = library.versions.find((v) => v.versionNumber === version.versionNumber - 1);
  return {
    __typename: "FormulaLibraryVersion",
    id: `flv_${library.id.slice(3)}_${version.versionNumber}`,
    libraryId: library.id,
    versionNumber: version.versionNumber,
    name: version.content.name,
    path: `org.${version.content.name}`,
    description: version.content.description,
    source: version.content.source,
    ...derived(state, library, version.content.source),
    status: version.status,
    changeMessage: version.changeMessage ?? "",
    changeSummary: previous ? diffContent(previous.content, version.content) : null,
    createdById: version.createdById,
    createdBy: userRecord(version.createdById),
    createdAt: version.createdAt,
  };
}

/** Every template version that imports the library, and the library version each pinned. */
function usagesOf(state, library) {
  const path = libraryPath(library);
  const rows = [];
  for (const t of studio(state).templates.values()) {
    for (const version of t.versions) {
      if (!version.content.source.includes(`use ${path}`)) continue;
      const pinned =
        [...library.versions]
          .reverse()
          .find((v) => v.status === "Active" && v.createdAt <= version.createdAt) ??
        approvedVersion(library);
      if (!pinned) continue;
      rows.push({
        templateId: t.id,
        templateName: t.name,
        templateStatus: t.status,
        templateVersion: version.versionNumber,
        versionStatus: version.status,
        libraryVersionNumber: pinned.versionNumber,
      });
    }
  }
  return rows.sort(
    (a, b) => a.templateName.localeCompare(b.templateName) || b.templateVersion - a.templateVersion,
  );
}

// --- Writes ---------------------------------------------------------------------------------

function normalize(body, fallback = {}) {
  const pick = (key) => (body?.[key] !== undefined ? body[key] : (fallback[key] ?? ""));
  return {
    name: String(pick("name") ?? "").trim(),
    description: String(pick("description") ?? ""),
    source: String(pick("source") ?? ""),
  };
}

/** formulalibrary.Validate: a camelCase name that is no keyword and no other library's. */
function contentErrors(state, content, self) {
  const errors = [];
  const add = (field, message, code = "INVALID") => errors.push({ field, code, message });
  if (!content.name) add("name", "Name is required", "REQUIRED");
  else if (content.name.length > MAX_NAME) add("name", "Name cannot be longer than 64 characters");
  else if (!NAME.test(content.name)) {
    add(
      "name",
      "Name must be one camelCase word, such as accessorials or zoneRates, so it can be imported as org.<name>",
    );
  } else if (KEYWORDS.has(content.name))
    add("name", `${content.name} is a keyword of the formula language and cannot name a library`);
  else if (
    [...studio(state).libraries.values()].some((l) => l.id !== self?.id && l.name === content.name)
  ) {
    add("name", `A library named org.${content.name} already exists`);
  }
  if (!content.source.trim()) add("source", "The library's source is required", "REQUIRED");
  return errors;
}

const refused = (errors) =>
  problem(400, "validation-error", "Validation Failed", errors[0].message, errors);

function newLibrary(state, content, extra = {}) {
  const s = studio(state);
  const now = state.now();
  const id = `fl_studio_new_${s.nextId}`;
  s.nextId += 1;
  const library = {
    id,
    ...content,
    status: "Draft",
    submittedById: null,
    submittedAt: null,
    approvedById: null,
    approvedAt: null,
    reviewComment: "",
    version: 1,
    sourceLibraryId: extra.sourceLibraryId ?? null,
    sourceVersionNumber: extra.sourceVersionNumber ?? null,
    currentVersionNumber: 1,
    createdAt: now,
    updatedAt: now,
    versions: [
      {
        versionNumber: 1,
        status: "Draft",
        createdById: USER.id,
        createdAt: now,
        changeMessage: extra.changeMessage ?? "Initial version",
        content: { ...content },
      },
    ],
    reviews: [],
  };
  s.libraries.set(id, library);
  return library;
}

function create(state, _path, body) {
  const content = normalize(body);
  const errors = contentErrors(state, content, null);
  if (errors.length > 0) return refused(errors);
  return { __status: 201, body: publicLibrary(state, newLibrary(state, content)) };
}

function update(state, path, body) {
  const library = libraryOf(state, path);
  if (!library) return notFound("Formula library");
  if (body?.version !== undefined && Number(body.version) !== library.version)
    return conflict("library");
  const content = normalize(body);
  const errors = contentErrors(state, content, library);
  if (errors.length > 0) return refused(errors);
  applyEdit(state, library, content);
  return publicLibrary(state, library);
}

function freeName(state, name) {
  const taken = new Set([...studio(state).libraries.values()].map((l) => l.name));
  for (let n = 2; ; n += 1) if (!taken.has(`${name}${n}`)) return `${name}${n}`;
}

function duplicate(state, _path, body) {
  const ids = Array.isArray(body?.libraryIds) ? body.libraryIds : [];
  if (ids.length === 0 || ids.length > 50)
    return invalid("libraryIds", "Choose between 1 and 50 libraries to duplicate");
  const copies = [];
  for (const id of ids) {
    const library = studio(state).libraries.get(id);
    if (!library) return notFound("Formula library");
    const name = freeName(state, library.name);
    const source = library.source.replace(
      `library "${libraryPath(library)}"`,
      `library "org.${name}"`,
    );
    copies.push(
      newLibrary(
        state,
        { name, description: library.description, source },
        {
          sourceLibraryId: library.id,
          sourceVersionNumber: library.currentVersionNumber,
          changeMessage: `Duplicated from ${libraryPath(library)} v${library.currentVersionNumber}`,
        },
      ),
    );
  }
  return copies.map((library) => publicLibrary(state, library));
}

function archive(state, path, body) {
  const library = libraryOf(state, path);
  if (!library) return notFound("Formula library");
  if (library.status === "Inactive") return business("This library is already archived");
  const usages = usagesOf(state, library);
  if (usages.length > 0 && !body?.confirmInUse) {
    const names = [...new Set(usages.map((usage) => usage.templateName))];
    const message = `Templates import ${libraryPath(library)}: ${names.join(", ")}. They keep the version they pinned, but no new template version can import it. Confirm to archive it anyway`;
    return problem(422, "validation-error", "Validation Failed", message, [
      { field: "confirmInUse", code: "INVALID", message },
    ]);
  }
  Object.assign(library, {
    status: "Inactive",
    submittedById: null,
    submittedAt: null,
    version: library.version + 1,
    updatedAt: state.now(),
  });
  return publicLibrary(state, library);
}

function rollback(state, path, body) {
  const library = libraryOf(state, path);
  if (!library) return notFound("Formula library");
  return rollbackRecord(state, library, body, "Formula library") ?? publicLibrary(state, library);
}

function compare(state, path, _body, query) {
  const library = libraryOf(state, path);
  if (!library) return notFound("Formula library");
  const from = library.versions.find((v) => v.versionNumber === Number(query?.get("from")));
  const to = library.versions.find((v) => v.versionNumber === Number(query?.get("to")));
  if (!from || !to) return invalid("versions", "from and to must be version numbers");
  const changes = diffContent(from.content, to.content);
  return {
    fromVersion: from.versionNumber,
    toVersion: to.versionNumber,
    changes,
    changeCount: Object.keys(changes).length,
  };
}

const readiness = (state, library) =>
  readinessOf(state, library, {
    kind: "library",
    context: contextFor(state, library),
    label: "library",
  });

const libraryReview = reviewFlow({
  find: libraryOf,
  present: publicLibrary,
  readinessFor: readiness,
  label: "library",
  review: { idPrefix: "flr_", key: "libraryId" },
});

// --- GraphQL ---------------------------------------------------------------------------------

function formulaLibraryTable(state, v) {
  const rows = [...studio(state).libraries.values()].sort((a, b) => b.updatedAt - a.updatedAt);
  return { formulaLibraries: connection(rows, v, (library) => publicLibrary(state, library)) };
}

function formulaLibraryRecord(state, v) {
  const library = studio(state).libraries.get(v.id) ?? null;
  return {
    formulaLibrary: library ? publicLibrary(state, library) : null,
    formulaLibraryVersions: library
      ? [...library.versions].reverse().map((version) => publicVersion(state, library, version))
      : [],
    formulaLibraryReviews: library
      ? [...library.reviews]
          .sort((a, b) => b.createdAt - a.createdAt || b.id.localeCompare(a.id))
          .map((review) => ({
            __typename: "FormulaLibraryReview",
            ...publicReview(review, "libraryId"),
          }))
      : [],
  };
}

export const FORMULA_LIBRARY_HANDLERS = {
  FormulaLibraryTable: formulaLibraryTable,
  FormulaLibrary: formulaLibraryRecord,
};

// --- Routes ----------------------------------------------------------------------------------

const guarded = (handler) => (state, path, body, query) => {
  try {
    return handler(state, path, body, query);
  } catch (error) {
    if (error instanceof RulesError)
      return business(`The rules module refused the request: ${error.detail}`);
    throw error;
  }
};

const route = (rest) => new RegExp(`^${BASE}${rest}/?$`);
const one = (rest = "") => route(`/${ID}${rest}`);
const withLibrary = (fn) => (s, p) => {
  const library = libraryOf(s, p);
  return library ? fn(s, library) : notFound("Formula library");
};

export const FORMULA_LIBRARY_ROUTES = [
  ["POST", route(""), create],
  ["POST", route("/duplicate"), duplicate],
  ["PUT", one(), update],
  ["POST", one("/rollback"), rollback],
  ["POST", one("/archive"), archive],
  ["POST", one("/submit"), libraryReview.submit],
  ["POST", one("/approve"), libraryReview.approve],
  ["POST", one("/reject"), libraryReview.reject],
  ["POST", one("/request-changes"), libraryReview.requestChanges],
  ["GET", one("/readiness"), withLibrary(readiness)],
  [
    "GET",
    one("/review-diff"),
    withLibrary((s, library) => reviewDiffOf(s, library, "library", contextFor(s, library))),
  ],
  ["GET", one("/compare"), compare],
  ["GET", one("/usages"), withLibrary(usagesOf)],
  ["GET", one("/reviewers"), withLibrary(() => reviewers())],
].map(([method, pattern, handler]) => [method, pattern, guarded(handler)]);
