import { useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, FormControlLabel, FormLabel, Radio, RadioGroup, SpeedDial, SpeedDialAction, SpeedDialIcon, Switch } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { SaveTwoTone, PrinterTwoTone, HeartTwoTone, ShareAltOutlined } from '@ant-design/icons';

// =============================|| SPEEDDIAL - SIMPLE ||============================= //

export type direction = 'up' | 'left' | 'right' | 'down' | undefined;

export default function SimpleSpeedDials() {
  const theme = useTheme();
  const [open, setOpen] = useState(false);

  // fab action options
  const actions = [
    { icon: <SaveTwoTone twoToneColor={theme.palette.grey[600]} style={{ fontSize: '1.15rem' }} />, name: 'Save' },
    { icon: <PrinterTwoTone twoToneColor={theme.palette.grey[600]} style={{ fontSize: '1.15rem' }} />, name: 'Print' },
    { icon: <ShareAltOutlined style={{ color: theme.palette.grey[600], fontSize: '1.15rem' }} />, name: 'Share' },
    { icon: <HeartTwoTone twoToneColor={theme.palette.grey[600]} style={{ fontSize: '1.15rem' }} />, name: 'Like' }
  ];

  const handleClose = () => {
    setOpen(false);
  };

  const handleOpen = () => {
    setOpen(true);
  };

  const [direction, setDirection] = useState<direction>('up');
  const handleDirectionChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setDirection(event.target.value as direction);
  };

  const [hidden, setHidden] = useState(false);
  const handleHiddenChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setHidden(event.target.checked);
  };

  const basicSpeeddialCodeString = `<FormControlLabel control={<Switch checked={hidden} onChange={handleHiddenChange} color="primary" />} label="Hidden" />
<FormLabel component="legend">Direction</FormLabel>
<RadioGroup sx={{ mt: 1 }} aria-label="direction" name="direction" value={direction} onChange={handleDirectionChange} row>
  <FormControlLabel value="up" control={<Radio />} label="Up" />
  <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
    <FormControlLabel value="right" control={<Radio />} label="Right" />
  </Box>
  <FormControlLabel value="down" control={<Radio />} label="Down" />
  <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
    <FormControlLabel value="left" control={<Radio />} label="Left" />
  </Box>
</RadioGroup>
<Box sx={{ position: 'relative', marginTop: theme.spacing(3), height: 300 }}>
  <SpeedDial
    sx={{
      position: 'absolute',
      '&.MuiSpeedDial-directionUp, &.MuiSpeedDial-directionLeft': {
        bottom: theme.spacing(2),
        right: theme.spacing(2)
      },
      '&.MuiSpeedDial-directionDown, &.MuiSpeedDial-directionRight': {
        top: theme.spacing(2),
        left: theme.spacing(2)
      }
    }}
    ariaLabel="SpeedDial example"
    hidden={hidden}
    icon={<SpeedDialIcon />}
    onClose={handleClose}
    onOpen={handleOpen}
    open={open}
    direction={direction}
  >
    {actions.map((action) => (
      <SpeedDialAction key={action.name} icon={action.icon} tooltipTitle={action.name} onClick={handleClose} />
    ))}
  </SpeedDial>
</Box>`;

  return (
    <MainCard title="Basic" codeString={basicSpeeddialCodeString}>
      <>
        <FormControlLabel control={<Switch checked={hidden} onChange={handleHiddenChange} color="primary" />} label="Hidden" />
        <FormLabel component="legend">Direction</FormLabel>
        <RadioGroup sx={{ mt: 1 }} aria-label="direction" name="direction" value={direction} onChange={handleDirectionChange} row>
          <FormControlLabel value="up" control={<Radio />} label="Up" />
          <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
            <FormControlLabel value="right" control={<Radio />} label="Right" />
          </Box>
          <FormControlLabel value="down" control={<Radio />} label="Down" />
          <Box sx={{ display: { xs: 'none', sm: 'block' } }}>
            <FormControlLabel value="left" control={<Radio />} label="Left" />
          </Box>
        </RadioGroup>
        <Box sx={{ position: 'relative', marginTop: theme.spacing(3), height: 300 }}>
          <SpeedDial
            sx={{
              position: 'absolute',
              '&.MuiSpeedDial-directionUp, &.MuiSpeedDial-directionLeft': {
                bottom: theme.spacing(2),
                right: theme.spacing(2)
              },
              '&.MuiSpeedDial-directionDown, &.MuiSpeedDial-directionRight': {
                top: theme.spacing(2),
                left: theme.spacing(2)
              }
            }}
            ariaLabel="SpeedDial example"
            hidden={hidden}
            icon={<SpeedDialIcon />}
            onClose={handleClose}
            onOpen={handleOpen}
            open={open}
            direction={direction}
          >
            {actions.map((action) => (
              <SpeedDialAction key={action.name} icon={action.icon} tooltipTitle={action.name} onClick={handleClose} />
            ))}
          </SpeedDial>
        </Box>
      </>
    </MainCard>
  );
}
