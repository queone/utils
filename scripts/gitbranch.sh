#!/usr/bin/env bash
# gitbranch.sh 1.0.0
# Fast git branch PS1 indicator in bash

# EXAMPLE USAGE:
# export Grn='\[\e[1;32m\]' Yel='\[\e[1;33m\]' Rst='\[\e[0m\]'
# if [[ -f "/path/to/gitbranch.sh" ]]; then
#     source "path/to/gitbranch.sh"
#     export PS1="${Grn}\h \W${Rst} \$(git_branch '${Yel}%s${Rst} ')$ "
# else
#     export PS1="${Grn}\h \W${Rst}$ "
# fi

# Simple PS1 without colors using format arg. Feel free to use PROMPT_COMMAND
export PS1="\u@\h \w \$(git_branch '[%s] ')$ "

# Optimized function for getting the current Git branch in Bash
function git_branch() {
    local format="${1:-(%s) }"
    
    # Invalidate cache if directory changed
    if [ "$GIT_BRANCH_CACHE_DIR" != "$PWD" ]; then
        unset GIT_BRANCH_CACHE
    fi

    # Use cached value if available
    if [ -n "$GIT_BRANCH_CACHE" ]; then
        printf "$format" "$GIT_BRANCH_CACHE"
        return
    fi

    local git_dir head_file branch
    local dir="$PWD"

    # Traverse up to find .git directory or file
    while [ -n "$dir" ]; do
        if [ -d "$dir/.git" ]; then
            git_dir="$dir/.git"
            break
        elif [ -f "$dir/.git" ]; then
            git_dir=$(sed 's/gitdir: //' "$dir/.git")  # Worktree support
            break
        fi
        dir="${dir%/*}"
    done

    # If no Git directory found, return early
    [ -z "$git_dir" ] && return 0

    # Read the HEAD reference
    head_file="$git_dir/HEAD"
    if [ -r "$head_file" ]; then
        read -r head < "$head_file" || return
        case "$head" in
            ref:*) branch="${head##*/}" ;;  # Extract branch name
            *) branch="${head:0:7}" ;;  # Detached HEAD
        esac
    fi

    # Cache result for performance
    export GIT_BRANCH_CACHE="$branch"
    export GIT_BRANCH_CACHE_DIR="$PWD"

    # Print branch name with format
    [ -n "$branch" ] && printf "$format" "$branch"
}
