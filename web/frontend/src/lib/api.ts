import axios from "@/lib/axiosConfig";
import { type AxiosResponse } from "axios";
import type { FieldValues } from "react-hook-form";

export async function executeApiMethod<T extends FieldValues>(
  method: "POST" | "PUT" | "PATCH" | "DELETE",
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
