#!/bin/bash
# decolor
sed 's/\x1B\[[0-9;]\{1,5\}[mGK]//g'
