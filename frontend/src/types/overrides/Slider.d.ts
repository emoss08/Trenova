// eslint-disable-next-line
import * as Slider from '@mui/material/Slider';

declare module '@mui/material/Slider' {
  interface SliderPropsColorOverrides {
    error;
    success;
    warning;
    info;
  }
}
