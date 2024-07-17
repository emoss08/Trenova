/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
 */

import { executeApiMethod } from "@/lib/api";
import { handleError } from "@/lib/error-handler";
import { useTableStore } from "@/stores/TableStore";
import type { QueryKeys, ValuesOf } from "@/types";
import {
  QueryClient,
  UseMutationResult,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { type AxiosResponse } from "axios";
import type { Control, FieldValues, UseFormReset } from "react-hook-form";
import { toast } from "sonner";

type DataProp = Record<string, unknown> | FormData;
type MutationOptions<K extends FieldValues> = {
  path: string;
  successMessage: string;
  errorMessage?: string;
  queryKeysToInvalidate?: ValuesOf<QueryKeys>;
  closeModal?: boolean;
  reset: UseFormReset<K>;
  method: "POST" | "PUT" | "PATCH" | "DELETE";
  onSettled?: () => void;
};

export function useCustomMutation<T extends FieldValues>(
  control: Control<T>,
  options: MutationOptions<T>,
): UseMutationResult<AxiosResponse, Error, DataProp> {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: DataProp) =>
      executeApiMethod(options.method, options.path, data),
    onSuccess: () => handleSuccess(options, queryClient),
    onError: (error: Error) => handleError(error, control),
    onSettled: options.onSettled,
  });
}

const broadcastChannel = new BroadcastChannel("query-invalidation");

async function handleSuccess<T extends FieldValues>(
  options: MutationOptions<T>,
  queryClient: QueryClient,
) {
  const notifySuccess = () => {
    toast.success(options.successMessage);
  };

  // Invalidate the queries that are passed in
  const invalidateQueries = async (queries?: string) => {
    if (queries) {
      await queryClient.invalidateQueries({
        predicate: (query) =>
          query.queryKey.some(
            (keyPart) =>
              typeof keyPart === "string" && keyPart.includes(queries),
          ),
      });

      // Broadcast a message to other windows to invalidate the same queries
      try {
        broadcastChannel.postMessage({
          type: "invalidate",
          queryKeys: [queries],
        });
      } catch (error) {
        console.error("[Trenova] BroadcastChannel not supported", error);
      }
    }
  };

  if (options.queryKeysToInvalidate) {
    await invalidateQueries(options.queryKeysToInvalidate).then(notifySuccess);
  } else {
    notifySuccess();
  }

  // Close the sheet depending on the method. If the sheet is not open, this will do nothing.
  const sheetKey = options.method === "POST" ? "sheetOpen" : "editSheetOpen";

  if (options.closeModal) {
    useTableStore.set(sheetKey, false);
  }

  // reset the form if `reset` is passed
  options.reset();
}
