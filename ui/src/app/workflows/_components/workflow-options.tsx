"use no memo";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import {
  faNetworkWired,
  faShareNodes,
} from "@fortawesome/pro-regular-svg-icons";
import { Activity, lazy, Suspense, useState } from "react";
import {
  AvailableNodesSkeleton,
  NodesInUseSkeleton,
} from "./workflow-skeletons";

const AvailableNodes = lazy(() => import("./available-nodes/node-palette"));
const NodesInUse = lazy(() => import("./nodes-in-use/nodes-in-use"));

type Option = "available-nodes" | "nodes-in-use";
type Options = {
  label: string;
  value: Option;
  icon: React.ReactNode;
  tooltip: string;
};

const options: Options[] = [
  {
    label: "Available Nodes",
    value: "available-nodes",
    icon: <Icon icon={faShareNodes} />,
    tooltip: "Available nodes",
  },
  {
    label: "Nodes in use",
    value: "nodes-in-use",
    icon: <Icon icon={faNetworkWired} />,
    tooltip: "Nodes in use",
  },
];

export default function WorkflowOptions() {
  const [selectedOption, setSelectedOption] =
    useState<Option>("available-nodes");

  const handleSelectOption = (option: Option) => {
    setSelectedOption(option);
  };

  return (
    <WorkflowOptionsOuter>
      <WorkflowOptionsInner>
        <Activity
          mode={selectedOption === "available-nodes" ? "visible" : "hidden"}
        >
          <Suspense fallback={<AvailableNodesSkeleton />}>
            <AvailableNodes />
          </Suspense>
        </Activity>
        <Activity
          mode={selectedOption === "nodes-in-use" ? "visible" : "hidden"}
        >
          <Suspense fallback={<NodesInUseSkeleton />}>
            <NodesInUse />
          </Suspense>
        </Activity>
      </WorkflowOptionsInner>
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
    </WorkflowOptionsOuter>
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
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant={isSelected ? "default" : "ghost"}
            size="icon"
            className="[&_svg]:size-4"
            onClick={() => handleSelectOption(option.value)}
          >
            {option.icon}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="left">{option.tooltip}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function WorkflowOptionsOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex w-fit max-w-sm shrink-0 divide-x divide-card-foreground/10 rounded-lg border border-border">
      {children}
    </div>
  );
}

function WorkflowOptionsInner({ children }: { children: React.ReactNode }) {
  return <div className="min-w-xs grow">{children}</div>;
}
