#!/bin/bash


if ! id "mattermost-minecraft-bot" &> /dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin mattermost-minecraft-bot
fi