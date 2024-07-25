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
  onSettled?: (data: AxiosResponse<any, any> | undefined) => void;
};

export function useCustomMutation<T extends FieldValues>(
  control: Control<T>,
  options: MutationOptions<T>,
): UseMutationResult<AxiosResponse<any, any>, Error, DataProp> {
  const queryClient = useQueryClient();

  return useMutation<AxiosResponse<any, any>, Error, DataProp>({
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
