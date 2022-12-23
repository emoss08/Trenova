// material-ui
import { Theme } from '@mui/material/styles';
import { PaginationProps } from '@mui/material';

// project import
import getColors from 'utils/getColors';

// types
import { ExtendedStyleProps } from 'types/extended';

// ==============================|| PAGINATION ITEM - COLORS ||============================== //

interface PaginationStyleProps extends ExtendedStyleProps {
  variant: PaginationProps['variant'];
}

function getColorStyle({ variant, color, theme }: PaginationStyleProps) {
  const colors = getColors(theme, color);
  const { lighter, light, dark, main, contrastText } = colors;

  const focusStyle = {
    '&:focus-visible': {
      outline: `2px solid ${dark}`,
      outlineOffset: 2,
      ...(variant === 'text' && {
        backgroundColor: theme.palette.background.paper
      })
    }
  };

  switch (variant) {
    case 'combined':
    case 'contained':
      return {
        color: contrastText,
        backgroundColor: main,
        '&:hover': {
          backgroundColor: light
        },
        ...focusStyle
      };
    case 'outlined':
      return {
        borderColor: main,
        '&:hover': {
          backgroundColor: lighter,
          borderColor: light
        },
        ...focusStyle
      };
    case 'text':
    default:
      return {
        color: main,
        '&:hover': {
          backgroundColor: main,
          color: lighter
        },
        ...focusStyle
      };
  }
}

// ==============================|| OVERRIDES - PAGINATION ITEM ||============================== //

export default function PaginationItem(theme: Theme) {
  return {
    MuiPaginationItem: {
      styleOverrides: {
        root: {
          '&:focus-visible': {
            outline: `2px solid ${theme.palette.secondary.dark}`,
            outlineOffset: 2
          }
        },
        text: {
          '&.Mui-selected': {
            backgroundColor: 'transparent',
            fontSize: '1rem',
            fontWeight: 500,
            '&.MuiPaginationItem-textPrimary': getColorStyle({ variant: 'text', color: 'primary', theme }),
            '&.MuiPaginationItem-textSecondary': getColorStyle({ variant: 'text', color: 'secondary', theme }),
            '&.MuiPaginationItem-textError': getColorStyle({ variant: 'text', color: 'error', theme }),
            '&.MuiPaginationItem-textSuccess': getColorStyle({ variant: 'text', color: 'success', theme }),
            '&.MuiPaginationItem-textInfo': getColorStyle({ variant: 'text', color: 'info', theme }),
            '&.MuiPaginationItem-textWarning': getColorStyle({ variant: 'text', color: 'warning', theme })
          }
        },
        contained: {
          '&.Mui-selected': {
            '&.MuiPaginationItem-containedPrimary': getColorStyle({ variant: 'contained', color: 'primary', theme }),
            '&.MuiPaginationItem-containedSecondary': getColorStyle({ variant: 'contained', color: 'secondary', theme }),
            '&.MuiPaginationItem-containedError': getColorStyle({ variant: 'contained', color: 'error', theme }),
            '&.MuiPaginationItem-containedSuccess': getColorStyle({ variant: 'contained', color: 'success', theme }),
            '&.MuiPaginationItem-containedInfo': getColorStyle({ variant: 'contained', color: 'info', theme }),
            '&.MuiPaginationItem-containedWarning': getColorStyle({ variant: 'contained', color: 'warning', theme })
          }
        },
        combined: {
          border: '1px solid',
          borderColor: theme.palette.divider,
          '&.MuiPaginationItem-ellipsis': {
            border: 'none'
          },
          '&.Mui-selected': {
            '&.MuiPaginationItem-combinedPrimary': getColorStyle({ variant: 'combined', color: 'primary', theme }),
            '&.MuiPaginationItem-combinedSecondary': getColorStyle({ variant: 'combined', color: 'secondary', theme }),
            '&.MuiPaginationItem-combinedError': getColorStyle({ variant: 'combined', color: 'error', theme }),
            '&.MuiPaginationItem-combinedSuccess': getColorStyle({ variant: 'combined', color: 'success', theme }),
            '&.MuiPaginationItem-combinedInfo': getColorStyle({ variant: 'combined', color: 'info', theme }),
            '&.MuiPaginationItem-combinedWarning': getColorStyle({ variant: 'combined', color: 'warning', theme })
          }
        },
        outlined: {
          borderColor: theme.palette.divider,
          '&.Mui-selected': {
            backgroundColor: 'transparent',
            '&.MuiPaginationItem-outlinedPrimary': getColorStyle({ variant: 'outlined', color: 'primary', theme }),
            '&.MuiPaginationItem-outlinedSecondary': getColorStyle({ variant: 'outlined', color: 'secondary', theme }),
            '&.MuiPaginationItem-outlinedError': getColorStyle({ variant: 'outlined', color: 'error', theme }),
            '&.MuiPaginationItem-outlinedSuccess': getColorStyle({ variant: 'outlined', color: 'success', theme }),
            '&.MuiPaginationItem-outlinedInfo': getColorStyle({ variant: 'outlined', color: 'info', theme }),
            '&.MuiPaginationItem-outlinedWarning': getColorStyle({ variant: 'outlined', color: 'warning', theme })
          }
        }
      }
    }
  };
}
