// eslint-disable-next-line
import * as createPalette from '@mui/material/styles';

declare module '@mui/material/styles' {
  interface SimplePaletteColorOptions {
    lighter?: string;
    darker?: string;
    0?: string;
    50?: string;
    100?: string;
    200?: string;
    300?: string;
    400?: string;
    500?: string;
    600?: string;
    700?: string;
    800?: string;
    900?: string;
    A50?: string;
    A100?: string;
    A200?: string;
    A300?: string;
    A400?: string;
    A700?: string;
    A800?: string;
  }

  interface PaletteColor {
    lighter: string;
    darker: string;
    0?: string;
    50?: string;
    100?: string;
    200?: string;
    300?: string;
    400?: string;
    500?: string;
    600?: string;
    700?: string;
    800?: string;
    900?: string;
    A50?: string;
    A100?: string;
    A200?: string;
    A300?: string;
    A400?: string;
    A700?: string;
    A800?: string;
  }
}
