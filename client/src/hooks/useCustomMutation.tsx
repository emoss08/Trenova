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

import { QueryClient, useMutation, useQueryClient } from "react-query";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import React from "react";
import { NotificationsEvents } from "@mantine/notifications/lib/events";
import { UseFormReturnType } from "@mantine/form";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { TableStoreProps } from "@/types/tables";
import { StoreType } from "@/helpers/useGlobalStore";

type MutationOptions = {
  path: string;
  successMessage: string;
  errorMessage?: string;
  queryKeysToInvalidate?: Array<string>;
  closeModal?: boolean;
  method: "POST" | "PUT" | "PATCH" | "DELETE";
};

const DEFAULT_ERROR_MESSAGE = "An error occurred.";
export function useCustomMutation<
  T extends Record<string, any>,
  K extends Omit<TableStoreProps<T>, "drawerOpen">,
>(
  form: UseFormReturnType<T>,
  store: StoreType<K>,
  notifications: NotificationsEvents,
  options: MutationOptions,
  onMutationSettled?: () => void,
) {
  const queryClient = useQueryClient();

  return useMutation(
    (values: T) => executeMethod(options.method, options.path, values),
    {
      onSuccess: () =>
        handleSuccess(options, notifications, queryClient, store),
      onError: (error: any) => handleError(error, options, form, notifications),
      onSettled: onMutationSettled,
    },
  );
}

function executeMethod(
  method: MutationOptions["method"],
  path: string,
  values: any,
): Promise<any> {
  switch (method) {
    case "POST":
      return axios.post(path, values);
    case "PUT":
      return axios.put(path, values);
    case "PATCH":
      return axios.patch(path, values);
    case "DELETE":
      return axios.delete(path);
    default:
      throw new Error(`Unsupported method: ${method}`);
  }
}

function handleSuccess<K>(
  options: MutationOptions,
  notifications: NotificationsEvents,
  queryClient: QueryClient,
  store: StoreType<K>,
): void {
  if (options.queryKeysToInvalidate) {
    queryClient.invalidateQueries(options.queryKeysToInvalidate).then(() => {
      showNotification(
        notifications,
        "Success",
        options.successMessage,
        "green",
        faCheck,
      );
    });
  }

  const modalKey =
    options.method === "POST" ? "createModalOpen" : "editModalOpen";
  if (options.closeModal) {
    store.set(modalKey as keyof K, false as any);
  }
}

function handleError(
  error: any,
  options: MutationOptions,
  form: UseFormReturnType<any>,
  notifications: NotificationsEvents,
): void {
  const { data } = error.response;
  if (data && data.type === "validationError") {
    handleValidationErrors(data.errors, form, notifications);
  } else {
    showErrorNotification(notifications, options.errorMessage);
  }
}

function showNotification(
  notifications: NotificationsEvents,
  title: string,
  message: string,
  color: string,
  icon: any,
): void {
  notifications.show({
    title,
    message,
    color,
    withCloseButton: true,
    icon: <FontAwesomeIcon icon={icon} />,
    autoClose: 10_000,
  });
}

function showErrorNotification(
  notifications: NotificationsEvents,
  errorMessage?: string,
): void {
  showNotification(
    notifications,
    "Error",
    errorMessage || DEFAULT_ERROR_MESSAGE,
    "red",
    faXmark,
  );
}

function handleValidationErrors<T extends Record<string, any>>(
  errors: APIError[],
  form: UseFormReturnType<T>,
  notifications: NotificationsEvents,
): void {
  errors.forEach((e: APIError) => {
    form.setFieldError(e.attr, e.detail);
    if (e.attr === "nonFieldErrors") {
      showErrorNotification(notifications, e.detail);
    }
  });
}
