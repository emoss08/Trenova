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

import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
} from "@/components/common/fields/select";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import { SelectValue } from "@radix-ui/react-select";
import axios, { AxiosProgressEvent } from "axios";
import { CheckIcon, XIcon } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import Dropzone from "react-dropzone";

const MAX_FILE_SIZE_MB = 2; // Maximum file size in MB
const MAX_FILE_SIZE_BYTES = MAX_FILE_SIZE_MB * 1024 * 1024; // Maximum file size in bytes

type FileWrapper = {
  file: File;
  classification: string;
  progress: number;
  status: "pending" | "uploaded" | "failed";
};

// This is going to change we're going to use the document classificatons from the backend.
// TODO(Wolfred): Fetch the classifications from the backend
const classificationChoices = [
  { value: "BOL", label: "Bill of Lading" },
  { value: "POD", label: "Proof of Delivery" },
  { value: "Receipt", label: "Receipt" },
  { value: "Invoice", label: "Invoice" },
  { value: "Other", label: "Other" },
];

/**
 *  Maps the status of the file to a color class
 *
 * @param status Status of the file
 * @returns Color class based on the status
 */
function mapStatusToColor(status: FileWrapper["status"]) {
  switch (status) {
    case "uploaded":
      return "text-green-500";
    case "failed":
      return "text-red-500";
    default:
      return "";
  }
}

/**
 * Uploads the files to the server
 *
 * @async
 * @param files  The files to upload
 * @param onUploadProgress  The function to call when the upload progress changes
 * @returns  The response from the server
 */
const uploadFiles = async (
  files: FileWrapper[],
  onUploadProgress: (event: AxiosProgressEvent) => void,
) => {
  // TODO(Wolfred): Change this to a mutation using react-query

  const formData = new FormData();
  files.forEach((fileWrapper) => {
    formData.append("files", fileWrapper.file);
    formData.append("classifications", fileWrapper.classification);
  });

  console.info("Uploading batch...");

  return axios.post("/upload-multiple-files-with-classification", formData, {
    headers: {
      "Content-Type": "multipart/form-data",
    },
    onUploadProgress,
  });
};

/**
 * Batches the files based on their size
 *
 * @param files  The files to batch
 * @returns  The batches of files
 */
const batchFilesBySize = (files: FileWrapper[]): FileWrapper[][] => {
  const batches: FileWrapper[][] = [];
  let currentBatch: FileWrapper[] = [];
  let currentBatchSize = 0;

  files.forEach((fileWrapper) => {
    const fileSize = fileWrapper.file.size;

    if (currentBatchSize + fileSize > MAX_FILE_SIZE_BYTES) {
      batches.push(currentBatch);
      currentBatch = [];
      currentBatchSize = 0;
    }

    currentBatch.push(fileWrapper);
    currentBatchSize += fileSize;
  });

  if (currentBatch.length > 0) {
    batches.push(currentBatch);
  }

  // Logging the batch details
  console.log(`Created ${batches.length} batches`);
  batches.forEach((batch, index) => {
    const batchSize = batch.reduce((acc, file) => acc + file.file.size, 0);
    console.log(
      `Batch ${index + 1}: ${batch.length} files, ${
        batchSize / (1024 * 1024)
      } MB`,
    );
  });

  return batches;
};

export default function Index() {
  const [files, setFiles] = useState<FileWrapper[]>([]);
  const [messages, setMessages] = useState<string[]>([]);
  const isUploading = useRef(false);

  const onDrop = useCallback((acceptedFiles: File[]) => {
    const validFiles = acceptedFiles.filter(
      (file) => file.size <= MAX_FILE_SIZE_BYTES,
    );
    const invalidFiles = acceptedFiles.filter(
      (file) => file.size > MAX_FILE_SIZE_BYTES,
    );

    if (invalidFiles.length > 0) {
      const newMessages = invalidFiles.map(
        (file) => `File ${file.name} is too large and was not added.`,
      );
      setMessages((prevMessages) => [...prevMessages, ...newMessages]);
    }

    const newFiles = validFiles.map((file) => ({
      file,
      classification: classificationChoices[0].value,
      progress: 0,
      status: "pending" as const, // Ensure the status is correctly typed
    }));
    setFiles((prevFiles) => [...prevFiles, ...newFiles]);
  }, []);

  const handleClassificationChange = (
    index: number,
    newClassification: string,
  ) => {
    setFiles((prevFiles) =>
      prevFiles.map((file, i) =>
        i === index ? { ...file, classification: newClassification } : file,
      ),
    );
  };

  const clearFiles = () => {
    setFiles([]);
    setMessages([]);
  };

  const handleUploadFiles = useCallback(async () => {
    if (isUploading.current) return;

    isUploading.current = true;
    setMessages([]);
    const newMessages: string[] = ["Starting file upload..."];
    setMessages(newMessages);

    const batches = batchFilesBySize(files);

    console.log(`Total files: ${files.length}`);
    console.log(`Total batches: ${batches.length}`);

    try {
      for (let i = 0; i < batches.length; i++) {
        const batch = batches[i];
        console.log(`Uploading batch ${i + 1} with ${batch.length} files`);
        await uploadFiles(batch, (event: AxiosProgressEvent) => {
          const total = event.total || 1;
          const progress = Math.round((100 * event.loaded) / total);
          setFiles((prevFiles) =>
            prevFiles.map((file) =>
              batch.some((batchFile) => batchFile.file.name === file.file.name)
                ? { ...file, progress }
                : file,
            ),
          );
        });
        setFiles((prevFiles) =>
          prevFiles.map((file) =>
            batch.some((batchFile) => batchFile.file.name === file.file.name)
              ? { ...file, status: "uploaded" as const }
              : file,
          ),
        );
        newMessages.push(`Batch ${i + 1} uploaded successfully.`);
        setMessages([...newMessages]);
      }
      newMessages.push("Uploaded all files successfully.");
      setMessages([...newMessages]);
    } catch (error) {
      console.error("Upload error: ", error);
      newMessages.push(`Could not upload the files: ${error}`);
      setMessages([...newMessages]);
      setFiles((prevFiles) =>
        prevFiles.map((file) =>
          batches.some((batch) =>
            batch.some((batchFile) => batchFile.file.name === file.file.name),
          )
            ? { ...file, status: "failed" as const }
            : file,
        ),
      );
    } finally {
      isUploading.current = false;
    }
  }, [files]);

  return (
    <>
      <div className="container mx-auto p-4">
        <Dropzone onDrop={onDrop}>
          {({ getRootProps, getInputProps }) => (
            <section className="border-2 border-dashed p-4 text-center">
              <div {...getRootProps()} className="dropzone">
                <input {...getInputProps()} />
                {files.length > 0 ? (
                  <div className="selected-file">
                    {files.length > 3
                      ? `${files.length} files`
                      : files
                          .map((fileWrapper) => fileWrapper.file.name)
                          .join(", ")}
                  </div>
                ) : (
                  "Drag and drop files here, or click to select files"
                )}
              </div>
            </section>
          )}
        </Dropzone>

        <ScrollArea className="h-82 px-4">
          <div className="pb-10">
            {files.map((fileWrapper, index) => (
              <div
                key={index}
                className="mt-4 flex items-center justify-between"
              >
                <div className="flex items-center">
                  <p
                    className={cn(
                      "w-64 shrink-0 truncate",
                      mapStatusToColor(fileWrapper.status),
                    )}
                  >
                    {fileWrapper.file.name}
                  </p>
                  {fileWrapper.status === "uploaded" && (
                    <CheckIcon className="ml-2 size-5 text-green-500" />
                  )}
                  {fileWrapper.status === "failed" && (
                    <XIcon className="ml-2 size-5 text-red-500" />
                  )}
                </div>
                <Select
                  value={fileWrapper.classification}
                  onValueChange={(value) =>
                    handleClassificationChange(index, value)
                  }
                  disabled={fileWrapper.status !== "pending"}
                >
                  <SelectTrigger className="w-32">
                    <SelectValue>{fileWrapper.classification}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {classificationChoices.map((choice) => (
                        <SelectItem key={choice.value} value={choice.value}>
                          {choice.label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </div>
            ))}
          </div>
          <div className="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-background to-transparent" />
        </ScrollArea>

        <div className="mt-4 flex gap-x-1">
          <Button
            size="xs"
            variant="outline"
            disabled={files.length === 0 || isUploading.current}
            onClick={clearFiles}
          >
            Clear
          </Button>
          <Button
            size="xs"
            disabled={
              files.length === 0 ||
              isUploading.current ||
              files.every((file) => file.status !== "pending")
            }
            onClick={handleUploadFiles}
          >
            {isUploading.current ? "Uploading..." : "Upload"}
          </Button>
        </div>

        {messages.length > 0 && (
          <div className="alert alert-secondary" role="alert">
            <ul>
              {messages.map((msg, i) => (
                <li key={i}>{msg}</li>
              ))}
            </ul>
          </div>
        )}
      </div>
    </>
  );
}
