<template>
<div>
  <b-table striped hover :items="requests" :fields="fields" :tbody-tr-class="rowClass"></b-table>

  <ul v-if="errors && errors.length">
    <li v-for="error of errors">
      {{error.message}}
    </li>
  </ul>
</div>
</template>

<script>
import axios from 'axios'

export default {
  data() {
    return {
      fields: [
        'timestamp',
        {
          key: 'remoteAddress',
          formatter: value => {
            return value.split(':')[0]
          }
        },
        {
          key: 'groupKey',
          label: 'Route',
          formatter: value => {
            value = value.split(':')[0]
            return value
          }
        },
        'groupLabels',
        {
          key: 'alerts',
          formatter: value => {
            var firing = 0
            value.forEach(function(item, index) {
              if (item.status == 'resolved') return
              firing++
            })
            return firing + ' firing / ' + (value.length - firing) + ' resolved'
          }
        }
      ],
      requests: [],
      errors: []
    }
  },

  // Fetches requests when the component is created.
  created() {
    axios.get('http://localhost:8080/requests/')
    .then(response => {
      // JSON responses are automatically parsed.
      this.requests = response.data
    })
    .catch(e => {
      this.errors.push(e)
    })
  },

  methods: {
    rowClass(item, type) {
      if (!item) return
      if (item.status === 'firing') return 'table-danger'
      return 'table-success'
    }
  }
}
</script>

<style>
#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}
</style>
