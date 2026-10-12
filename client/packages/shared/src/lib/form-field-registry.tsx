import { createContext, useContext, useEffect, useRef, type ReactNode } from "react";

/** Writes one of a field's values the way the field shows it, such as a record's name. */
export type FieldValueFormat = (value: unknown) => ReactNode | undefined;

/**
 * The fields a form, or one section of it, has drawn, each with the label a
 * person sees and, where the value is not readable as stored (a record's ID, an
 * option's key), how the field writes it. Fields register when they mount and
 * leave when they unmount, so a change review can name and show a field the way
 * the form does and a section can tell which fields it holds, without either
 * being told.
 *
 * Registration is the only write and happens once per mount; typing never touches
 * it. Notifications for registrations that land in the same tick are batched into
 * one, so a form mounting fifty fields wakes its subscribers once. Formats are read
 * only when a review asks for them and never notify.
 */
export type FieldRegistry = {
  register: (name: string, label: string | undefined) => () => void;
  setFormat: (name: string, format: FieldValueFormat) => () => void;
  label: (name: string) => string | undefined;
  format: (name: string) => FieldValueFormat | undefined;
  names: () => readonly string[];
  subscribe: (listener: () => void) => () => void;
  version: () => number;
};

export function createFieldRegistry(): FieldRegistry {
  const entries = new Map<string, { label: string | undefined; count: number }>();
  const formats = new Map<string, FieldValueFormat>();
  const listeners = new Set<() => void>();
  let version = 0;
  let names: readonly string[] = [];
  let scheduled = false;

  const notify = () => {
    if (scheduled) return;
    scheduled = true;
    queueMicrotask(() => {
      scheduled = false;
      version += 1;
      names = Array.from(entries.keys());
      for (const listener of listeners) listener();
    });
  };

  return {
    register(name, label) {
      const entry = entries.get(name);
      if (entry) {
        entry.count += 1;
        if (label) entry.label = label;
      } else {
        entries.set(name, { label, count: 1 });
        notify();
      }
      return () => {
        const current = entries.get(name);
        if (!current) return;
        current.count -= 1;
        if (current.count === 0) {
          entries.delete(name);
          notify();
        }
      };
    },
    setFormat(name, format) {
      formats.set(name, format);
      return () => {
        if (formats.get(name) === format) formats.delete(name);
      };
    },
    label: (name) => entries.get(name)?.label,
    format: (name) => formats.get(name),
    names: () => names,
    subscribe(listener) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    version: () => version,
  };
}

const FormFieldsContext = createContext<FieldRegistry | null>(null);
const SectionFieldsContext = createContext<FieldRegistry | null>(null);
const FieldNameContext = createContext<string | undefined>(undefined);

export function FormFieldsProvider({
  registry,
  children,
}: {
  registry: FieldRegistry;
  children: ReactNode;
}) {
  return <FormFieldsContext.Provider value={registry}>{children}</FormFieldsContext.Provider>;
}

export function SectionFieldsProvider({
  registry,
  children,
}: {
  registry: FieldRegistry;
  children: ReactNode;
}) {
  return (
    <SectionFieldsContext.Provider value={registry}>{children}</SectionFieldsContext.Provider>
  );
}

/** Names the field a control is drawn in, so the control can say how its values read. */
export function FieldNameProvider({ name, children }: { name: string; children: ReactNode }) {
  return <FieldNameContext.Provider value={name}>{children}</FieldNameContext.Provider>;
}

/** The registry of the form this is drawn in, or null outside one. */
export function useFormFieldRegistry(): FieldRegistry | null {
  return useContext(FormFieldsContext);
}

/**
 * Registers how a field writes its values. The function is held behind a ref, so
 * one that is a new closure every render costs nothing and never re-registers.
 */
function useValueFormat(name: string | undefined, format: FieldValueFormat | undefined) {
  const form = useContext(FormFieldsContext);
  const latest = useRef(format);
  useEffect(() => {
    latest.current = format;
  });
  const hasFormat = format !== undefined;

  useEffect(() => {
    if (!name || !form || !hasFormat) return;
    return form.setFormat(name, (value) => latest.current?.(value));
  }, [name, form, hasFormat]);
}

/**
 * Tells the form and the section a field is drawn in that it is there, what it is
 * called and, given a format, how its values read. A field outside any form
 * registers nowhere and costs nothing.
 */
export function useFieldRegistration(
  name: string | undefined,
  label: ReactNode,
  format?: FieldValueFormat,
) {
  const form = useContext(FormFieldsContext);
  const section = useContext(SectionFieldsContext);
  const text = typeof label === "string" ? label : undefined;

  useEffect(() => {
    if (!name || (!form && !section)) return;
    const leaveForm = form?.register(name, text);
    const leaveSection = section?.register(name, text);
    return () => {
      leaveForm?.();
      leaveSection?.();
    };
  }, [name, text, form, section]);

  useValueFormat(name, format);
}

/**
 * For a control drawn inside a field wrapper, such as a record picker: says how the
 * field's values read, using the field's name from the wrapper around it.
 */
export function useFieldValueFormat(format: FieldValueFormat) {
  useValueFormat(useContext(FieldNameContext), format);
}
