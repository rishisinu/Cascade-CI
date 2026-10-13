#!/usr/bin/env bash
set -euo pipefail

# ruleid: bash-curl-pipe-shell
curl -fsSL https://example.com/install.sh | bash
# ruleid: bash-curl-pipe-shell
wget -qO- https://example.com/install.sh | sudo sh
# ok: bash-curl-pipe-shell
curl -fsSL -o install.sh https://example.com/install.sh

# ruleid: bash-tls-verification-disabled
curl -sSLk https://internal.example/api
# ruleid: bash-tls-verification-disabled
wget --no-check-certificate https://internal.example/file
# ok: bash-tls-verification-disabled
curl -sSL --keepalive-time 10 https://internal.example/api

# ruleid: bash-chmod-world-writable
chmod 777 /srv/shared
# ruleid: bash-chmod-world-writable
chmod -R o+w /srv/shared
# ok: bash-chmod-world-writable
chmod 755 /srv/app
# ok: bash-chmod-world-writable
chmod u+w /srv/app

# ruleid: bash-eval-variable
eval "$USER_COMMAND"
# ok: bash-eval-variable
eval "$(ssh-agent -s)"
