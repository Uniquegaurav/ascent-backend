#!/usr/bin/env bash
# End-to-end check of the explore-driven flow against a running server.
set -e
B="${1:-http://localhost:8080}"
say() { printf "\n# %s\n" "$1"; }

say "health";      curl -s "$B/health"; echo
TOKEN=$(curl -s -XPOST "$B/auth/verify-otp" -d '{"phone":"+919888800000","code":"'"${SMOKE_OTP:-000000}"'"}' \
  | sed -E 's/.*"token":"([^"]+)".*/\1/')
A="Authorization: Bearer $TOKEN"

say "explore";        curl -s "$B/explore" -H "$A" | grep -oE '"title":"[^"]+","layout":"[^"]+"'
say "search trek";    curl -s "$B/explore/search?q=trek" -H "$A" | grep -oE '"title":"[^"]+"'
say "add to ascent";  AID=$(curl -s -XPOST "$B/ascents" -H "$A" -d '{"fromExploreItemId":"ex_valley"}' | sed -E 's/.*"id":"([^"]+)".*/\1/'); echo "ascentId=$AID"
say "my ascents";     curl -s "$B/ascents" -H "$A" | grep -oE '"title":"[^"]+"'
say "add log";        curl -s -XPOST "$B/ascents/$AID/logs" -H "$A" -d '{"title":"Reached Ghangaria","note":"base camp","moodScore":5}'; echo
say "ascent detail";  curl -s "$B/ascents/$AID" -H "$A" | grep -oE '"title":"[^"]+"'
say "feed";           curl -s "$B/feed" -H "$A" | grep -oE '"authorName":"[^"]+"'
say "friends";        curl -s "$B/friends/list" -H "$A" | grep -oE '"name":"[^"]+"'
say "places";         curl -s "$B/places?type=trekking" -H "$A" | grep -oE '"name":"[^"]+"' | head -2
