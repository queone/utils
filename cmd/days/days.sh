#!/bin/bash
# days
# Number of calendar days calculator

# Original bash script.

OSDist=$(uname | tr [A-Z] [a-z])
if [[ "$OSDist" == "darwin" ]]; then
    Date=$(which gdate)
    if [[ -z "$Date" ]]; then
        printf "Install GNU /usr/local/bin/gdate\n"
        exit 1
    fi
else
    Date=$(which date)
fi

if [[ -z "$1" || -z "$(echo $1 | grep -Eo '[0-9]{4}-[0-9]{2}-[0-9]{2}')" ]]; then
    printf "Usage: $(basename $0) 2121-01-03\n"
    exit 1
fi

Now=$($Date +%s)
Target=$($Date +%s --date "$1")
Dif=$(($Target-$Now))
Days=$(($Dif / 86400)) # 86400 seconds in a day
echo ${Days#-}

exit 0
