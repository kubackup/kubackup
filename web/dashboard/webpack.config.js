'use strict'
const path = require('path')

module.exports = {
  context: path.resolve(__dirname, './'),
  resolve: {
    extensions: ['.js', '.ts', '.vue', '.json', '.css', '.node', '.sass'],
    alias: {
      '@': path.resolve('src'),
      'vue$': 'vue/dist/vue.esm.js'
    }
  }
}
