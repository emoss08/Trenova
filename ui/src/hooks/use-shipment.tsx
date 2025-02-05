import type { Shipment } from "@/types/shipment";
import React, { createContext, useContext, useMemo } from "react";

type ShipmentContextType = {
  shipment: Shipment | null;
  isLoading: boolean;
  error: Error | null;
};

const ShipmentContext = createContext<ShipmentContextType | undefined>(
  undefined,
);

type ShipmentProvderProps = {
  children: React.ReactNode;
  initialShipment?: Shipment | null;
  isLoading?: boolean;
};

export function ShipmentProvider({
  children,
  initialShipment = null,
  isLoading = false,
}: ShipmentProvderProps) {
  const value = useMemo(
    () => ({
      shipment: initialShipment,
      isLoading,
      error: null,
    }),
    [initialShipment, isLoading],
  );

  return (
    <ShipmentContext.Provider value={value}>
      {children}
    </ShipmentContext.Provider>
  );
}

export function useShipment() {
  const context = useContext(ShipmentContext);

  if (context === undefined) {
    throw new Error("useShipment must be used within a ShipmentProvider");
  }

  return context;
}
