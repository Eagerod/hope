package hope

var NginxConfig = `user  nginx;
worker_processes  1;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;


events {
    worker_connections  1024;
}

stream {
    log_format basic '$remote_addr [$time_local] '
                     '$protocol $status $bytes_sent $bytes_received '
                     '$session_time';

    upstream backend {%s
    }

    server {
        listen            6443;
        proxy_pass        backend;
    }
}
`
