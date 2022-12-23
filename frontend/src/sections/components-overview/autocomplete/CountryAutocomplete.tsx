// material-ui
import { Autocomplete, Box, TextField } from '@mui/material';

// project import
import countries from 'data/countries';
import MainCard from 'components/MainCard';

// ==============================|| AUTOCOMPLETE - COUNTRY ||============================== //

export default function CountryAutocomplete() {
  const countryAutocompleteCodeString = `<Autocomplete
  id="country-select-demo"
  fullWidth
  options={countries}
  autoHighlight
  getOptionLabel={(option) => option.label}
  renderOption={(props, option) => (
    <Box component="li" sx={{ '& > img': { mr: 2, flexShrink: 0 } }} {...props}>
      {option.code && (
        <img
          loading="lazy"
          width="20"
          src={https://flagcdn.com/w20/{option.code.toLowerCase()}.png}
          srcSet={https://flagcdn.com/w40/{option.code.toLowerCase()}.png 2x}
          alt=""
        />
      )}
      {option.label} ({option.code}) +{option.phone}
    </Box>
  )}
  renderInput={(params) => (
    <TextField
      {...params}
      label="Choose a country"
      inputProps={{
        ...params.inputProps,
        autoComplete: 'new-password' // disable autocomplete and autofill
      }}
    />
  )}
/>`;

  return (
    <MainCard title="With Image" codeString={countryAutocompleteCodeString}>
      <Autocomplete
        id="country-select-demo"
        fullWidth
        options={countries}
        autoHighlight
        getOptionLabel={(option) => option.label}
        renderOption={(props, option) => (
          <Box component="li" sx={{ '& > img': { mr: 2, flexShrink: 0 } }} {...props}>
            {option.code && (
              <img
                loading="lazy"
                width="20"
                src={`https://flagcdn.com/w20/${option.code.toLowerCase()}.png`}
                srcSet={`https://flagcdn.com/w40/${option.code.toLowerCase()}.png 2x`}
                alt=""
              />
            )}
            {option.label} ({option.code}) +{option.phone}
          </Box>
        )}
        renderInput={(params) => (
          <TextField
            {...params}
            label="Choose a country"
            inputProps={{
              ...params.inputProps,
              autoComplete: 'new-password' // disable autocomplete and autofill
            }}
          />
        )}
      />
    </MainCard>
  );
}
