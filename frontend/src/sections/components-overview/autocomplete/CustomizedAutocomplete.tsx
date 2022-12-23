// material-ui
import { useAutocomplete, AutocompleteGetTagProps } from '@mui/base/AutocompleteUnstyled';
import { styled } from '@mui/material/styles';

// project import
import MainCard from 'components/MainCard';
import data from 'data/movies';

// assets
import { CloseOutlined, CheckOutlined } from '@ant-design/icons';

const Root = styled('div')(
  ({ theme }) => `
  color: ${theme.palette.mode === 'dark' ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,.85)'};
  font-size: 14px;
`
);
Root.displayName = 'Root';

const InputWrapper = styled('div')(
  ({ theme }) => `
  width: 100%;
  border: 1px solid ${theme.palette.mode === 'dark' ? '#434343' : '#d9d9d9'};
  background-color: ${theme.palette.mode === 'dark' ? '#141414' : '#fff'};
  border-radius: 4px;
  padding: 8px;
  display: flex;
  flex-wrap: wrap;

  &:hover {
    border-color: ${theme.palette.primary.main};
  }

  &.focused {
    border-color: ${theme.palette.primary.main};
    box-shadow: ${theme.customShadows.primary};
  }

  & input {
    background-color: transparent;
    height: 30px;
    box-sizing: border-box;
    padding: 4px 6px;
    width: 0;
    min-width: 30px;
    flex-grow: 1;
    border: 0;
    margin: 0;
    outline: 0;
  }
`
);

interface TagProps extends ReturnType<AutocompleteGetTagProps> {
  label: string;
}

function Tag(props: TagProps) {
  const { label, onDelete, ...other } = props;
  return (
    <div {...other}>
      <span>{label}</span>
      <CloseOutlined onClick={onDelete} />
    </div>
  );
}

const StyledTag = styled(Tag)<TagProps>(
  ({ theme }) => `
  display: flex;
  align-items: center;
  height: 24px;
  margin: 4px;
  line-height: 22px;
  background-color: ${theme.palette.secondary.lighter};
  border: 1px solid ${theme.palette.secondary.light};
  border-radius: 2px;
  box-sizing: content-box;
  padding: 0 4px 0 10px;
  outline: 0;
  overflow: hidden;

  & span {
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  & svg {
    font-size: 0.625rem;
    cursor: pointer;
    padding: 4px;
  }
`
);

const Listbox = styled('ul')(
  ({ theme }) => `
  width: 300px;
  margin: 2px 0 0;
  padding: 0;
  position: absolute;
  list-style: none;
  background-color: ${theme.palette.background.paper};
  overflow: auto;
  max-height: 250px;
  border-radius: 4px;
  box-shadow: ${theme.customShadows.z1};
  z-index: 1;

  & li {
    padding: 5px 12px;
    display: flex;

    & span {
      flex-grow: 1;
    }

    & svg {
      color: transparent;
    }
  }

  & li[aria-selected='true'] {
    background-color: ${theme.palette.primary.lighter};
    font-weight: 600;

    & svg {
      color: ${theme.palette.primary.main};
    }
  }

  & li[data-focus='true'] {
    background-color: ${theme.palette.primary.lighter};
    cursor: pointer;

    & svg {
      color: currentColor;
    }
  }
`
);

// ==============================|| AUTOCOMPLETE - CUSTOMIZED ||============================== //

export default function CustomizedAutocomplete() {
  const { getRootProps, getInputProps, getTagProps, getListboxProps, getOptionProps, groupedOptions, value, focused, setAnchorEl } =
    useAutocomplete({
      id: 'customized-hook-demo',
      defaultValue: [data[1], data[3], data[8]],
      multiple: true,
      options: data,
      getOptionLabel: (option) => option.label
    });

  const customAutocompleteCodeString = `// CustomizedAutocomplete.tsx
<Root>
  <div {...getRootProps()}>
    <InputWrapper ref={setAnchorEl} className={focused ? 'focused' : ''}>
      {value.map((option: FilmOptionType, index: number) => (
        <StyledTag label={option.label} {...getTagProps({ index })} />
      ))}
      <input {...getInputProps()} />
    </InputWrapper>
  </div>
  {groupedOptions.length > 0 ? (
    <Listbox {...getListboxProps()}>
      {(groupedOptions as typeof data).map((option, index) => (
        <li {...getOptionProps({ option, index })}>
          <span>{option.label}</span>
          <CheckOutlined style={{ marginTop: 2 }} />
        </li>
      ))}
    </Listbox>
  ) : null}
</Root>`;

  return (
    <MainCard title="Customized" sx={{ overflow: 'visible' }} codeString={customAutocompleteCodeString}>
      <Root>
        <div {...getRootProps()}>
          <InputWrapper ref={setAnchorEl} className={focused ? 'focused' : ''}>
            {value.map((option: FilmOptionType, index: number) => (
              <StyledTag label={option.label} {...getTagProps({ index })} />
            ))}
            <input {...getInputProps()} />
          </InputWrapper>
        </div>
        {groupedOptions.length > 0 ? (
          <Listbox {...getListboxProps()}>
            {(groupedOptions as typeof data).map((option, index) => (
              <li {...getOptionProps({ option, index })}>
                <span>{option.label}</span>
                <CheckOutlined style={{ marginTop: 2 }} />
              </li>
            ))}
          </Listbox>
        ) : null}
      </Root>
    </MainCard>
  );
}

interface FilmOptionType {
  label: string;
  year: number;
}
