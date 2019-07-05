A minimalist UI to receive and display AlertManager webhook requests.

# Usage

```
  -help
        Help message
  -listen-address string
        Listen address (default ":8080")
```

You can make the server wait an extra period of time before returning to the client.

```
curl http://localhost:8080/?sleep=1s
```

The `sleep` parameter can be any duration string supported by
[`time.ParseDuration()`](https://golang.org/pkg/time/#ParseDuration).

You can randomize the sleep duration too.

```
curl 'http://localhost:8080/?sleep=1s&random'
```
