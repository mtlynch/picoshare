FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update

RUN apt-get install -y \
    lxde \
    tightvncserver \
    novnc \
    net-tools \
    dbus-x11 \
    x11-xserver-utils

ENV USER=vncuser
ENV HOME=/home/vncuser

RUN useradd -m -s /bin/bash ${USER}
RUN mkdir -p /home/vncuser/.vnc
RUN chown -R ${USER}:${USER} /home/vncuser

USER ${USER}
WORKDIR ${HOME}

COPY startup.sh /home/vncuser/startup.sh

EXPOSE 5901
EXPOSE 6080

CMD ["/home/vncuser/startup.sh"]
