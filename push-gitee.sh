#!/bin/bash
# 一键推送本地 commit 到 Gitee (绕过全局代理)
cd "$(dirname "$0")"
git -c http.proxy= -c https.proxy= push gitee master
