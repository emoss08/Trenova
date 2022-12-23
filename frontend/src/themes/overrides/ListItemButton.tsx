// material-ui
import { Theme } from '@mui/material/styles';

// ==============================|| OVERRIDES - LIST ITEM ICON ||============================== //

export default function ListItemButton(theme: Theme) {
  return {
    MuiListItemButton: {
      styleOverrides: {
        root: {
          '&.Mui-selected': {
            color: theme.palette.primary.main,
            '& .MuiListItemIcon-root': {
              color: theme.palette.primary.main
            }
          }
        }
      }
    }
  };
}
