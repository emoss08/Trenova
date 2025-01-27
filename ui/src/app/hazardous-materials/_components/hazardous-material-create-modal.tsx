import { Button } from "@/components/ui/button";
import { FormCreateModal } from "@/components/ui/form-create-modal";
import { Icon } from "@/components/ui/icons";
import { ExternalLink } from "@/components/ui/link";
import {
  hazardousMaterialSchema,
  type HazardousMaterialSchema,
} from "@/lib/schemas/hazardous-material-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
} from "@/types/hazardous-material";
import { faInfoCircle, faXmark } from "@fortawesome/pro-regular-svg-icons";
import { yupResolver } from "@hookform/resolvers/yup";
import { useLocalStorage } from "@uidotdev/usehooks";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-form";

export function CreateHazardousMaterialModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<HazardousMaterialSchema>({
    resolver: yupResolver(hazardousMaterialSchema),
    defaultValues: {
      code: "",
      status: Status.Active,
      description: "",
      class: HazardousClassChoiceProps.HazardClass1And1,
      packingGroup: PackingGroupChoiceProps.PackingGroupIII,
      properShippingName: "",
      handlingInstructions: "",
      emergencyContact: "",
      placardRequired: false,
      isReportableQuantity: false,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Hazardous Material"
      formComponent={<HazardousMaterialForm />}
      form={form}
      schema={hazardousMaterialSchema}
      url="/hazardous-materials/"
      queryKey="hazardous-material-list"
      className="max-w-[550px]"
      notice={<HazardousMaterialNotice />}
    />
  );
}

function HazardousMaterialNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    "showHazardousMaterialNotice",
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
              This notice is provided to emphasize the importance of complying
              with federal Hazardous Material regulations. For details on proper
              handling and safety, please consult the official{" "}
              <ExternalLink href="https://www.fmcsa.dot.gov/regulations/hazardous-materials/how-comply-federal-hazardous-materials-regulations">
                FMCSA documentation
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
