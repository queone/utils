#!/usr/bin/env bash
# bashrc_root.sh 1.0.0
# Generic interactive bash settings for the root account on macOS

[[ $- == *i* ]] || return # skip all of this for non-interactive shells

echo "history -c" > ~/.bash_logout
on_exit() { [ -f ~/.bash_history ] && rm ~/.bash_history && sh ~/.bash_logout ; }
trap on_exit EXIT

export BASH_SILENCE_DEPRECATION_WARNING=1
export HISTCONTROL=ignoreboth
export HISTIGNORE='ls:cd:ll:h'
export EDITOR=vi

Red='\[\e[1;31m\]' Blu='\[\e[1;34m\]' Mag='\[\e[0;35m\]'
Grn='\[\e[1;32m\]' Yel='\[\e[1;33m\]' Rst='\[\e[0m\]'
PS1="${Red}\h \W${Rst}$ "

alias grep='grep --color=auto'
alias ls='gls -N --color --group-directories-first'
alias ll='ls -la'
alias h='history'
alias vi='vim'
alias ipa="ifconfig | grep 'inet ' | grep -v 127.0.0.1 | cut -d' ' -f2"
