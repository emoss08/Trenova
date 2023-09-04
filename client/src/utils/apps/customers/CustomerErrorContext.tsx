/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import React, {
  createContext,
  useContext,
  useMemo,
  useState,
  ReactNode,
} from "react";

interface ErrorCounts {
  customerInfo: number;
  deliverySlots: number;
  // ... other tabs
}

interface ErrorContextProps {
  errors: Record<string, any>;
  setErrors: React.Dispatch<React.SetStateAction<Record<string, any>>>;
  errorCounts: ErrorCounts;
}

const ErrorContext = createContext<ErrorContextProps | null>(null);

export function useCustomerErrorContext() {
  const context = useContext(ErrorContext);
  if (!context) {
    throw new Error(
      "useCustomerErrorContext must be used within an ErrorProvider",
    );
  }
  return context;
}

interface ErrorProviderProps {
  children: ReactNode;
}

export function ErrorProvider({ children }: ErrorProviderProps) {
  const [errors, setErrors] = useState<Record<string, any>>({});

  const errorCounts = useMemo(() => getErrorCounts(errors), [errors]);

  const contextValue = useMemo(
    () => ({
      errors,
      setErrors,
      errorCounts,
    }),
    [errors, setErrors, errorCounts],
  );

  return (
    <ErrorContext.Provider value={contextValue}>
      {children}
    </ErrorContext.Provider>
  );
}

export const customerInfoFields = [
  "name",
  "addressLine1",
  "addressLine2",
  "city",
  "state",
  "zipCode",
];

function getErrorCounts(errors: Record<string, any>): ErrorCounts {
  const counts: ErrorCounts = {
    customerInfo: 0,
    deliverySlots: 0,
  };

  Object.keys(errors).forEach((field) => {
    if (field.startsWith("deliverySlots")) {
      counts.deliverySlots += 1;
    }
  });

  customerInfoFields.forEach((field) => {
    if (errors[field]) {
      counts.customerInfo += 1;
    }
  });

  return counts;
}
