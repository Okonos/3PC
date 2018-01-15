#!/bin/bash

# allow guest user to connect from a remote host
mkdir /etc/rabbitmq
echo 'loopback_users = none' > /etc/rabbitmq/rabbitmq.conf

# add erlang repository entry
wget https://packages.erlang-solutions.com/erlang-solutions_1.0_all.deb
sudo dpkg -i erlang-solutions_1.0_all.deb

# add apt repository
echo "deb https://dl.bintray.com/rabbitmq/debian xenial main" | \
	sudo tee /etc/apt/sources.list.d/bintray.rabbitmq.list
# add public key
wget -O- https://www.rabbitmq.com/rabbitmq-release-signing-key.asc | \
	sudo apt-key add -

sudo apt-get update
# install erlang
sudo apt-get install -y esl-erlang
# dpkg -i /vagrant/esl-erlang_20.1-1~ubuntu~xenial_amd64.deb
sudo apt-get install -y rabbitmq-server
