import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { faGear, faShareNodes } from "@fortawesome/pro-regular-svg-icons";
import { Activity, lazy, Suspense, useState } from "react";
import { AvailableNodesSkeleton } from "./workflow-skeletons";

const AvailableNodes = lazy(() => import("./available-nodes/node-palette"));

type Option = "available-nodes" | "workflow-settings";
type Options = {
  label: string;
  value: Option;
  icon: React.ReactNode;
};

const options: Options[] = [
  {
    label: "Available Nodes",
    value: "available-nodes",
    icon: <Icon icon={faShareNodes} className="size-4 shrink-0" />,
  },
  {
    label: "Workflow Settings",
    value: "workflow-settings",
    icon: <Icon icon={faGear} className="size-4 shrink-0" />,
  },
];

export default function WorkflowOptions() {
  const [selectedOption, setSelectedOption] =
    useState<Option>("available-nodes");

  const handleSelectOption = (option: Option) => {
    setSelectedOption(option);
  };

  return (
    <div className="relative flex w-fit max-w-sm shrink-0 divide-x divide-card-foreground/10 rounded-lg border border-border">
      <div className="min-w-xs grow">
        <Activity
          mode={selectedOption === "available-nodes" ? "visible" : "hidden"}
        >
          <Suspense fallback={<AvailableNodesSkeleton />}>
            <AvailableNodes />
          </Suspense>
        </Activity>
      </div>
      <div className="shrink-0 p-1.5">
        <div className="flex h-full flex-col gap-2">
          {options.map((option) => (
            <WorkflowOptionsButton
              key={option.value}
              option={option}
              selectedOption={selectedOption}
              handleSelectOption={handleSelectOption}
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function WorkflowOptionsButton({
  option,
  selectedOption,
  handleSelectOption,
}: {
  option: Options;
  selectedOption: Option;
  handleSelectOption: (option: Option) => void;
}) {
  const isSelected = selectedOption === option.value;

  return (
    <Button
      variant={isSelected ? "default" : "ghost"}
      size="icon"
      onClick={() => handleSelectOption(option.value)}
    >
      {option.icon}
    </Button>
  );
}
