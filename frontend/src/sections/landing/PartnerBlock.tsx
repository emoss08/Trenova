// material-ui
import { Box, Container, Grid, Typography } from '@mui/material';

// third party
import Marquee from 'react-fast-marquee';

// assets
import imgcust1 from 'assets/images/landing/client-01.png';
import imgcust2 from 'assets/images/landing/client-02.png';
import imgcust3 from 'assets/images/landing/client-03.png';
import imgcust4 from 'assets/images/landing/client-04.png';

// ================================|| SLIDER - ITEMS ||================================ //

const Item = ({ item }: { item: { text: string; highlight?: boolean } }) => (
  <Typography
    variant="h2"
    sx={{
      cursor: 'pointer',
      fontWeight: 600,
      my: 1,
      mx: 2.5,
      transition: 'all 0.3s ease-in-out',
      opacity: item.highlight ? 0.75 : 0.4,
      '&:hover': { opacity: '1' }
    }}
  >
    {item.text}
  </Typography>
);

// ==============================|| LANDING - PARTNER PAGE ||============================== //

const PartnerBlock = () => {
  const partnerimage = [{ image: imgcust1 }, { image: imgcust2 }, { image: imgcust3 }, { image: imgcust4 }];
  const items = [
    { text: 'Auth Methods' },
    { text: '150+ Pages' },
    { text: '6+ Preset Colors' },
    { text: '50+ Widgets' },
    { text: 'Best User Experience' },
    { text: 'Live Customizer' },
    { text: '5+ Apps' },
    { text: 'Material UI v5' },
    { text: 'Highly Flexible' },
    { text: 'Always Updated' },
    { text: 'Professional Design' },
    { text: 'TypeScript Support' },
    { text: 'Figma Design' },
    { text: 'Dark Layout' },
    { text: 'RTL Support' },
    { text: 'Retina Ready' },
    { text: 'Prettier Code' },
    { text: 'i18n Support' }
  ];
  return (
    <Box sx={{ overflowX: 'hidden' }}>
      <Container>
        <Grid container alignItems="center" justifyContent="center" spacing={2} sx={{ mt: { md: 15, xs: 2.5 }, mb: { md: 5, xs: 2.5 } }}>
          <Grid item xs={12}>
            <Grid container spacing={1} justifyContent="center" sx={{ mb: 4, textAlign: 'center' }}>
              <Grid item sm={10} md={6}>
                <Grid container spacing={1} justifyContent="center">
                  <Grid item xs={12}>
                    <Typography variant="subtitle1" color="primary">
                      Customer Portfolio
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="h2">They Trust Us</Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body1">
                      Our Products are use by top companies around the globe. More than 50K+ customers believe in us for their end product.
                    </Typography>
                  </Grid>
                </Grid>
              </Grid>
            </Grid>
          </Grid>
          <Grid item xs={12}>
            <Grid container spacing={5} justifyContent="center" sx={{ mb: 4, textAlign: 'center' }}>
              {partnerimage.map((item, index) => (
                <Grid item key={index}>
                  <img src={item.image} alt="feature" />
                </Grid>
              ))}
            </Grid>
          </Grid>
        </Grid>
      </Container>
      {/* <Grid container spacing={0} sx={{ mb: 15, mt: 10 }}>
        <Grid item xs={12}>
          <Box sx={{ bgcolor: 'secondary.lighter', py: 5 }}>
            <Container>
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                  <Stack spacing={3} alignItems="center">
                    <Typography align="center" variant="subtitle1" color="primary">
                      Mantis
                    </Typography>
                    <Typography align="center" variant="h2">
                      SEO
                    </Typography>
                    <Typography align="center" variant="caption" color="secondary">
                      Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's
                      standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a
                      type specimen book. It has survived not only five centuries,
                    </Typography>
                    <Typography align="center" variant="caption" color="secondary">
                      Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's
                      standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a
                      type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining
                      essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum
                      passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.
                    </Typography>
                  </Stack>
                </Grid>
                <Grid item xs={12} sm={6}>
                  2
                </Grid>
              </Grid>
            </Container>
          </Box>
        </Grid>
      </Grid> */}
      <Grid container spacing={0}>
        <Grid item xs={12}>
          <Marquee pauseOnHover gradient={false}>
            {items.map((item, index) => (
              <Item key={index} item={item} />
            ))}
          </Marquee>
        </Grid>
        <Grid item xs={12}>
          <Marquee pauseOnHover direction="right" gradient={false}>
            {items.map((item, index) => (
              <Item key={index} item={item} />
            ))}
          </Marquee>
        </Grid>
      </Grid>
    </Box>
  );
};

export default PartnerBlock;
