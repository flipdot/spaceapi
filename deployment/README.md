
# Deployment via Ansible

Ansible does DNS, targeting, updating, deployment

## Requirements

Credentials needs to be setup from https://gitlab.com/flipdot/hosting/passwordstore

`pip3 install hcloud`

Access to an `HCLOUD_TOKEN` for the prod environment. When you need one, ask your admin

Access to the prod-docker host https://gitlab.com/flipdot/hosting/docker/. When need you need one, ask your admin

## Deployment

Example run for the prod environment

`source prod.env; PASSWORD_STORE_DIR="see requirements" HCLOUD_TOKEN=xxx ansible-playbook -i inventory/hcloud.yml -e env=prod deploy.yml`

# Deployment via SSH

This update the spaceapi app only

## Requirements

SSH access to api.flipdot.org. When you need one, ask your admin

There should be an traefik (https://gitlab.com/flipdot/hosting/traefik) running on the target host

## Deployment

connect to docker host via ssh

change dir to current spaceapi git folder and pull latest changes

`cd spaceapi`

`git pull`

Example run for the prod environment

`docker-compose build --pull`
`source prod.env; docker-compose up --force-recreate -d`
