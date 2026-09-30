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
    MODRINTH_PROJECTS=luckperms:v5.5.71-bukkit \
    ONLINE_MODE=TRUE \
    SYNC_SKIP_NEWER_IN_DESTINATION=false

# Fixed filename so an upgrade overwrites the previous jar in /data/plugins
# instead of leaving two copies of Warden side by side.
COPY --from=builder /Warden.jar /plugins/Warden.jar
COPY minecraft-server/config.yml /plugins/Warden/config.yml
