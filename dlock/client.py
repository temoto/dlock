# coding: utf-8
"""Dlock client. Locks keys on server and executes command.
"""
import argparse
import gevent, gevent.socket
import gevent.baseserver # parse_address()
import os
import time

from . import util


API_VERSION = 1

command_line = argparse.ArgumentParser(description=__doc__)
command_line.add_argument('--connect', type=str, required=True,
                          help=u"Address of lock server.")
command_line.add_argument('--auto-key', type=str, default=False,
                          help=u"Prepend this string to full command including all arguments and use it as key. Appends to --keys.")
command_line.add_argument('--keys', type=str, default=[], nargs='*',
                          help=u"Keys to lock.")
command_line.add_argument('--lock-wait', type=float, default=0,
                          help=u"Lock acquire timeout.")
command_line.add_argument('--lock-release', type=float, default=False,
                          help=u"Tell server to hold lock for exactly this time. No implicit unlocking at disconnect is performed.")
command_line.add_argument('--fork-sleep', type=float, default=False,
                          help=u"Fork+sleep to hold locks at least this time.")
command_line.add_argument('--connect-timeout', type=float, default=10,
                          help=u"Maximum time to establish TCP connection with server.")
command_line.add_argument('--idle-timeout', type=float, default=10,
                          help=u"Maximum time to wait for beginning of server response.")
command_line.add_argument('--read-timeout', type=float, default=10,
                          help=u"Maximum time to receive a single message.")
command_line.add_argument('--send-timeout', type=float, default=10,
                          help=u"Maximum time to send a single message.")
command_line.add_argument('--max-message', type=int, default=128 * 1024,
                          help=u"Maximum message length accepted by client. If server sends more - we disconnect.")
command_line.add_argument('--read-buffer', type=int, default=-1,
                          help=u"Read buffer size for sockets.")
command_line.add_argument('--exec', dest='exec_', type=str, nargs="+",
                          help=u"Command to execute.")


def run(flags):
    _family, address = gevent.baseserver.parse_address(flags.connect)

    socket = util._missing
    with gevent.Timeout(flags.connect_timeout, False):
        socket = gevent.socket.create_connection(address)

    if socket is util._missing:
        util.log("error", "Could not connect to server")
        return 1

    socket.setsockopt(gevent.socket.SOL_SOCKET, gevent.socket.SO_KEEPALIVE, 1)
    try:
        socket.setsockopt(gevent.socket.SOL_TCP, gevent.socket.TCP_KEEPIDLE, 8)
        socket.setsockopt(gevent.socket.SOL_TCP, gevent.socket.TCP_KEEPINTVL, 5)
        socket.setsockopt(gevent.socket.SOL_TCP, gevent.socket.TCP_KEEPCNT, 2)
    finally:
        pass

    read_file = socket.makefile("rb", flags.read_buffer)

    if flags.auto_key != False:
        flags.keys.append(flags.auto_key + " ".join(flags.exec_))

    command = [API_VERSION, 'lock', flags.lock_wait, flags.lock_release, flags.keys]
    if not util.send_message(socket, command, flags.send_timeout):
        util.log("error", "Failed to send message within timeout")
        return 2

    response, error = util.read_message(read_file, flags.idle_timeout, flags.read_timeout, flags.max_message)
    if response is None and error is None:
        util.log("error", "No response")
        return 3

    if error is not None:
        util.log("error", "Error while reading response: {e}", e=repr(error))
        return 4

    if response[0] != "ok":
        util.log("error", "Did not acquire locks. Response: {r}", r=repr(response))
        return 5

    if flags.fork_sleep:
        if os.fork() == 0:
            # child
            # TODO: set alarm
            util.log("debug", "Exec command")
            os.execvp(flags.exec_[0], flags.exec_)
        else:
            # parent
            time.sleep(flags.fork_sleep)
    else:
        # TODO: set alarm
        util.log("debug", "Exec command")
        os.execvp(flags.exec_[0], flags.exec_)


def main():
    flags = command_line.parse_args()
    try:
        exit(run(flags))
    except KeyboardInterrupt:
        exit(1)
