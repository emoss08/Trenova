import { Badge } from "@/components/ui/badge";
import { Icon } from "@/components/ui/icons";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { formatFileSize } from "@/lib/utils";
import { Document as DocumentFile } from "@/types/document";
import { faFileImage } from "@fortawesome/pro-regular-svg-icons";
import {
  faFileAlt,
  faFileContract,
  faFileExcel,
  faFilePdf,
  faFileWord,
} from "@fortawesome/pro-solid-svg-icons";
import { useCallback } from "react";

export function DocumentPreview({
  doc,
  handleDocumentClick,
  handleDocumentDoubleClick,
}: {
  doc: DocumentFile;
  handleDocumentClick: (doc: DocumentFile) => void;
  handleDocumentDoubleClick: (doc: DocumentFile) => void;
}) {
  // Get document preview URL
  const getDocumentPreviewUrl = (document: DocumentFile) => {
    // In a real implementation, you would get a thumbnail or preview URL
    return `/api/v1/documents/${document.id}/preview`;
  };

  const getFileIcon = useCallback((fileType: string) => {
    const type = fileType.toLowerCase();
    if (type.includes("pdf")) return faFilePdf;
    if (
      type.includes("image") ||
      type.includes("jpg") ||
      type.includes("png") ||
      type.includes("jpeg")
    )
      return faFileImage;
    if (
      type.includes("excel") ||
      type.includes("spreadsheet") ||
      type.includes("csv") ||
      type.includes("xlsx")
    )
      return faFileExcel;
    if (type.includes("word") || type.includes("doc")) return faFileWord;
    if (type.includes("contract")) return faFileContract;
    return faFileAlt;
  }, []);

  return (
    <div
      key={doc.id}
      className="border rounded-lg overflow-hidden hover:shadow-md transition-shadow"
      onClick={() => handleDocumentClick(doc)}
      onDoubleClick={() => handleDocumentDoubleClick(doc)}
    >
      <div className="h-32 bg-gray-100 flex items-center justify-center border-b">
        {doc.fileType.includes("image") ? (
          <img
            src={getDocumentPreviewUrl(doc)}
            alt={doc.originalName}
            className="max-h-full max-w-full object-contain"
          />
        ) : (
          <Icon
            icon={getFileIcon(doc.fileType)}
            className="text-5xl text-blue-500"
          />
        )}
      </div>
      <div className="p-3">
        <div className="flex items-center justify-between mb-1">
          <h3 className="font-medium text-sm truncate" title={doc.originalName}>
            {doc.originalName}
          </h3>
        </div>
        <div className="flex justify-between items-center">
          <Badge withDot={false} variant="purple" className="text-xs">
            {doc.documentType}
          </Badge>
          <span className="text-xs text-gray-500">
            {formatFileSize(doc.fileSize)}
          </span>
        </div>
        <p className="text-xs text-gray-500 mt-1">
          {generateDateTimeStringFromUnixTimestamp(doc.createdAt)}
        </p>
      </div>
    </div>
  );
}
