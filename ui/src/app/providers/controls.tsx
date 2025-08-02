/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// Credit: https://github.com/openstatusHQ/data-table-filters/blob/main/src/providers/controls.tsx#L11

import { useLocalStorage } from "@/hooks/use-local-storage";
import { createContext, useContext } from "react";

interface ControlsContextType {
  open: boolean;
  setOpen: React.Dispatch<React.SetStateAction<boolean>>;
}

export const ControlsContext = createContext<ControlsContextType | null>(null);

export function ControlsProvider({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useLocalStorage("data-table-controls", true);

  return (
    <ControlsContext.Provider value={{ open, setOpen }}>
      <div
        // REMINDER: access the data-expanded state with tailwind via `group-data-[expanded=true]/controls:block`
        // In tailwindcss v4, we could even use `group-data-expanded/controls:block`
        className="group/controls mt-2 flex flex-col gap-3 w-full"
        data-expanded={open}
      >
        {children}
      </div>
    </ControlsContext.Provider>
  );
}

export function useControls() {
  const context = useContext(ControlsContext);

  if (!context) {
    throw new Error("useControls must be used within a ControlsProvider");
  }

  return context as ControlsContextType;
}
