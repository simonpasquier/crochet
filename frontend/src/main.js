import Vue from 'vue'
import BootstrapVue from 'bootstrap-vue'
import './custom.scss'
import App from './App.vue'
import moment from 'moment'

Vue.use(BootstrapVue)
Vue.config.productionTip = false
Vue.filter('formatDate', function(value) {
  if (value) {
    if (String(value) == "0001-01-01T00:00:00Z") {
        return ""
    }
    return moment(String(value)).format('YYYY/MM/DD kk:mm:ss.SSS')
  }
})

new Vue({
  render: h => h(App),
}).$mount('#app')
