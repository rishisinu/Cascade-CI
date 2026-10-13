# ruleid: dockerfile-latest-tag
FROM ubuntu:latest AS build
# ruleid: dockerfile-curl-pipe-shell
RUN curl -fsSL https://example.com/install.sh | sh
# ok: dockerfile-curl-pipe-shell
RUN curl -fsSL -o /tmp/install.sh https://example.com/install.sh && sha256sum -c /tmp/install.sh.sha256
# ruleid: dockerfile-secret-in-env-arg
ARG NPM_TOKEN
# ruleid: dockerfile-secret-in-env-arg
ENV DB_PASSWORD=placeholder
# ok: dockerfile-secret-in-env-arg
ENV NODE_ENV=production
# ruleid: dockerfile-add-remote-url
ADD https://example.com/tool.tar.gz /opt/
# ok: dockerfile-add-remote-url
ADD --checksum=sha256:0000000000000000000000000000000000000000000000000000000000000000 https://example.com/tool.tar.gz /opt/
# ok: dockerfile-user-root
USER root
RUN apt-get update
USER app

# ok: dockerfile-latest-tag
FROM debian:12-slim
COPY --from=build /out /app
# ruleid: dockerfile-user-root
USER root
CMD ["/app/server"]
