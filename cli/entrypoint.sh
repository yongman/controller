#! /bin/sh
#
# entrypoint.sh
# Copyright (C) 2017 yanming02 <yanming02@baidu.com>
#
# Distributed under terms of the MIT license.
#
zkhosts=$1

sed -i "s/ZKHOSTS/$zkhosts/g" ~/.cc_cli_config

shift
cli $@
