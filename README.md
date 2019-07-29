Crochet is an [AlertManager](https://prometheus.io/docs/alerting/alertmanager/)
receiver that stores notifications in memory with a simple UI to view/filter.

Notifications are processed by the `/api/notifications/` endpoint

## Usage

Configuration of AlertManager:

```
route:
  receiver: webhook
  [...]

receivers:
- name: webhook
  webhook_configs:
  - url: 'http://localhost:8080/api/notifications/'
    send_resolved: true
```

Start `crochet`:

```
docker run -p 8080:8080 quay.io/simonpasquier/crochet
```

## License

Apache License 2.0, see [LICENSE](https://github.com/simonpasquier/crochet/blob/master/LICENSE).
