user  nginx;
worker_processes  1;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;


events {
    worker_connections  1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  65;

    gzip             on;
    gzip_min_length  1000;
    gzip_types       text/html text/plain application/xml text/css application/javascript application/json application/x-javascript text/javascript;
    gzip_disable     "MSIE [1-6]\.";
    gzip_static      on;

{{ range . }}
    server {
        listen 80;
        server_name {{ .Host }};
        {{ if ne .Root "" }}
        root {{ .Root }}
        index index.htm index.html;

        location ~* \.(?:ico|css|js|gif|jpe?g|png)$ {
            expires 1w;
            add_header Cache-Control "public";
        }

        location = /service-worker.js {
            expires -1;
        }
        {{ end }}
        {{ if ne .Redirect "" }}
        location / {
            rewrite ^(.*)$ https://{{ .Redirect }}$1 permanent;
        }
        {{ end }}
    }
{{ end }}
}

