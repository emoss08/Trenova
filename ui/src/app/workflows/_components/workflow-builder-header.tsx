import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { useNavigate } from "react-router";

export function WorkflowBuilderHeader() {
  const navigate = useNavigate();

  return (
    <div className="flex items-center gap-4">
      <Button
        variant="link"
        className="text-lg [&_svg]:size-4"
        onClick={() => navigate("/organization/workflows")}
      >
        <ArrowLeft className="size-6" />
        Back to Workflows
      </Button>
    </div>
  );
}
