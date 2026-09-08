#!/usr/bin/env bash
# mac_screencap.sh 1.2.1
# Adjusts macOS SHIFT-CMD-4 screen capture settings

set -euo pipefail  # Fail immediately on any error
Gre='\e[1;32m' Red='\e[1;31m' Mag='\e[1;35m' Yel='\e[1;33m' Blu='\e[1;34m' Rst='\e[0m'

defaults write com.apple.screencapture location ~/Downloads
defaults write com.apple.screencapture type jpg
defaults write com.apple.screencapture name "screenshot"
defaults write com.apple.screencapture include-date -bool false
defaults write com.apple.screencapture disable-shadow -bool false
killall SystemUIServer
