import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, CardContent, CardMedia, Chip, Divider, Grid, Rating, Stack, Typography } from '@mui/material';

// types
import { ProductCardProps } from 'types/cart';

// project import
import MainCard from 'components/MainCard';
import IconButton from 'components/@extended/IconButton';
import SkeletonProductPlaceholder from 'components/cards/skeleton/ProductPlaceholder';
import { useDispatch, useSelector } from 'store';
import { addProduct } from 'store/reducers/cart';
import { openSnackbar } from 'store/reducers/snackbar';

// assets
import { HeartOutlined, HeartFilled } from '@ant-design/icons';

const prodImage = require.context('assets/images/e-commerce', true);

// ==============================|| PRODUCT CARD ||============================== //

const ProductCard = ({
  id,
  color,
  name,
  brand,
  offer,
  isStock,
  image,
  description,
  offerPrice,
  salePrice,
  rating,
  open
}: ProductCardProps) => {
  const theme = useTheme();
  const dispatch = useDispatch();

  const prodProfile = image && prodImage(`./${image}`);

  const [productRating] = useState<number | undefined>(rating);
  const [wishlisted, setWishlisted] = useState<boolean>(false);
  const cart = useSelector((state) => state.cart);

  const addCart = () => {
    dispatch(addProduct({ id, name, image, salePrice, offerPrice, color, size: 8, quantity: 1, description }, cart.checkout.products));
    dispatch(
      openSnackbar({
        open: true,
        message: 'Add To Cart Success',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        close: false
      })
    );
  };

  const addToFavourite = () => {
    setWishlisted(!wishlisted);
    dispatch(
      openSnackbar({
        open: true,
        message: 'Added to favourites',
        variant: 'alert',
        alert: {
          color: 'success'
        },
        close: false
      })
    );
  };

  const [isLoading, setLoading] = useState(true);
  useEffect(() => {
    setLoading(false);
  }, []);

  return (
    <>
      {isLoading ? (
        <SkeletonProductPlaceholder />
      ) : (
        <MainCard
          content={false}
          boxShadow
          sx={{
            '&:hover': {
              transform: 'scale3d(1.02, 1.02, 1)',
              transition: 'all .4s ease-in-out'
            }
          }}
        >
          <Box sx={{ width: 250, m: 'auto' }}>
            <CardMedia
              sx={{ height: 250, textDecoration: 'none', opacity: isStock ? 1 : 0.25 }}
              image={prodProfile}
              component={Link}
              to={`/apps/e-commerce/product-details/${id}`}
            />
          </Box>
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{ width: '100%', position: 'absolute', top: 0, pt: 1.75, pl: 2, pr: 1 }}
          >
            {!isStock && <Chip variant="light" color="error" size="small" label="Sold out" />}
            {offer && <Chip label={offer} variant="combined" color="success" size="small" />}
            <IconButton color="secondary" sx={{ ml: 'auto', '&:hover': { background: 'transparent' } }} onClick={addToFavourite}>
              {wishlisted ? (
                <HeartFilled style={{ fontSize: '1.15rem', color: theme.palette.error.main }} />
              ) : (
                <HeartOutlined style={{ fontSize: '1.15rem' }} />
              )}
            </IconButton>
          </Stack>
          <Divider />
          <CardContent sx={{ p: 2 }}>
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <Stack>
                  <Typography
                    component={Link}
                    to={`/apps/e-commerce/product-details/${id}`}
                    color="textPrimary"
                    variant="h5"
                    sx={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', display: 'block', textDecoration: 'none' }}
                  >
                    {name}
                  </Typography>
                  <Typography variant="h6" color="textSecondary">
                    {brand}
                  </Typography>
                </Stack>
              </Grid>
              <Grid item xs={12}>
                <Stack direction="row" justifyContent="space-between" alignItems="flex-end">
                  <Stack>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Typography variant="h5">${offerPrice}</Typography>
                      {salePrice && (
                        <Typography variant="h6" color="textSecondary" sx={{ textDecoration: 'line-through' }}>
                          ${salePrice}
                        </Typography>
                      )}
                    </Stack>
                    <Stack direction="row" alignItems="flex-start">
                      <Rating precision={0.5} name="size-small" value={productRating} size="small" readOnly />
                      <Typography variant="caption">({productRating?.toFixed(1)})</Typography>
                    </Stack>
                  </Stack>

                  <Button variant="contained" onClick={addCart} disabled={!isStock}>
                    {!isStock ? 'Sold Out' : 'Add to Cart'}
                  </Button>
                </Stack>
              </Grid>
            </Grid>
          </CardContent>
        </MainCard>
      )}
    </>
  );
};

export default ProductCard;
