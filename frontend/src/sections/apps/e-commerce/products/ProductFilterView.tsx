// material-ui
import { useTheme } from '@mui/material/styles';
import { Button, Chip, Divider, Grid, Stack, Typography, useMediaQuery } from '@mui/material';

// types
import { ProductsFilter } from 'types/e-commerce';

// project imports
import ColorOptions from './ColorOptions';
import IconButton from 'components/@extended/IconButton';

// assets
import { CloseOutlined } from '@ant-design/icons';

function getColor(color: string) {
  return ColorOptions.filter((item: any) => item.value === color);
}

// ==============================|| PRODUCT GRID - FILTER VIEW ||============================== //

interface ProductFilterViewProps {
  filter: ProductsFilter;
  initialState: ProductsFilter;
  filterIsEqual: (initialState: ProductsFilter, filter: ProductsFilter) => boolean;
  handelFilter: (type: string, params: string, rating?: number) => void;
}

const ProductFilterView = ({ filter, filterIsEqual, handelFilter, initialState }: ProductFilterViewProps) => {
  const theme = useTheme();
  const matchDownMD = useMediaQuery(theme.breakpoints.down('lg'));

  return (
    <>
      {!filterIsEqual(initialState, filter) && (
        <Stack spacing={2}>
          <Typography variant="h5">Active Filters</Typography>
          {!(initialState.search === filter.search) && (
            <Grid item>
              <Stack direction="row" alignItems="center" sx={{ ml: '-10px' }}>
                <Chip
                  size={matchDownMD ? 'small' : undefined}
                  label={filter.search}
                  sx={{
                    borderRadius: '4px',
                    textTransform: 'capitalize',
                    color: `grey.500`,
                    bgcolor: 'inherit',
                    '& .MuiSvgIcon-root': { color: `grey` }
                  }}
                />
                <IconButton
                  color="secondary"
                  size="small"
                  sx={{ '&:hover': { bgcolor: 'transparent' }, ml: -1.5 }}
                  onClick={() => handelFilter('search', '')}
                >
                  <CloseOutlined />
                </IconButton>
              </Stack>
            </Grid>
          )}
          {!(initialState.sort === filter.sort) && (
            <Grid item>
              <Stack>
                <Typography variant="subtitle1">Sort</Typography>
                <Stack direction="row" alignItems="center" sx={{ ml: '-10px' }}>
                  <Chip
                    size={matchDownMD ? 'small' : undefined}
                    label={filter.sort}
                    sx={{
                      borderRadius: '4px',
                      textTransform: 'capitalize',
                      color: `grey.500`,
                      bgcolor: 'inherit',
                      '& .MuiSvgIcon-root': { color: `grey` }
                    }}
                  />
                  <IconButton
                    color="secondary"
                    size="small"
                    sx={{ '&:hover': { bgcolor: 'transparent' }, ml: -1.5 }}
                    onClick={() => handelFilter('sort', initialState.sort)}
                  >
                    <CloseOutlined />
                  </IconButton>
                </Stack>
              </Stack>
            </Grid>
          )}
          {!(JSON.stringify(initialState.gender) === JSON.stringify(filter.gender)) && (
            <Grid item>
              <Stack>
                <Typography variant="subtitle1">Gender</Typography>
                <Grid container item sx={{ ml: '-10px' }}>
                  {filter.gender.map((item: string, index: number) => (
                    <Stack direction="row" alignItems="center" key={index}>
                      <Chip
                        size={matchDownMD ? 'small' : undefined}
                        label={item}
                        sx={{
                          borderRadius: '4px',
                          textTransform: 'capitalize',
                          color: `grey.500`,
                          bgcolor: 'inherit',
                          '& .MuiSvgIcon-root': { color: `grey` }
                        }}
                      />
                      <IconButton
                        color="secondary"
                        size="small"
                        sx={{ '&:hover': { bgcolor: 'transparent' }, ml: -1.5 }}
                        onClick={() => handelFilter('gender', item)}
                      >
                        <CloseOutlined />
                      </IconButton>
                    </Stack>
                  ))}
                </Grid>
              </Stack>
            </Grid>
          )}
          {!(JSON.stringify(initialState.categories) === JSON.stringify(filter.categories)) && filter.categories.length > 0 && (
            <Grid item>
              <Stack>
                <Typography variant="subtitle1">Categories</Typography>
                <Grid container item sx={{ ml: '-10px' }}>
                  {filter.categories.map((item: string, index: number) => (
                    <Stack direction="row" alignItems="center" key={index}>
                      <Chip
                        size={matchDownMD ? 'small' : undefined}
                        label={item}
                        sx={{
                          borderRadius: '4px',
                          textTransform: 'capitalize',
                          color: `grey.500`,
                          bgcolor: 'inherit',
                          '& .MuiSvgIcon-root': { color: `grey` }
                        }}
                      />
                      <IconButton
                        color="secondary"
                        size="small"
                        sx={{ '&:hover': { bgcolor: 'transparent' }, ml: -1.5 }}
                        onClick={() => handelFilter('categories', item)}
                      >
                        <CloseOutlined />
                      </IconButton>
                    </Stack>
                  ))}
                </Grid>
              </Stack>
            </Grid>
          )}
          {!(JSON.stringify(initialState.colors) === JSON.stringify(filter.colors)) && (
            <Grid item>
              <Stack>
                <Typography variant="subtitle1">Colors</Typography>
                <Grid container item sx={{ ml: '-10px' }}>
                  {filter.colors.map((item: string, index: number) => {
                    const colorsData = getColor(item);
                    return (
                      <Stack direction="row" alignItems="center" key={index}>
                        <Chip
                          size={matchDownMD ? 'small' : undefined}
                          label={colorsData[0].label}
                          sx={{
                            borderRadius: '4px',
                            textTransform: 'capitalize',
                            color: `grey.500`,
                            bgcolor: 'inherit',
                            '& .MuiSvgIcon-root': { color: `grey` }
                          }}
                        />
                        <IconButton
                          color="secondary"
                          size="small"
                          sx={{ '&:hover': { bgcolor: 'transparent' }, ml: -1.5 }}
                          onClick={() => handelFilter('colors', item)}
                        >
                          <CloseOutlined />
                        </IconButton>
                      </Stack>
                    );
                  })}
                </Grid>
              </Stack>
            </Grid>
          )}
          {!(initialState.price === filter.price) && (
            <Grid item>
              <Stack>
                <Typography variant="subtitle1">Price</Typography>
                <Grid item sx={{ ml: '-10px' }}>
                  <Chip
                    size={matchDownMD ? 'small' : undefined}
                    label={filter.price}
                    sx={{
                      borderRadius: '4px',
                      textTransform: 'capitalize',
                      color: `grey.500`,
                      bgcolor: 'inherit',
                      '& .MuiSvgIcon-root': { color: `grey` }
                    }}
                  />
                </Grid>
              </Stack>
            </Grid>
          )}
          {!(initialState.rating === filter.rating) && (
            <Grid item>
              <Stack>
                <Typography variant="subtitle1">Rating</Typography>
                <Grid item sx={{ ml: '-10px' }}>
                  <Stack direction="row" alignItems="center">
                    <Chip
                      size={matchDownMD ? 'small' : undefined}
                      label={String(filter.rating)}
                      sx={{
                        borderRadius: '4px',
                        textTransform: 'capitalize',
                        color: `grey.500`,
                        bgcolor: 'inherit',
                        '& .MuiSvgIcon-root': { color: `grey` }
                      }}
                    />
                    <IconButton
                      color="secondary"
                      size="small"
                      sx={{ '&:hover': { bgcolor: 'transparent' }, ml: -1.5 }}
                      onClick={() => handelFilter('rating', '', 0)}
                    >
                      <CloseOutlined />
                    </IconButton>
                  </Stack>
                </Grid>
              </Stack>
            </Grid>
          )}
          <Grid item>
            <Button variant="text" color="primary" sx={{ ml: '-10px' }} onClick={() => handelFilter('reset', '')}>
              Reset all filters
            </Button>
          </Grid>
          <Grid item>
            <Divider sx={{ ml: '-8%', mr: '-8%' }} />
          </Grid>
        </Stack>
      )}
    </>
  );
};

export default ProductFilterView;
