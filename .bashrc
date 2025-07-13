# Git branch and dirty marker

# Source ~/.bash_aliases if it exists
if [ -f ~/.bash_aliases ]; then
    . ~/.bash_aliases
fi
parse_git_branch() {
    git rev-parse --is-inside-work-tree &>/dev/null || return
    local branch
    branch=$(git symbolic-ref --short HEAD 2>/dev/null || git describe --tags --exact-match 2>/dev/null)
    local status
    if ! git diff --quiet --ignore-submodules HEAD 2>/dev/null; then
        status="*"
    else
        status=""
    fi
    echo "$branch$status"
}

# Git status summary (adds, mods, dels, untracked)
parse_git_status_summary() {
    git rev-parse --is-inside-work-tree &>/dev/null || return

    local status=""
    local added modified deleted untracked

    added=$(git diff --cached --name-only | wc -l)
    modified=$(git diff --name-only | wc -l)
    deleted=$(git ls-files --deleted | wc -l)
    untracked=$(git ls-files --others --exclude-standard | wc -l)

    status+="A:$added M:$modified D:$deleted U:$untracked"
    echo "$status"
}

# Date/time formatter
prompt_datetime() {
    date "+%Y-%m-%d %H:%M:%S %Z"
}

# Fancy boxed PS1
export PS1='\[\e[0;36m\]┌───────────────[ \[\e[0;34m\]\w\[\e[0;36m\] ]\n'\
'│ \[\e[0;33m\]Branch:\[\e[0m\] \[\e[0;33m\]$(parse_git_branch)\n'\
'│ \[\e[0;35m\]Status:\[\e[0m\] \[\e[0;35m\]$(parse_git_status_summary)\n'\
'│ \[\e[0;32m\]Time:  \[\e[0m\] \[\e[0;32m\]$(prompt_datetime)\n'\
'\[\e[0;36m\]└─\[\e[0;32m\]\\$\[\e[0m\] '

