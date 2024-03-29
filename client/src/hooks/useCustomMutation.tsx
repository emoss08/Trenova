/*
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

import axios from "@/lib/axiosConfig";
import { TOAST_STYLE } from "@/lib/constants";
import { useTableStore } from "@/stores/TableStore";
import type { QueryKeys, QueryKeyWithParams } from "@/types";
import { type APIError } from "@/types/server";
import { faX } from "@fortawesome/pro-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  QueryClient,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { type AxiosResponse } from "axios";
import type { Control, FieldValues, Path, UseFormReset } from "react-hook-form";
import toast from "react-hot-toast";

type DataProp = Record<string, unknown> | FormData;
type MutationOptions = {
  path: string;
  successMessage: string;
  errorMessage?: string;
  queryKeysToInvalidate?: QueryKeys | QueryKeyWithParams<any, any>;
  closeModal?: boolean;
  method: "POST" | "PUT" | "PATCH" | "DELETE";
  additionalInvalidateQueries?: QueryKeys;
};

export function useCustomMutation<T extends FieldValues>(
  control: Control<T>,
  options: MutationOptions,
  onMutationSettled?: () => void,
  reset?: UseFormReset<T>,
) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: DataProp) =>
      executeApiMethod(options.method, options.path, data),
    onSuccess: () => handleSuccess(options, queryClient, reset),
    onError: (error: Error) => handleError(error, options, control),
    onSettled: onMutationSettled,
  });
}

async function executeApiMethod(
  method: MutationOptions["method"],
  path: string,
  data: Record<string, unknown> | FormData,
): Promise<AxiosResponse> {
  const fileData = extractFileFromData(data);

  // Send the JSON data first (this assumes no file in the payload)
  const axiosConfig = {
    method,
    url: path,
    data: JSON.stringify(data),
    headers: {
      "Content-Type": "application/json",
    },
  };

  const response = await axios(axiosConfig);

  if (fileData) {
    const newPath = method === "POST" ? `${path}${response.data.id}/` : path;
    await sendFileData(newPath, fileData);
  }

  return response;
}

function extractFileFromData(
  data: Record<string, unknown> | FormData,
): { fieldName: string; file: File | Blob } | null {
  if (!data) {
    return null;
  }

  if (data instanceof FormData) {
    for (const pair of data.entries()) {
      const [key, value]: [string, any] = pair;
      if (value instanceof File || value instanceof Blob) {
        return { fieldName: key, file: value };
      }
    }
    return null;
  }

  for (const key of Object.keys(data)) {
    const item = data[key];

    if (item instanceof File || item instanceof Blob) {
      delete data[key];
      return { fieldName: key, file: item };
    } else if (item instanceof FileList && item.length > 0) {
      const file = item[0];
      delete data[key];
      return { fieldName: key, file };
    }
  }
  return null;
}

function sendFileData(
  path: string,
  fileData: { fieldName: string; file: File | Blob },
) {
  const formData = new FormData();
  formData.append(fileData.fieldName, fileData.file);
  return axios.patch(path, formData);
}

const broadcastChannel = new BroadcastChannel("query-invalidation");

async function handleSuccess<T extends FieldValues>(
  options: MutationOptions,
  queryClient: QueryClient,
  reset?: UseFormReset<T>,
) {
  const notifySuccess = () => {
    toast.success(
      () => (
        <div className="flex items-start justify-between">
          <div className="flex flex-col space-y-1">
            <span className="font-semibold">Success!</span>
            <span className="text-xs">{options.successMessage}</span>
          </div>
          <button
            onClick={() => toast.dismiss("notification-toast")}
            aria-label="Close"
            className="-mt-2 ml-4"
          >
            <FontAwesomeIcon icon={faX} className="size-2.5" />
          </button>
        </div>
      ),
      {
        duration: 4000,
        id: "notification-toast",
        style: TOAST_STYLE,
        ariaProps: {
          role: "status",
          "aria-live": "polite",
        },
      },
    );
  };

  // Invalidate the queries that are passed in
  const invalidateQueries = async (queries?: string[]) => {
    if (queries) {
      await queryClient.invalidateQueries({
        queryKey: queries,
      });

      // Broadcast a message to other windows to invalidate the same queries
      try {
        broadcastChannel.postMessage({
          type: "invalidate",
          queryKeys: queries,
        });
      } catch (error) {
        console.error("[Trenova] BroadcastChannel not supported", error);
      }
    }
  };

  // Invalidate the queries that are passed in
  await invalidateQueries(options.queryKeysToInvalidate).then(notifySuccess);
  await invalidateQueries(options.additionalInvalidateQueries);

  // Close the sheet depending on the method. If the sheet is not open, this will do nothing.
  const sheetKey = options.method === "POST" ? "sheetOpen" : "editSheetOpen";

  if (options.closeModal) {
    useTableStore.set(sheetKey, false);
  }

  // Reset the form if `reset` is passed
  reset?.();
}

async function handleError<T extends FieldValues>(
  error: any,
  options: MutationOptions,
  control: Control<T>,
) {
  if (!error.response) {
    console.error("[Trenova] Network or other error", error);
    showErrorNotification("A network or system error occurred.");
    return;
  }

  const { data } = error?.response || {};
  if (data?.type === "validationError") {
    handleValidationErrors(data.errors, control);
  } else if (data?.type === "databaseError") {
    handleDatabaseErrors(data.errors, control);
  }
}

function showErrorNotification(errorMessage?: string) {
  toast.error(
    () => (
      <div className="flex flex-col space-y-1">
        <span className="font-semibold">Uh Oh! Something went wrong.</span>
        <span className="text-xs">{errorMessage}</span>
      </div>
    ),
    {
      duration: 4000,
      id: "notification-toast",
      style: TOAST_STYLE,
      ariaProps: {
        role: "status",
        "aria-live": "polite",
      },
    },
  );
}

/**
 * Handle validation errors by setting errors on the form control and showing notifications.
 *
 * @param {APIError[]} errors - Array of errors from the API.
 * @param {Control<T>} control - React Hook Form control object.
 */
function handleValidationErrors<T extends FieldValues>(
  errors: APIError[],
  control: Control<T>,
) {
  errors.forEach((error: APIError) => {
    const { attr, detail } = error;

    // Set error on the control
    control.setError(attr as Path<T>, { type: "manual", message: detail });

    // Show appropriate notification based on the error attribute
    if (attr === "nonFieldErrors" || attr === "databaseError") {
      showErrorNotification(detail);
    } else {
      showErrorNotification("Please fix the errors and try again.");
    }
  });
}

/**
 * Handle database errors by showing notifications.
 *
 * @param {APIError[]} errors - Array of errors from the API.
 * @param {Control<T>} control - React Hook Form control object.
 */
function handleDatabaseErrors<T extends FieldValues>(
  errors: APIError[],
  control: Control<T>,
) {
  errors.forEach((error: APIError) => {
    const { attr, detail } = error;

    // Set error on the control
    control.setError(attr as Path<T>, { type: "manual", message: detail });

    // Show appropriate notification based on the error attribute
    if (attr === "nonFieldErrors" || attr === "databaseError") {
      showErrorNotification(detail);
    } else {
      showErrorNotification("Please fix the errors and try again.");
    }
  });
}
