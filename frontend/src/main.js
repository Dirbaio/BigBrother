import Vue from 'vue'
import App from './App'
import router from './router'
import moment from 'moment'

Vue.config.productionTip = false

Vue.filter('datetime', function (value) {
  if (value) {
    return moment(value).format('ddd DD/MM/YYYY HH:mm')
  }
})
Vue.filter('date', function (value) {
  if (value) {
    return moment(value).format('ddd DD/MM/YYYY')
  }
})
Vue.filter('time', function (value) {
  if (value) {
    return moment(value).format('HH:mm:ss')
  }
})

/* eslint-disable no-new */
new Vue({
  el: '#app',
  router,
  render: h => h(App)
})
