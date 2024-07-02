import { truncateText } from "@/lib/utils";
import { CaretSortIcon } from "@radix-ui/react-icons";

// TODO: Implement this when workflows are available
export function WorkflowPlaceholder() {
  return (
    <div className="group col-span-full flex w-full select-none items-center gap-x-4 rounded-lg border border-dashed border-blue-200 bg-blue-200 p-1 px-4 hover:cursor-pointer dark:border-blue-500 dark:bg-blue-600/20 dark:text-blue-400">
      <div className="flex flex-1 flex-col">
        <p className="text-foreground text-sm dark:text-blue-100">Workflow</p>
        <h2 className="truncate text-lg font-semibold leading-7 text-blue-600 dark:text-blue-400">
          {truncateText("Operation Management", 20)}
        </h2>
      </div>
      <div className="ml-auto flex items-center justify-center">
        <CaretSortIcon className="size-6 text-blue-600 dark:text-blue-400" />
      </div>
    </div>
  );
}
