FROM ubuntu:20.04

ARG GOOS="linux"
ARG GOARCH="amd64"
ARG APP_HOME="/app"

ARG RUNTIME_USER="krossboard"

RUN apt-get update --no-install-recommends && \
    apt-get install -y rrdtool && \
    rm -rf /var/lib/apt/lists/* && \
    addgroup $RUNTIME_USER && \
    adduser --disabled-password --no-create-home  --gecos "" --home $APP_HOME --ingroup $RUNTIME_USER $RUNTIME_USER

COPY bin/krossboard-data-processor \
     LICENSE.md \
     $APP_HOME/

RUN chmod 755 $APP_HOME/krossboard-data-processor && \
    chown -R $RUNTIME_USER:$RUNTIME_USER $APP_HOME/

WORKDIR $APP_HOME/

ENTRYPOINT ["/app/krossboard-data-processor"]