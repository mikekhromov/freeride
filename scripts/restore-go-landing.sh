#!/bin/bash
set -e

MAP=/opt/hiddify-manager/haproxy/maps/http_domain
CFG=/opt/hiddify-manager/haproxy/go-landing.cfg
BACKEND_SRC=/root/vpn/scripts/go-landing.cfg

if [ ! -f "$CFG" ]; then
    cp "$BACKEND_SRC" "$CFG"
fi

if ! grep -q "^go.arengate.tech" "$MAP"; then
    echo "go.arengate.tech go_landing" >> "$MAP"
fi

if systemctl is-active --quiet haproxy; then
    systemctl reload haproxy
else
    systemctl restart haproxy
fi
