// material-ui
import { Stack } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ExclusiveToggleButtons from './ExclusiveToggleButtons';
import MultipleToggleButtons from './MultipleToggleButtons';
import ColorToggleButton from './ColorToggleButton';
import TextToggleButtons from './TextToggleButtons';
import VariantToggleButtons from './VariantToggleButtons';
import VerticalToggleButtons from './VerticalToggleButtons';

// ==============================|| TOGGLE BUTTON ||============================== //

const toggleButtonCodeString = `// ExclusiveToggleButtons.tsx
<ToggleButtonGroup value={alignment} exclusive onChange={handleAlignment} aria-label="text alignment">
  <ToggleButton value="left" aria-label="left aligned">
    <AlignLeftOutlined />
  </ToggleButton>
  <ToggleButton value="center" aria-label="centered">
    <AlignCenterOutlined />
  </ToggleButton>
  <ToggleButton value="right" aria-label="right aligned">
    <AlignRightOutlined />
  </ToggleButton>
  <ToggleButton value="list" aria-label="list" disabled sx={{ '&.Mui-disabled': { color: 'text.disabled' } }}>
    <UnorderedListOutlined />
  </ToggleButton>
</ToggleButtonGroup>

// ColorToggleButton.tsx
<ToggleButtonGroup value={formats} onChange={handleFormat} aria-label="text formatting">
  <ToggleButton value="bold" aria-label="bold">
    <BoldOutlined />
  </ToggleButton>
  <ToggleButton value="italic" aria-label="italic">
    <ItalicOutlined />
  </ToggleButton>
  <ToggleButton value="underlined" aria-label="underlined">
    <UnderlineOutlined />
  </ToggleButton>
  <ToggleButton value="color" aria-label="color" disabled>
    <BgColorsOutlined />
    <DownOutlined style={{ fontSize: '0.625rem', marginLeft: 6 }} />
  </ToggleButton>
</ToggleButtonGroup>

// ColorToggleButton.tsx
<ToggleButtonGroup color="primary" value={alignment} exclusive onChange={handleAlignment} aria-label="text alignment">
  <ToggleButton value="left" aria-label="left aligned">
    <AlignLeftOutlined />
  </ToggleButton>
  <ToggleButton value="center" aria-label="centered">
    <AlignCenterOutlined />
  </ToggleButton>
  <ToggleButton value="right" aria-label="right aligned">
    <AlignRightOutlined />
  </ToggleButton>
  <ToggleButton value="list" aria-label="list">
    <UnorderedListOutlined />
  </ToggleButton>
</ToggleButtonGroup>

// TextToggleButtons.tsx
<ToggleButtonGroup value={alignment} exclusive onChange={handleAlignment} aria-label="text alignment">
  <ToggleButton value="one" aria-label="first">
    One
  </ToggleButton>
  <ToggleButton value="two" aria-label="second">
    Two
  </ToggleButton>
  <ToggleButton value="three" aria-label="third">
    Three
  </ToggleButton>
  <ToggleButton value="four" aria-label="fourth">
    Four
  </ToggleButton>
</ToggleButtonGroup>

// VariantToggleButtons.tsx
<ToggleButtonGroup
  value={alignment}
  color="primary"
  exclusive
  onChange={handleAlignment}
  aria-label="text alignment"
  sx={{
    '& .MuiToggleButton-root': {
      '&:not(.Mui-selected)': {
        borderTopColor: 'transparent',
        borderBottomColor: 'transparent'
      },
      '&:first-of-type': {
        borderLeftColor: 'transparent'
      },
      '&:last-of-type': {
        borderRightColor: 'transparent'
      },
      '&.Mui-selected': {
        borderColor: 'inherit',
        borderLeftColor: theme.palette.primary.main !important,
        '&:hover': {
          bgcolor: theme.palette.primary.lighter
        }
      },
      '&:hover': {
        bgcolor: 'transparent',
        borderColor: theme.palette.primary.main,
        borderLeftColor: theme.palette.primary.main !important,
        zIndex: 2
      }
    }
  }}
>
  <ToggleButton value="web" aria-label="web">
    Web
  </ToggleButton>
  <ToggleButton value="android" aria-label="android">
    Android
  </ToggleButton>
  <ToggleButton value="ios" aria-label="ios">
    iOS
  </ToggleButton>
  <ToggleButton value="all" aria-label="all">
    All
  </ToggleButton>
</ToggleButtonGroup>

// VerticalToggleButtons.tsx
<ToggleButtonGroup orientation="vertical" value={view} exclusive onChange={handleChange}>
  <ToggleButton value="tree" aria-label="tree">
    <ApartmentOutlined />
  </ToggleButton>
  <ToggleButton value="grid" aria-label="grid">
    <AppstoreOutlined />
  </ToggleButton>
  <ToggleButton value="table" aria-label="table">
    <TableOutlined />
  </ToggleButton>
</ToggleButtonGroup>`;

const ToggleButtons = () => (
  <MainCard title="Toggle Button" codeString={toggleButtonCodeString}>
    <Stack spacing={2} sx={{ mb: 2 }}>
      <ExclusiveToggleButtons />
      <MultipleToggleButtons />
      <ColorToggleButton />
      <TextToggleButtons />
      <VariantToggleButtons />
    </Stack>
    <VerticalToggleButtons />
  </MainCard>
);

export default ToggleButtons;
