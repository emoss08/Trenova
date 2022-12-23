import { useEffect, useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';
import { Button, Grid, LinearProgress, Rating, Stack, Typography, TextField, InputAdornment } from '@mui/material';

// types
import { Products, Reviews } from 'types/e-commerce';

// project imports
import ProductReview from 'components/cards/e-commerce/ProductReview';
import MainCard from 'components/MainCard';
import Avatar from 'components/@extended/Avatar';
import { useDispatch, useSelector } from 'store';
import { getProductReviews } from 'store/reducers/product';

// assets
import { PaperClipOutlined, PictureOutlined, SmileOutlined, StarFilled, StarOutlined } from '@ant-design/icons';
import userItself from 'assets/images/users/avatar-4.png';
import IconButton from 'components/@extended/IconButton';

interface ProgressProps {
  star: number;
  value: number;
  color?: 'inherit' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning' | undefined;
}

// progress
function LinearProgressWithLabel({ star, color, value, ...others }: ProgressProps) {
  return (
    <>
      <Stack direction="row" spacing={1} alignItems="center">
        <LinearProgress
          value={value}
          variant="determinate"
          color={color}
          {...others}
          sx={{ width: '100%', bgcolor: 'secondary.lighter' }}
        />
        <Typography variant="body2" sx={{ minWidth: 50 }} color="textSecondary">{`${Math.round(star)} Star`}</Typography>
      </Stack>
    </>
  );
}

// ==============================|| PRODUCT DETAILS - REVIEWS ||============================== //

const ProductReviews = ({ product }: { product: Products }) => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const [reviews, setReviews] = useState<Reviews[]>([]);
  const productState = useSelector((state) => state.product);

  useEffect(() => {
    setReviews(productState.reviews);
  }, [productState]);

  useEffect(() => {
    dispatch(getProductReviews());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Grid container spacing={3}>
      <Grid item xs={12}>
        <MainCard>
          <Grid container justifyContent="space-between" alignItems="center" spacing={2.5}>
            <Grid item>
              {product && (
                <Stack spacing={1} sx={{ height: '100%' }}>
                  <Stack spacing={1}>
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Typography variant="h2">
                        {Number((product.rating! < 4 ? product.rating! + 1 : product.rating!).toFixed(1))}
                      </Typography>
                      <Typography variant="h4" color="textSecondary">
                        /5
                      </Typography>
                    </Stack>
                    <Typography color="textSecondary">Based on {product.offerPrice?.toFixed(0)} reviews</Typography>
                  </Stack>
                  <Rating
                    name="simple-controlled"
                    value={product.rating! < 4 ? product.rating! + 1 : product.rating}
                    icon={<StarFilled style={{ fontSize: 'inherit' }} />}
                    emptyIcon={<StarOutlined style={{ fontSize: 'inherit' }} />}
                    readOnly
                    precision={0.1}
                  />
                </Stack>
              )}
            </Grid>
            <Grid item>
              <Grid container alignItems="center" justifyContent="space-between" spacing={1}>
                <Grid item xs={12}>
                  <LinearProgressWithLabel color="warning" star={5} value={100} />
                </Grid>
                <Grid item xs={12}>
                  <LinearProgressWithLabel color="warning" star={4} value={80} />
                </Grid>
                <Grid item xs={12}>
                  <LinearProgressWithLabel color="warning" star={3} value={60} />
                </Grid>
                <Grid item xs={12}>
                  <LinearProgressWithLabel color="warning" star={2} value={40} />
                </Grid>
                <Grid item xs={12}>
                  <LinearProgressWithLabel color="warning" star={1} value={20} />
                </Grid>
              </Grid>
            </Grid>
          </Grid>
        </MainCard>
      </Grid>

      {reviews &&
        reviews.map((review, index) => (
          <Grid item xs={12} key={index}>
            <MainCard sx={{ bgcolor: theme.palette.grey.A50 }}>
              <ProductReview
                avatar={review.profile.avatar}
                date={review.date}
                name={review.profile.name}
                rating={review.rating}
                review={review.review}
              />
            </MainCard>
          </Grid>
        ))}
      <Grid item xs={12}>
        <Stack direction="row" justifyContent="center">
          <Button variant="text" sx={{ textTransform: 'none' }}>
            {' '}
            View more comments{' '}
          </Button>
        </Stack>
      </Grid>
      <Grid item xs={12}>
        <Stack direction="row" spacing={0.5} alignItems="center">
          <Avatar alt="user" src={userItself} />
          <TextField
            placeholder="Write a Review"
            fullWidth
            sx={{ bgcolor: theme.palette.grey.A50 }}
            InputProps={{
              endAdornment: (
                <InputAdornment position="end" sx={{ opacity: 0.5 }}>
                  <Stack direction="row" spacing={0}>
                    <IconButton size="small" color="secondary" sx={{ '&:hover': { bgcolor: 'transparent' } }}>
                      <PaperClipOutlined />
                    </IconButton>
                    <IconButton size="small" color="secondary" sx={{ '&:hover': { bgcolor: 'transparent' } }}>
                      <PictureOutlined />
                    </IconButton>
                    <IconButton size="small" color="secondary" sx={{ '&:hover': { bgcolor: 'transparent' } }}>
                      <SmileOutlined />
                    </IconButton>
                  </Stack>
                </InputAdornment>
              )
            }}
          />
        </Stack>
      </Grid>
    </Grid>
  );
};

export default ProductReviews;
