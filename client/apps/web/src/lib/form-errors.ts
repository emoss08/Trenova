// The server addresses fields with bracket indices (`lines[0].glAccountId`, from
// errortypes.MultiError.WithIndex); react-hook-form registers them with dot indices
// (`lines.0.glAccountId`). Without this translation every nested field error would look
// unregistered and get routed to the form root.
export function normalizeFieldPath(field: string): string {
  return field.replace(/\[(\d+)\]/g, ".$1").replace(/^\./, "");
}

type RegisteredField = { _f?: unknown };

// Structural rather than Control<T>: react-hook-form's Control carries an invariant
// transformed-values generic, and this only ever reads the registry.
export type FieldRegistry = { _fields?: Record<string, unknown> };

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
 * The messages of a server refusal that land on a field of the form, each once. The
 * ones for no field of the form go to the form-wide root instead and are said there.
 */
export function refusedFieldMessages(
  fieldErrors: readonly { field: string; message: string }[],
  registry: FieldRegistry,
): string[] {
  return [
    ...new Set(
      fieldErrors
        .filter((error) => isRegisteredLeaf(registry, normalizeFieldPath(error.field)))
        .map((error) => error.message),
    ),
  ];
}
