# enable color support of ls and also add handy aliases
alias ls='ls --color=auto'
alias grep='grep --color=auto'
alias fgrep='fgrep --color=auto'
alias egrep='egrep --color=auto'

# some more ls aliases
alias ll='ls -alF'
alias la='ls -A'
alias l='ls -CF'
# Add an "alert" alias for long running commands.  Use like so:
# Add an "alert" alias for long running commands.  Use like so:
#   sleep 10; alert
alias alert='notify-send --urgency=low -i "$([ $? = 0 ] && echo terminal || echo error)" "$(history|tail -n1|sed -e '\''s/^\s*[0-9]\+\s*//;s/[;&|]\s*alert$//'\'')"'
alias clipit='xclip -selection c'
alias kc='kubectl --namespace=secc-prod '
alias rm_tilde='find . -iname "*~" -exec rm {} \;'
alias bs='baus search '
alias run_gologesd='go run cmd/gologesd/gologesd.go -config ./base/generic/etc/gologesd/gologesd.conf.json'
alias dcls='docker container ls -a'
alias goto_leap_cluster='kubectl --namespace=leap-staging exec -it clickhouse-leap-1 /bin/bash'
alias git_update='git fetch --all ; git pull origin master'
alias ks='kubectl -n secc-prod'
alias ksl='kubectl -n leap-staging '
alias connect_ruby='bluetoothctl connect C8:7B:23:89:E2:09'
alias show_blue='kubectl -n secc-prod  get pods | grep -i blue'
alias watch='watch ' # so we can use alias defined in the interactive shell
alias git_up='git fetch --all && git pull origin main'
alias show_green='ks get pods | grep green'
alias show_blue='ks get pods | grep blue'
alias connect_soundcore='bluetoothctl connect B4:9A:95:B4:ED:48'
alias connect_bose='bluetoothctl connect C8:7B:23:89:E2:09'
alias blue_0='ks exec -it clickhouse-blue-0 /bin/bash'
alias green_0='ks exec -it clickhouse-green-0 /bin/bash'
alias start_hurl_docker="docker container start $hurl_docker"
alias run_hurl="docker exec $hurl_docker hurl "
alias prsm='ssh -4 prsmusa.com'
