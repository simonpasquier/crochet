<template>
<div>
  <div v-if="apiError">
    <b-alert show variant="warning">{{ apiError.message }}</b-alert>
  </div>

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
    <template v-slot:top-row="{ fields }">
      <td v-for="field in fields" :key="field.key">
        <input v-if="field.key != 'show_details' && field.key != 'alerts'" v-model="filters[field.key]" :placeholder="field.label">
      </td>
    </template>
    <template v-slot:cell(show_details)="row">
      <b-button size="sm" @click="row.toggleDetails" class="mr-2">
        {{ row.detailsShowing ? 'Hide' : 'Show'}}
      </b-button>
    </template>
    <template v-slot:row-details="row">
      <b-container class="border">
        <template v-for="(alert, index) in row.item.alerts">
          <b-row :key="index">
            <b-col :key="start" :variant="alert.status === 'firing' ? 'danger' : 'success'">{{alert.startsAt | formatDate }}</b-col>
            <b-col :key="end" :variant="alert.status === 'firing' ? 'danger' : 'success'">{{alert.endsAt | formatDate }}</b-col>
            <b-col>
              <template v-for="(value, name) in alert.labels">
                <b-list-group-item :key="name" :variant="alert.status === 'firing' ? 'danger' : 'success'"  class="flex-fill">{{name}}: {{value}}</b-list-group-item>
              </template>
            </b-col>
          </b-row>
        </template>
      </b-container>
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
        'show_details'
      ],
      filters: {
        'timestamp': '',
        'receiver': '',
        'remoteAddress': '',
        'groupKey': '',
        'groupLabels': '',
      },
      items: [],
      apiError: "",
    }
  },

  // Fetches items when the component is created.
  created() {
    this.fetchItems();

    // Refresh list every minute.
    setInterval(function () {
      this.fetchItems()
    }.bind(this), 60000);
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
    fetchItems() {
      axios.get('/api/notifications/')
      .then(response => {
        // JSON responses are automatically parsed.
        this.items = response.data
        this.apiError = null
      })
      .catch(e => {
        this.apiError = e
      })
    },

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
