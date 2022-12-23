// material-ui
import { Theme } from '@mui/material/styles';

// ==============================|| OVERRIDES - ALERT TITLE ||============================== //

export default function AccordionDetails(theme: Theme) {
  return {
    MuiAccordionDetails: {
      styleOverrides: {
        root: {
          padding: theme.spacing(2),
          borderTop: `1px solid ${theme.palette.secondary.light}`
        }
      }
    }
  };
}
