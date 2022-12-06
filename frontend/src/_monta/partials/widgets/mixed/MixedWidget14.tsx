// @ts-nocheck
/* eslint-disable jsx-a11y/anchor-is-valid */
import React, {useEffect, useRef} from 'react'
import ApexCharts, {ApexOptions} from 'apexcharts'
import {getCSSVariableValue} from '../../../assets/ts/_utils'
import {useThemeMode} from '../../layout/theme-mode/ThemeModeProvider'

type Props = {
  className: string
  chartHeight: string
  backGroundColor: string
}

const MixedWidget14: React.FC<Props> = ({className, backGroundColor, chartHeight = '150px'}) => {
  const chartRef = useRef<HTMLDivElement | null>(null)
  const {mode} = useThemeMode()

  const refreshChart = () => {
    if (!chartRef.current) {
      return
    }

    const chart = new ApexCharts(chartRef.current, chartOptions(chartHeight))
    if (chart) {
      chart.render()
    }

    return chart
  }

  useEffect(() => {
    const chart = refreshChart()

    return () => {
      if (chart) {
        chart.destroy()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chartRef, mode])

  return (
    <div className={`card ${className} theme-dark-bg-body`} style={{backgroundColor: backGroundColor}}>
      {/* begin::Body */}
      <div className='card-body d-flex flex-column'>
        {/* begin::Wrapper */}
        <div className='d-flex flex-column flex-grow-1'>
          {/* begin::Title                    */}
          <a href='#' className='text-dark text-hover-primary fw-bolder fs-3'>
            Contributors
          </a>

          {/* end::Title */}

          <div
            ref={chartRef}
            className='mixed-widget-14-chart'
            style={{height: chartHeight, minHeight: chartHeight}}
          ></div>
        </div>
        {/* end::Wrapper */}

        {/* begin::Stats */}
        <div className='pt-5'>
          {/* begin::Symbol */}
          <span className='text-dark fw-bolder fs-2x lh-0'>$</span>
          {/* end::Symbol */}

          {/* begin::Number */}
          <span className='text-dark fw-bolder fs-3x me-2 lh-0'>47</span>
          {/* end::Number */}

          {/* begin::Text */}
          <span className='text-dark fw-bolder fs-6 lh-0'>- 12% this week</span>
          {/* end::Text */}
        </div>
        {/* end::Stats */}
      </div>
    </div>
  )
}

const chartOptions = (chartHeight: string): ApexOptions => {
  const labelColor = getCSSVariableValue('--kt-gray-800')

  return {
    series: [
      {
        name: 'Inflation',
        data: [1, 2.1, 1, 2.1, 4.1, 6.1, 4.1, 4.1, 2.1, 4.1, 2.1, 3.1, 1, 1, 2.1],
      },
    ],
    chart: {
      fontFamily: 'inherit',
      height: chartHeight,
      type: 'bar',
      toolbar: {
        show: false,
      },
    },
    grid: {
      show: false,
      padding: {
        top: 0,
        bottom: 0,
        left: 0,
        right: 0,
      },
    },
    colors: ['#ffffff'],
    plotOptions: {
      bar: {
        borderRadius: 2.5,
        dataLabels: {
          position: 'top', // top, center, bottom
        },
        columnWidth: '20%',
      },
    },
    dataLabels: {
      enabled: false,
      formatter: function (val) {
        return val + '%'
      },
      offsetY: -20,
      style: {
        fontSize: '12px',
        colors: ['#304758'],
      },
    },
    xaxis: {
      labels: {
        show: false,
      },
      categories: [
        'Jan',
        'Feb',
        'Mar',
        'Apr',
        'May',
        'Jun',
        'Jul',
        'Aug',
        'Sep',
        'Oct',
        'Nov',
        'Dec',
        'Jan',
        'Feb',
        'Mar',
      ],
      position: 'top',
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      crosshairs: {
        show: false,
      },
      tooltip: {
        enabled: false,
      },
    },
    yaxis: {
      show: false,
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
        background: labelColor,
      },
      labels: {
        show: false,
        formatter: function (val) {
          return val + '%'
        },
      },
    },
  }
}

export {MixedWidget14}
