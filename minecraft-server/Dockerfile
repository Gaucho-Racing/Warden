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

# Warden trusts the Mojang-verified UUID, so online mode must stay on.
ENV ONLINE_MODE=TRUE \
    ENFORCE_SECURE_PROFILE=TRUE \
    SERVER_NAME="Gaucho Racing" \
    MOTD="Gaucho Racing" \
    MODE=survival \
    FORCE_GAMEMODE=TRUE \
    DIFFICULTY=hard \
    PVP=TRUE \
    MAX_PLAYERS=100 \
    VIEW_DISTANCE=12 \
    SIMULATION_DISTANCE=8 \
    SPAWN_PROTECTION=0 \
    ENABLE_COMMAND_BLOCK=FALSE

# Fixed filename so an upgrade overwrites the previous jar in /data/plugins
# instead of leaving two copies of Warden side by side.
COPY --from=builder /Warden.jar /plugins/Warden.jar
COPY minecraft-server/plugins.txt /opt/warden/plugins.txt
COPY minecraft-server/plugins/ /plugins/
