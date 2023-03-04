const path = require('path')
const del = require('del')
const MiniCssExtractPlugin = require('mini-css-extract-plugin')
const RtlCssPlugin = require('rtlcss-webpack-plugin')

// global variables
const rootPath = path.resolve(__dirname)
const distPath = rootPath + '/src/_metronic/assets'
const entries = {
  'css/style': './src/_metronic/assets/sass/style.scss',
}

// remove older folders and files
;(async () => {
  await del(distPath + '/css', {force: true})
})()

module.exports = {
  mode: 'development',
  stats: 'verbose',
  performance: {
    hints: 'error',
    maxAssetSize: 10000000,
    maxEntrypointSize: 4000000,
  },
  entry: entries,
  output: {
    // main output path in assets folder
    path: distPath,
    // output path based on the entries' filename
    filename: '[name].js',
  },
  resolve: {
    extensions: ['.scss'],
  },
  plugins: [
    new MiniCssExtractPlugin({
      filename: '[name].rtl.css',
    }),
    new RtlCssPlugin({
      filename: '[name].rtl.css',
    }),
    {
      apply: (compiler) => {
        // hook name
        compiler.hooks.afterEmit.tap('AfterEmitPlugin', () => {
          ;(async () => {
            await del(distPath + '/css/*.js', {force: true})
          })()
        })
      },
    },
  ],
  module: {
    rules: [
      {
        test: /\.scss$/,
        use: [
          MiniCssExtractPlugin.loader,
          'css-loader',
          {
            loader: 'sass-loader',
            options: {
              sourceMap: true,
            },
          },
        ],
      },
    ],
  },
}
