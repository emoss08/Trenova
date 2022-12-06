/* eslint-disable jsx-a11y/anchor-is-valid */
import React, {useEffect, useRef} from 'react'
import ApexCharts, {ApexOptions} from 'apexcharts'
import {MTSVG, toAbsoluteUrl} from '../../../helpers'
import {getCSSVariableValue} from '../../../assets/ts/_utils'
import {Dropdown1} from '../../content/dropdown/Dropdown1'
import {useThemeMode} from '../../layout/theme-mode/ThemeModeProvider'

type Props = {
  className: string
  chartColor: string
  chartHeight: string
}

const MixedWidget8: React.FC<Props> = ({className, chartColor, chartHeight}) => {
  const chartRef = useRef<HTMLDivElement | null>(null)
  const {mode} = useThemeMode()
  const refreshChart = () => {
    if (!chartRef.current) {
      return
    }

    const chart1 = new ApexCharts(chartRef.current, chart1Options(chartColor, chartHeight))
    if (chart1) {
      chart1.render()
    }

    return chart1
  }

  useEffect(() => {
    const chart1 = refreshChart()

    return () => {
      if (chart1) {
        chart1.destroy()
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chartRef, mode])

  return (
    <div className={`card ${className}`}>
      {/* begin::Beader */}
      <div className='card-header border-0 py-5'>
        <h3 className='card-title align-items-start flex-column'>
          <span className='card-label fw-bold fs-3 mb-1'>Trends</span>

          <span className='text-muted fw-semibold fs-7'>Latest trends</span>
        </h3>

        <div className='card-toolbar'>
          {/* begin::Menu */}
          <button
            type='button'
            className='btn btn-sm btn-icon btn-color-primary btn-active-light-primary'
            data-kt-menu-trigger='click'
            data-kt-menu-placement='bottom-end'
            data-kt-menu-flip='top-end'
          >
            <MTSVG path='/media/icons/duotune/general/gen024.svg' className='svg-icon-2' />
          </button>
          <Dropdown1 />
          {/* end::Menu */}
        </div>
      </div>
      {/* end::Header */}

      {/* begin::Body */}
      <div className='card-body d-flex flex-column'>
        {/* begin::Chart */}
        <div ref={chartRef} className='mixed-widget-5-chart card-rounded-top'></div>
        {/* end::Chart */}

        {/* begin::Items */}
        <div className='mt-5'>
          {/* begin::Item */}
          <div className='d-flex flex-stack mb-5'>
            {/* begin::Section */}
            <div className='d-flex align-items-center me-2'>
              {/* begin::Symbol */}
              <div className='symbol symbol-50px me-3'>
                <div className='symbol-label bg-light'>
                  <img
                    src={toAbsoluteUrl('/media/svg/brand-logos/plurk.svg')}
                    alt=''
                    className='h-50'
                  />
                </div>
              </div>
              {/* end::Symbol */}

              {/* begin::Title */}
              <div>
                <a href='#' className='fs-6 text-gray-800 text-hover-primary fw-bold'>
                  Top Authors
                </a>
                <div className='fs-7 text-muted fw-semibold mt-1'>Ricky Hunt, Sandra Trepp</div>
              </div>
              {/* end::Title */}
            </div>
            {/* end::Section */}

            {/* begin::Label */}
            <div className='badge badge-light fw-semibold py-4 px-3'>+82$</div>
            {/* end::Label */}
          </div>
          {/* end::Item */}

          {/* begin::Item */}
          <div className='d-flex flex-stack mb-5'>
            {/* begin::Section */}
            <div className='d-flex align-items-center me-2'>
              {/* begin::Symbol */}
              <div className='symbol symbol-50px me-3'>
                <div className='symbol-label bg-light'>
                  <img
                    src={toAbsoluteUrl('/media/svg/brand-logos/figma-1.svg')}
                    alt=''
                    className='h-50'
                  />
                </div>
              </div>
              {/* end::Symbol */}

              {/* begin::Title */}
              <div>
                <a href='#' className='fs-6 text-gray-800 text-hover-primary fw-bold'>
                  Top Sales
                </a>
                <div className='fs-7 text-muted fw-semibold mt-1'>PitStop Emails</div>
              </div>
              {/* end::Title */}
            </div>
            {/* end::Section */}

            {/* begin::Label */}
            <div className='badge badge-light fw-semibold py-4 px-3'>+82$</div>
            {/* end::Label */}
          </div>
          {/* end::Item */}

          {/* begin::Item */}
          <div className='d-flex flex-stack'>
            {/* begin::Section */}
            <div className='d-flex align-items-center me-2'>
              {/* begin::Symbol */}
              <div className='symbol symbol-50px me-3'>
                <div className='symbol-label bg-light'>
                  <img
                    src={toAbsoluteUrl('/media/svg/brand-logos/vimeo.svg')}
                    alt=''
                    className='h-50'
                  />
                </div>
              </div>
              {/* end::Symbol */}

              {/* begin::Title */}
              <div className='py-1'>
                <a href='#' className='fs-6 text-gray-800 text-hover-primary fw-bold'>
                  Top Engagement
                </a>

                <div className='fs-7 text-muted fw-semibold mt-1'>KT.com</div>
              </div>
              {/* end::Title */}
            </div>
            {/* end::Section */}

            {/* begin::Label */}
            <div className='badge badge-light fw-semibold py-4 px-3'>+82$</div>
            {/* end::Label */}
          </div>
          {/* end::Item */}
        </div>
        {/* end::Items */}
      </div>
      {/* end::Body */}
    </div>
  )
}

const chart1Options = (chartColor: string, chartHeight: string): ApexOptions => {
  const labelColor = getCSSVariableValue('--kt-gray-800')
  const strokeColor = getCSSVariableValue('--kt-gray-300')
  const baseColor = getCSSVariableValue('--kt-' + chartColor) as string
  const lightColor = getCSSVariableValue('--kt-' + chartColor + '-light')

  return {
    series: [
      {
        name: 'Net Profit',
        data: [30, 30, 60, 25, 25, 40],
      },
    ],
    chart: {
      fontFamily: 'inherit',
      type: 'area',
      height: chartHeight,
      toolbar: {
        show: false,
      },
      zoom: {
        enabled: false,
      },
      sparkline: {
        enabled: true,
      },
    },
    plotOptions: {},
    legend: {
      show: false,
    },
    dataLabels: {
      enabled: false,
    },
    fill: {
      type: 'solid',
      opacity: 1,
    },
    fill1: {
      type: 'gradient',
      opacity: 1,
      gradient: {
        type: 'vertical',
        shadeIntensity: 0.5,
        gradientToColors: undefined,
        inverseColors: true,
        opacityFrom: 1,
        opacityTo: 0.375,
        stops: [25, 50, 100],
        colorStops: [],
      },
    },
    stroke: {
      curve: 'smooth',
      show: true,
      width: 3,
      colors: [baseColor],
    },
    xaxis: {
      categories: ['Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul'],
      axisBorder: {
        show: false,
      },
      axisTicks: {
        show: false,
      },
      labels: {
        show: false,
        style: {
          colors: labelColor,
          fontSize: '12px',
        },
      },
      crosshairs: {
        show: false,
        position: 'front',
        stroke: {
          color: strokeColor,
          width: 1,
          dashArray: 3,
        },
      },
      tooltip: {
        enabled: true,
        formatter: undefined,
        offsetY: 0,
        style: {
          fontSize: '12px',
        },
      },
    },
    yaxis: {
      min: 0,
      max: 65,
      labels: {
        show: false,
        style: {
          colors: labelColor,
          fontSize: '12px',
        },
      },
    },
    states: {
      normal: {
        filter: {
          type: 'none',
          value: 0,
        },
      },
      hover: {
        filter: {
          type: 'none',
          value: 0,
        },
      },
      active: {
        allowMultipleDataPointsSelection: false,
        filter: {
          type: 'none',
          value: 0,
        },
      },
    },
    tooltip: {
      style: {
        fontSize: '12px',
      },
      y: {
        formatter: function (val) {
          return '$' + val + ' thousands'
        },
      },
    },
    colors: [lightColor],
    markers: {
      colors: [lightColor],
      // @ts-ignore
      strokeColor: [baseColor],
      strokeWidth: 3,
    },
  }
}

export {MixedWidget8}
