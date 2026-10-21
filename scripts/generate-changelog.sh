#!/usr/bin/env bash
set -e

# Find the latest semver tag
PREV_TAG=$(git tag -l | grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n 1 || true)

if [ -z "$PREV_TAG" ]; then
    echo "## What's Changed"
    echo ""
    COMMITS=$(git log -n 25 --pretty=format:"%h|%s|%an")
else
    echo "## What's Changed since ${PREV_TAG}"
    echo ""
    COMMITS=$(git log "${PREV_TAG}..HEAD" --pretty=format:"%h|%s|%an")
fi

FEATS=""
FIXES=""
DOCS=""
REFACTOR=""
OTHER=""

while IFS="|" read -r hash subject author; do
    [ -z "$hash" ] && continue
    line="* ${subject} (${hash}) by @${author}"
    if echo "$subject" | grep -qiE "^feat(\(.*\))?:"; then
        FEATS="${FEATS}${line}"$'\n'
    elif echo "$subject" | grep -qiE "^fix(\(.*\))?:"; then
        FIXES="${FIXES}${line}"$'\n'
    elif echo "$subject" | grep -qiE "^docs(\(.*\))?:"; then
        DOCS="${DOCS}${line}"$'\n'
    elif echo "$subject" | grep -qiE "^(perf|refactor)(\(.*\))?:"; then
        REFACTOR="${REFACTOR}${line}"$'\n'
    else
        OTHER="${OTHER}${line}"$'\n'
    fi
done <<< "$COMMITS"

if [ -n "$FEATS" ]; then
    echo "### 🚀 Features"
    echo "$FEATS"
fi
if [ -n "$FIXES" ]; then
    echo "### 🐛 Bug Fixes"
    echo "$FIXES"
fi
if [ -n "$REFACTOR" ]; then
    echo "### ⚡ Performance & Refactoring"
    echo "$REFACTOR"
fi
if [ -n "$DOCS" ]; then
    echo "### 📝 Documentation"
    echo "$DOCS"
fi
if [ -n "$OTHER" ]; then
    echo "### 📦 Other Changes"
    echo "$OTHER"
fi

