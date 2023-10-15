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

import { ToasterToast } from "@/components/ui/use-toast";
import axios from "@/lib/AxiosConfig";
import { StoreType } from "@/lib/useGlobalStore";
import { QueryKeys } from "@/types";
import { APIError } from "@/types/server";
import { AxiosResponse } from "axios";
import { UseFormProps } from "react-hook-form";
import { QueryClient, useMutation, useQueryClient } from "react-query";

type MutationOptions = {
  path: string;
  successMessage: string;
  errorMessage?: string;
  queryKeysToInvalidate?: QueryKeys[];
  closeModal?: boolean;
  method: "POST" | "PUT" | "PATCH" | "DELETE";
  additionalInvalidateQueries?: QueryKeys[];
};

const DEFAULT_ERROR_MESSAGE = "An error occurred.";
type Toast = Omit<ToasterToast, "id">;

export function useCustomMutation<T, K>(
  form: UseFormReturnType<T>,
  toast: (toast: Toast) => void,
  options: MutationOptions,
  onMutationSettled?: () => void,
  store?: StoreType<K>,
) {
  const queryClient = useQueryClient();

  return useMutation(
    (values: T) => executeApiMethod(options.method, options.path, values),
    {
      onSuccess: () => handleSuccess(options, toast, queryClient, store),
      onError: (error: Error) => handleError(error, options, form, toast),
      onSettled: onMutationSettled,
    },
  );
}

async function executeApiMethod(
  method: MutationOptions["method"],
  path: string,
  data?: any,
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

  // If there's a file to send and the JSON data was sent successfully
  if (fileData) {
    const newPath = method === "POST" ? `${path}${response.data.id}/` : path;
    await sendFileData(newPath, fileData);
  }

  return response;
}
function extractFileFromData(
  data: any,
): { fieldName: string; file: File | Blob } | null {
  // eslint-disable-next-line no-restricted-syntax
  for (const key of Object.keys(data)) {
    if (data[key] instanceof File || data[key] instanceof Blob) {
      const file = data[key];
      // eslint-disable-next-line no-param-reassign
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
function handleSuccess<K>(
  options: MutationOptions,
  toast: (toast: Toast) => void,
  queryClient: QueryClient,
  store?: StoreType<K>,
) {
  const notifySuccess = () => {
    showNotification(toast, "Success", options.successMessage);
  };

  const invalidateQueries = async (queries?: string[]) => {
    if (queries) {
      await queryClient.invalidateQueries(queries);
    }
  };

  invalidateQueries(options.queryKeysToInvalidate).then(notifySuccess);
  invalidateQueries(options.additionalInvalidateQueries);

  const modalKey = options.method === "POST" ? "createModalOpen" : "drawerOpen";
  if (options.closeModal) {
    store && store.set(modalKey as keyof K, false as any);
  }
}

function handleError(
  error: any,
  options: MutationOptions,
  form: UseFormProps<any>,
  toast: (toast: Toast) => void,
) {
  const { data } = error?.response || {};
  if (data?.type === "validationError") {
    handleValidationErrors(data.errors, form, toast);
  } else {
    showErrorNotification(toast, options.errorMessage);
  }
}

function showNotification(
  toast: (toast: Toast) => void,
  title: string,
  message: string,
) {
  toast({
    title: title,
    description: message,
  });
}

function showErrorNotification(
  toast: (toast: Toast) => void,
  errorMessage?: string,
) {
  showNotification(toast, "Error", errorMessage || DEFAULT_ERROR_MESSAGE);
}

function handleValidationErrors<T>(
  errors: APIError[],
  form: UseFormReturnType<T>,
  toast: (toast: Toast) => void,
) {
  errors.forEach((e: APIError) => {
    form.setFieldError(e.attr, e.detail);
    if (e.attr === "nonFieldErrors") {
      showErrorNotification(toast, e.detail);
    }
  });
}
