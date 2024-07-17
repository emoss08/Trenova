/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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



import { Control, FieldValues, Path } from "react-hook-form";
import { toast } from "sonner";
import { API_BASE_URL } from "./constants";

interface ErrorResponse {
  type: string;
  title: string;
  status: number;
  detail: string;
  instance?: string;
  "invalid-params"?: { name: string; reason: string }[];
  message?: string;
}

export async function handleError<T extends FieldValues>(
  error: any,
  control: Control<T>,
) {
  if (!error.response) {
    console.error("[Trenova] Network or other error", error);
    showErrorNotification("A network or system error occurred.");
    return;
  }

  const { data } = error.response as { data?: ErrorResponse };

  if (!data) {
    console.error("[Trenova] Error without data", error);
    showErrorNotification("An unknown error occurred.");
    return;
  }

  const invalidParams = data["invalid-params"] || [];

  switch (data.instance) {
    case `${API_BASE_URL}/probs/validation-errors`:
      console.info("[Trenova] Validation error", invalidParams);
      handleValidationErrors(invalidParams, control);
      break;
    case `${API_BASE_URL}/probs/database-errors`:
    case `${API_BASE_URL}/probs/invalid-request`: // Combined case for both types of errors
      handleInvalidRequest(invalidParams, control);
      break;
    default:
      console.error("[Trenova] Unhandled error type", data.message);
      showErrorNotification(
        data.message || "An unhandled error type occurred.",
      );
      break;
  }
}

function showErrorNotification(errorMessage?: string) {
  toast.error(
    <div className="flex flex-col space-y-1">
      <span className="font-semibold">Uh Oh! Something went wrong.</span>
      <span className="text-xs">{errorMessage}</span>
    </div>,
  );
}

function handleValidationErrors<T extends FieldValues>(
  invalidParams: { name: string; reason: string }[],
  control: Control<T>,
) {
  invalidParams.forEach((param) => {
    const { name, reason } = param;

    // Set error on the control
    control.setError(name as Path<T>, { type: "manual", message: reason });

    // Show appropriate notification based on the error attribute
    if (name === "nonFieldErrors" || name === "databaseError") {
      showErrorNotification(reason);
    } else {
      showErrorNotification("Please fix the errors and try again.");
    }
  });
}

function handleInvalidRequest<T extends FieldValues>(
  invalidParams: { name: string; reason: string }[],
  control: Control<T>,
) {
  invalidParams.forEach((param) => {
    const { name, reason } = param;

    // Set error on the control
    control.setError(name as Path<T>, { type: "manual", message: reason });

    // Show appropriate notification based on the error attribute
    if (name === "nonFieldErrors" || name === "databaseError") {
      console.log(reason);
      showErrorNotification(reason);
    } else {
      showErrorNotification("Please fix the errors and try again.");
    }
  });
}
