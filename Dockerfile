FROM alpine:3.11.6

ARG GOOS="linux"
ARG GOARCH="amd64"
ARG APP_HOME="/app"

ARG RUNTIME_USER="krossboard"

RUN addgroup -g $RUNTIME_USER_UID $RUNTIME_USER && \
    adduser --disabled-password --no-create-home  --gecos "" --home $APP_HOME --ingroup $RUNTIME_USER $RUNTIME_USER

COPY entrypoint.sh \
     bin/krossboard-data-processor \
     LICENSE.md \
     $APP_HOME/

RUN chown -R $RUNTIME_USER:$RUNTIME_USER $APP_HOME/

WORKDIR $APP_HOME/
ENTRYPOINT ["sh", "./entrypoint.sh"]