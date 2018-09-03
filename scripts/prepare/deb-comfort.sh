#!/bin/bash

export DEBIAN_FRONTEND="noninteractive"
sudo -E apt-get -y -qq install progress mc powerline locate man || exit $?

sudo bash -c "cat > /etc/profile.d/powerline.sh" <<- EOS

if ! shopt -oq posix; then
  if [ -f /usr/share/powerline/bindings/bash/powerline.sh ]; then
    . /usr/share/powerline/bindings/bash/powerline.sh
  fi
  alias ll="ls -la --color"
  alias e="mcedit"
fi
EOS

# remove public access for home directories
sudo chmod o= /home/*/ /etc/skel