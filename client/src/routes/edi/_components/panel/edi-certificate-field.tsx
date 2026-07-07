import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { useDebounce } from "@/hooks/use-debounce";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { CommunicationProfileFormValues } from "@/routes/edi/_components/edi-schemas";
import type { EDICertificateSummary } from "@/types/edi";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { UploadIcon } from "lucide-react";
import { useRef } from "react";
import { useController, useWatch, type Control, type FieldPath } from "react-hook-form";
import { toast } from "sonner";

const CERTIFICATE_INSPECTION_DEBOUNCE_MS = 500;
const CERTIFICATE_EXPIRY_WARNING_DAYS = 30;
const CERTIFICATE_ACCEPT = ".pem,.crt,.cer,.txt";

type CertificateFieldName = FieldPath<CommunicationProfileFormValues>;

type EDICertificateFieldProps = {
  control: Control<CommunicationProfileFormValues>;
  name: CertificateFieldName;
  label: string;
  description?: string;
};

export function EDICertificateField({
  control,
  name,
  label,
  description,
}: EDICertificateFieldProps) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const { field } = useController({ control, name });
  const rawValue = useWatch({ control, name });
  const value = typeof rawValue === "string" ? rawValue.trim() : "";
  const debouncedValue = useDebounce(value, CERTIFICATE_INSPECTION_DEBOUNCE_MS);

  const inspection = useQuery({
    queryKey: ["edi-certificate-inspection", debouncedValue],
    queryFn: () => apiService.ediService.inspectCertificate(debouncedValue),
    enabled: debouncedValue.length > 0,
    staleTime: Infinity,
    retry: false,
  });

  const handleFileSelected = async (files: FileList | null) => {
    const file = files?.item(0);
    if (!file) return;
    try {
      const contents = await file.text();
      field.onChange(contents.trim());
    } catch {
      toast.error("The selected certificate file could not be read");
    }
  };

  return (
    <div className="flex flex-col gap-1">
      <TextareaField
        control={control}
        name={name}
        label={label}
        description={description}
        placeholder="-----BEGIN CERTIFICATE-----"
      />
      <div className="flex items-start justify-between gap-2">
        <CertificateSummaryLine
          hasValue={debouncedValue.length > 0}
          inspection={inspection}
        />
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="shrink-0"
          onClick={() => fileInputRef.current?.click()}
        >
          <UploadIcon className="size-3.5" />
          Upload
        </Button>
        <input
          ref={fileInputRef}
          type="file"
          accept={CERTIFICATE_ACCEPT}
          className="hidden"
          onChange={(event) => {
            void handleFileSelected(event.target.files);
            event.target.value = "";
          }}
        />
      </div>
    </div>
  );
}

function CertificateSummaryLine({
  hasValue,
  inspection,
}: {
  hasValue: boolean;
  inspection: UseQueryResult<EDICertificateSummary, Error>;
}) {
  if (!hasValue) {
    return (
      <p className="text-xs text-muted-foreground">
        Paste a PEM certificate or upload a .pem/.crt file.
      </p>
    );
  }
  if (inspection.isPending) {
    return <p className="text-xs text-muted-foreground">Inspecting certificate…</p>;
  }
  if (inspection.isError || !inspection.data) {
    return (
      <p className="text-xs text-red-600 dark:text-red-400">
        The value is not a valid PEM certificate.
      </p>
    );
  }

  const summary = inspection.data;
  const expiresOn = new Date(summary.notAfter * 1000).toLocaleDateString();
  const expiryText = summary.expired
    ? `Expired on ${expiresOn}`
    : `Expires in ${summary.expiresInDays} day(s) (${expiresOn})`;
  const expiryTone = summary.expired
    ? "text-red-600 dark:text-red-400"
    : summary.expiresInDays <= CERTIFICATE_EXPIRY_WARNING_DAYS
      ? "text-yellow-700 dark:text-yellow-400"
      : "text-muted-foreground";

  return (
    <div className="min-w-0 text-xs">
      <p className="truncate text-muted-foreground" title={summary.subject}>
        {summary.subject}
      </p>
      <p className={cn("font-medium", expiryTone)}>{expiryText}</p>
      <p
        className="truncate font-mono text-2xs text-muted-foreground"
        title={`SHA-256 ${summary.sha256Fingerprint}`}
      >
        SHA-256 {summary.sha256Fingerprint}
      </p>
    </div>
  );
}
