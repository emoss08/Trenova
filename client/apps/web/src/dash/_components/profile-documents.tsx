import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { formatRange } from "@/lib/date";
import {
  fetchMyProfileDocuments,
  fetchPortalWorkerDocumentTypes,
  uploadMyProfileDocument,
} from "@/lib/portal";
import { cn, formatFileSize } from "@/lib/utils";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CameraIcon, IdCardIcon, PaperclipIcon, XIcon } from "lucide-react";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { useDashFeatures } from "./use-dash-features";

export function ProfileDocuments() {
  const queryClient = useQueryClient();
  const features = useDashFeatures();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [documentTypeId, setDocumentTypeId] = useState<string | null>(null);

  const documents = useQuery({
    queryKey: ["dash-profile-documents"],
    queryFn: fetchMyProfileDocuments,
  });
  const documentTypes = useQuery({
    queryKey: ["dash-worker-document-types"],
    queryFn: fetchPortalWorkerDocumentTypes,
    staleTime: 5 * 60 * 1000,
  });

  const upload = useMutation({
    mutationFn: (file: File) => uploadMyProfileDocument(file, documentTypeId ?? undefined),
    onSuccess: async () => {
      toast.success("Document uploaded — your carrier will see it in your file.");
      setPendingFile(null);
      setDocumentTypeId(null);
      await queryClient.invalidateQueries({ queryKey: ["dash-profile-documents"] });
    },
    onError: (error: Error) => toast.error(error.message || "Upload failed. Try again."),
  });

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (file) {
      setPendingFile(file);
    }
  };

  return (
    <div className="rounded-2xl border border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <IdCardIcon className="size-4 text-muted-foreground" />
          <h2 className="text-sm font-semibold">My documents</h2>
        </div>
        {!pendingFile && features.allowProfileDocumentUpload ? (
          <Button
            variant="outline"
            size="sm"
            className="h-8"
            onClick={() => fileInputRef.current?.click()}
          >
            <CameraIcon className="size-3.5" />
            Add
          </Button>
        ) : null}
      </div>

      <input
        ref={fileInputRef}
        type="file"
        accept="image/*,application/pdf"
        capture="environment"
        className="hidden"
        onChange={handleFileChange}
      />

      {pendingFile ? (
        <div className="mb-3 rounded-xl border border-border bg-muted/40 p-3">
          <div className="flex items-center justify-between gap-2">
            <p className="min-w-0 truncate text-sm font-medium">{pendingFile.name}</p>
            <button
              type="button"
              aria-label="Cancel upload"
              className="text-muted-foreground hover:text-foreground"
              onClick={() => {
                setPendingFile(null);
                setDocumentTypeId(null);
              }}
            >
              <XIcon className="size-4" />
            </button>
          </div>
          <p className="text-xs text-muted-foreground">{formatFileSize(pendingFile.size)}</p>
          {documentTypes.data && documentTypes.data.length > 0 ? (
            <div className="mt-2 flex flex-wrap gap-1.5">
              {documentTypes.data.map((type) => (
                <button
                  key={type.id}
                  type="button"
                  onClick={() =>
                    setDocumentTypeId((current) => (current === type.id ? null : type.id))
                  }
                  className={cn(
                    "rounded-full border border-border px-2.5 py-1 text-xs font-medium text-muted-foreground transition-colors",
                    documentTypeId === type.id &&
                      "border-primary bg-primary text-primary-foreground",
                  )}
                >
                  {type.name}
                </button>
              ))}
            </div>
          ) : null}
          <Button
            size="sm"
            className="mt-3 w-full"
            disabled={upload.isPending}
            onClick={() => upload.mutate(pendingFile)}
          >
            <PaperclipIcon className="size-3.5" />
            {upload.isPending ? "Uploading..." : "Upload"}
          </Button>
        </div>
      ) : null}

      {documents.isPending ? (
        <Skeleton className="h-16 w-full rounded-xl" />
      ) : documents.data && documents.data.length > 0 ? (
        <ul className="divide-y divide-border">
          {documents.data.map((doc) => (
            <li key={doc.id} className="flex items-center justify-between gap-3 py-2.5">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{doc.fileName}</p>
                <p className="text-xs text-muted-foreground">
                  {formatRange(doc.createdAt, doc.createdAt)} · {formatFileSize(doc.fileSize)}
                </p>
              </div>
              {doc.documentTypeName ? (
                <Badge variant="secondary">{doc.documentTypeName}</Badge>
              ) : null}
            </li>
          ))}
        </ul>
      ) : !pendingFile && features.allowProfileDocumentUpload ? (
        <p className="text-xs text-muted-foreground">
          Snap photos of your CDL, medical card, and anything else your carrier needs for your
          qualification file — front and back for cards.
        </p>
      ) : null}
    </div>
  );
}
