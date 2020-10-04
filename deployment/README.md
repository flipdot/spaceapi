# Ansible
Ansible does
DNS, targeting, updating, deployment


Needed for DNS entries
Set INWX_USER
Set INWX_PASSWORD

Example run for the test environment
cd deployment
source test.env; HCLOUD_TOKEN=xxx ansible-playbook -i inventory/hcloud.yml -e env=test deploy.yml


# Docker
## Setup
Install hcloud to list servers in Hetzner cloud and connect to them
pip install hcloud

## deployment
list hetzner cloud server
hcloud server list

connect to docker host via ssh
hcloud server ssh [name or id]

change dir to current spaceapi git folder
cd spaceapi

Example run for the test environment
source test.env;  docker-compose --build up
