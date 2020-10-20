# Ansible
Ansible does DNS, targeting, updating, deployment

## Requirements

Credentials needs to be setup from https://gitlab.com/flipdot/hosting/passwordstore

## deployment
Example run for the test environment

`cd deployment`

`source test.env; PASSWORD_STORE_DIR="see requirements" HCLOUD_TOKEN=xxx ansible-playbook -i inventory/hcloud.yml -e env=test deploy.yml`


# Docker
## Requirements

`pip3 install hcloud`

## deployment
list hetzner cloud server

`hcloud server list`

connect to docker host via ssh

`hcloud server ssh [name or id]`

change dir to current spaceapi git folder

`cd spaceapi`

Example run for the test environment

`source test.env;  docker-compose --build up`
