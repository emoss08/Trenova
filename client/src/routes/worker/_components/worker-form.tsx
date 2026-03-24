import { ComplianceTab, EmploymentTab, GeneralTab } from "./worker-form-tabs";

export function WorkerCreateForm() {
  return (
    <div className="space-y-8">
      <GeneralTab />
      <EmploymentTab />
      <ComplianceTab />
    </div>
  );
}
