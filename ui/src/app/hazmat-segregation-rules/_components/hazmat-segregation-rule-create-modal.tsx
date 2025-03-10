import { Button } from "@/components/ui/button";
import { FormCreateModal } from "@/components/ui/form-create-modal";
import { Icon } from "@/components/ui/icons";
import { ExternalLink } from "@/components/ui/link";
import { HAZMAT_SEGREGATION_RULE_NOTICE_KEY } from "@/constants/env";
import {
  hazmatSegregationRuleSchema,
  type HazmatSegregationRuleSchema,
} from "@/lib/schemas/hazmat-segregation-rule-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { HazardousClassChoiceProps } from "@/types/hazardous-material";
import { SegregationType } from "@/types/hazmat-segregation-rule";
import { faInfoCircle, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useForm } from "react-hook-form";
import { HazmatSegregationRuleForm } from "./hazmat-segregation-rule-form";

export function CreateHazmatSegregationRuleModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<HazmatSegregationRuleSchema>({
    resolver: yupResolver(hazmatSegregationRuleSchema),
    defaultValues: {
      status: Status.Active,
      name: "",
      description: "",
      classA: HazardousClassChoiceProps.HazardClass1And1,
      classB: HazardousClassChoiceProps.HazardClass1And1,
      segregationType: SegregationType.Separated,
      minimumDistance: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Hazmat Segregation Rule"
      formComponent={<HazmatSegregationRuleForm />}
      form={form}
      schema={hazmatSegregationRuleSchema}
      url="/hazmat-segregation-rules/"
      queryKey="hazmat-segregation-rule-list"
      className="max-w-[550px]"
      notice={<HazmatSegregationRuleNotice />}
    />
  );
}

function HazmatSegregationRuleNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    HAZMAT_SEGREGATION_RULE_NOTICE_KEY,
    true,
  );

  const handleClose = () => {
    setNoticeVisible(false);
  };

  return noticeVisible ? (
    <div className="bg-muted px-4 py-3 text-foreground">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-foreground"
            aria-hidden="true"
          />
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <span className="text-sm">
              This notice is provided to emphasize the importance of segregating
              hazardous materials in accordance with federal regulations. For
              details on proper handling and safety, please consult the official{" "}
              <ExternalLink href="https://www.ecfr.gov/current/title-49/subtitle-B/chapter-I/subchapter-C/part-177/subpart-C/section-177.848">
                CFR 177.848
              </ExternalLink>
              .
            </span>
          </div>
        </div>
        <Button
          variant="secondary"
          className="group -my-1.5 -me-2 size-8 shrink-0 p-0 hover:bg-transparent"
          onClick={handleClose}
          aria-label="Close banner"
        >
          <Icon
            icon={faXmark}
            className="opacity-60 transition-opacity group-hover:opacity-100"
            aria-hidden="true"
          />
        </Button>
      </div>
    </div>
  ) : null;
}
