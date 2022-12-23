import * as React from 'react';

// material-ui
import { useTheme, styled } from '@mui/material/styles';
import {
  autocompleteClasses,
  Autocomplete,
  AutocompleteCloseReason,
  Box,
  ButtonBase,
  ClickAwayListener,
  InputBase,
  Popper
} from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { CloseOutlined, CheckOutlined, SettingFilled } from '@ant-design/icons';

interface PopperComponentProps {
  anchorEl?: any;
  disablePortal?: boolean;
  open: boolean;
}

const StyledAutocompletePopper = styled('div')(({ theme }) => ({
  [`& .${autocompleteClasses.paper}`]: {
    boxShadow: 'none',
    margin: 0,
    color: 'inherit',
    fontSize: 13
  },
  [`& .${autocompleteClasses.listbox}`]: {
    backgroundColor: theme.palette.mode === 'light' ? '#fff' : '#1c2128',
    padding: 0,
    [`& .${autocompleteClasses.option}`]: {
      minHeight: 'auto',
      alignItems: 'flex-start',
      padding: 8,
      borderBottom: `1px solid  ${theme.palette.mode === 'light' ? ' #eaecef' : '#30363d'}`,
      '&[aria-selected="true"]': {
        backgroundColor: 'transparent'
      },
      '&[data-focus="true"], &[data-focus="true"][aria-selected="true"]': {
        backgroundColor: theme.palette.action.hover
      }
    }
  },
  [`&.${autocompleteClasses.popperDisablePortal}`]: {
    position: 'relative'
  }
}));

function PopperComponent({ disablePortal, anchorEl, open, ...other }: PopperComponentProps) {
  return <StyledAutocompletePopper {...other} />;
}

const StyledPopper = styled(Popper)(({ theme }) => ({
  border: `1px solid ${theme.palette.mode === 'light' ? '#e1e4e8' : '#30363d'}`,
  boxShadow: `0 8px 24px ${theme.palette.mode === 'light' ? 'rgba(149, 157, 165, 0.2)' : 'rgb(1, 4, 9)'}`,
  borderRadius: 6,
  width: 300,
  zIndex: theme.zIndex.modal,
  fontSize: 13,
  color: theme.palette.mode === 'light' ? '#24292e' : '#c9d1d9',
  backgroundColor: theme.palette.mode === 'light' ? '#fff' : '#1c2128'
}));

const StyledInput = styled(InputBase)(({ theme }) => ({
  padding: 10,
  width: '100%',
  borderBottom: `1px solid ${theme.palette.divider}`,
  '& input': {
    borderRadius: 4,
    backgroundColor: theme.palette.background.paper,
    padding: 8,
    transition: theme.transitions.create(['border-color', 'box-shadow']),
    border: `1px solid ${theme.palette.primary.main}`,
    fontSize: 14,
    '&:focus-visible': {
      boxShadow: theme.customShadows.primary,
      borderColor: theme.palette.primary.main
    }
  }
}));

const Button = styled(ButtonBase)(({ theme }) => ({
  fontSize: 13,
  width: '100%',
  textAlign: 'left',
  marginBottom: 8,
  color: theme.palette.text.primary,
  fontWeight: 600,
  '&:hover': {
    color: theme.palette.primary.main
  },
  '&:focus-visible': {
    borderRadius: 2,
    outline: `2px solid ${theme.palette.secondary.dark}`,
    outlineOffset: 2
  },
  '& span': {
    width: '100%'
  },
  '& svg': {
    width: 16,
    height: 16
  }
}));

export default function GitHubLabel() {
  const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
  const [value, setValue] = React.useState<LabelType[]>([labels[1], labels[11]]);
  const [pendingValue, setPendingValue] = React.useState<LabelType[]>([]);
  const theme = useTheme();

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setPendingValue(value);
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setValue(pendingValue);
    if (anchorEl) {
      anchorEl.focus();
    }
    setAnchorEl(null);
  };

  const open = Boolean(anchorEl);
  const id = open ? 'github-label' : undefined;

  const gitAutocompleteCodeString = `// GitHubAutocomplete.tsx
<Box sx={{ width: 221, fontSize: 13 }}>
  <Button
    disableRipple
    aria-describedby={id}
    onClick={handleClick}
    sx={{ justifyContent: 'space-between', '& span': { width: 'auto' } }}
  >
    <span>Labels</span>
    <SettingFilled />
  </Button>
  {value.map((label, index) => (
    <Box
      key={index}
      sx={{
        mt: '3px',
        height: 20,
        padding: '.15em 4px',
        fontWeight: 600,
        lineHeight: '15px',
        borderRadius: '2px'
      }}
      style={{
        backgroundColor: label.color,
        color: theme.palette.getContrastText(label.color)
      }}
    >
      {label.name}
    </Box>
  ))}
</Box>
<StyledPopper id={id} open={open} anchorEl={anchorEl} placement="bottom-start">
  <ClickAwayListener onClickAway={handleClose}>
    <div>
      <Box
        sx={{
          borderBottom: '1px solid {theme.palette.mode === 'light' ? '#eaecef' : '#30363d'}',
          padding: '8px 10px',
          fontWeight: 600
        }}
      >
        Apply labels to this pull request
      </Box>
      <Autocomplete
        open
        multiple
        onClose={(event: React.ChangeEvent<{}>, reason: AutocompleteCloseReason) => {
          if (reason === 'escape') {
            handleClose();
          }
        }}
        value={pendingValue}
        onChange={(event, newValue, reason) => {
          if (event.type === 'keydown' && (event as React.KeyboardEvent).key === 'Backspace' && reason === 'removeOption') {
            return;
          }
          setPendingValue(newValue);
        }}
        disableCloseOnSelect
        PopperComponent={PopperComponent}
        renderTags={() => null}
        noOptionsText="No labels"
        renderOption={(props, option, { selected }) => (
          <li {...props}>
            <Box
              component={CheckOutlined}
              sx={{ width: 17, height: 17, mr: '5px', ml: '-2px', mt: 0.25 }}
              style={{
                visibility: selected ? 'visible' : 'hidden'
              }}
            />
            <Box
              component="span"
              sx={{
                width: 14,
                height: 14,
                flexShrink: 0,
                borderRadius: '3px',
                mr: 1,
                mt: '2px'
              }}
              style={{ backgroundColor: option.color }}
            />
            <Box
              sx={{
                flexGrow: 1,
                '& span': {
                  color: theme.palette.mode === 'light' ? '#586069' : '#8b949e'
                }
              }}
            >
              {option.name}
              <br />
              <span>{option.description}</span>
            </Box>
            <Box
              component={CloseOutlined}
              sx={{ opacity: 0.6, width: 18, height: 18, mt: 0.25 }}
              style={{
                visibility: selected ? 'visible' : 'hidden'
              }}
            />
          </li>
        )}
        options={[...labels].sort((a, b) => {
          // Display the selected labels first.
          let ai = value.indexOf(a);
          ai = ai === -1 ? value.length + labels.indexOf(a) : ai;
          let bi = value.indexOf(b);
          bi = bi === -1 ? value.length + labels.indexOf(b) : bi;
          return ai - bi;
        })}
        getOptionLabel={(option) => option.name}
        renderInput={(params) => (
          <StyledInput ref={params.InputProps.ref} inputProps={params.inputProps} autoFocus placeholder="Filter labels" />
        )}
      />
    </div>
  </ClickAwayListener>
</StyledPopper>`;

  return (
    <MainCard title="GitHub's Picker" codeString={gitAutocompleteCodeString}>
      <Box sx={{ width: 221, fontSize: 13 }}>
        <Button
          disableRipple
          aria-describedby={id}
          onClick={handleClick}
          sx={{ justifyContent: 'space-between', '& span': { width: 'auto' } }}
        >
          <span>Labels</span>
          <SettingFilled />
        </Button>
        {value.map((label, index) => (
          <Box
            key={index}
            sx={{
              mt: '3px',
              height: 20,
              padding: '.15em 4px',
              fontWeight: 600,
              lineHeight: '15px',
              borderRadius: '2px'
            }}
            style={{
              backgroundColor: label.color,
              color: theme.palette.getContrastText(label.color)
            }}
          >
            {label.name}
          </Box>
        ))}
      </Box>
      <StyledPopper id={id} open={open} anchorEl={anchorEl} placement="bottom-start">
        <ClickAwayListener onClickAway={handleClose}>
          <div>
            <Box
              sx={{
                borderBottom: `1px solid ${theme.palette.mode === 'light' ? '#eaecef' : '#30363d'}`,
                padding: '8px 10px',
                fontWeight: 600
              }}
            >
              Apply labels to this pull request
            </Box>
            <Autocomplete
              open
              multiple
              onClose={(event: React.ChangeEvent<{}>, reason: AutocompleteCloseReason) => {
                if (reason === 'escape') {
                  handleClose();
                }
              }}
              value={pendingValue}
              onChange={(event, newValue, reason) => {
                if (event.type === 'keydown' && (event as React.KeyboardEvent).key === 'Backspace' && reason === 'removeOption') {
                  return;
                }
                setPendingValue(newValue);
              }}
              disableCloseOnSelect
              PopperComponent={PopperComponent}
              renderTags={() => null}
              noOptionsText="No labels"
              renderOption={(props, option, { selected }) => (
                <li {...props}>
                  <Box
                    component={CheckOutlined}
                    sx={{ width: 17, height: 17, mr: '5px', ml: '-2px', mt: 0.25 }}
                    style={{
                      visibility: selected ? 'visible' : 'hidden'
                    }}
                  />
                  <Box
                    component="span"
                    sx={{
                      width: 14,
                      height: 14,
                      flexShrink: 0,
                      borderRadius: '3px',
                      mr: 1,
                      mt: '2px'
                    }}
                    style={{ backgroundColor: option.color }}
                  />
                  <Box
                    sx={{
                      flexGrow: 1,
                      '& span': {
                        color: theme.palette.mode === 'light' ? '#586069' : '#8b949e'
                      }
                    }}
                  >
                    {option.name}
                    <br />
                    <span>{option.description}</span>
                  </Box>
                  <Box
                    component={CloseOutlined}
                    sx={{ opacity: 0.6, width: 18, height: 18, mt: 0.25 }}
                    style={{
                      visibility: selected ? 'visible' : 'hidden'
                    }}
                  />
                </li>
              )}
              options={[...labels].sort((a, b) => {
                // Display the selected labels first.
                let ai = value.indexOf(a);
                ai = ai === -1 ? value.length + labels.indexOf(a) : ai;
                let bi = value.indexOf(b);
                bi = bi === -1 ? value.length + labels.indexOf(b) : bi;
                return ai - bi;
              })}
              getOptionLabel={(option) => option.name}
              renderInput={(params) => (
                <StyledInput ref={params.InputProps.ref} inputProps={params.inputProps} autoFocus placeholder="Filter labels" />
              )}
            />
          </div>
        </ClickAwayListener>
      </StyledPopper>
    </MainCard>
  );
}

interface LabelType {
  name: string;
  color: string;
  description?: string;
}

// From https://github.com/abdonrd/github-labels
const labels = [
  {
    name: 'good first issue',
    color: '#7057ff',
    description: 'Good for newcomers'
  },
  {
    name: 'help wanted',
    color: '#008672',
    description: 'Extra attention is needed'
  },
  {
    name: 'priority: critical',
    color: '#b60205',
    description: ''
  },
  {
    name: 'priority: high',
    color: '#d93f0b',
    description: ''
  },
  {
    name: 'priority: low',
    color: '#0e8a16',
    description: ''
  },
  {
    name: 'priority: medium',
    color: '#fbca04',
    description: ''
  },
  {
    name: "status: can't reproduce",
    color: '#fec1c1',
    description: ''
  },
  {
    name: 'status: confirmed',
    color: '#215cea',
    description: ''
  },
  {
    name: 'status: duplicate',
    color: '#cfd3d7',
    description: 'This issue or pull request already exists'
  },
  {
    name: 'status: needs information',
    color: '#fef2c0',
    description: ''
  },
  {
    name: 'status: wont do/fix',
    color: '#eeeeee',
    description: 'This will not be worked on'
  },
  {
    name: 'type: bug',
    color: '#d73a4a',
    description: "Something isn't working"
  },
  {
    name: 'type: discussion',
    color: '#d4c5f9',
    description: ''
  },
  {
    name: 'type: documentation',
    color: '#006b75',
    description: ''
  },
  {
    name: 'type: enhancement',
    color: '#84b6eb',
    description: ''
  },
  {
    name: 'type: epic',
    color: '#3e4b9e',
    description: 'A theme of work that contain sub-tasks'
  },
  {
    name: 'type: feature request',
    color: '#fbca04',
    description: 'New feature or request'
  },
  {
    name: 'type: question',
    color: '#d876e3',
    description: 'Further information is requested'
  }
];
