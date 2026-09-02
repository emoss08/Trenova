import type { FormulaReceipt } from "@trenova/shared/types/formula-template";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  type ReactNode,
} from "react";
import type { HoverVariable } from "./expr-hover";

type PreviewValuesContextValue = {
  getValues: () => ReadonlyMap<string, HoverVariable>;
};

const EMPTY: ReadonlyMap<string, HoverVariable> = new Map();

const PreviewValuesContext = createContext<PreviewValuesContextValue | null>(null);

/**
 * Publishes the latest preview receipt's variables to every editor beneath,
 * through a ref so a new preview never rebuilds an editor's extensions.
 */
export function PreviewValuesProvider({
  receipt,
  children,
}: {
  receipt: FormulaReceipt | null | undefined;
  children: ReactNode;
}) {
  const valuesRef = useRef<ReadonlyMap<string, HoverVariable>>(EMPTY);

  useEffect(() => {
    valuesRef.current = receipt
      ? new Map(
          receipt.variables.map((variable) => [
            variable.name,
            { name: variable.name, value: variable.value, source: variable.source },
          ]),
        )
      : EMPTY;
  }, [receipt]);

  const getValues = useCallback(() => valuesRef.current, []);
  const value = useMemo(() => ({ getValues }), [getValues]);

  return <PreviewValuesContext.Provider value={value}>{children}</PreviewValuesContext.Provider>;
}

/** The getter editors hand to the hover extension; empty outside a provider. */
export function usePreviewValues(): () => ReadonlyMap<string, HoverVariable> {
  const context = useContext(PreviewValuesContext);
  return context?.getValues ?? (() => EMPTY);
}
