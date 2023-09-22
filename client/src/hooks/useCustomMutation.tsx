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

import { useMutation, useQueryClient } from "react-query";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import React from "react";
import { NotificationsEvents } from "@mantine/notifications/lib/events";
import { UseFormReturnType } from "@mantine/form";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";

type MutationOptions = {
  path: string;
  successMessage: string;
  errorMessage?: string;
  notificationId: string;
  queryKeysToInvalidate?: Array<string>;
  closeModal?: boolean;
  validationDetail: string;
  validationFieldName: string;
};

export function useCustomMutation<T extends Record<string, any>>(
  form: UseFormReturnType<T>,
  store: any,
  notifications: NotificationsEvents,
  options: MutationOptions,
  onMutationSettled?: () => void,
) {
  const queryClient = useQueryClient();

  return useMutation((values: T) => axios.post(options.path, values), {
    onSuccess: () => {
      if (options.queryKeysToInvalidate) {
        queryClient
          .invalidateQueries(options.queryKeysToInvalidate)
          .then(() => {
            notifications.show({
              title: "Success",
              message: options.successMessage,
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
          });
      }
      if (options.closeModal) {
        store.set("createModalOpen", false);
      }
    },
    onError: (error: any) => {
      const { data } = error.response;
      if (data && data.type === "validationError") {
        handleValidationErrors(
          data.errors,
          form,
          notifications,
          options.validationDetail,
          options.validationFieldName,
        );
      } else {
        // General error handling
        notifications.show({
          title: "Error",
          message: options.errorMessage || "An error occurred.",
          color: "red",
          withCloseButton: true,
          icon: <FontAwesomeIcon icon={faXmark} />,
          autoClose: 10_000, // 10 seconds
        });
      }
    },
    onSettled: () => {
      if (onMutationSettled) {
        onMutationSettled();
      }
    },
  });
}

function handleValidationErrors<T extends Record<string, any>>(
  errors: APIError[],
  form: UseFormReturnType<T>,
  notifications: NotificationsEvents,
  notificationDetail: string,
  validationFieldName: string,
) {
  errors.forEach((e: APIError) => {
    form.setFieldError(e.attr, e.detail);
    if (e.attr === "nonFieldErrors") {
      notifications.show({
        title: "Error",
        message: e.detail,
        color: "red",
        withCloseButton: true,
        icon: <FontAwesomeIcon icon={faXmark} />,
        autoClose: 10_000, // 10 seconds
      });
    } else if (e.attr === "All" && e.detail === notificationDetail) {
      form.setFieldError(validationFieldName, e.detail);
    }
  });
}

export function usePutMutation<T extends Record<string, any>>(
  form: UseFormReturnType<T>,
  store: any,
  notifications: NotificationsEvents,
  options: MutationOptions,
  onMutationSettled?: () => void,
) {
  const queryClient = useQueryClient();

  return useMutation((values: T) => axios.put(options.path, values), {
    onSuccess: () => {
      if (options.queryKeysToInvalidate) {
        queryClient
          .invalidateQueries(options.queryKeysToInvalidate)
          .then(() => {
            notifications.show({
              title: "Success",
              message: options.successMessage,
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
          });
      }
      if (options.closeModal) {
        store.set("editModalOpen", false);
      }
    },
    onError: (error: any) => {
      const { data } = error.response;
      if (data && data.type === "validationError") {
        handleValidationErrors(
          data.errors,
          form,
          notifications,
          options.validationDetail,
          options.validationFieldName,
        );
      } else {
        // General error handling
        notifications.show({
          title: "Error",
          message: options.errorMessage || "An error occurred.",
          color: "red",
          withCloseButton: true,
          icon: <FontAwesomeIcon icon={faXmark} />,
          autoClose: 10_000, // 10 seconds
        });
      }
    },
    onSettled: () => {
      if (onMutationSettled) {
        onMutationSettled();
      }
    },
  });
}
