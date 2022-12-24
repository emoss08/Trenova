import { Box, IconButton } from '@mui/material';
import { ConfigContext } from 'contexts/ConfigContext';
import { useContext } from 'react';
import Brightness4OutlinedIcon from '@mui/icons-material/Brightness4Outlined';
// @ts-ignore
import Brightness7OutlinedIcon from '@mui/icons-material/Brightness7Outlined';
import { useTheme } from '@mui/material/styles';

const ThemeSwitcher = () => {
  const config = useContext(ConfigContext);
  const theme = useTheme();

  const iconBackColor = theme.palette.mode === 'dark' ? 'background.default' : 'grey.100';

  const handleClick = () => {
    config.onChangeMode(config.mode === 'dark' ? 'light' : 'dark');
  };

  return (
    <Box sx={{ flexShrink: 0, ml: 0.75 }}>
      <IconButton color="secondary" onClick={handleClick} sx={{ color: 'text.primary', bgcolor: iconBackColor }}>
        {config.mode === 'dark' ? <Brightness4OutlinedIcon /> : <Brightness7OutlinedIcon />}
      </IconButton>
    </Box>
  );
};

export default ThemeSwitcher;
