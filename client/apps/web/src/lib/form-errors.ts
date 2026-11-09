type RegisteredField = { _f?: unknown };

// Structural rather than Control<T>: react-hook-form's Control carries an invariant
// transformed-values generic, and this only ever reads the registry, the values and
// whether the form validates with a resolver.
export type FieldRegistry = {
  _fields?: Record<string, unknown>;
  _formValues?: unknown;
  _options?: { resolver?: unknown };
};

// isRegisteredLeaf answers the only question that matters before calling setError: is
// there an input mounted against this exact path?
//
// Probing `form.getValues(path)` does NOT answer it. `getValues("lines[0]")` returns a
// populated object because defaultValues are fully shaped, so a non-leaf sails through
// and setError lands on a node no input renders — an error the user can neither see nor
// clear. react-hook-form's field registry is the real source: only a registered leaf
// carries the `_f` descriptor.
//
// Reads `control._fields` rather than `control._names.mount` deliberately: mount is
// pruned on unmount, so a field on an inactive tab of a tabbed panel would be
// misclassified as unknown and its error hidden in the root summary instead of waiting
// on the tab that owns it.
export function isRegisteredLeaf(control: FieldRegistry, path: string): boolean {
  if (path === "") {
    return false;
  }

  const fields = control._fields;
  if (!fields) {
    return false;
  }

  let node: unknown = fields;
  for (const segment of path.split(".")) {
    if (node === null || typeof node !== "object") {
      return false;
    }
    node = (node as Record<string, unknown>)[segment];
  }

  return node !== null && typeof node === "object" && (node as RegisteredField)._f !== undefined;
}

// The parts of each form that a component shows every error under, by the path that
// heads them, each with the number of components showing it. Keyed by the form's
// control, so a form's surfaces leave with it.
const errorSurfaces = new WeakMap<object, Map<string, number>>();

/**
 * Declares that a component shows every error under `prefix` (a form path such as
 * `moves.0`), whether or not a field is mounted there, until the returned function is
 * called. A server error on a path under it is then set on that path, where the component
 * shows it, instead of being sent to the form root.
 *
 * A move card is one: it says each stop's problems, while the stops' fields mount only in
 * the move editor.
 */
export function addErrorSurface(control: object, prefix: string): () => void {
  let surfaces = errorSurfaces.get(control);
  if (!surfaces) {
    surfaces = new Map();
    errorSurfaces.set(control, surfaces);
  }
  surfaces.set(prefix, (surfaces.get(prefix) ?? 0) + 1);
  let removed = false;
  return () => {
    if (removed) return;
    removed = true;
    const count = surfaces.get(prefix) ?? 0;
    if (count <= 1) surfaces.delete(prefix);
    else surfaces.set(prefix, count - 1);
  };
}

function isUnderErrorSurface(control: object, path: string): boolean {
  const surfaces = errorSurfaces.get(control);
  if (!surfaces) return false;
  for (const prefix of surfaces.keys()) {
    if (path === prefix || path.startsWith(`${prefix}.`)) return true;
  }
  return false;
}

function valueAt(control: FieldRegistry, path: string): unknown {
  let node: unknown = control._formValues;
  for (const segment of path.split(".")) {
    if (node === null || typeof node !== "object") return undefined;
    node = (node as Record<string, unknown>)[segment];
  }
  return node;
}

/**
 * Where a server error on `path` belongs on the form, or null when nothing on the form
 * would show it and it belongs on the form root.
 *
 * A registered leaf shows its own error. A path under an error surface is shown by the
 * surface: on the path itself for a value, and under `root` for an object or a list, where
 * react-hook-form keeps an error on a whole (a field array's own error), beside the errors
 * of the fields inside it. A surface counts only on a form that validates with a resolver:
 * each submit replaces every error with what the resolver finds, so a server error on a
 * path no field registers cannot outlive the attempt that earned it. A form validated
 * field by field would never clear it, and could never be submitted again.
 */
export function serverErrorTarget(control: FieldRegistry, path: string): string | null {
  if (isRegisteredLeaf(control, path)) return path;
  if (path === "" || !control._options?.resolver || !isUnderErrorSurface(control, path)) {
    return null;
  }
  const value = valueAt(control, path);
  return value !== null && typeof value === "object" ? `${path}.root` : path;
}

/**
 * Every message in a form's errors, nested ones included, each once and in the order
 * the form holds them. The form-wide root message is left out: it has its own channel.
 */
export function fieldErrorMessages(errors: object): string[] {
  const messages = new Set<string>();
  const visit = (node: unknown, key: string) => {
    if (!node || typeof node !== "object" || key === "root" || key === "ref") {
      return;
    }
    const message = (node as { message?: unknown }).message;
    if (typeof message === "string" && message.trim() !== "") {
      messages.add(message);
    }
    for (const [child, value] of Object.entries(node)) {
      visit(value, child);
    }
  };
  for (const [key, value] of Object.entries(errors)) {
    visit(value, key);
  }
  return [...messages];
}

/**
 * The messages of a server refusal that land on the form (a field, or a surface that
 * shows the path), each once. The ones nothing on the form shows go to the form-wide
 * root instead and are said there.
 */
export function refusedFieldMessages(
  fieldErrors: readonly { field: string; message: string }[],
  registry: FieldRegistry,
): string[] {
  return [
    ...new Set(
      fieldErrors
        .filter((error) => serverErrorTarget(registry, error.field) !== null)
        .map((error) => error.message),
    ),
  ];
}
