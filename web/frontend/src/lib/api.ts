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
