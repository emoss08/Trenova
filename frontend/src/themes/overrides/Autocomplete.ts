// ==============================|| OVERRIDES - AUTOCOMPLETE ||============================== //

export default function Autocomplete() {
  return {
    MuiAutocomplete: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            padding: '3px 9px'
          }
        },
        popupIndicator: {
          width: 'auto',
          height: 'auto'
        },
        clearIndicator: {
          width: 'auto',
          height: 'auto'
        }
      }
    }
  };
}
