#!/usr/bin/env bash
# bashrc_user.sh 1.0.0
# Generic interactive bash settings for a user account on macOS

[[ $- == *i* ]] || return # skip all of this for non-interactive shells

echo "history -c" > ~/.bash_logout
on_exit() { [ -f ~/.bash_history ] && rm ~/.bash_history && sh ~/.bash_logout ; }
trap on_exit EXIT

export GOPATH=~/.go
export PATH="$PATH:/usr/local/bin:$GOPATH/bin:$HOME/.local/bin:$HOME/.cargo/bin"

Red='\[\e[1;31m\]' Blu='\[\e[1;34m\]' Mag='\[\e[0;35m\]'
Grn='\[\e[1;32m\]' Yel='\[\e[1;33m\]' Rst='\[\e[0m\]'
if [[ -f "$HOME/.gitbranch.sh" ]]; then
    source "$HOME/.gitbranch.sh"
    PS1="${Grn}\h \W${Rst} \$(git_branch '${Yel}%s${Rst} ')$ "
else
    PS1="${Grn}\h \W${Rst}$ "
fi

export BASH_SILENCE_DEPRECATION_WARNING=1
export HOMEBREW_NO_ANALYTICS=1 # Disable homebrew Google Analytics collection
export HISTCONTROL=ignoreboth
export HISTIGNORE='ls:cd:ll:h'
export EDITOR=vi
alias grep='grep --color=auto'
alias ls='gls -N --color --group-directories-first'
alias ll='ls -la'
alias h='history'
alias vi='vim'
alias vs='/Applications/VSCodium.app/Contents/Resources/app/bin/codium -r'
alias ipa="ifconfig | grep 'inet ' | grep -v 127.0.0.1 | cut -d' ' -f2"
alias d='date +"%Y-$(date +%b | tr A-Z a-z)-%d %a %I:%M$(date +%p | tr A-Z a-z | cut -c1)" | tee >(pbcopy)'
alias vm='limactl'
alias tagv='git tag --sort=-v:refname | head -1'

[[ -f "$HOME/.cargo/env" ]] && source "$HOME/.cargo/env" # PATH already has cargo/bin above, so its prepend is a no-op

# Private settings (tokens, tenant IDs, account aliases) live outside this file
[[ -f ~/.bashrc.local ]] && source ~/.bashrc.local
