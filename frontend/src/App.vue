<template>
<div>
  <ul v-if="errors && errors.length">
    <li v-for="(error, index) in errors" :key="index">
      {{error.message}}
    </li>
  </ul>

  <b-table striped hover :items="requests" :fields="fields" :tbody-tr-class="rowClass">
    <template slot="details" slot-scope="row">
      <b-button size="sm" @click="row.toggleDetails" class="mr-2">
        {{ row.detailsShowing ? 'Hide' : 'Show'}}
      </b-button>
    </template>
    <template slot="row-details" slot-scope="row">
      <b-list-group>
        <template v-for="(alert, index) in row.item.alerts">
          <b-list-group horizontal :key="index">
            <template v-for="(value, name) in alert.labels">
              <b-list-group-item :key="name" :variant="alert.status === 'firing' ? 'danger' : 'success'">{{name}}: {{value}}</b-list-group-item>
            </template>
          </b-list-group>
        </template>
      </b-list-group>
    </template>
  </b-table>
</div>
</template>

<script>
import axios from 'axios'

export default {
  data() {
    return {
      sortBy: 'timestamp',
      sortDesc: 'true',
      fields: [
        {key: 'timestamp', sortable: true},
        'receiver',
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
            value.forEach(function(item) {
              if (item.status == 'resolved') return
              firing++
            })
            return firing + ' firing / ' + (value.length - firing) + ' resolved'
          }
        },
        'details'
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
    rowClass(item) {
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
