// material-ui
import { useTheme } from '@mui/material/styles';
import { Box, Button, Container, CardMedia, Grid, Link, Typography } from '@mui/material';

// third party
import { motion } from 'framer-motion';

// project import
import AnimateButton from 'components/@extended/AnimateButton';

// assets
import { DownloadOutlined } from '@ant-design/icons';
import imgbg from 'assets/images/landing/img-bg-screen.svg';

// ==============================|| LANDING - CALL TO ACTION PAGE ||============================== //

const CallToActionPage = () => {
  const theme = useTheme();

  return (
    <Box
      sx={{
        position: 'relative',
        overflow: 'hidden',
        bgcolor: theme.palette.mode === 'dark' ? theme.palette.grey[100] : theme.palette.secondary[800],
        '&:after': {
          content: '""',
          position: 'absolute',
          width: '100%',
          height: '80%',
          bottom: 0,
          left: 0,
          background: `linear-gradient(180deg, transparent, ${
            theme.palette.mode === 'dark' ? theme.palette.grey[100] : theme.palette.secondary[800]
          })`
        }
      }}
    >
      <CardMedia component="img" image={imgbg} sx={{ width: 'auto', position: 'absolute', top: 0, right: 0 }} />
      <Container>
        <Grid
          container
          alignItems="center"
          justifyContent="space-between"
          spacing={2}
          sx={{
            position: 'relative',
            zIndex: 1,
            pt: { md: 18.75, xs: 7.5 },
            pb: { md: 10, xs: 3.75 }
          }}
        >
          <Grid item xs={12} md={7} sx={{ position: 'relative', zIndex: 1 }}>
            <Grid container spacing={2} sx={{ [theme.breakpoints.down('md')]: { pr: 0, textAlign: 'center' } }}>
              <Grid item xs={12}>
                <motion.div
                  initial={{ opacity: 0, translateY: 550 }}
                  animate={{ opacity: 1, translateY: 0 }}
                  transition={{
                    type: 'spring',
                    stiffness: 150,
                    damping: 30
                  }}
                >
                  <Typography
                    variant="h2"
                    sx={{
                      color: theme.palette.common.white,
                      fontSize: { xs: '1.25rem', sm: '1.56rem', md: '1.875rem' },
                      fontWeight: 700,
                      lineHeight: { xs: 1.4, sm: 1.4, md: 1.4 }
                    }}
                  >
                    <Box component="span" sx={{ mx: 0 }}>
                      Check Mantis
                    </Box>
                    <Box component="span" sx={{ mx: 0, color: theme.palette.primary.main }}>
                      &nbsp;Free&nbsp;
                    </Box>
                    Version Before Purchase
                  </Typography>
                </motion.div>
              </Grid>
              <Grid item xs={12} sx={{ my: 3.25 }}>
                <motion.div
                  initial={{ display: 'inline-block', opacity: 0, translateY: 550 }}
                  animate={{ opacity: 1, translateY: 0 }}
                  transition={{
                    type: 'spring',
                    stiffness: 150,
                    damping: 30,
                    delay: 0.2
                  }}
                >
                  <AnimateButton>
                    <Button
                      component={Link}
                      target="_blank"
                      href="https://github.com/codedthemes/mantis-free-react-admin-template"
                      size="large"
                      color="primary"
                      variant="contained"
                      startIcon={<DownloadOutlined />}
                    >
                      Download Now
                    </Button>
                  </AnimateButton>
                </motion.div>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
      </Container>
    </Box>
  );
};

export default CallToActionPage;
