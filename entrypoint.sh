#!/bin/sh

if [ "${WEB:-}" = "true" ]; then
    exec nginx -g "daemon off;"
fi

set -- /usr/bin/jms-go

if [ "${API:-}" = "true" ]; then
    set -- "$@" api
elif [ "${SCHEDULER:-}" = "true" ]; then
    set -- "$@" scheduler
else
    set -- "$@" sshd
fi

if [ "${DEBUG:-}" = "true" ]; then
    set -- "$@" --debug
fi

if [ -n "${TIMEOUT:-}" ] && [ "${API:-}" != "true" ] && [ "${SCHEDULER:-}" != "true" ]; then
    set -- "$@" --timeout "$TIMEOUT"
fi

exec "$@"
