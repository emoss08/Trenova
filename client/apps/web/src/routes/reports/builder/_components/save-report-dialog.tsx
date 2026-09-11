import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  REPORT_CATEGORY_CHOICES,
  REPORT_DEFINITION_STATUS_LABELS,
  REPORT_FORMAT_CHOICES,
  REPORT_VISIBILITY_LABELS,
} from "@/types/report";

const VISIBILITY_CHOICES = Object.entries(REPORT_VISIBILITY_LABELS).map(([value, label]) => ({
  value,
  label,
}));

const STATUS_CHOICES = Object.entries(REPORT_DEFINITION_STATUS_LABELS)
  .filter(([value]) => value !== "needs_attention")
  .map(([value, label]) => ({ value, label }));

export type ReportMeta = {
  name: string;
  description: string;
  category: string;
  tags: string[];
  visibility: string;
  status: string;
  defaultFormat: string;
};

export const DEFAULT_REPORT_META: ReportMeta = {
  name: "",
  description: "",
  category: "custom",
  tags: [],
  visibility: "private",
  status: "active",
  defaultFormat: "xlsx",
};

type SaveReportDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  meta: ReportMeta;
  onMetaChange: (meta: ReportMeta) => void;
  onSave: () => void;
  saving: boolean;
  isNew: boolean;
};

export function SaveReportDialog({
  open,
  onOpenChange,
  meta,
  onMetaChange,
  onSave,
  saving,
  isNew,
}: SaveReportDialogProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isNew ? "Save Report" : "Save Changes"}</DialogTitle>
          <DialogDescription>
            {t("Saving creates a new revision — runs always execute against a specific revision.")}
          </DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="report-meta-name">{t("Name")}</Label>
            <Input
              id="report-meta-name"
              value={meta.name}
              onChange={(event) => onMetaChange({ ...meta, name: event.target.value })}
              placeholder={t("Weekly Revenue by Customer")}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="report-meta-description">{t("Description")}</Label>
            <Textarea
              id="report-meta-description"
              value={meta.description}
              onChange={(event) => onMetaChange({ ...meta, description: event.target.value })}
              rows={2}
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="report-meta-category">{t("Category")}</Label>
              <Select
                value={meta.category}
                onValueChange={(category) => {
                  if (category) onMetaChange({ ...meta, category });
                }}
                items={REPORT_CATEGORY_CHOICES}
              >
                <SelectTrigger id="report-meta-category">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {REPORT_CATEGORY_CHOICES.map((choice) => (
                    <SelectItem key={choice.value} value={choice.value}>
                      {choice.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="report-meta-format">{t("Default Format")}</Label>
              <Select
                value={meta.defaultFormat}
                onValueChange={(defaultFormat) => {
                  if (defaultFormat) onMetaChange({ ...meta, defaultFormat });
                }}
                items={REPORT_FORMAT_CHOICES}
              >
                <SelectTrigger id="report-meta-format">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {REPORT_FORMAT_CHOICES.map((choice) => (
                    <SelectItem key={choice.value} value={choice.value}>
                      {choice.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="report-meta-visibility">{t("Visibility")}</Label>
              <Select
                value={meta.visibility}
                onValueChange={(visibility) => {
                  if (visibility) onMetaChange({ ...meta, visibility });
                }}
                items={VISIBILITY_CHOICES}
              >
                <SelectTrigger id="report-meta-visibility">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {VISIBILITY_CHOICES.map((choice) => (
                    <SelectItem key={choice.value} value={choice.value}>
                      {choice.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="report-meta-status">{t("Status")}</Label>
              <Select
                value={meta.status}
                onValueChange={(status) => {
                  if (status) onMetaChange({ ...meta, status });
                }}
                items={STATUS_CHOICES}
              >
                <SelectTrigger id="report-meta-status">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {STATUS_CHOICES.map((choice) => (
                    <SelectItem key={choice.value} value={choice.value}>
                      {choice.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="report-meta-tags">{t("Tags")}</Label>
            <Input
              id="report-meta-tags"
              value={meta.tags.join(", ")}
              placeholder={t("revenue, weekly")}
              onChange={(event) =>
                onMetaChange({
                  ...meta,
                  tags: event.target.value
                    .split(",")
                    .map((tag) => tag.trim())
                    .filter(Boolean),
                })
              }
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button onClick={onSave} disabled={saving || meta.name.trim() === ""}>
            {saving ? "Saving..." : "Save Report"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
