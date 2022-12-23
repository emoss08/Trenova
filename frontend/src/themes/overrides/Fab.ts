// material-ui
import { alpha, Theme } from '@mui/material/styles';

// project import
import getColors from 'utils/getColors';
import getShadow from 'utils/getShadow';

// types
import { ExtendedStyleProps } from 'types/extended';

// ==============================|| BUTTON - COLORS ||============================== //

function getColorStyle({ color, theme }: ExtendedStyleProps) {
  const colors = getColors(theme, color);
  const { main, dark, contrastText } = colors;

  const buttonShadow = `${color}Button`;
  const shadows = getShadow(theme, buttonShadow);

  return {
    color: contrastText,
    backgroundColor: main,
    boxShadow: shadows,
    '&:hover': {
      boxShadow: 'none',
      backgroundColor: dark
    },
    '&:focus-visible': {
      outline: `2px solid ${dark}`,
      outlineOffset: 2
    },
    '&::after': {
      borderRadius: '50px',
      boxShadow: `0 0 5px 5px ${alpha(main, 0.9)}`
    },
    '&:active::after': {
      borderRadius: '50px',
      boxShadow: `0 0 0 0 ${alpha(main, 0.9)}`
    }
  };
}

// ==============================|| OVERRIDES - BUTTON ||============================== //

export default function Button(theme: Theme) {
  return {
    MuiFab: {
      styleOverrides: {
        root: {
          fontWeight: 400,
          '&.Mui-disabled': {
            backgroundColor: theme.palette.grey[200]
          },
          '&.MuiFab-primary': getColorStyle({ color: 'primary', theme }),
          '&.MuiFab-secondary': getColorStyle({ color: 'secondary', theme }),
          '&.Mui-error': getColorStyle({ color: 'error', theme }),
          '&.MuiFab-success': getColorStyle({ color: 'success', theme }),
          '&.MuiFab-info': getColorStyle({ color: 'info', theme }),
          '&.MuiFab-warning': getColorStyle({ color: 'warning', theme }),
          '&::after': {
            content: '""',
            display: 'block',
            position: 'absolute',
            left: 0,
            top: 0,
            width: '100%',
            height: '100%',
            borderRadius: 4,
            opacity: 0,
            transition: 'all 0.5s'
          },

          '&:active::after': {
            position: 'absolute',
            borderRadius: 4,
            left: 0,
            top: 0,
            opacity: 1,
            transition: '0s'
          }
        }
      }
    }
  };
}
