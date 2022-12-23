import { useState } from 'react';

// material-ui
import { createFilterOptions, Autocomplete, TextField } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import dataCreatable from 'data/movies';

interface FilmOptionType {
  inputValue?: string;
  label: string;
  year?: number;
}

const data: readonly FilmOptionType[] = dataCreatable;

const filter = createFilterOptions<FilmOptionType>();

// ==============================|| AUTOCOMPLETE - CREATABLE ||============================== //

export default function CreatableAutocomplete() {
  const [value, setValue] = useState<FilmOptionType | null>(null);

  const createAutocompleteCodeString = `<Autocomplete
  fullWidth
  value={value}
  onChange={(event, newValue) => {
    if (typeof newValue === 'string') {
      setValue({
        label: newValue
      });
    } else if (newValue && newValue.inputValue) {
      setValue({
        label: newValue.inputValue
      });
    } else {
      setValue(newValue);
    }
  }}
  filterOptions={(options, params) => {
    const filtered = filter(options, params);

    const { inputValue } = params;

    const isExisting = options.some((option) => inputValue === option.label);
    if (inputValue !== '' && !isExisting) {
      filtered.push({
        inputValue,
        label: 'Add {inputValue}'
      });
    }

    return filtered;
  }}
  selectOnFocus
  clearOnBlur
  handleHomeEndKeys
  id="free-solo-with-text-demo"
  options={data}
  getOptionLabel={(option) => {
    // Value selected with enter, right from the input
    if (typeof option === 'string') {
      return option;
    }
    // Add "xxx" option created dynamically
    if (option.inputValue) {
      return option.inputValue;
    }
    // Regular option
    return option.label;
  }}
  renderOption={(props, option) => <li {...props}>{option.label}</li>}
  freeSolo
  renderInput={(params) => <TextField {...params} label="Free solo with text demo" />}
/>`;

  return (
    <MainCard title="Creatable" codeString={createAutocompleteCodeString}>
      <Autocomplete
        fullWidth
        value={value}
        onChange={(event, newValue) => {
          if (typeof newValue === 'string') {
            setValue({
              label: newValue
            });
          } else if (newValue && newValue.inputValue) {
            setValue({
              label: newValue.inputValue
            });
          } else {
            setValue(newValue);
          }
        }}
        filterOptions={(options, params) => {
          const filtered = filter(options, params);

          const { inputValue } = params;

          const isExisting = options.some((option) => inputValue === option.label);
          if (inputValue !== '' && !isExisting) {
            filtered.push({
              inputValue,
              label: `Add "${inputValue}"`
            });
          }

          return filtered;
        }}
        selectOnFocus
        clearOnBlur
        handleHomeEndKeys
        id="free-solo-with-text-demo"
        options={data}
        getOptionLabel={(option) => {
          // Value selected with enter, right from the input
          if (typeof option === 'string') {
            return option;
          }
          // Add "xxx" option created dynamically
          if (option.inputValue) {
            return option.inputValue;
          }
          // Regular option
          return option.label;
        }}
        renderOption={(props, option) => <li {...props}>{option.label}</li>}
        freeSolo
        renderInput={(params) => <TextField {...params} label="Free solo with text demo" />}
      />
    </MainCard>
  );
}
