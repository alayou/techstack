FROM ubuntu:20.04

ENV ServiceName="techstack"
ENV LaunchFlag=""

COPY build/$ServiceName /opt/$ServiceName/
COPY config.yml /opt/$ServiceName/

RUN ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo Asia/Shanghai > /etc/timezone \
    && apt update \
    && apt -y upgrade \
    && apt install -y tzdata \
    && apt -y clean \
    && apt -y autoclean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* \
    && chmod +x /opt/$ServiceName/$ServiceName

EXPOSE 8775

WORKDIR /opt/$ServiceName

CMD /opt/$ServiceName/$ServiceName $LaunchFlag
