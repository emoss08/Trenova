// src/components/form/form-save-context.tsx
import { ReactNode, createContext, useContext, useState } from "react";

interface FormSaveContextType {
  lastSaved: string | null;
  setLastSavedNow: () => void;
  resetLastSaved: () => void;
}

const FormSaveContext = createContext<FormSaveContextType | undefined>(
  undefined,
);

interface FormSaveProviderProps {
  children: ReactNode;
}

export function FormSaveProvider({ children }: FormSaveProviderProps) {
  const [lastSaved, setLastSaved] = useState<string | null>(null);

  const setLastSavedNow = () => {
    setLastSaved(new Date().toLocaleTimeString());
  };

  const resetLastSaved = () => {
    setLastSaved(null);
  };

  return (
    <FormSaveContext.Provider
      value={{
        lastSaved,
        setLastSavedNow,
        resetLastSaved,
      }}
    >
      {children}
    </FormSaveContext.Provider>
  );
}

export function useFormSave() {
  const context = useContext(FormSaveContext);

  if (context === undefined) {
    throw new Error("useFormSave must be used within a FormSaveProvider");
  }

  return context;
}
