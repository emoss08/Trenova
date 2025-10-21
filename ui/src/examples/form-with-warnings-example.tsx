/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/**
 * Example of how to use validation warnings in a form
 * This demonstrates handling LOW priority validation errors as warnings
 * that don't block form submission but provide helpful feedback
 */

import { useForm } from "react-hook-form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useValidationWarnings } from "@/hooks/use-validation-warnings";
import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";

interface HoldReasonFormData {
  code: string;
  label: string;
  description?: string;
  defaultSeverity: string;
}

export function HoldReasonFormExample() {
  const form = useForm<HoldReasonFormData>();
  const { warnings, addWarnings, getWarning, clearAllWarnings } =
    useValidationWarnings();

  const mutation = useApiMutation({
    mutationFn: async (data: HoldReasonFormData) => {
      // Your API call here
      const response = await fetch("/api/hold-reasons", {
        method: "POST",
        body: JSON.stringify(data),
      });
      return response.json();
    },
    setFormError: form.setError,
    resourceName: "Hold Reason",
    allowLowPrioritySubmission: true,
    onLowPriorityErrors: (lowPriorityErrors) => {
      // Add warnings to state
      addWarnings(lowPriorityErrors);

      // Optionally show a toast notification
      if (lowPriorityErrors.length > 0) {
        toast.info("Form submitted with warnings", {
          description: `${lowPriorityErrors.length} warning(s) were found but the form was still submitted.`,
        });
      }
    },
    onSuccess: () => {
      toast.success("Hold reason created successfully");
      form.reset();
      clearAllWarnings();
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    // Clear previous warnings
    clearAllWarnings();

    // Submit the form
    await mutation.mutateAsync(data);
  });

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <InputField
        name="code"
        control={form.control}
        label="Code"
        description="A unique identifier for this hold reason"
        rules={{ required: "Code is required" }}
        warning={getWarning("code")}
        placeholder="e.g., DAMAGED_GOODS"
      />

      <InputField
        name="label"
        control={form.control}
        label="Label"
        description="Human-readable name for this hold reason"
        rules={{ required: "Label is required" }}
        warning={getWarning("label")}
        placeholder="e.g., Damaged Goods"
      />

      <InputField
        name="description"
        control={form.control}
        label="Description"
        description="Optional detailed description"
        warning={getWarning("description")}
        placeholder="Provide additional context..."
      />

      <div className="flex gap-2">
        <Button type="submit" disabled={mutation.isPending}>
          {mutation.isPending ? "Saving..." : "Save Hold Reason"}
        </Button>

        {warnings && Object.keys(warnings).length > 0 && (
          <div className="text-sm text-yellow-600">
            Form has {Object.keys(warnings).length} warning(s) but can still be
            submitted
          </div>
        )}
      </div>
    </form>
  );
}

/**
 * Example API response with validation priorities:
 *
 * {
 *   "type": "validation-error",
 *   "status": 400,
 *   "invalidParams": [
 *     {
 *       "name": "code",
 *       "reason": "Code must be at least 3 characters",
 *       "code": "MinLength",
 *       "priority": "HIGH"  // Blocks submission
 *     },
 *     {
 *       "name": "description",
 *       "reason": "Consider adding a description for better documentation",
 *       "code": "Suggestion",
 *       "priority": "LOW"   // Shows as warning, doesn't block
 *     }
 *   ]
 * }
 */
