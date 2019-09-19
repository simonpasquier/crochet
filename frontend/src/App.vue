<template>
<div>
  <ul v-if="errors && errors.length">
    <li v-for="(error, index) in errors" :key="index">
      {{error.message}}
    </li>
  </ul>

  <b-table
    hover
    striped
    show-empty
    :items="filtered"
    :fields="fields"
    :sort-by.sync="sortBy"
    :sort-desc.sync="sortDesc"
    :tbody-tr-class="rowClass"
  >
    <template slot="top-row" slot-scope="{ fields }">
      <td v-for="field in fields" :key="field.key">
        <input v-if="field.key != 'details' && field.key != 'alerts'" v-model="filters[field.key]" :placeholder="field.label">
      </td>
    </template>
    <template slot="details" slot-scope="row">
      <b-button size="sm" @click="row.toggleDetails" class="mr-2">
        {{ row.detailsShowing ? 'Hide' : 'Show'}}
      </b-button>
    </template>
    <template slot="row-details" slot-scope="row">
      <b-list-group>
        <template v-for="(alert, index) in row.item.alerts">
          <b-list-group horizontal :key="index">
            <b-list-group-item :key="start" :variant="alert.status === 'firing' ? 'danger' : 'success'">{{alert.startsAt | formatDate }}</b-list-group-item>
            <b-list-group-item :key="end" :variant="alert.status === 'firing' ? 'danger' : 'success'">{{alert.endsAt | formatDate }}</b-list-group-item>
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
      sortDesc: true,
      fields: [
        {
          key: 'timestamp',
          sortable: true,
          formatter: value => {
            return this.$options.filters.formatDate(value)
          }
        },
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
      filters: {
        'timestamp': '',
        'receiver': '',
        'remoteAddress': '',
        'groupKey': '',
        'groupLabels': '',
      },
      items: [],
      errors: []
    }
  },

  // Fetches items when the component is created.
  created() {
    axios.get('http://localhost:8080/api/notifications/')
    .then(response => {
      // JSON responses are automatically parsed.
      this.items = response.data
    })
    .catch(e => {
      this.errors.push(e)
    })
  },

  computed: {
    filtered() {
      const filtered = this.items.filter(item => {
        return Object.keys(this.filters).every(key =>
            JSON.stringify(item[key]).includes(this.filters[key]))
      })
      return filtered.length > 0 ? filtered : []
    }
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
