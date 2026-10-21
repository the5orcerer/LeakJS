#!/usr/bin/env bash
set -e

echo "[*] Purging historical 'leakjs' binary blobs from Git history..."
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch --force --index-filter 'git rm --cached --ignore-unmatch leakjs' --prune-empty --tag-name-filter cat -- --all

echo "[*] Removing original refs backup..."
rm -rf .git/refs/original/

echo "[*] Expiring reflogs..."
git reflog expire --expire=now --all

echo "[*] Running aggressive garbage collection and pruning..."
git gc --prune=now --aggressive

echo "[+] Done! Repository size:"
du -sh .git
echo "[!] To update the remote GitHub repository, run:"
echo "    git push --force --all origin"
echo "    git push --force --tags origin"

