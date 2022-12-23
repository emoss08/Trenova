import { useState } from 'react';

// material-ui
import { Accordion, AccordionDetails, AccordionSummary, Typography } from '@mui/material';

// project import
import MainCard from 'components/MainCard';

// ==============================|| ACCORDION - FIXED ||============================== //

const FixedAccordion = () => {
  const [expanded, setExpanded] = useState<string | false>(false);

  const handleChange = (panel: string) => (event: React.SyntheticEvent, newExpanded: boolean) => {
    setExpanded(newExpanded ? panel : false);
  };

  const fixedAccordionCodeString = `<Accordion expanded={expanded === 'panel1'} onChange={handleChange('panel1')}>
  <AccordionSummary aria-controls="panel1d-content" id="panel1d-header">
    <Typography variant="h6">Accordion 01</Typography>
  </AccordionSummary>
  <AccordionDetails>
    <Typography>
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
    </Typography>
  </AccordionDetails>
</Accordion>
<Accordion expanded={expanded === 'panel2'} onChange={handleChange('panel2')}>
  <AccordionSummary aria-controls="panel2d-content" id="panel2d-header">
    <Typography variant="h6">Accordion 02</Typography>
  </AccordionSummary>
  <AccordionDetails>
    <Typography>
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
    </Typography>
  </AccordionDetails>
</Accordion>
<Accordion expanded={expanded === 'panel3'} onChange={handleChange('panel3')}>
  <AccordionSummary aria-controls="panel3d-content" id="panel3d-header">
    <Typography variant="h6">Accordion 03</Typography>
  </AccordionSummary>
  <AccordionDetails>
    <Typography>
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
    </Typography>
  </AccordionDetails>
</Accordion>
<Accordion expanded>
  <AccordionSummary aria-controls="panel4d-content" id="panel4d-header">
    <Typography variant="h6">Accordion 04</Typography>
  </AccordionSummary>
  <AccordionDetails>
    <Typography>
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
      Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
    </Typography>
  </AccordionDetails>
</Accordion>`;

  return (
    <MainCard title="Fixed" codeString={fixedAccordionCodeString}>
      <>
        <Accordion expanded={expanded === 'panel1'} onChange={handleChange('panel1')}>
          <AccordionSummary aria-controls="panel1d-content" id="panel1d-header">
            <Typography variant="h6">Accordion 01</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Typography>
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
            </Typography>
          </AccordionDetails>
        </Accordion>
        <Accordion expanded={expanded === 'panel2'} onChange={handleChange('panel2')}>
          <AccordionSummary aria-controls="panel2d-content" id="panel2d-header">
            <Typography variant="h6">Accordion 02</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Typography>
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
            </Typography>
          </AccordionDetails>
        </Accordion>
        <Accordion expanded={expanded === 'panel3'} onChange={handleChange('panel3')}>
          <AccordionSummary aria-controls="panel3d-content" id="panel3d-header">
            <Typography variant="h6">Accordion 03</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Typography>
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
            </Typography>
          </AccordionDetails>
        </Accordion>
        <Accordion expanded>
          <AccordionSummary aria-controls="panel4d-content" id="panel4d-header">
            <Typography variant="h6">Accordion 04</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Typography>
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
              Lorem ipsum dolor sit amet, consectetur adipiscing elit. Suspendisse malesuada lacus ex, sit amet blandit leo lobortis eget.
            </Typography>
          </AccordionDetails>
        </Accordion>
      </>
    </MainCard>
  );
};

export default FixedAccordion;
