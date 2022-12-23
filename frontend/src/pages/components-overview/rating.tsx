import { useState } from 'react';

// material-ui
import { Box, Grid, Rating, Stack, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';
import ComponentHeader from 'components/cards/ComponentHeader';
import ComponentWrapper from 'sections/components-overview/ComponentWrapper';
import ComponentSkeleton from 'sections/components-overview/ComponentSkeleton';

// assets
import { HeartFilled, HeartOutlined, SmileOutlined, StarOutlined } from '@ant-design/icons';

const labels: { [index: string]: string } = {
  0.5: 'Useless',
  1: 'Useless+',
  1.5: 'Poor',
  2: 'Poor+',
  2.5: 'Ok',
  3: 'Ok+',
  3.5: 'Good',
  4: 'Good+',
  4.5: 'Excellent',
  5: 'Excellent+'
};

// ==============================|| COMPONENTS - RATING ||============================== //

const ComponentRating = () => {
  const [value, setValue] = useState<number | null>(2);
  const [hover, setHover] = useState(-1);

  const basicRatingCodeString = `<Rating
  name="simple-controlled"
  value={value}
  onChange={(event, newValue) => {
    setValue(newValue);
  }}
/>
<Rating name="read-only" value={3} readOnly />
<Rating name="disabled" value={4} disabled />
<Rating name="no-value" value={null} />`;

  const precisionRatingCodeString = `<Rating name="half-rating" defaultValue={3.6} precision={0.1} />
<Rating name="half-rating-read" defaultValue={2.5} precision={0.5} readOnly />`;

  const hoverRatingCodeString = `<Box
  sx={{
    width: 200,
    display: 'flex',
    alignItems: 'center'
  }}
>
  <Rating
    name="hover-feedback"
    value={value}
    precision={0.5}
    onChange={(event, newValue) => {
      setValue(newValue);
    }}
    onChangeActive={(event, newHover) => {
      setHover(newHover);
    }}
  />
  {value !== null && <Box sx={{ ml: 2 }}>{labels[hover !== -1 ? hover : value]}</Box>}
</Box>`;

  const sizeRatingCodeString = `<Rating name="size-small" defaultValue={2} size="small" />
  <Rating name="size-medium" defaultValue={2} />
  <Rating name="size-large" defaultValue={2} size="large" />`;

  const customizationRatingCodeString = `<Rating name="customized-10" defaultValue={2} max={10} />
<Rating
  name="customized-color"
  defaultValue={2}
  precision={0.5}
  icon={<HeartFilled style={{ fontSize: '1.3rem', margin: 2 }} />}
  emptyIcon={<HeartOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
  sx={{ color: 'error.main' }}
/>
<Rating
  name="customized-color"
  defaultValue={3}
  icon={<SmileOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
  emptyIcon={<SmileOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
  sx={{ color: 'warning.main' }}
/>`;

  const halfRatingCodeString = `<Rating name="half-rating-read" defaultValue={2.5} precision={0.5} />
<Rating
  name="customized-color"
  defaultValue={3.5}
  precision={0.5}
  icon={<StarOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
  emptyIcon={<StarOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
  sx={{ color: 'warning.main' }}
/>`;

  return (
    <ComponentSkeleton>
      <ComponentHeader
        title="Rating"
        caption="Ratings provide insight regarding others' opinions and experiences, and can allow the user to submit a rating of their own."
        directory="src/pages/components-overview/rating"
        link="https://mui.com/material-ui/react-rating/"
      />
      <ComponentWrapper>
        <Grid container spacing={3}>
          <Grid item xs={12} lg={6}>
            <Stack spacing={3}>
              <MainCard title="Basic" codeHighlight codeString={basicRatingCodeString}>
                <Grid container spacing={1}>
                  <Grid item xs={5} sm={3}>
                    <Typography variant="h6">Controlled</Typography>
                  </Grid>
                  <Grid item xs={7} sm={9}>
                    <Rating
                      name="simple-controlled"
                      value={value}
                      onChange={(event, newValue) => {
                        setValue(newValue);
                      }}
                    />
                  </Grid>
                  <Grid item xs={5} sm={3}>
                    <Typography variant="h6">Read Only</Typography>
                  </Grid>
                  <Grid item xs={7} sm={9}>
                    <Rating name="read-only" value={3} readOnly />
                  </Grid>
                  <Grid item xs={5} sm={3}>
                    <Typography variant="h6">Disabled</Typography>
                  </Grid>
                  <Grid item xs={7} sm={9}>
                    <Rating name="disabled" value={4} disabled />
                  </Grid>
                  <Grid item xs={5} sm={3}>
                    <Typography variant="h6">No rating</Typography>
                  </Grid>
                  <Grid item xs={7} sm={9}>
                    <Rating name="no-value" value={null} />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Precision" codeString={precisionRatingCodeString}>
                <Grid container spacing={1}>
                  <Grid item xs={12}>
                    <Rating name="half-rating" defaultValue={3.6} precision={0.1} />
                  </Grid>
                  <Grid item xs={12}>
                    <Rating name="half-rating-read" defaultValue={2.5} precision={0.5} readOnly />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Hover Feedback" codeString={hoverRatingCodeString}>
                <Box
                  sx={{
                    width: 200,
                    display: 'flex',
                    alignItems: 'center'
                  }}
                >
                  <Rating
                    name="hover-feedback"
                    value={value}
                    precision={0.5}
                    onChange={(event, newValue) => {
                      setValue(newValue);
                    }}
                    onChangeActive={(event, newHover) => {
                      setHover(newHover);
                    }}
                  />
                  {value !== null && <Box sx={{ ml: 2 }}>{labels[hover !== -1 ? hover : value]}</Box>}
                </Box>
              </MainCard>
            </Stack>
          </Grid>
          <Grid item xs={12} lg={6}>
            <Stack spacing={3}>
              <MainCard title="Sizes" codeString={sizeRatingCodeString}>
                <Grid container spacing={1}>
                  <Grid item xs={12}>
                    <Rating name="size-small" defaultValue={2} size="small" />
                  </Grid>
                  <Grid item xs={12}>
                    <Rating name="size-medium" defaultValue={2} />
                  </Grid>
                  <Grid item xs={12}>
                    <Rating name="size-large" defaultValue={2} size="large" />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Customization" codeString={customizationRatingCodeString}>
                <Grid container spacing={1}>
                  <Grid item xs={12}>
                    <Rating name="customized-10" defaultValue={2} max={10} />
                  </Grid>
                  <Grid item xs={12}>
                    <Rating
                      name="customized-color-heart"
                      defaultValue={2}
                      precision={0.5}
                      icon={<HeartFilled style={{ fontSize: '1.3rem', margin: 2 }} />}
                      emptyIcon={<HeartOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
                      sx={{ color: 'error.main' }}
                    />
                  </Grid>
                  <Grid item xs={12}>
                    <Rating
                      name="customized-color-smily"
                      defaultValue={3}
                      icon={<SmileOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
                      emptyIcon={<SmileOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
                      sx={{ color: 'warning.main' }}
                    />
                  </Grid>
                </Grid>
              </MainCard>
              <MainCard title="Half" codeString={halfRatingCodeString}>
                <Grid container spacing={1}>
                  <Grid item xs={12}>
                    <Rating name="half-rating-read-custom" defaultValue={2.5} precision={0.5} />
                  </Grid>
                  <Grid item xs={12}>
                    <Rating
                      name="customized-color-half"
                      defaultValue={3.5}
                      precision={0.5}
                      icon={<StarOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
                      emptyIcon={<StarOutlined style={{ fontSize: '1.3rem', margin: 2 }} />}
                      sx={{ color: 'warning.main' }}
                    />
                  </Grid>
                </Grid>
              </MainCard>
            </Stack>
          </Grid>
        </Grid>
      </ComponentWrapper>
    </ComponentSkeleton>
  );
};

export default ComponentRating;
