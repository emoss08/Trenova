// lading-service.ts is what the Lading editor's extensions read: the loaded EDI module, the
// project as the studio holds it now (every file's current text and the samples), the context it
// is checked against, the file being edited, and the last run's values. Each is a getter, so a
// change never rebuilds the editor. `checkProject` asks `lading.check` once per project: the
// linter, the folding and the studio's panels share the answer.

import { Facet } from "@codemirror/state";
import {
  ladingCheck,
  type LadingCheck,
  type LadingCheckedFile,
  type LadingContext,
  type LadingEdit,
  type LadingLocation,
  type LadingProjectInput,
  type LadingRunValue,
} from "@trenova/shared/edi/lading";
import type { EdiRuntime } from "@trenova/shared/edi/runtime";
import { ModuleRequestError } from "@trenova/shared/lib/wasm-module";

export type LadingLanguageService = {
  edi: () => EdiRuntime;
  /** The project as the studio holds it, the edited file's text included. */
  project: () => LadingProjectInput;
  context: () => LadingContext;
  /** The path of the file the editor shows. */
  path: () => string;
  /** The last run's values, for hover and inlay hints. */
  values: () => readonly LadingRunValue[];
  /** Opens a location: in this file it moves the cursor, in another the studio opens it. */
  openLocation: (location: LadingLocation) => void;
  /** Applies a rename's or a refactoring's edits, which may reach other files of the project. */
  applyEdits: (edits: readonly LadingEdit[]) => void;
};

export const ladingServiceFacet = Facet.define<LadingLanguageService, LadingLanguageService | null>(
  { combine: (values) => values[0] ?? null },
);

const EMPTY_CHECK: LadingCheck = {
  files: [],
  partner: null,
  maps: [],
  flows: [],
  rules: [],
  libraries: [],
  guides: [],
  errors: 0,
  warnings: 0,
};

const checks = new WeakMap<LadingProjectInput, LadingCheck>();

/** The module's check of a project, asked once per project object and context. */
export function checkProject(
  edi: EdiRuntime,
  project: LadingProjectInput,
  context: LadingContext,
): LadingCheck {
  const cached = checks.get(project);
  if (cached) return cached;
  const check = unlessRefused(() => ladingCheck(edi, { project, context }), EMPTY_CHECK);
  checks.set(project, check);
  return check;
}

/** The check of the file the editor shows, or null while the project has none at its path. */
export function checkedFile(service: LadingLanguageService): LadingCheckedFile | null {
  const check = checkProject(service.edi(), service.project(), service.context());
  return check.files.find((file) => file.path === service.path()) ?? null;
}

/**
 * The project with the edited file as the editor holds it now. The studio learns of an edit after
 * the editor's transaction, so a tool answering inside one reads the file from the editor.
 */
export function liveProject(service: LadingLanguageService, source: string): LadingProjectInput {
  const project = service.project();
  const path = service.path();
  if (project.files.find((file) => file.path === path)?.source === source) return project;
  return {
    ...project,
    files: project.files.map((file) => (file.path === path ? { path, source } : file)),
  };
}

const reported = new Set<string>();

/**
 * Calls one of the module's operations for the editor. A request the module refuses is a caller
 * bug: it is logged once per operation and the editor carries on as if the operation had nothing
 * to offer, so typing never breaks. A trap or any other failure still throws.
 */
export function unlessRefused<T>(ask: () => T, nothing: T): T {
  try {
    return ask();
  } catch (error) {
    if (!(error instanceof ModuleRequestError)) throw error;
    if (!reported.has(error.op)) {
      reported.add(error.op);
      console.error(error);
    }
    return nothing;
  }
}
