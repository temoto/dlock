# coding: utf-8
"""Dlock server. Listens for connections, manages locks.
"""
import argparse
from collections import namedtuple
import gevent, gevent.server
import time

from . import util


API_VERSION = 1

command_line = argparse.ArgumentParser(description=__doc__)
command_line.add_argument('--bind', type=str, required=True, nargs='+',
                          help=u"Bind to these address:port pairs.")
command_line.add_argument('--idle-timeout', type=float, default=60,
                          help=u"Disconnect clients without any activity within this time.")
command_line.add_argument('--read-timeout', type=float, default=10,
                          help=u"Maximum time to receive a single message.")
command_line.add_argument('--send-timeout', type=float, default=10,
                          help=u"Maximum time to send a single message.")
command_line.add_argument('--max-message', type=int, default=128 * 1024,
                          help=u"Maximum message length accepted by server. Clients trying to send more will be disconnected.")
command_line.add_argument('--read-buffer', type=int, default=-1,
                          help=u"Read buffer size for sockets.")


StateT = namedtuple("StateT", ('flags', 'key_locks', 'client_locks'))


def release(state, client_id, keys):
    client_locks = state.client_locks.setdefault(client_id, set())

    for key in keys:
        del state.key_locks[key]
        client_locks.remove(key)
    if not client_locks:
        del state.client_locks[client_id]


def command_lock(message, socket, state, client_id):
    """Returns tuple(release-on-disconnect, response)
    """
    command_time = time.time()

    if len(message) < 5:
        return (), ["error", 4, "Not enough arguments. Syntax: api_version lock wait release key [key...]"]

    acquire_timeout = message[2]
    release_timeout = message[3]
    keys = message[4]

    if not isinstance(acquire_timeout, (int, long, float)) or acquire_timeout < 0:
        return (), ["error", 100, "Acquire timeout must be int or float >= 0"]
    if release_timeout != False and (not isinstance(release_timeout, (int, long, float)) or release_timeout <= 0):
        return (), ["error", 101, "Release timeout must be False or int or float > 0"]

    if len(keys) > 100:
        return (), ["error", 102, "Attempt to lock too many keys"]
    if not all([isinstance(key, basestring) for key in keys]):
        return (), ["error", 103, "Keys must be simple strings"]

    # TODO: benchmark, optimize. Maybe intersect sets?
    held = [key for key in keys if key in state.key_locks]

    if held:
        util.log("debug", "client:{client_id} some requested keys are held: {held}", client_id=client_id, held=repr(held))
        # TODO: implement efficient wait on locked keys, detect deadlocks
        while held and time.time() - command_time < acquire_timeout:
            gevent.sleep(0.1)
            held = [key for key in keys if key in state.key_locks]

        # Still holding after timeout.
        if held:
            return (), ["error", 120, "Acquire timeout", held]

    util.log("debug", "client:{client_id} all requested keys are free", client_id=client_id)
    client_locks = state.client_locks.setdefault(client_id, set())
    for key in keys:
        state.key_locks[key] = (command_time, client_id, socket.fileno())
        client_locks.add(key)

    if release_timeout:
        gevent.spawn_later(release_timeout, release, state, client_id, keys)
        release_on_disconnect = ()
    else:
        release_on_disconnect = keys

    return release_on_disconnect, ["ok"]


def client_handler(socket, address, state):
    assert isinstance(address, tuple)
    client_id = "{0}:{1}".format(*address)
    util.log("debug", "client:{client_id} connected", client_id=client_id)
    client_locks = state.client_locks.setdefault(client_id, set())
    release_keys = set()

    socket.setsockopt(gevent.socket.SOL_SOCKET, gevent.socket.SO_KEEPALIVE, 1)
    try:
        socket.setsockopt(gevent.socket.SOL_TCP, gevent.socket.TCP_KEEPIDLE, 8)
        socket.setsockopt(gevent.socket.SOL_TCP, gevent.socket.TCP_KEEPINTVL, 5)
        socket.setsockopt(gevent.socket.SOL_TCP, gevent.socket.TCP_KEEPCNT, 2)
    finally:
        pass

    # Provides socket.recv buffer.
    read_file = socket.makefile("rb", state.flags.read_buffer)

    while True:
        try:
            message, error = util.read_message(read_file, state.flags.idle_timeout, state.flags.read_timeout, state.flags.max_message)
        except util.IdleTimeout:
            util.log("debug", "client:{client_id} idle timeout", client_id=client_id)
            break
        except util.ReadTimeout:
            util.log("error", "client:{client_id} read timeout", client_id=client_id)
            break
        except Exception as e:
            try:
                error_str = unicode(e).encode("utf-8", "replace")
            except Exception:
                error_str = "Input is so invalid - could not even format error message"

            if not isinstance(e, ValueError):
                util.log("error", "client:{client_id} unhandled error: {e}", client_id=client_id, e=error_str)
            break

        # Client closed connection
        if message is None and error is None:
            util.log("debug", "client:{client_id} remote closed", client_id=client_id)
            break

        if error is not None:
            util.log("error", "client:{client_id} {e}", client_id=client_id, e=error)
            util.send_message(socket, ["error", 1, error], state.flags.send_timeout)
            break

        # Have valid message at this point. Message is list, first two fields are API version and message type.
        util.log("debug", "client:{client_id} received message {message}", client_id=client_id, message=repr(message))
        if message[0] != API_VERSION:
            util.send_message(socket, ["error", 2, "Unsupported API version: {client_api_version}".format(client_api_version=message[0])], state.flags.send_timeout)
            break

        message_type = message[1]
        if message_type == 'lock':
            release_on_disconnect, response = command_lock(message, socket, state, client_id)
            release_keys.update(release_on_disconnect)
            if response is not None and not util.send_message(socket, response, state.flags.send_timeout):
                break
        elif message_type == 'ping':
            if not util.send_message(socket, ["ok"], state.flags.send_timeout):
                break
        else:
            util.send_message(socket, ["error", 3, "Unknown message type: '{message_type}'".format(message_type=message_type)], state.flags.send_timeout)
            break

    try:
        read_file.close()
    except IOError:
        pass
    try:
        socket.close()
    except IOError:
        pass

    # TODO: in finally
    util.log("debug", "client:{client_id} clean up {n} locks: {cl}", client_id=client_id, n=len(client_locks), cl=repr(client_locks))
    release(state, client_id, release_keys)


def run(flags):
    # TODO: use Murmur or random prefixed key dict to avoid
    # Hash collision security issue http://bugs.python.org/issue13703
    state = StateT(flags=flags, key_locks={}, client_locks={})

    servers = []
    handler = lambda sock, addr: client_handler(sock, addr, state)

    for address_str in flags.bind:
        # gevent.server.BaseServer parses address:port strings.
        server = gevent.server.StreamServer(address_str, handler)
        server.start()
        servers.append(server)

    util.log("debug", "Entering main loop")
    while True:
        gevent.sleep(1)


def main():
    flags = command_line.parse_args()
    try:
        exit(run(flags))
    except KeyboardInterrupt:
        exit(1)
