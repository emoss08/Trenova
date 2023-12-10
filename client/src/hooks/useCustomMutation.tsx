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

import { ToastAction } from "@/components/ui/toast";
import { toast } from "@/components/ui/use-toast";
import axios from "@/lib/axiosConfig";
import { useTableStore } from "@/stores/TableStore";
import { QueryKeys } from "@/types";
import { APIError } from "@/types/server";
import {
  QueryClient,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { AxiosResponse } from "axios";
import {
  Control,
  ErrorOption,
  FieldValues,
  Path,
  UseFormReset,
} from "react-hook-form";

type DataProp = Record<string, unknown> | FormData;
type MutationOptions = {
  path: string;
  successMessage: string;
  errorMessage?: string;
  queryKeysToInvalidate?: QueryKeys[];
  closeModal?: boolean;
  method: "POST" | "PUT" | "PATCH" | "DELETE";
  additionalInvalidateQueries?: QueryKeys[];
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
  data: Record<string, unknown> | FormData | null,
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
  data: any,
): { fieldName: string; file: File | Blob } | null {
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
    toast({
      variant: "success",
      title: "Great! Changes saved successfully.",
      description: options.successMessage,
    });
  };

  // Invalidate the queries that are passed in
  const invalidateQueries = async (queries?: string[]) => {
    if (queries) {
      await queryClient.invalidateQueries({
        queryKey: queries,
      });

      // Broadcast a message to other windows to invalidate the same queries
      broadcastChannel.postMessage({
        type: "invalidate",
        queryKeys: queries,
      });
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
  const { data } = error?.response || {};
  if (data?.type === "validationError") {
    handleValidationErrors(data.errors, control);
  } else {
    showErrorNotification(options.errorMessage);
  }
}

function showErrorNotification(errorMessage?: string) {
  toast({
    variant: "destructive",
    title: "Uh Oh! Something went wrong.",
    description: errorMessage || "An error occurred.",
    action: <ToastAction altText="Try Again">Try Again.</ToastAction>,
  });
}

function handleValidationErrors<T extends FieldValues>(
  errors: APIError[],
  control: Control<T>,
) {
  errors.forEach((e: APIError) => {
    control.setError(
      e.attr as Path<T>,
      {
        type: "manual",
        message: e.detail,
      } as ErrorOption,
    );
    if (e.attr === "nonFieldErrors") {
      showErrorNotification(e.detail);
    }
  });
}
