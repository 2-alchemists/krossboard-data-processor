WORKING_DIR=$(dirname $0)
/usr/bin/docker run --rm \
    --name krossboard-ui \
    --net host \
    -v $WORKING_DIR/etc/Caddyfile:/etc/caddy/Caddyfile \
    -v $WORKING_DIR/etc/.caddy:/root/.caddy \
    krossboard/krossboard-ui:6c0b56e