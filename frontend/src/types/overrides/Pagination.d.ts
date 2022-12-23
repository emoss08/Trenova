// eslint-disable-next-line
import * as Pagination from '@mui/material/Pagination';

declare module '@mui/material/Pagination' {
  interface PaginationPropsColorOverrides {
    error;
    success;
    warning;
    info;
  }
  interface PaginationPropsVariantOverrides {
    contained;
    combined;
  }
}
