// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, CardMedia, Grid, Stack, Typography, useMediaQuery } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// assets
import { RightOutlined } from '@ant-design/icons';
import imageEmpty from 'assets/images/e-commerce/empty.svg';
import imageDarkEmpty from 'assets/images/e-commerce/empty-dark.svg';

// ==============================|| PRODUCT - NO/EMPTY FILTER ITEMS ||============================== //

interface ProductEmptyProps {
  handelFilter: () => void;
}

const ProductEmpty = ({ handelFilter }: ProductEmptyProps) => {
  const theme = useTheme();
  const matchDownMD = useMediaQuery(theme.breakpoints.down('lg'));

  return (
    <MainCard content={false}>
      <Grid
        container
        alignItems="center"
        justifyContent="center"
        spacing={3}
        sx={{ my: 3, height: { xs: 'auto', md: 'calc(100vh - 240px)' }, p: { xs: 2.5, md: 'auto' } }}
      >
        <Grid item>
          <CardMedia
            component="img"
            image={theme.palette.mode === 'dark' ? imageDarkEmpty : imageEmpty}
            title="Cart Empty"
            sx={{ width: { xs: 240, md: 320, lg: 440 } }}
          />
        </Grid>
        <Grid item>
          <Stack spacing={0.5}>
            <Typography variant={matchDownMD ? 'h3' : 'h1'} color="inherit">
              There is no Product
            </Typography>
            <Typography variant="h5" color="textSecondary">
              Try checking your spelling or use more general terms
            </Typography>
            <Box sx={{ pt: 3 }}>
              <Button variant="contained" size="large" color="error" endIcon={<RightOutlined />} onClick={() => handelFilter()}>
                Reset Filter
              </Button>
            </Box>
          </Stack>
        </Grid>
      </Grid>
    </MainCard>
  );
};

export default ProductEmpty;
