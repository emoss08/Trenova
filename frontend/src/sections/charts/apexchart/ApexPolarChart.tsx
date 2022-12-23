import { useEffect, useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';

// third-party
import ReactApexChart, { Props as ChartProps } from 'react-apexcharts';

// project import
import useConfig from 'hooks/useConfig';

// chart options
const polarChartOptions = {
  chart: {
    width: 450,
    height: 450,
    type: 'polarArea'
  },
  fill: {
    opacity: 1
  },
  legend: {
    show: true,
    fontFamily: `'Roboto', sans-serif`,
    offsetX: 10,
    offsetY: 10,
    labels: {
      useSeriesColors: false
    },
    markers: {
      width: 12,
      height: 12,
      radius: 5
    },
    itemMargin: {
      horizontal: 25,
      vertical: 4
    }
  },
  responsive: [
    {
      breakpoint: 450,
      chart: {
        width: 280,
        height: 280
      },
      options: {
        legend: {
          show: false,
          position: 'bottom'
        }
      }
    }
  ]
};

// ==============================|| APEXCHART - POLAR ||============================== //

const ApexPolarChart = () => {
  const theme = useTheme();
  const { mode } = useConfig();

  const { primary } = theme.palette.text;
  const line = theme.palette.divider;
  const grey200 = theme.palette.grey[200];
  const backColor = theme.palette.background.paper;

  const [series] = useState<number[]>([14, 23, 21, 17, 15, 10, 12, 17, 21]);
  const [options, setOptions] = useState<ChartProps>(polarChartOptions);

  const secondary = theme.palette.primary[700];
  const primaryMain = theme.palette.primary.main;
  const successDark = theme.palette.success.main;
  const error = theme.palette.error.main;
  const warningDark = theme.palette.warning.main;

  useEffect(() => {
    setOptions((prevState) => ({
      ...prevState,
      colors: [secondary, primaryMain, successDark, error, warningDark, error],
      xaxis: {
        labels: {
          style: {
            colors: [primary, primary, primary, primary, primary, primary, primary]
          }
        }
      },
      yaxis: {
        labels: {
          style: {
            colors: [primary]
          }
        }
      },
      grid: {
        borderColor: line
      },
      legend: {
        labels: {
          colors: 'grey.500'
        }
      },
      stroke: {
        colors: [backColor]
      },
      plotOptions: {
        polarArea: {
          rings: {
            strokeColor: line
          },
          spokes: {
            connectorColors: line
          }
        }
      }
    }));
  }, [mode, primary, line, grey200, backColor, secondary, primaryMain, successDark, error, warningDark]);

  return (
    <div id="chart">
      <ReactApexChart options={options} series={series} type="polarArea" />
    </div>
  );
};

export default ApexPolarChart;
