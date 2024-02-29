import type {
  DataTableFilterableColumn,
  DataTableSearchableColumn,
} from "@/types/tables";
import {
  type ColumnDef,
  type ColumnFiltersState,
  getCoreRowModel,
  getFacetedRowModel,
  getFacetedUniqueValues,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type PaginationState,
  type SortingState,
  useReactTable,
  type VisibilityState,
} from "@tanstack/react-table";
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { useDebounce } from "./useDebounce";
import { z } from "zod";
import { ApiResponse } from "@/types/server";
import axios from "@/lib/axiosConfig";
import { API_URL } from "@/lib/constants";
import { useQuery } from "@tanstack/react-query";
import { QueryKeys } from "@/types";

function useDataTableQuery<K>(
  queryKey: QueryKeys | string,
  link: string,
  pageIndex: number,
  pageSize: number,
  extraSearchParams?: Record<string, any>,
) {
  return useQuery<ApiResponse<K>, Error>({
    queryKey: [queryKey, link, pageIndex, pageSize, extraSearchParams],
    queryFn: () => fetchData<K>(link, pageIndex, pageSize, extraSearchParams),
  });
}

// Separate function for the fetch logic

async function fetchData<K>(
  link: string,
  pageIndex: number,
  pageSize: number,
  extraSearchParams?: Record<string, any>,
): Promise<ApiResponse<K>> {
  const fetchURL = new URL(`${API_URL}${link}`);
  fetchURL.searchParams.set("limit", pageSize.toString());
  fetchURL.searchParams.set("offset", (pageIndex * pageSize).toString());
  if (extraSearchParams) {
    Object.entries(extraSearchParams).forEach(([key, value]) =>
      fetchURL.searchParams.set(key, value),
    );
  }

  const response = await axios.get<ApiResponse<K>>(fetchURL.href);
  if (response.status !== 200) {
    throw new Error("Failed to fetch data from server");
  }
  return response.data;
}

interface UseDataTableProps<TData, TValue> {
  /**
   * The data for the table
   * @default []
   * @type TData[]
   */
  data: TData[];

  /**
   * The columns of the table
   * @default []
   * @type ColumnDef<TData, TValue>[]
   */
  columns: ColumnDef<TData, TValue>[];

  /**
   * The number of pages in the table
   * @type number
   */
  pageCount: number;

  /**
   * The searchable columns of the table
   * @default []
   * @type {id: keyof TData, title: string}[]
   * @example searchableColumns={[{ id: "title", title: "titles" }]}
   */
  searchableColumns?: DataTableSearchableColumn<TData>[];

  /**
   * The filterable columns of the table. When provided, renders dynamic faceted filters, and the advancedFilter prop is ignored.
   * @default []
   * @type {id: keyof TData, title: string, options: { label: string, value: string, icon?: React.ComponentType<{ className?: string }> }[]}[]
   * @example filterableColumns={[{ id: "status", title: "Status", options: ["todo", "in-progress", "done", "canceled"]}]}
   */
  filterableColumns?: DataTableFilterableColumn<TData>[];
}

const schema = z.object({
  page: z.coerce.number().default(1),
  per_page: z.coerce.number().default(10),
  sort: z.string().optional(),
});

export function useDataTable<TData, TValue>({
  data,
  columns,
  pageCount,
  searchableColumns = [],
  filterableColumns = [],
}: UseDataTableProps<TData, TValue>) {
  const navigate = useNavigate();
  const pathname = location.pathname;
  const searchParams = new URLSearchParams();

  // Search params
  const {
    page,
    per_page: perPage,
    sort,
  } = schema.parse(Object.fromEntries(searchParams));
  const [column, order] = sort?.split(".") ?? [];

  // Create query string
  const createQueryString = React.useCallback(
    (params: Record<string, string | number | null>) => {
      const newSearchParams = new URLSearchParams(searchParams?.toString());

      for (const [key, value] of Object.entries(params)) {
        if (value === null) {
          newSearchParams.delete(key);
        } else {
          newSearchParams.set(key, String(value));
        }
      }

      return newSearchParams.toString();
    },
    [searchParams],
  );
  // Initial column filters
  const initialColumnFilters: ColumnFiltersState = React.useMemo(() => {
    return Array.from(searchParams.entries()).reduce<ColumnFiltersState>(
      (filters, [key, value]) => {
        const filterableColumn = filterableColumns.find(
          (column) => column.id === key,
        );
        const searchableColumn = searchableColumns.find(
          (column) => column.id === key,
        );

        if (filterableColumn) {
          filters.push({
            id: key,
            value: value.split("."),
          });
        } else if (searchableColumn) {
          filters.push({
            id: key,
            value: [value],
          });
        }

        return filters;
      },
      [],
    );
  }, [filterableColumns, searchableColumns, searchParams]);

  // Table states
  const [rowSelection, setRowSelection] = React.useState({});
  const [columnVisibility, setColumnVisibility] =
    React.useState<VisibilityState>({});
  const [columnFilters, setColumnFilters] =
    React.useState<ColumnFiltersState>(initialColumnFilters);

  // Handle server-side pagination
  const [{ pageIndex, pageSize }, setPagination] =
    React.useState<PaginationState>({
      pageIndex: page - 1,
      pageSize: perPage,
    });

  const pagination = React.useMemo(
    () => ({
      pageIndex,
      pageSize,
    }),
    [pageIndex, pageSize],
  );

  React.useEffect(() => {
    setPagination({
      pageIndex: page - 1,
      pageSize: perPage,
    });
  }, [page, perPage]);

  React.useEffect(() => {
    navigate(
      `${pathname}?${createQueryString({
        page: pageIndex + 1,
        per_page: pageSize,
      })}`,
    );

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pageIndex, pageSize]);

  // Handle server-side sorting
  const [sorting, setSorting] = React.useState<SortingState>([
    {
      id: column ?? "",
      desc: order === "desc",
    },
  ]);

  React.useEffect(() => {
    navigate(
      `${pathname}?${createQueryString({
        page,
        sort: sorting[0]?.id
          ? `${sorting[0]?.id}.${sorting[0]?.desc ? "desc" : "asc"}`
          : null,
      })}`,
    );

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sorting]);

  // Handle server-side filtering
  const debouncedSearchableColumnFilters = JSON.parse(
    useDebounce(
      JSON.stringify(
        columnFilters.filter((filter) => {
          return searchableColumns.find((column) => column.id === filter.id);
        }),
      ),
      500,
    ),
  ) as ColumnFiltersState;

  const filterableColumnFilters = columnFilters.filter((filter) => {
    return filterableColumns.find((column) => column.id === filter.id);
  });

  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    // Prevent resetting the page on initial render
    if (!mounted) {
      setMounted(true);
      return;
    }

    // Initialize new params
    const newParamsObject = {
      page: 1,
    };

    // Handle debounced searchable column filters
    for (const column of debouncedSearchableColumnFilters) {
      if (typeof column.value === "string") {
        Object.assign(newParamsObject, {
          [column.id]: typeof column.value === "string" ? column.value : null,
        });
      }
    }

    // Handle filterable column filters
    for (const column of filterableColumnFilters) {
      if (typeof column.value === "object" && Array.isArray(column.value)) {
        Object.assign(newParamsObject, { [column.id]: column.value.join(".") });
      }
    }

    // Remove deleted values
    for (const key of searchParams.keys()) {
      if (
        (searchableColumns.find((column) => column.id === key) &&
          !debouncedSearchableColumnFilters.find(
            (column) => column.id === key,
          )) ||
        (filterableColumns.find((column) => column.id === key) &&
          !filterableColumnFilters.find((column) => column.id === key))
      ) {
        Object.assign(newParamsObject, { [key]: null });
      }
    }

    // After cumulating all the changes, push new params
    navigate(`${pathname}?${createQueryString(newParamsObject)}`);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    // eslint-disable-next-line react-hooks/exhaustive-deps
    JSON.stringify(debouncedSearchableColumnFilters),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    JSON.stringify(filterableColumnFilters),
  ]);

  const table = useReactTable({
    data: data ?? [],
    columns,
    pageCount: pageCount ?? -1,
    state: {
      pagination,
      sorting,
      columnVisibility,
      rowSelection,
      columnFilters,
    },
    enableRowSelection: true,
    onRowSelectionChange: setRowSelection,
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFacetedRowModel: getFacetedRowModel(),
    getFacetedUniqueValues: getFacetedUniqueValues(),
    manualPagination: true,
    manualSorting: true,
    manualFiltering: true,
  });

  return { table };
}
