#!/usr/bin/env sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repository_root"

gitee_url='https://gitee.com/lnsyzjw/yi-kd-web-client-go.git'
github_url='https://github.com/1609676823/YiKdWebClient-Go.git'

git rev-parse --is-inside-work-tree >/dev/null

if git remote get-url origin >/dev/null 2>&1; then
    git remote set-url origin "$gitee_url"
else
    git remote add origin "$gitee_url"
fi

unset_status=0
git config --unset-all remote.origin.pushurl || unset_status=$?
if [ "$unset_status" -ne 0 ] && [ "$unset_status" -ne 5 ]; then
    exit "$unset_status"
fi

git config --add remote.origin.pushurl "$gitee_url"
git config --add remote.origin.pushurl "$github_url"
git config remote.pushDefault origin
git config push.default current

branch=$(git branch --show-current)
if [ -n "$branch" ]; then
    git config "branch.$branch.remote" origin
    git config "branch.$branch.merge" "refs/heads/$branch"
fi

printf '%s\n' 'Fetch URL:'
git remote get-url origin
printf '%s\n' 'Push URLs:'
git remote get-url --push --all origin
