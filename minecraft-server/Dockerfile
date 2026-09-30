FROM --platform=$BUILDPLATFORM maven:3.9.16-eclipse-temurin-25 AS builder

WORKDIR /build

COPY plugin/pom.xml ./
RUN mvn -B --no-transfer-progress dependency:go-offline

COPY plugin/src src
RUN mvn -B --no-transfer-progress package \
    && cp target/Warden-*.jar /Warden.jar

FROM itzg/minecraft-server:2026.9.2-java25

# Paper build must match paper.version in pom.xml.
ENV TYPE=PAPER \
    VERSION=26.2 \
    PAPER_BUILD=129 \
    MODRINTH_PROJECTS=@/opt/warden/plugins.txt \
    SYNC_SKIP_NEWER_IN_DESTINATION=false

# server.properties is copied from /config rather than generated from env.
# rcon-cli and graceful shutdown read RCON_PASSWORD, so it must match the file.
ENV COPY_CONFIG_DEST=/data \
    SKIP_SERVER_PROPERTIES=true \
    RCON_PASSWORD=GauchoRacing123

# Fixed filename so an upgrade overwrites the previous jar in /data/plugins
# instead of leaving two copies of Warden side by side.
COPY --from=builder /Warden.jar /plugins/Warden.jar
COPY minecraft-server/plugins.txt /opt/warden/plugins.txt
COPY minecraft-server/plugins/ /plugins/
COPY minecraft-server/config/ /config/
