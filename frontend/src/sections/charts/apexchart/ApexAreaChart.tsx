import { useEffect, useState } from 'react';

// material-ui
import { useTheme } from '@mui/material/styles';

// third-party
import ReactApexChart, { Props as ChartProps } from 'react-apexcharts';

// project imports
import useConfig from 'hooks/useConfig';

// chart options
const areaChartOptions = {
  chart: {
    height: 350,
    type: 'area'
  },
  colors: ['primary.700', 'primary.main'],
  dataLabels: {
    enabled: false
  },
  stroke: {
    curve: 'smooth'
  },
  xaxis: {
    type: 'datetime',
    categories: [
      '2018-09-19T00:00:00.000Z',
      '2018-09-19T01:30:00.000Z',
      '2018-09-19T02:30:00.000Z',
      '2018-09-19T03:30:00.000Z',
      '2018-09-19T04:30:00.000Z',
      '2018-09-19T05:30:00.000Z',
      '2018-09-19T06:30:00.000Z'
    ]
  },
  tooltip: {
    x: {
      format: 'dd/MM/yy HH:mm'
    }
  },
  legend: {
    show: true,
    fontFamily: `'Roboto', sans-serif`,
    position: 'bottom',
    offsetX: 10,
    offsetY: 10,
    labels: {
      useSeriesColors: false
    },
    markers: {
      width: 16,
      height: 16,
      radius: 5
    },
    itemMargin: {
      horizontal: 15,
      vertical: 8
    }
  }
};

// ==============================|| APEXCHART - AREA ||============================== //

const ApexAreaChart = () => {
  const theme = useTheme();

  const { mode } = useConfig();
  const line = theme.palette.divider;
  const { primary, secondary } = theme.palette.text;

  const [series] = useState([
    {
      name: 'Series 1',
      data: [31, 40, 28, 51, 42, 109, 100]
    },
    {
      name: 'Series 2',
      data: [11, 32, 45, 32, 34, 52, 41]
    }
  ]);

  const [options, setOptions] = useState<ChartProps>(areaChartOptions);
  useEffect(() => {
    setOptions((prevState) => ({
      ...prevState,
      colors: [theme.palette.primary[700], theme.palette.primary.main],
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
      tooltip: {
        theme: mode === 'dark' ? 'dark' : 'light'
      },
      legend: {
        labels: {
          colors: 'grey.500'
        }
      }
    }));
  }, [mode, primary, secondary, line, theme]);

  return (
    <div id="chart">
      <ReactApexChart options={options} series={series} type="area" height={350} />
    </div>
  );
};

export default ApexAreaChart;
