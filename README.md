# Moquette — MQTT Service Dispatcher

Moquette is to MQTT what inetd is to IP. Moquette listens for events from a MQTT broker and executes a process (event handler) found in its configuration directory if its name matches the event topic. The matching obeys to the MQTT topic rules. Slashes in the topic are replaced by commas (:).

For instance, the following file names will all match the topic `home/office/lamp/setOn`:

  * `home:office:lamp:setOn`
  * `home:office:+:setOn`
  * `home:office:#`
  * `#`

Event handler files must be at the root of the Moquette configuration directory and be executable. The default directory is `/etc/moquette.d` and can be changed using the `--conf` option. New files can be added/removed while Moquette is running, without the need to restart it.

When an event handler is executed, Moquette sends the event payload as first argument to the command and the event topic is set as the `$MQTT_TOPIC` environment variable. The message ID is also transmitted using the `$MQTT_MSGID` environment variable.

A command can send events back by writing to the file descriptor number 3. The Format is as follow:

    PUB <topic> <qos> <payload length>\n
    <payload>

For instance, to send "hello world" to the topic `example/somewhere` using bash:

    msg="hello world"
    echo -e "PUB example/somewhere 0 ${#msg}\n$msg" >&3

Moquette will wait as long as necessary for the process to finish its execution. This way it is possible to delay the response to an event, or send multiple events spreaded in time to implement a timer for instance.

## Handler Examples

The examples below are written in bash, but a handler can be written in any language.

### example:echo:+:in

    #!/bin/bash

    echo -e "PUB ${MQTT_TOPIC%*in}out 0 ${#1}\n$1" >&3

This handler responds to any event written on a matching topic, and sends back an event on the same topic by replacing `in` by `out`.

For instance, sending "hello world" to `example/echo/test/in` will send back "hello world" to the topic `example/echo/test/out`.

### example:timer:+:set

    #!/bin/bash

    n=$1
    while [ $n -ge 0 ]; do
        sleep 1
        echo -e "PUB ${MQTT_TOPIC%*set}tick 0 ${#n}\n$n" >&3
        ((n--))
    done

This handler sends a tick every second for `n` seconds when `n` is sent to a matching topic. Ticks are sent on the same topic with the last `set` component replaced by `tick`.

## Install

From source:

    go get -u github.com/rs/moquette

Using docker (assuming the `conf/` directory contains your handlers):

    docker run -it --rm -v $(pwd)/conf:/etc/moquette.d poitrus/moquette -broker tcp://<broker_ip>:1883

## Licenses

All source code is licensed under the [MIT License](https://raw.github.com/rs/moquette/master/LICENSE).