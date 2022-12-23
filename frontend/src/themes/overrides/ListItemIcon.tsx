// material-ui
import { Theme } from '@mui/material/styles';

// ==============================|| OVERRIDES - LIST ITEM ICON ||============================== //

export default function ListItemIcon(theme: Theme) {
  return {
    MuiListItemIcon: {
      styleOverrides: {
        root: {
          minWidth: 24,
          color: theme.palette.text.primary
        }
      }
    }
  };
}
