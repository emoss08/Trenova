// material-ui
import { Theme } from '@mui/material/styles';

// ==============================|| OVERRIDES - DIALOG CONTENT TEXT ||============================== //

export default function Popover(theme: Theme) {
  return {
    MuiPopover: {
      styleOverrides: {
        paper: {
          boxShadow: theme.customShadows.z1
        }
      }
    }
  };
}
