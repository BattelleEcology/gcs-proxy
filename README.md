# gcs-proxy

gcs-proxy is a tool that acts as a server for your GCS bucket.

This project is forked from [NYTimes/gcs-helper](https://github.com/nytimes/gcs-helper). Since the original repository is archived, we have forked it to maintain and further improve it.

It was designed to be used with [Kaltura's nginx-vod-module](https://github.com/kaltura/nginx-vod-module), but it can also work as a standalone proxy.

Specific to nginx-vod-module, gcs-helper provides support for the mapped mode
(when using the proper environment variables - ``GCS_HELPER_PROXY_PREFIX``,
``GCS_HELPER_MAP_PREFIX`` and ``GCS_HELPER_MAP_REGEX_FILTER``).

## Configuration

The following environment variables control the behavior of gcs-helper:

| Variable                         | Default value | Required | Description                                                                                                  |
| -------------------------------- | ------------- | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GCS_HELPER_LISTEN                | :8080         | No       | Address to bind the server                                                                                                                                               |
| GCS_HELPER_BUCKET_NAME           |               | Yes      | Name of the bucket                                                                                                                                                       |
| GCS_HELPER_ROOT_REDIRECT         |               | No       | Redirect all requests from `/` to this path
| GCS_HELPER_LOG_LEVEL             | info          | No       | Logging level. At ``debug`` every proxied request is logged                                                                                                             |
| GCS_HELPER_PROXY_PREFIX          |               | No       | Prefix to use for the proxy binding, matched on whole path segments. Required if running in map and proxy modes (example value: ``/proxy/``)                           |
| GCS_HELPER_PROXY_TIMEOUT         | 10s           | No       | Defines the maximum time in serving the proxy requests, including streaming the response body. This is a hard timeout                                                  |
| GCS_HELPER_PROXY_LOG_HEADERS     |               | No       | Comma-separated list of request headers to include in the proxy request logs (example value: ``Accept,Range``)                                                         |
| GCS_HELPER_PROXY_BUCKET_ON_PATH  | false         | No       | Send proxy requests path-style (``storage.googleapis.com/<path>``) instead of to ``<bucket>.storage.googleapis.com``, so the request path must start with the bucket name. Only ``GCS_HELPER_BUCKET_NAME`` and ``GCS_HELPER_PROXY_ALLOWED_BUCKETS`` can be requested, and bucket listings are refused |
| GCS_HELPER_PROXY_ALLOWED_BUCKETS |               | No       | Comma-separated list of extra buckets that may be requested when ``GCS_HELPER_PROXY_BUCKET_ON_PATH`` is ``true``                                                       |
| GCS_HELPER_MAP_PREFIX            |               | No       | Prefix to use for the map binding, matched on whole path segments. Required if running in map and proxy modes (example value: ``/map/``)                               |
| GCS_HELPER_MAP_REGEX_FILTER      |               | No       | A regular expression that is used to deliver only those files that match the specified naming convention (example value: \d{3,4}p(\.mp4|[a-z0-9_-]{37}\.(vtt|srt))$) |

The proxy only forwards the ``Accept``, ``Accept-Encoding``, ``Range`` and
conditional (``If-*``) request headers, and only the ``generation`` query
parameter, to GCS.

The are also some configuration variables for network communication with Google
Cloud Storage API:

| Variable                     | Default value | Required | Description                                                                                                  |
| ---------------------------- | ------------- | -------- | ------------------------------------------------------------------------------------------------------------ |
| GCS_CLIENT_TIMEOUT           | 2s            | No       | Timeout for connecting to the Google Storage API and for waiting for its response headers. Also a hard timeout on each object listing call |
| GCS_CLIENT_IDLE_CONN_TIMEOUT | 120s          | No       | Maximum duration of idle connections between gcs-helper and the Google Storage API                           |
| GCS_CLIENT_MAX_IDLE_CONNS    | 10            | No       | Maximum number of idle connections to keep open. This doesn't control the maximum number of connections      |

### GCS_HELPER_PROXY_TIMEOUT x GCS_CLIENT_TIMEOUT

The timeout configuration is mainly controlled by two environment variables:
``GCS_HELPER_PROXY_TIMEOUT`` and ``GCS_CLIENT_TIMEOUT``.

For proxied requests, ``GCS_CLIENT_TIMEOUT`` limits how long gcs-helper waits
to connect to Google's API and for the response headers, and
``GCS_HELPER_PROXY_TIMEOUT`` limits the whole request, including streaming the
body. Set ``GCS_HELPER_PROXY_TIMEOUT`` above the time your largest responses
take to download; a download still running when it expires is cut off. Proxied
requests aren't retried, and when GCS fails or times out the proxy responds
with ``502`` or ``504``.

For map requests, ``GCS_CLIENT_TIMEOUT`` limits each listing call. A listing
that fails with a server error, a rate limit or a network error is retried up
to five times with exponential backoff; missing buckets and other client errors
aren't retried.
