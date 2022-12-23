// ==============================|| OVERRIDES - LOADING BUTTON ||============================== //

export default function LoadingButton() {
  return {
    MuiLoadingButton: {
      styleOverrides: {
        root: {
          padding: '6px 16px',
          '&.MuiLoadingButton-loading': {
            opacity: 0.6,
            textShadow: 'none'
          }
        }
      }
    }
  };
}
