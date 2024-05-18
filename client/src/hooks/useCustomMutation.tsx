import axios from "@/lib/axiosConfig";
import { useTableStore } from "@/stores/TableStore";
import type { QueryKeys, ValuesOf } from "@/types";
import { type APIError } from "@/types/server";
import {
  QueryClient,
  UseMutationResult,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { type AxiosResponse } from "axios";
import type { Control, FieldValues, Path, UseFormReset } from "react-hook-form";
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
  });
}

async function executeApiMethod<T extends FieldValues>(
  method: MutationOptions<T>["method"],
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

interface ErrorResponse {
  type: "validationError" | "databaseError" | "invalidRequest";
  errors: any;
}

async function handleError<T extends FieldValues>(
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

  switch (data.type) {
    case "validationError":
      handleValidationErrors(data.errors, control);
      break;
    case "databaseError":
    case "invalidRequest": // Combined case for both types of errors
      handleInvalidRequest(data.errors, control);
      break;
    default:
      console.error("[Trenova] Unhandled error type", data);
      showErrorNotification("An unhandled error type occurred.");
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
function handleInvalidRequest<T extends FieldValues>(
  errors: APIError[],
  control: Control<T>,
) {
  errors.forEach((error: APIError) => {
    const { attr, detail } = error;

    // Set error on the control
    control.setError(attr as Path<T>, { type: "manual", message: detail });

    // Show appropriate notification based on the error attribute
    if (attr === "nonFieldErrors" || attr === "databaseError") {
      console.log(detail);
      showErrorNotification(detail);
    } else {
      showErrorNotification("Please fix the errors and try again.");
    }
  });
}
