// material-ui
import { Box, Container, Grid, Rating, Typography } from '@mui/material';

// third party
import Slider from 'react-slick';

// project import
import Avatar from 'components/@extended/Avatar';
import MainCard from 'components/MainCard';

// assets
import imgfeature1 from 'assets/images/landing/img-user1.svg';

// ================================|| TESTIMONIAL - ITEMS ||================================ //

interface Props {
  item: { image: string; title: string; review: string; rating: number; client: string };
}

const Item = ({ item }: Props) => (
  <MainCard sx={{ mx: 2 }} contentSX={{ p: 3 }}>
    <Grid container spacing={1}>
      <Grid item>
        <Avatar src={item.image} alt="feature" />
      </Grid>
      <Grid item sm zeroMinWidth>
        <Grid container spacing={1}>
          <Grid item xs={12}>
            <Typography variant="h5" sx={{ fontWeight: 600 }}>
              {item.title}
            </Typography>
            <Rating name="read-only" readOnly value={item.rating} size="small" precision={0.5} />
          </Grid>
          <Grid item xs={12}>
            <Typography variant="body1" color="secondary">
              {item.review}
            </Typography>
          </Grid>
          <Grid item xs={12}>
            <Typography variant="subtitle2">{item.client}</Typography>
          </Grid>
        </Grid>
      </Grid>
    </Grid>
  </MainCard>
);

// ==============================|| LANDING - TESTIMONIAL PAGE ||============================== //

const TestimonialBlock = () => {
  const settings = {
    autoplay: true,
    arrows: false,
    dots: false,
    infinite: true,
    speed: 500,
    slidesToShow: 1,
    slidesToScroll: 1
  };

  const items = [
    {
      image: imgfeature1,
      title: 'Design Quality',
      review:
        'One of the better themes Ive used. Beautiful and clean design. Also included a NextJS project which is pretty rare in what Ive seen on MUI templates. Ultimately it didnt work out for my specific use case, but this is a well organized theme. Definitely keeping it in mind for future projects.',
      rating: 5,
      client: 'William S.'
    },
    {
      image: imgfeature1,
      title: 'Customizability',
      review:
        'Excellent design, you can use in a new project or include in your current project. Multiple components for any use. Good code quality. Great customer service and support.',
      rating: 5,
      client: 'Rodrigo J.'
    },
    {
      image: imgfeature1,
      title: 'Design Quality',
      review: 'there is no mistake, great design and organized code, thank you ...',
      rating: 4,
      client: 'Yang Z.'
    },
    {
      image: imgfeature1,
      title: 'Code Quality',
      review:
        'Fantastic design and good code quality. Its a great starting point for any new project. They provide plenty of premade components, page views, and authentication options. Definitely the best Ive found for Material UI in Typescript',
      rating: 5,
      client: 'Felipe F.'
    },
    {
      image: imgfeature1,
      title: 'Code Quality ',
      review:
        'Great template. Very well written code and good structure. Very customizable and tons of nice components. Good documentation. Team is very responsive too.',
      rating: 5,
      client: 'Besart M.'
    },
    {
      image: imgfeature1,
      title: 'Code Quality',
      review:
        'We are just getting started with this new theme, but we liked it enough that we decided to import our application into this codebase rather than the other way around. Impressive number of custom components and original work VS some other themes that seem to just be repackaged versions of Material UI.',
      rating: 5,
      client: 'Oxbird'
    }
  ];
  return (
    <Box sx={{ overflowX: 'hidden' }}>
      <Container>
        <Grid container alignItems="center" justifyContent="center" spacing={2} sx={{ mt: { md: 15, xs: 2.5 }, mb: { md: 10, xs: 2.5 } }}>
          <Grid item xs={12}>
            <Grid container spacing={1} justifyContent="center" sx={{ mb: 4, textAlign: 'center' }}>
              <Grid item sm={10} md={6}>
                <Grid container spacing={1} justifyContent="center">
                  <Grid item xs={12}>
                    <Typography variant="subtitle1" color="primary">
                      Testament
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="h2">Customers Voice</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body1">
                      We have proven records in Dashboard development with an average 4.9/5 ratings. We are glad to show such a warm reveiws
                      from our loyal customers.
                    </Typography>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item lg={6} md={8} xs={12} sx={{ '& .slick-list': { overflow: 'visible' } }}>
            <Slider {...settings}>
              {items.map((item, index) => (
                <Item key={index} item={item} />
              ))}
            </Slider>
          </Grid>
        </Grid>
      </Container>
    </Box>
  );
};

export default TestimonialBlock;
