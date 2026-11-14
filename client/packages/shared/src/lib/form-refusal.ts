/**
 * The `type` a form error carries when the server set it by refusing a save, as opposed to
 * one the form's own validation found. The form's refusal alert lists only these.
 */
export const SERVER_ERROR_TYPE = "validation";

/**
 * Every message of a server refusal held in a form's errors: the form-wide one first, then
 * each field's, nested ones included, each once and in the order the form holds them. A
 * field the server said several things about shows the first and keeps them all under
 * `types`; all of them are returned.
 */
export function refusalMessages(errors: object): string[] {
  const messages = new Set<string>();
  const add = (message: unknown) => {
    if (typeof message === "string" && message.trim() !== "") {
      messages.add(message);
    }
  };
  const visit = (node: unknown, key: string) => {
    if (!node || typeof node !== "object" || key === "ref") {
      return;
    }
    const { message, type, types } = node as { message?: unknown; type?: unknown; types?: unknown };
    if (type === SERVER_ERROR_TYPE) {
      add(message);
      if (types && typeof types === "object") {
        Object.values(types).forEach(add);
      }
      return;
    }
    for (const [child, value] of Object.entries(node)) {
      visit(value, child);
    }
  };

  const { root, ...fields } = errors as Record<string, unknown>;
  visit(root, "root");
  for (const [key, value] of Object.entries(fields)) {
    visit(value, key);
  }
  return [...messages];
}
