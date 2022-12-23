import { useEffect, useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { ButtonBase, Grid, Skeleton, Stack, Tooltip, Typography } from '@mui/material';

// types
import { ColorsOptionsProps } from 'types/e-commerce';

// project imports
import ColorOptions from './ColorOptions';
import Avatar from 'components/@extended/Avatar';

// ==============================|| PRODUCT - COLOR OPTIONS ||============================== //

interface ColorProps {
  bg: string;
  id: string;
  colors: string[];
  label: string;
  handelFilter: (type: string, params: string) => void;
}

const Color = ({ bg, id, colors, label, handelFilter }: ColorProps) => {
  const theme = useTheme();

  return (
    <Grid item>
      <Tooltip title={label}>
        <ButtonBase
          sx={{
            borderRadius: '50%',
            '&:focus-visible': {
              outline: `2px solid ${theme.palette.secondary.dark}`,
              outlineOffset: 2
            }
          }}
          onClick={() => handelFilter('colors', id)}
        >
          <Avatar
            color="inherit"
            size="sm"
            sx={{
              bgcolor: bg,
              color: theme.palette.mode === 'light' ? theme.palette.grey[50] : theme.palette.grey[800],
              border: '3px solid',
              borderColor: colors.some((item: string) => item === id) ? theme.palette.secondary.light : theme.palette.background.paper
            }}
          >
            {' '}
          </Avatar>
        </ButtonBase>
      </Tooltip>
    </Grid>
  );
};

// ==============================|| PRODUCT - COLOR ||============================== //

const Colors = ({ colors, handelFilter }: { colors: string[]; handelFilter: (type: string, params: string) => void }) => {
  const [isColorsLoading, setColorLoading] = useState(true);
  useEffect(() => {
    setColorLoading(false);
  }, []);

  return (
    <>
      {isColorsLoading ? (
        <Grid item xs={12}>
          <Skeleton variant="rectangular" width="100%" height={158} />
        </Grid>
      ) : (
        <Stack>
          <Typography variant="h5" sx={{ mb: 1 }}>
            Color
          </Typography>
          <Grid container spacing={1.5} alignItems="center">
            {ColorOptions.map((color: ColorsOptionsProps, index) => (
              <Color key={index} id={color.value} bg={color.bg} label={color.label} colors={colors} handelFilter={handelFilter} />
            ))}
          </Grid>
        </Stack>
      )}
    </>
  );
};

export default Colors;
