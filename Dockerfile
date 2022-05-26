FROM alpine:3.16.0

ARG GOOS="linux"
ARG GOARCH="amd64"
ARG APP_HOME="/app"

ARG RUNTIME_USER="krossboard"

RUN addgroup $RUNTIME_USER && \
    adduser --disabled-password --no-create-home  --gecos "" --home $APP_HOME --ingroup $RUNTIME_USER $RUNTIME_USER

COPY entrypoint.sh \
     bin/krossboard-data-processor \
     LICENSE.md \
     $APP_HOME/

RUN chmod 755 $APP_HOME/krossboard-data-processor && \
    chown -R $RUNTIME_USER:$RUNTIME_USER $APP_HOME/

WORKDIR $APP_HOME/
ENTRYPOINT ["sh", "./entrypoint.sh"]