// Agent extensions: the marketplace, each extension's settings, and its connection test.
import { EXTENSION_CATALOG } from "../fixtures/aicontrol.mjs";

function extensions(state) {
  if (!state.ai.extensions) {
    state.ai.extensions = EXTENSION_CATALOG.items.map((item) => ({
      item: structuredClone(item),
      values: Object.fromEntries(
        item.configSpec.filter((field) => !field.sensitive).map((field) => [field.key, field.default ?? ""]),
      ),
      keyStored: false,
    }));
  }
  return state.ai.extensions;
}

function entryOf(state, type) {
  return extensions(state).find((entry) => entry.item.type === type) ?? null;
}

function configOf(entry) {
  return {
    type: entry.item.type,
    enabled: entry.item.enabled,
    availability: entry.item.availability,
    fields: entry.item.configSpec.map((field) =>
      field.sensitive
        ? { key: field.key, hasValue: entry.keyStored }
        : { key: field.key, value: entry.values[field.key] ?? "", hasValue: (entry.values[field.key] ?? "") !== "" },
    ),
    spec: entry.item.configSpec,
    version: entry.item.version,
    updatedAt: entry.item.updatedAt,
  };
}

const typeOf = (path) => decodeURIComponent(path.split("/")[4]);

export const EXTENSION_ROUTES = [
  [
    "GET",
    /^\/api\/v1\/agent-extensions\/catalog\/?$/,
    (state) => ({ ...EXTENSION_CATALOG, items: extensions(state).map((entry) => entry.item) }),
  ],
  [
    "GET",
    /^\/api\/v1\/agent-extensions\/[^/]+\/config\/?$/,
    (state, path) => {
      const entry = entryOf(state, typeOf(path));
      return entry ? configOf(entry) : { __status: 404, body: { message: "Extension not found" } };
    },
  ],
  [
    "PUT",
    /^\/api\/v1\/agent-extensions\/[^/]+\/config\/?$/,
    (state, path, body) => {
      const entry = entryOf(state, typeOf(path));
      if (!entry) return { __status: 404, body: { message: "Extension not found" } };
      if (body.version !== entry.item.version) {
        return { __status: 409, body: { type: "version_mismatch", message: "Someone else saved this extension" } };
      }
      for (const field of entry.item.configSpec) {
        const value = body.configuration?.[field.key];
        if (value === undefined) continue;
        if (field.sensitive) {
          if (value !== "") entry.keyStored = true;
        } else {
          entry.values[field.key] = value;
        }
      }
      const limit = Number(entry.values.dailyRequestLimit);
      Object.assign(entry.item, {
        enabled: !!body.enabled,
        availability: body.availability,
        configured: entry.keyStored,
        dailyRequestLimit: Number.isFinite(limit) && limit > 0 ? limit : entry.item.dailyRequestLimit,
        version: entry.item.version + 1,
        updatedAt: state.now(),
        enabledAt: body.enabled ? (entry.item.enabledAt ?? state.now()) : entry.item.enabledAt,
      });
      return configOf(entry);
    },
  ],
  [
    "POST",
    /^\/api\/v1\/agent-extensions\/[^/]+\/test\/?$/,
    (state, path) => {
      const entry = entryOf(state, typeOf(path));
      if (!entry) return { __status: 404, body: { message: "Extension not found" } };
      return {
        type: entry.item.type,
        success: entry.keyStored,
        checkedAt: state.now(),
        latencyMs: entry.keyStored ? 412 : 0,
        message: entry.keyStored ? "Exa answered a test search" : "No API key is saved",
      };
    },
  ],
];
