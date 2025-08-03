/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// import { Button } from "@/components/ui/button";
// import {
//   Sheet,
//   SheetBody,
//   SheetContent,
//   SheetDescription,
//   SheetFooter,
//   SheetHeader,
//   SheetTitle,
// } from "@/components/ui/sheet";
// import { useApiMutation } from "@/hooks/use-api-mutation";
// import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
// import {
//   consolidationGroupSchema,
//   type ConsolidationGroupSchema,
// } from "@/lib/schemas/consolidation-schema";
// import { api } from "@/services/api";
// import { type EditTableSheetProps } from "@/types/data-table";
// import { zodResolver } from "@hookform/resolvers/zod";
// import { useQuery, useQueryClient } from "@tanstack/react-query";
// import { useCallback, useEffect } from "react";
// import { FormProvider, useForm } from "react-hook-form";
// import { toast } from "sonner";
// import { ConsolidationForm } from "./consolidation-form";

// export function ConsolidationEditSheet({
//   open,
//   onOpenChange,
//   currentRecord,
// }: EditTableSheetProps<ConsolidationGroupSchema>) {
//   const queryClient = useQueryClient();

//   const { data: consolidation, isLoading } = useQuery({
//     queryKey: ["consolidation", currentRecord?.id],
//     queryFn: () =>
//       api.consolidations.getConsolidationGroupByID(currentRecord?.id),
//     enabled: !!currentRecord?.id,
//   });

//   const form = useForm<ConsolidationGroupSchema>({
//     resolver: zodResolver(consolidationGroupSchema),
//     defaultValues: consolidation || {},
//   });

//   useEffect(() => {
//     if (consolidation) {
//       form.reset(consolidation);
//     }
//   }, [consolidation, form]);

//   const updateMutation = useApiMutation({
//     mutationFn: (values: Partial<ConsolidationGroupSchema>) =>
//       api.consolidations.update(currentRecord!.id, values),
//     onSuccess: async () => {
//       await broadcastQueryInvalidation("consolidation-list");
//       await queryClient.invalidateQueries({
//         queryKey: ["consolidation-list"],
//       });
//       await queryClient.invalidateQueries({
//         queryKey: ["consolidation", currentRecord?.id],
//       });
//       toast.success("Consolidation updated successfully");
//       onOpenChange(false);
//     },
//     onError: (error) => {
//       toast.error("Failed to update consolidation", {
//         description: error.message,
//       });
//     },
//   });

//   const handleSubmit = useCallback(
//     async (values: ConsolidationGroupSchema) => {
//       if (!currentRecord?.id) return;
//       await updateMutation.mutateAsync(values);
//     },
//     [currentRecord?.id, updateMutation],
//   );

//   if (isLoading) {
//     return (
//       <Sheet open={open} onOpenChange={onOpenChange}>
//         <SheetContent className="w-full max-w-[600px]">
//           <div className="flex items-center justify-center h-full">
//             <p>Loading...</p>
//           </div>
//         </SheetContent>
//       </Sheet>
//     );
//   }

//   return (
//     <Sheet open={open} onOpenChange={onOpenChange}>
//       <SheetContent className="w-full max-w-[600px]">
//         <SheetHeader>
//           <SheetTitle>Edit Consolidation</SheetTitle>
//           <SheetDescription>
//             Update the consolidation group details and manage shipments.
//           </SheetDescription>
//         </SheetHeader>
//         <FormProvider {...form}>
//           <form onSubmit={form.handleSubmit(handleSubmit)}>
//             <SheetBody>
//               <ConsolidationForm isEdit />
//             </SheetBody>
//             <SheetFooter>
//               <Button
//                 type="button"
//                 variant="outline"
//                 onClick={() => onOpenChange(false)}
//                 disabled={updateMutation.isPending}
//               >
//                 Cancel
//               </Button>
//               <Button type="submit" disabled={updateMutation.isPending}>
//                 {updateMutation.isPending ? "Updating..." : "Update"}
//               </Button>
//             </SheetFooter>
//           </form>
//         </FormProvider>
//       </SheetContent>
//     </Sheet>
//   );
// }
