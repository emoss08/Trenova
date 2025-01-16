import { ExtendedSortingState, Filter } from "@/types/data-table";
import { PaginationState } from "@tanstack/react-table";

interface QueryBuilderOptions<TData> {
  baseUrl: string;
  pagination: PaginationState;
  filters?: Filter<TData>[];
  sorting?: ExtendedSortingState<TData>;
  joinOperator?: "and" | "or";
  extraParams?: Record<string, any>;
}

interface QueryParams {
  limit: number;
  offset: number;
  filter?: string;
  sort?: string;
  [key: string]: any;
}

export class QueryBuilder<TData> {
  private params: QueryParams;

  constructor(private options: QueryBuilderOptions<TData>) {
    this.params = {
      limit: options.pagination.pageSize,
      offset: options.pagination.pageIndex * options.pagination.pageSize,
    };
  }

  private buildFilterQuery(): void {
    if (!this.options.filters?.length) return;

    const filterQueries = this.options.filters
      .map((filter) => {
        let value = filter.value;

        // Handle array values (multi-select)
        if (Array.isArray(value)) {
          value = value.join(",");
        }

        // Handle special operators
        switch (filter.operator) {
          case "isEmpty":
            return `${filter.id} is null`;
          case "isNotEmpty":
            return `${filter.id} is not null`;
          case "iLike":
            return `${filter.id} ilike '%${value}%'`;
          case "isBetween":
            if (Array.isArray(filter.value) && filter.value.length === 2) {
              return `${filter.id} between '${filter.value[0]}' and '${filter.value[1]}'`;
            }
            return "";
          default:
            return `${filter.id} ${filter.operator} '${value}'`;
        }
      })
      .filter(Boolean);

    if (filterQueries.length) {
      this.params.filter = filterQueries.join(
        ` ${this.options.joinOperator || "and"} `,
      );
    }
  }

  private buildSortQuery(): void {
    if (!this.options.sorting?.length) return;

    this.params.sort = this.options.sorting
      .map((sort) => `${sort.id} ${sort.desc ? "desc" : "asc"}`)
      .join(",");
  }

  private addExtraParams(): void {
    if (this.options.extraParams) {
      Object.entries(this.options.extraParams).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          this.params[key] = value;
        }
      });
    }
  }

  build(): string {
    this.buildFilterQuery();
    this.buildSortQuery();
    this.addExtraParams();

    const url = new URL(this.options.baseUrl);
    Object.entries(this.params).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        url.searchParams.set(key, value.toString());
      }
    });

    return url.toString();
  }
}
