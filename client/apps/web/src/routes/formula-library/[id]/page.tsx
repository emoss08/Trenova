import { StudioSkeleton } from "@/components/tariff-studio/studio-skeletons";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { formulaLibraryRoutes } from "@/lib/formula-library-routes";
import { FormulaLibraryNotFoundError } from "@/lib/graphql/formula-library";
import { formulaLibrary, invalidateFormulaLibrary } from "@/lib/queries/formula-library";
import { formulaTemplate } from "@/lib/queries/formula-template";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertCircleIcon } from "@trenova/shared/components/icons";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formulaLibrarySchema,
  type FormulaLibrary,
  type FormulaLibraryFormValues,
} from "@trenova/shared/types/formula-library";
import { useEffect, useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { useNavigate, useParams } from "react-router";
import { LibraryStudio } from "../_components/studio/library-studio";
import {
  EMPTY_LIBRARY,
  libraryFormValues,
  saveDemotesLibrary,
} from "../_components/studio/library-form";

export const prefetch: RoutePrefetch = ({ params }) => [
  formulaTemplate.context,
  ...(params.id ? [formulaLibrary.get(params.id)] : []),
];

export function FormulaLibraryEditPage() {
  const t = useT();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [seatedVersion, setSeatedVersion] = useState<number | null>(null);
  const [pendingSave, setPendingSave] = useState<FormulaLibraryFormValues | null>(null);

  const {
    data: library,
    isLoading,
    isError,
    error,
  } = useQuery({
    ...formulaLibrary.get(id ?? ""),
    enabled: !!id,
  });
  useBreadcrumbLabel(library ? `org.${library.name}` : undefined);

  const form = useForm<FormulaLibraryFormValues>({
    resolver: zodResolver(formulaLibrarySchema),
    defaultValues: EMPTY_LIBRARY,
  });

  useEffect(() => {
    if (library && library.version !== seatedVersion) {
      form.reset(libraryFormValues(library));
      // oxlint-disable-next-line react/set-state-in-effect -- form.reset must run before seatedVersion lets the studio mount, so both stay in the effect
      setSeatedVersion(library.version);
    }
  }, [library, seatedVersion, form]);

  const reseat = (saved: FormulaLibrary) => {
    form.reset(libraryFormValues(saved));
    setSeatedVersion(saved.version ?? null);
  };

  const { mutate } = useApiMutation({
    mutationFn: (values: FormulaLibraryFormValues) =>
      apiService.formulaLibraryService.update(id ?? "", {
        name: values.name.trim(),
        description: values.description ?? "",
        source: values.source,
        version: library?.version,
      }),
    onSuccess: async (updated) => {
      reseat(updated);
      await invalidateFormulaLibrary(queryClient);
    },
    form,
    resourceName: "Formula Library",
  });

  const onSave = form.handleSubmit((values) => {
    if (seatedVersion === null) return;
    if (saveDemotesLibrary(library ?? null, values)) {
      setPendingSave(values);
      return;
    }
    mutate(values);
  });

  if (isError) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
        <AlertCircleIcon className="text-destructive size-10" />
        <p className="text-sm font-medium">
          {error instanceof FormulaLibraryNotFoundError
            ? t("This formula library does not exist, or it is not your organization's.")
            : t("This formula library could not be loaded.")}
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => void navigate(formulaLibraryRoutes.list)}
        >
          {t("Back to formula libraries")}
        </Button>
      </div>
    );
  }

  if (isLoading || seatedVersion === null || !library) {
    return <StudioSkeleton inputs={false} label={t("Loading the library")} />;
  }

  return (
    <FormProvider {...form}>
      <LibraryStudio
        mode="edit"
        library={library}
        isSubmitting={form.formState.isSubmitting}
        onSave={() => void onSave()}
        onLibraryChanged={reseat}
      />
      <AlertDialog
        open={pendingSave !== null}
        onOpenChange={(open) => {
          if (!open) setPendingSave(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle className="text-lg font-semibold">
              {library.status === "InReview"
                ? t("Cancel the review and save?")
                : t("Return the library to draft?")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {library.status === "InReview"
                ? t(
                    "Saving makes a new version and cancels the review in progress. Templates keep the version they pinned.",
                  )
                : t(
                    "Saving makes a new version as a draft. {0, plural, one {# template version keeps} other {# template versions keep}} the version they pinned until a new template version pins this one.",
                    library.usageCount,
                  )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="outline" size="default">
              {t("Keep editing")}
            </AlertDialogCancel>
            <AlertDialogAction
              size="default"
              onClick={() => {
                if (pendingSave) mutate(pendingSave);
                setPendingSave(null);
              }}
            >
              {t("Save as draft")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </FormProvider>
  );
}
